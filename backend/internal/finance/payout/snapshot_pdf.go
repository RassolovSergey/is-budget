package payout

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"isbudget/backend/internal/db/sqlc"

	"github.com/go-pdf/fpdf"
	"github.com/google/uuid"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
)

func (s *Service) saveSnapshotPDF(ctx context.Context, projectID uuid.UUID, calculatedAt time.Time, preview PreviewResponse) (SnapshotFile, error) {
	project, err := s.repo.GetProjectByID(ctx, projectID)
	if err != nil {
		return SnapshotFile{}, err
	}

	projectDir := filepath.Join(s.snapshotsDir, projectID.String())
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		return SnapshotFile{}, err
	}

	fileName := fmt.Sprintf("payout_snapshot_%s_%s.pdf", calculatedAt.Format("20060102_150405"), uuid.NewString()[:8])
	filePath := filepath.Join(projectDir, fileName)

	if err := renderPayoutSnapshotPDF(filePath, project, calculatedAt, preview); err != nil {
		return SnapshotFile{}, err
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return SnapshotFile{}, err
	}

	return SnapshotFile{
		FileName:  fileName,
		SavedAt:   calculatedAt,
		SizeBytes: info.Size(),
	}, nil
}

func renderPayoutSnapshotPDF(filePath string, project sqlc.Project, calculatedAt time.Time, preview PreviewResponse) error {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.AddUTF8FontFromBytes("Go", "", goregular.TTF)
	pdf.AddUTF8FontFromBytes("Go", "B", gobold.TTF)
	if err := pdf.Error(); err != nil {
		return err
	}
	pdf.SetMargins(12, 12, 12)
	pdf.SetAutoPageBreak(true, 12)
	pdf.AddPage()

	pageW, _ := pdf.GetPageSize()
	pdf.SetFillColor(18, 38, 83)
	pdf.Rect(0, 0, pageW, 28, "F")

	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Go", "B", 18)
	pdf.SetXY(12, 9)
	pdf.CellFormat(0, 8, "Снимок выплат", "", 1, "L", false, 0, "")

	pdf.SetFont("Go", "", 10)
	pdf.SetXY(12, 18)
	pdf.CellFormat(pageW-24, 5, "Дата формирования: "+calculatedAt.Format("02.01.2006 15:04 UTC"), "", 1, "R", false, 0, "")

	pdf.SetTextColor(24, 33, 56)
	pdf.SetY(34)
	pdf.SetFont("Go", "B", 16)
	pdf.CellFormat(0, 8, project.Name, "", 1, "L", false, 0, "")
	if strings.TrimSpace(project.Description) != "" {
		pdf.SetFont("Go", "", 10)
		pdf.SetTextColor(80, 94, 120)
		pdf.MultiCell(0, 5, project.Description, "", "L", false)
		pdf.SetTextColor(24, 33, 56)
	}

	distributed := 0.0
	for _, item := range preview.Items {
		distributed = round2(distributed + item.PayoutAmount)
	}
	remaining := round2(preview.Summary.AvailableForSharing - distributed)
	if remaining < 0 {
		remaining = 0
	}

	distributedPercent := 0.0
	remainingPercent := 0.0
	if preview.Summary.AvailableForSharing > 0 {
		distributedPercent = round2((distributed / preview.Summary.AvailableForSharing) * 100)
		remainingPercent = round2((remaining / preview.Summary.AvailableForSharing) * 100)
	}

	cardY := pdf.GetY() + 4
	cardW := 58.0
	cardH := 19.0
	gap := 6.0
	x1 := 12.0
	x2 := x1 + cardW + gap
	x3 := x2 + cardW + gap

	drawMetricCard(pdf, x1, cardY, cardW, cardH, "Чистая прибыль", formatMoney(preview.Summary.NetProfitToDate, project.CurrencyCode))
	drawMetricCard(pdf, x2, cardY, cardW, cardH, fmt.Sprintf("Налоги и комиссии (%.2f%%)", preview.Summary.TaxPercentTotal), formatMoney(preview.Summary.TaxTotal, project.CurrencyCode))
	drawMetricCard(pdf, x3, cardY, cardW, cardH, "Доступно к распределению", formatMoney(preview.Summary.AvailableForSharing, project.CurrencyCode))

	cardY2 := cardY + cardH + 4
	drawMetricCard(pdf, x1, cardY2, cardW, cardH, fmt.Sprintf("Распределено (%.2f%%)", distributedPercent), formatMoney(distributed, project.CurrencyCode))
	drawMetricCard(pdf, x2, cardY2, cardW, cardH, fmt.Sprintf("Остаток (%.2f%%)", remainingPercent), formatMoney(remaining, project.CurrencyCode))
	drawMetricCard(pdf, x3, cardY2, cardW, cardH, "ID проекта", truncateText(project.ID.String(), 20))

	pdf.SetY(cardY2 + cardH + 8)
	pdf.SetFont("Go", "B", 13)
	pdf.SetTextColor(24, 33, 56)
	pdf.CellFormat(0, 7, "Выплаты по участникам", "", 1, "L", false, 0, "")

	headers := []string{"Получатель", "Ответственность", "Уровень", "Процент", "Сумма"}
	widths := []float64{46, 58, 22, 24, 36}

	pdf.SetFillColor(24, 45, 96)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Go", "B", 10)
	for i := range headers {
		pdf.CellFormat(widths[i], 8, headers[i], "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	pdf.SetTextColor(30, 40, 64)
	pdf.SetFont("Go", "", 10)
	alt := false
	if len(preview.Items) == 0 {
		pdf.SetFillColor(247, 248, 252)
		pdf.CellFormat(186, 8, "Правила распределения не настроены.", "1", 1, "C", true, 0, "")
	} else {
		for _, item := range preview.Items {
			if alt {
				pdf.SetFillColor(247, 248, 252)
			} else {
				pdf.SetFillColor(255, 255, 255)
			}
			alt = !alt

			pdf.CellFormat(widths[0], 8, truncateText(item.RecipientName, 22), "1", 0, "L", true, 0, "")
			pdf.CellFormat(widths[1], 8, truncateText(item.Responsibility, 30), "1", 0, "L", true, 0, "")
			pdf.CellFormat(widths[2], 8, fmt.Sprintf("%d", item.Level), "1", 0, "C", true, 0, "")
			pdf.CellFormat(widths[3], 8, formatPercent(item.Percent), "1", 0, "R", true, 0, "")
			pdf.CellFormat(widths[4], 8, formatMoney(item.PayoutAmountConverted, item.CurrencyCode), "1", 1, "R", true, 0, "")
		}
	}

	pdf.Ln(4)
	pdf.SetFont("Go", "B", 11)
	pdf.SetTextColor(24, 33, 56)
	pdf.CellFormat(0, 7, fmt.Sprintf("Итого распределено: %s", formatMoney(distributed, project.CurrencyCode)), "", 1, "R", false, 0, "")
	pdf.CellFormat(0, 7, fmt.Sprintf("Нераспределенный остаток: %s (%s)", formatMoney(remaining, project.CurrencyCode), formatPercent(remainingPercent)), "", 1, "R", false, 0, "")

	pdf.Ln(2)
	pdf.SetFont("Go", "", 9)
	pdf.SetTextColor(100, 110, 130)
	pdf.MultiCell(0, 5, "Документ сформирован автоматически на основе текущих финансов проекта и правил распределения прибыли.", "", "L", false)

	return pdf.OutputFileAndClose(filePath)
}

func drawMetricCard(pdf *fpdf.Fpdf, x, y, w, h float64, title, value string) {
	pdf.SetFillColor(245, 247, 252)
	pdf.SetDrawColor(220, 226, 238)
	pdf.Rect(x, y, w, h, "DF")

	pdf.SetXY(x+2, y+2)
	pdf.SetFont("Go", "", 9)
	pdf.SetTextColor(88, 104, 132)
	pdf.CellFormat(w-4, 4.5, truncateText(title, 40), "", 1, "L", false, 0, "")

	pdf.SetX(x + 2)
	pdf.SetFont("Go", "B", 11)
	pdf.SetTextColor(24, 33, 56)
	pdf.CellFormat(w-4, 6, truncateText(value, 30), "", 1, "L", false, 0, "")
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

func formatPercent(value float64) string {
	v := round2(value)
	s := fmt.Sprintf("%.2f", v)
	s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	return s + "%"
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
