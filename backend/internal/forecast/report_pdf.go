package forecast

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"isbudget/backend/internal/db/sqlc"

	"github.com/go-pdf/fpdf"
	"github.com/google/uuid"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
)

type forecastReportRow struct {
	DateISO    string
	DailyBad   float64
	DailyBase  float64
	DailyGood  float64
	TotalBad   float64
	TotalBase  float64
	TotalGood  float64
	ActualBase *float64
}

type forecastReportData struct {
	project      sqlc.GetProjectByIDRow
	runResult    RunResult
	actualTotal  float64
	rows         []forecastReportRow
	actualSeries []struct {
		DateISO string
		Total   float64
	}
}

func (s *Service) BuildRunReportPDF(ctx context.Context, projectID, runID uuid.UUID) ([]byte, string, error) {
	reportData, err := s.buildForecastReportData(ctx, projectID, runID)
	if err != nil {
		return nil, "", err
	}

	pdf, err := renderForecastReportPDF(reportData.project, reportData.runResult.Run, reportData.actualTotal, reportData.rows, reportData.actualSeries)
	if err != nil {
		return nil, "", err
	}
	fileName := fmt.Sprintf("forecast_report_%s_%s.pdf", reportData.runResult.Run.CreatedAt.UTC().Format("20060102_150405"), reportData.runResult.Run.ID.String()[:8])
	return pdf, fileName, nil
}

func (s *Service) buildForecastReportData(ctx context.Context, projectID, runID uuid.UUID) (forecastReportData, error) {
	project, err := s.repo.GetProjectByID(ctx, projectID)
	if err != nil {
		return forecastReportData{}, err
	}
	runResult, err := s.GetByID(ctx, projectID, runID)
	if err != nil {
		return forecastReportData{}, err
	}

	incomeRows, err := s.repo.ListIncomeDailyByProject(ctx, projectID)
	if err != nil {
		return forecastReportData{}, err
	}

	startDate := project.CreatedAt.UTC()
	if project.ActivatedAt != nil {
		startDate = project.ActivatedAt.UTC()
	}
	startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.UTC)
	durationDays := int(project.DurationDays)
	if durationDays < 1 {
		durationDays = 1
	}
	endDate := startDate.AddDate(0, 0, durationDays-1)

	incomeByDate := make(map[string]float64)
	for _, row := range incomeRows {
		date := time.Date(row.IncomeDate.Year(), row.IncomeDate.Month(), row.IncomeDate.Day(), 0, 0, 0, 0, time.UTC)
		if date.Before(startDate) || date.After(endDate) {
			continue
		}
		key := date.Format("2006-01-02")
		incomeByDate[key] = round2(incomeByDate[key] + row.Amount)
	}

	actualDates := make([]string, 0, len(incomeByDate))
	for date := range incomeByDate {
		actualDates = append(actualDates, date)
	}
	sort.Strings(actualDates)
	lastActualDate := ""
	if len(actualDates) > 0 {
		lastActualDate = actualDates[len(actualDates)-1]
	}

	actualCumulative := make([]struct {
		DateISO string
		Total   float64
	}, 0, durationDays)
	runningActual := 0.0
	for day := startDate; !day.After(endDate); day = day.AddDate(0, 0, 1) {
		iso := day.Format("2006-01-02")
		if amount, ok := incomeByDate[iso]; ok {
			runningActual = round2(runningActual + amount)
		} else if iso <= lastActualDate {
			runningActual = round2(runningActual)
		}
		if lastActualDate != "" && iso <= lastActualDate {
			value := runningActual
			actualCumulative = append(actualCumulative, struct {
				DateISO string
				Total   float64
			}{DateISO: iso, Total: value})
		}
	}

	forecastPoints := append([]sqlc.ForecastPoint(nil), runResult.Points...)
	sort.SliceStable(forecastPoints, func(i, j int) bool {
		return forecastPoints[i].ForecastDate.Before(forecastPoints[j].ForecastDate)
	})

	rows := make([]forecastReportRow, 0, len(forecastPoints))
	cumBad := runningActual
	cumBase := runningActual
	cumGood := runningActual
	for _, point := range forecastPoints {
		dateISO := point.ForecastDate.UTC().Format("2006-01-02")
		if lastActualDate != "" && dateISO <= lastActualDate {
			continue
		}
		cumBad = round2(cumBad + point.P10)
		cumBase = round2(cumBase + point.Yhat)
		cumGood = round2(cumGood + point.P90)
		rows = append(rows, forecastReportRow{
			DateISO:   dateISO,
			DailyBad:  round2(point.P10),
			DailyBase: round2(point.Yhat),
			DailyGood: round2(point.P90),
			TotalBad:  cumBad,
			TotalBase: cumBase,
			TotalGood: cumGood,
		})
	}

	return forecastReportData{
		project:      project,
		runResult:    runResult,
		actualTotal:  runningActual,
		rows:         rows,
		actualSeries: actualCumulative,
	}, nil
}

