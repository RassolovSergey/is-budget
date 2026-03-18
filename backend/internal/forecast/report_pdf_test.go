package forecast

import (
	"bytes"
	"testing"
	"time"

	"isbudget/backend/internal/db/sqlc"

	"github.com/google/uuid"
)

func TestRenderForecastReportPDFProducesValidPDF(t *testing.T) {
	project := sqlc.GetProjectByIDRow{
		ID:           uuid.New(),
		Name:         "Demo Project",
		CurrencyCode: "EUR",
		DurationDays: 10,
	}
	run := sqlc.ForecastRun{
		ID:           uuid.New(),
		ProjectID:    project.ID,
		CreatedAt:    time.Date(2026, 3, 1, 10, 30, 0, 0, time.UTC),
		HorizonDays:  7,
		ModelVersion: "hybrid-v2",
	}
	rows := []forecastReportRow{
		{
			DateISO:   "2026-03-02",
			DailyBad:  100,
			DailyBase: 150,
			DailyGood: 190,
			TotalBad:  900,
			TotalBase: 950,
			TotalGood: 990,
		},
	}
	actualSeries := []struct {
		DateISO string
		Total   float64
	}{
		{DateISO: "2026-03-01", Total: 800},
	}

	pdfBytes, err := renderForecastReportPDF(project, run, 800, rows, actualSeries)
	if err != nil {
		t.Fatalf("renderForecastReportPDF returned error: %v", err)
	}
	if len(pdfBytes) < 2000 {
		t.Fatalf("expected non-trivial pdf size, got %d bytes", len(pdfBytes))
	}
	if !bytes.HasPrefix(pdfBytes, []byte("%PDF")) {
		t.Fatalf("expected valid PDF header, got: %q", string(pdfBytes[:4]))
	}
}
