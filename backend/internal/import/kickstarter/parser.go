package kickstarter

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	ErrInvalidCSVStructure = errors.New("kickstarter csv has invalid structure")
	ErrNoEligibleRows      = errors.New("kickstarter csv contains no eligible collected rows")
)

type parsedImportRow struct {
	RowNumber        int32
	Status           string
	ErrorMessage     *string
	PledgedAt        *time.Time
	PledgedAmount    *float64
	BackerIdentifier *string
	RewardTier       *string
}

type parseResult struct {
	Rows             []parsedImportRow
	DailyAmounts     map[time.Time]float64
	RowsProcessed    int32
	RowsSucceeded    int32
	RowsFailed       int32
	RowsSkipped      int32
	TotalPledged     *float64
	BackerCount      *int32
	RewardTiersCount *int32
}

func parseKickstarterCSV(content []byte) (parseResult, error) {
	reader := csv.NewReader(bytes.NewReader(content))
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = false
	reader.Comma = sniffCSVDelimiter(content)

	rows, err := reader.ReadAll()
	if err != nil {
		return parseResult{}, fmt.Errorf("%w: %v", ErrInvalidCSVStructure, err)
	}
	if len(rows) < 2 {
		return parseResult{}, ErrInvalidCSVStructure
	}

	headers := rows[0]
	idxPledgeAmount := headerIndex(headers, []string{"Pledge Amount"})
	idxPledgedAt := headerIndex(headers, []string{"Pledged At (UTC)", "Pledged At"})
	idxRefunded := headerIndex(headers, []string{"Pledge Total Refunded"})
	idxPMRefunded := headerIndex(headers, []string{"Pledge Manager Total Refunded"})
	idxStatus := headerIndex(headers, []string{"Pledged Status"})
	idxBackerID := headerIndex(headers, []string{"Backer UID", "Backer ID", "Backer Number", "Backer Email"})
	idxRewardTier := headerIndex(headers, []string{"Reward Title", "Reward Name", "Reward", "Reward Tier"})
	if idxPledgeAmount < 0 || idxPledgedAt < 0 {
		return parseResult{}, ErrInvalidCSVStructure
	}

	out := parseResult{
		Rows:         make([]parsedImportRow, 0, len(rows)-1),
		DailyAmounts: make(map[time.Time]float64),
	}
	backers := make(map[string]struct{})
	rewardTiers := make(map[string]struct{})
	totalPledged := 0.0

	for i := 1; i < len(rows); i++ {
		row := rows[i]
		if isEmptyCSVRow(row) {
			continue
		}
		out.RowsProcessed++
		rowNumber := int32(i + 1)

		status := ""
		if idxStatus >= 0 {
			status = strings.ToLower(strings.TrimSpace(csvValue(row, idxStatus)))
		}
		if status != "" && status != "collected" {
			out.RowsSkipped++
			out.Rows = append(out.Rows, parsedImportRow{
				RowNumber: rowNumber,
				Status:    "skipped",
			})
			continue
		}

		pledgedAt, err := parseKickstarterDate(csvValue(row, idxPledgedAt))
		if err != nil {
			msg := "invalid pledged date"
			out.RowsFailed++
			out.Rows = append(out.Rows, parsedImportRow{RowNumber: rowNumber, Status: "failed", ErrorMessage: &msg})
			continue
		}
		pledge, _, ok := parseMoneyWithCurrency(csvValue(row, idxPledgeAmount))
		if !ok {
			msg := "invalid pledge amount"
			out.RowsFailed++
			out.Rows = append(out.Rows, parsedImportRow{RowNumber: rowNumber, Status: "failed", ErrorMessage: &msg})
			continue
		}
		refunded, _, ok := parseOptionalMoney(csvValue(row, idxRefunded))
		if !ok {
			msg := "invalid refunded amount"
			out.RowsFailed++
			out.Rows = append(out.Rows, parsedImportRow{RowNumber: rowNumber, Status: "failed", ErrorMessage: &msg})
			continue
		}
		pmRefunded, _, ok := parseOptionalMoney(csvValue(row, idxPMRefunded))
		if !ok {
			msg := "invalid pledge manager refunded amount"
			out.RowsFailed++
			out.Rows = append(out.Rows, parsedImportRow{RowNumber: rowNumber, Status: "failed", ErrorMessage: &msg})
			continue
		}

		netAmount := round2(pledge - refunded - pmRefunded)
		day := normalizeDate(pledgedAt)
		out.DailyAmounts[day] = round2(out.DailyAmounts[day] + netAmount)
		out.RowsSucceeded++
		totalPledged = round2(totalPledged + netAmount)

		var backerID *string
		if value := strings.TrimSpace(csvValue(row, idxBackerID)); value != "" {
			backerID = &value
			backers[value] = struct{}{}
		}

		var rewardTier *string
		if value := strings.TrimSpace(csvValue(row, idxRewardTier)); value != "" {
			rewardTier = &value
			rewardTiers[value] = struct{}{}
		}

		amount := netAmount
		out.Rows = append(out.Rows, parsedImportRow{
			RowNumber:        rowNumber,
			Status:           "success",
			PledgedAt:        &day,
			PledgedAmount:    &amount,
			BackerIdentifier: backerID,
			RewardTier:       rewardTier,
		})
	}

	if out.RowsSucceeded == 0 {
		return out, ErrNoEligibleRows
	}
	if out.RowsProcessed > 0 {
		out.TotalPledged = &totalPledged
	}
	backerCount := int32(len(backers))
	if backerCount == 0 {
		backerCount = out.RowsSucceeded
	}
	rewardCount := int32(len(rewardTiers))
	out.BackerCount = &backerCount
	out.RewardTiersCount = &rewardCount
	return out, nil
}

