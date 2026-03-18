package forecast

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

type HistoryPoint struct {
	Date   string  `json:"date"`
	Amount float64 `json:"amount"`
}

type AnalyticsHistoricalExpense struct {
	Category string  `json:"category"`
	Amount   float64 `json:"amount"`
}

type AnalyticsHistoricalDay struct {
	DayNumber    int                          `json:"day_number"`
	IncomeAmount float64                      `json:"income_amount"`
	Expenses     []AnalyticsHistoricalExpense `json:"expenses"`
}

type AnalyticsHistoricalCampaign struct {
	Name         string                   `json:"name"`
	DurationDays int                      `json:"duration_days"`
	CurrencyCode string                   `json:"currency_code"`
	StartedAt    string                   `json:"started_at,omitempty"`
	EndedAt      string                   `json:"ended_at,omitempty"`
	Days         []AnalyticsHistoricalDay `json:"days"`
}

type AnalyticsRequest struct {
	ProjectID            string                        `json:"project_id"`
	HorizonDays          int                           `json:"horizon_days"`
	History              []HistoryPoint                `json:"history"`
	HistoricalCampaigns  []AnalyticsHistoricalCampaign `json:"historical_campaigns,omitempty"`
	CurrentIncomeToDate  float64                       `json:"current_income_to_date"`
	CurrentAdSpendToDate float64                       `json:"current_ad_spend_to_date"`
	ModelVersion         string                        `json:"model_version"`
	Seed                 int64                         `json:"seed"`
}

type AnalyticsPoint struct {
	Date string  `json:"date"`
	Yhat float64 `json:"yhat"`
	P10  float64 `json:"p10"`
	P90  float64 `json:"p90"`
}

type AnalyticsResponse struct {
	ModelVersion string           `json:"model_version"`
	HorizonDays  int              `json:"horizon_days"`
	Forecast     []AnalyticsPoint `json:"forecast"`
}

type AnalyticsClient interface {
	Forecast(ctx context.Context, req AnalyticsRequest) (AnalyticsResponse, error)
}

type HTTPAnalyticsClient struct {
	baseURL string
	http    *http.Client
}

func NewHTTPAnalyticsClient(baseURL string) *HTTPAnalyticsClient {
	return &HTTPAnalyticsClient{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		http:    &http.Client{Timeout: 20 * time.Second},
	}
}

func (c *HTTPAnalyticsClient) Forecast(ctx context.Context, req AnalyticsRequest) (AnalyticsResponse, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return AnalyticsResponse{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/forecast", bytes.NewReader(payload))
	if err != nil {
		return AnalyticsResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return AnalyticsResponse{}, fmt.Errorf("%w: %v", ErrAnalyticsTimeout, err)
		}
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return AnalyticsResponse{}, fmt.Errorf("%w: %v", ErrAnalyticsTimeout, err)
		}
		return AnalyticsResponse{}, fmt.Errorf("%w: %v", ErrAnalyticsUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return AnalyticsResponse{}, fmt.Errorf("%w: status %d", ErrAnalyticsUnavailable, resp.StatusCode)
	}

	var out AnalyticsResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return AnalyticsResponse{}, fmt.Errorf("%w: %v", ErrAnalyticsInvalidResponse, err)
	}
	return out, nil
}