func renderForecastReportPDF(
	project sqlc.GetProjectByIDRow,
	run sqlc.ForecastRun,
	actualTotal float64,
	rows []forecastReportRow,
	actualSeries []struct {
		DateISO string
		Total   float64
	},
) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.AddUTF8FontFromBytes("Go", "", goregular.TTF)
	pdf.AddUTF8FontFromBytes("Go", "B", gobold.TTF)
	if err := pdf.Error(); err != nil {
		return nil, err
	}
	pdf.SetMargins(12, 12, 12)
	pdf.SetAutoPageBreak(true, 12)
	pdf.AddPage()

	currency := strings.ToUpper(strings.TrimSpace(project.CurrencyCode))
	if currency == "" {
		currency = "EUR"
	}
	totalBad := actualTotal
	totalBase := actualTotal
	totalGood := actualTotal
	for _, row := range rows {
		totalBad = row.TotalBad
		totalBase = row.TotalBase
		totalGood = row.TotalGood
	}

	pageW, _ := pdf.GetPageSize()
	pdf.SetFillColor(18, 38, 83)
	pdf.Rect(0, 0, pageW, 28, "F")

	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Go", "B", 18)
	pdf.SetXY(12, 9)
	pdf.CellFormat(0, 8, "Отчёт по прогнозу", "", 1, "L", false, 0, "")

	pdf.SetFont("Go", "", 10)
	pdf.SetXY(12, 18)
	pdf.CellFormat(pageW-24, 5, "Запуск: "+run.CreatedAt.UTC().Format("02.01.2006 15:04 UTC"), "", 1, "R", false, 0, "")

	pdf.SetTextColor(24, 33, 56)
	pdf.SetY(34)
	pdf.SetFont("Go", "B", 16)
	pdf.CellFormat(0, 8, project.Name, "", 1, "L", false, 0, "")

	pdf.SetFont("Go", "", 10)
	pdf.SetTextColor(80, 94, 120)
	pdf.CellFormat(0, 5, "Модель: "+run.ModelVersion, "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 5, "Валюта: "+currency, "", 1, "L", false, 0, "")
	pdf.SetTextColor(24, 33, 56)

	cardY := pdf.GetY() + 4
	cardW := 58.0
	cardH := 18.0
	gap := 6.0
	drawMetricCard(pdf, 12, cardY, cardW, cardH, "Факт на сегодня", formatMoney(actualTotal, currency))
	drawMetricCard(pdf, 12+cardW+gap, cardY, cardW, cardH, "Плохой итог", formatMoney(totalBad, currency))
	drawMetricCard(pdf, 12+(cardW+gap)*2, cardY, cardW, cardH, "Средний итог", formatMoney(totalBase, currency))
	drawMetricCard(pdf, 12+(cardW+gap)*3, cardY, cardW, cardH, "Хороший итог", formatMoney(totalGood, currency))

	chartY := cardY + cardH + 7
	drawForecastChart(pdf, 12, chartY, pageW-24, 74, actualSeries, rows)
	pdf.SetY(chartY + 80)

	pdf.SetFont("Go", "B", 12)
	pdf.SetTextColor(24, 33, 56)
	pdf.CellFormat(0, 6, "Прогноз по дням", "", 1, "L", false, 0, "")
	pdf.Ln(1)

	headers := []string{"Дата", "Плохой", "Средний", "Хороший", "Нак. плохой", "Нак. средний", "Нак. хороший"}
	widths := []float64{23, 24, 24, 24, 32, 32, 32}

	pdf.SetFillColor(24, 45, 96)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Go", "B", 9)
	for i := range headers {
		pdf.CellFormat(widths[i], 8, headers[i], "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	pdf.SetFont("Go", "", 8.7)
	pdf.SetTextColor(30, 40, 64)
	alt := false
	if len(rows) == 0 {
		pdf.SetFillColor(247, 248, 252)
		pdf.CellFormat(191, 8, "Нет прогнозных точек для отчёта.", "1", 1, "C", true, 0, "")
	} else {
		for _, row := range rows {
			if pdf.GetY() > 274 {
				pdf.AddPage()
				pdf.SetFillColor(24, 45, 96)
				pdf.SetTextColor(255, 255, 255)
				pdf.SetFont("Go", "B", 9)
				for i := range headers {
					pdf.CellFormat(widths[i], 8, headers[i], "1", 0, "C", true, 0, "")
				}
				pdf.Ln(-1)
				pdf.SetFont("Go", "", 8.7)
				pdf.SetTextColor(30, 40, 64)
			}

			if alt {
				pdf.SetFillColor(247, 248, 252)
			} else {
				pdf.SetFillColor(255, 255, 255)
			}
			alt = !alt

			values := []string{
				row.DateISO,
				fmtNum(row.DailyBad),
				fmtNum(row.DailyBase),
				fmtNum(row.DailyGood),
				fmtNum(row.TotalBad),
				fmtNum(row.TotalBase),
				fmtNum(row.TotalGood),
			}
			for i := range values {
				pdf.CellFormat(widths[i], 8, values[i], "1", 0, "R", true, 0, "")
			}
			pdf.Ln(-1)
		}
	}

	var out bytes.Buffer
	if err := pdf.Output(&out); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func drawForecastChart(
	pdf *fpdf.Fpdf,
	x, y, w, h float64,
	actualSeries []struct {
		DateISO string
		Total   float64
	},
	forecastRows []forecastReportRow,
) {
	pdf.SetFillColor(245, 247, 252)
	pdf.SetDrawColor(220, 226, 238)
	pdf.Rect(x, y, w, h, "DF")

	plotX := x + 8
	plotY := y + 6
	plotW := w - 14
	plotH := h - 14

	points := make([]float64, 0, len(actualSeries)+len(forecastRows))
	for _, row := range actualSeries {
		points = append(points, row.Total)
	}
	for _, row := range forecastRows {
		points = append(points, row.TotalBase)
	}
	if len(points) == 0 {
		return
	}

	maxValue := 0.0
	for _, value := range points {
		if value > maxValue {
			maxValue = value
		}
	}
	if maxValue <= 0 {
		maxValue = 1
	}
	maxValue *= 1.08

	pdf.SetDrawColor(210, 218, 230)
	pdf.Line(plotX, plotY+plotH, plotX+plotW, plotY+plotH)
	pdf.Line(plotX, plotY, plotX, plotY+plotH)

	totalCount := len(actualSeries) + len(forecastRows)
	if totalCount < 2 {
		totalCount = 2
	}
	stepX := plotW / float64(totalCount-1)
	toY := func(value float64) float64 {
		ratio := value / maxValue
		ratio = math.Min(1, math.Max(0, ratio))
		return plotY + plotH - ratio*plotH
	}

	if len(actualSeries) > 1 {
		pdf.SetDrawColor(34, 197, 94)
		for i := 1; i < len(actualSeries); i++ {
			x1 := plotX + float64(i-1)*stepX
			y1 := toY(actualSeries[i-1].Total)
			x2 := plotX + float64(i)*stepX
			y2 := toY(actualSeries[i].Total)
			pdf.Line(x1, y1, x2, y2)
		}
	}

	if len(forecastRows) > 0 {
		pdf.SetDrawColor(245, 158, 11)
		startOffset := len(actualSeries)
		if startOffset > 0 {
			startOffset--
		}
		prevX := plotX + float64(startOffset)*stepX
		prevY := plotY + plotH
		if len(actualSeries) > 0 {
			prevY = toY(actualSeries[len(actualSeries)-1].Total)
		}
		for i := 0; i < len(forecastRows); i++ {
			x2 := plotX + float64(startOffset+i+1)*stepX
			y2 := toY(forecastRows[i].TotalBase)
			pdf.Line(prevX, prevY, x2, y2)
			prevX = x2
			prevY = y2
		}
	}

	pdf.SetFont("Go", "", 8)
	pdf.SetTextColor(90, 104, 125)
	pdf.Text(plotX, y+h-2, "Факт")
	pdf.SetTextColor(245, 158, 11)
	pdf.Text(plotX+24, y+h-2, "Прогноз")
	pdf.SetTextColor(90, 104, 125)
}

func drawMetricCard(pdf *fpdf.Fpdf, x, y, w, h float64, title, value string) {
	pdf.SetFillColor(245, 247, 252)
	pdf.SetDrawColor(220, 226, 238)
	pdf.Rect(x, y, w, h, "DF")

	pdf.SetXY(x+2, y+2)
	pdf.SetFont("Go", "", 8)
	pdf.SetTextColor(88, 104, 132)
	pdf.CellFormat(w-4, 4.5, truncateText(title, 36), "", 1, "L", false, 0, "")

	pdf.SetX(x + 2)
	pdf.SetFont("Go", "B", 10)
	pdf.SetTextColor(24, 33, 56)
	pdf.CellFormat(w-4, 6, truncateText(value, 24), "", 1, "L", false, 0, "")
}

func formatMoney(amount float64, currencyCode string) string {
	return fmt.Sprintf("%.2f %s", round2(amount), currencySymbol(currencyCode))
}

func currencySymbol(currencyCode string) string {
	switch strings.ToUpper(currencyCode) {
	case "USD":
		return "$"
	case "RUB":
		return "₽"
	default:
		return "€"
	}
}

func truncateText(value string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	r := []rune(strings.TrimSpace(value))
	if len(r) <= maxRunes {
		return string(r)
	}
	if maxRunes == 1 {
		return string(r[:1])
	}
	return string(r[:maxRunes-1]) + "."
}

func fmtNum(value float64) string {
	return fmt.Sprintf("%.2f", round2(value))
}