func sniffCSVDelimiter(content []byte) rune {
	firstLine := string(content)
	if idx := strings.IndexByte(firstLine, '\n'); idx >= 0 {
		firstLine = firstLine[:idx]
	}
	if strings.Count(firstLine, ";") > strings.Count(firstLine, ",") {
		return ';'
	}
	return ','
}

func headerIndex(headers []string, candidates []string) int {
	normalized := make([]string, 0, len(headers))
	for _, header := range headers {
		normalized = append(normalized, strings.ToLower(strings.TrimSpace(header)))
	}
	for _, candidate := range candidates {
		target := strings.ToLower(strings.TrimSpace(candidate))
		for idx, header := range normalized {
			if header == target {
				return idx
			}
		}
	}
	return -1
}

func csvValue(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return row[idx]
}

func isEmptyCSVRow(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

func parseKickstarterDate(value string) (time.Time, error) {
	raw := strings.Trim(strings.TrimSpace(value), `"`)
	if raw == "" {
		return time.Time{}, errors.New("empty date")
	}
	layouts := []string{
		"2006-01-02 15:04:05 MST",
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006/01/02 15:04:05",
		"2006-01-02",
		"2006/01/02",
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return parsed.UTC(), nil
		}
	}
	if len(raw) >= 10 {
		prefix := raw[:10]
		if parsed, err := time.Parse("2006-01-02", strings.ReplaceAll(strings.ReplaceAll(prefix, "/", "-"), ".", "-")); err == nil {
			return parsed.UTC(), nil
		}
	}
	return time.Time{}, errors.New("unsupported date")
}

func parseOptionalMoney(value string) (float64, string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, "", true
	}
	return parseMoneyWithCurrency(trimmed)
}

func parseMoneyWithCurrency(value string) (float64, string, bool) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return 0, "", false
	}
	upper := strings.ToUpper(raw)
	currency := ""
	switch {
	case strings.Contains(upper, "EUR") || strings.Contains(raw, "€"):
		currency = "EUR"
	case strings.Contains(upper, "USD") || strings.Contains(raw, "$"):
		currency = "USD"
	case strings.Contains(upper, "RUB") || strings.Contains(raw, "₽"):
		currency = "RUB"
	}

	cleaned := strings.ReplaceAll(raw, " ", "")
	var decimalSep rune
	if strings.Contains(cleaned, ",") && strings.Contains(cleaned, ".") {
		if strings.LastIndex(cleaned, ",") > strings.LastIndex(cleaned, ".") {
			decimalSep = ','
		} else {
			decimalSep = '.'
		}
	} else if strings.Contains(cleaned, ",") {
		decimalSep = ','
	} else {
		decimalSep = '.'
	}

	b := strings.Builder{}
	for _, ch := range cleaned {
		if (ch >= '0' && ch <= '9') || ch == '-' {
			b.WriteRune(ch)
			continue
		}
		if ch == '.' || ch == ',' {
			if ch == decimalSep {
				b.WriteRune('.')
			}
		}
	}
	normalized := b.String()
	if normalized == "" || normalized == "-" || normalized == "." || normalized == "-." {
		return 0, currency, false
	}
	amount, err := strconvParseFloat(normalized)
	if err != nil || math.IsNaN(amount) || math.IsInf(amount, 0) {
		return 0, currency, false
	}
	return round2(amount), currency, true
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func sortedImportDates(values map[time.Time]float64) []time.Time {
	dates := make([]time.Time, 0, len(values))
	for date := range values {
		dates = append(dates, date)
	}
	sort.Slice(dates, func(i, j int) bool { return dates[i].Before(dates[j]) })
	return dates
}

func strconvParseFloat(value string) (float64, error) {
	return strconv.ParseFloat(value, 64)
}
