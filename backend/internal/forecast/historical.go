package forecast

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

type HistoricalCampaignExpense struct {
	Category string  `json:"category"`
	Amount   float64 `json:"amount"`
}

type HistoricalCampaignDay struct {
	DayNumber    int32                       `json:"day_number"`
	IncomeAmount float64                     `json:"income_amount"`
	Expenses     []HistoricalCampaignExpense `json:"expenses"`
}

type HistoricalCampaign struct {
	ID           uuid.UUID               `json:"id"`
	ProjectID    *uuid.UUID              `json:"project_id,omitempty"`
	OwnerUserID  uuid.UUID               `json:"owner_user_id"`
	Category     string                  `json:"category"`
	Source       string                  `json:"source,omitempty"`
	Name         string                  `json:"name"`
	DurationDays int32                   `json:"duration_days"`
	CurrencyCode string                  `json:"currency_code"`
	StartedAt    string                  `json:"started_at"`
	EndedAt      string                  `json:"ended_at"`
	CreatedAt    time.Time               `json:"created_at"`
	UpdatedAt    time.Time               `json:"updated_at"`
	Days         []HistoricalCampaignDay `json:"days"`
}

type UpsertHistoricalCampaignRequest struct {
	Name         string                  `json:"name"`
	Category     string                  `json:"category"`
	DurationDays int32                   `json:"duration_days"`
	CurrencyCode string                  `json:"currency_code"`
	StartedAt    string                  `json:"started_at"`
	EndedAt      string                  `json:"ended_at"`
	Days         []HistoricalCampaignDay `json:"days"`
}

var (
	ErrInvalidHistoricalCampaignName         = errors.New("name must be between 2 and 120 characters")
	ErrInvalidHistoricalCampaignCategory     = errors.New("category must be between 2 and 80 characters")
	ErrInvalidHistoricalCampaignDuration     = errors.New("duration_days must be between 3 and 60")
	ErrInvalidHistoricalCampaignCurrency     = errors.New("currency_code must be one of EUR, USD, RUB")
	ErrInvalidHistoricalCampaignDays         = errors.New("days must contain at least one item")
	ErrInvalidHistoricalCampaignDayNumber    = errors.New("day_number must be within campaign duration and >= 1")
	ErrInvalidHistoricalCampaignDayDuplicate = errors.New("day_number values must be unique")
	ErrInvalidHistoricalCampaignIncome       = errors.New("income_amount must be a finite number")
	ErrInvalidHistoricalCampaignExpense      = errors.New("expense amount must be >= 0 and category must be non-empty")
	ErrInvalidHistoricalCampaignStartDate    = errors.New("started_at must be in YYYY-MM-DD format")
	ErrInvalidHistoricalCampaignEndDate      = errors.New("ended_at must be in YYYY-MM-DD format")
	ErrInvalidHistoricalCampaignPeriod       = errors.New("ended_at must be greater than or equal to started_at")
	ErrInvalidHistoricalCampaignDateDuration = errors.New("campaign period must match duration_days")
)

func normalizeCurrency(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

func isValidCurrency(code string) bool {
	switch normalizeCurrency(code) {
	case "EUR", "USD", "RUB":
		return true
	default:
		return false
	}
}

func normalizeHistoricalCampaignInput(req UpsertHistoricalCampaignRequest) (UpsertHistoricalCampaignRequest, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Category = strings.TrimSpace(req.Category)
	req.CurrencyCode = normalizeCurrency(req.CurrencyCode)
	req.StartedAt = strings.TrimSpace(req.StartedAt)
	req.EndedAt = strings.TrimSpace(req.EndedAt)
	if req.CurrencyCode == "" {
		req.CurrencyCode = "EUR"
	}

	if len(req.Name) < 2 || len(req.Name) > 120 {
		return UpsertHistoricalCampaignRequest{}, ErrInvalidHistoricalCampaignName
	}
	if len(req.Category) < 2 || len(req.Category) > 80 {
		return UpsertHistoricalCampaignRequest{}, ErrInvalidHistoricalCampaignCategory
	}
	if req.DurationDays < 3 || req.DurationDays > 60 {
		return UpsertHistoricalCampaignRequest{}, ErrInvalidHistoricalCampaignDuration
	}
	if !isValidCurrency(req.CurrencyCode) {
		return UpsertHistoricalCampaignRequest{}, ErrInvalidHistoricalCampaignCurrency
	}
	startDate, err := parseDate(req.StartedAt)
	if err != nil {
		return UpsertHistoricalCampaignRequest{}, ErrInvalidHistoricalCampaignStartDate
	}
	endDate, err := parseDate(req.EndedAt)
	if err != nil {
		return UpsertHistoricalCampaignRequest{}, ErrInvalidHistoricalCampaignEndDate
	}
	if endDate.Before(startDate) {
		return UpsertHistoricalCampaignRequest{}, ErrInvalidHistoricalCampaignPeriod
	}
	if int32(dateDiffDaysInclusive(startDate, endDate)) != req.DurationDays {
		return UpsertHistoricalCampaignRequest{}, ErrInvalidHistoricalCampaignDateDuration
	}
	if len(req.Days) == 0 {
		return UpsertHistoricalCampaignRequest{}, ErrInvalidHistoricalCampaignDays
	}

	usedDays := make(map[int32]struct{}, len(req.Days))
	for i := range req.Days {
		day := &req.Days[i]
		if day.DayNumber < 1 || day.DayNumber > req.DurationDays {
			return UpsertHistoricalCampaignRequest{}, fmt.Errorf("%w: %d", ErrInvalidHistoricalCampaignDayNumber, day.DayNumber)
		}
		if _, exists := usedDays[day.DayNumber]; exists {
			return UpsertHistoricalCampaignRequest{}, fmt.Errorf("%w: %d", ErrInvalidHistoricalCampaignDayDuplicate, day.DayNumber)
		}
		usedDays[day.DayNumber] = struct{}{}
		if !isFinite(day.IncomeAmount) {
			return UpsertHistoricalCampaignRequest{}, fmt.Errorf("%w at day %d", ErrInvalidHistoricalCampaignIncome, day.DayNumber)
		}
		for j := range day.Expenses {
			expense := &day.Expenses[j]
			expense.Category = strings.TrimSpace(expense.Category)
			if expense.Category == "" || expense.Amount < 0 {
				return UpsertHistoricalCampaignRequest{}, fmt.Errorf("%w at day %d", ErrInvalidHistoricalCampaignExpense, day.DayNumber)
			}
		}
		sort.SliceStable(day.Expenses, func(a, b int) bool {
			return strings.ToLower(day.Expenses[a].Category) < strings.ToLower(day.Expenses[b].Category)
		})
	}

	sort.SliceStable(req.Days, func(i, j int) bool {
		return req.Days[i].DayNumber < req.Days[j].DayNumber
	})
	req.StartedAt = startDate.Format("2006-01-02")
	req.EndedAt = endDate.Format("2006-01-02")
	return req, nil
}

func parseDate(value string) (time.Time, error) {
	return time.Parse("2006-01-02", value)
}

func dateDiffDaysInclusive(start, end time.Time) int {
	startUTC := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
	endUTC := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, time.UTC)
	return int(endUTC.Sub(startUTC).Hours()/24) + 1
}

func isAdCategory(category string) bool {
	cat := strings.ToLower(strings.TrimSpace(category))
	switch cat {
	case "ads", "ad", "advertising", "marketing", "реклама", "маркетинг", "рекламный бюджет":
		return true
	default:
		return false
	}
}

func isFinite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}
