package fx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"isbudget/backend/internal/db/sqlc"
)

var supportedCurrencies = []string{"EUR", "USD", "RUB"}

var (
	ErrFXUnavailable     = errors.New("fx provider unavailable")
	ErrFXTimeout         = errors.New("fx provider timeout")
	ErrFXInvalidResponse = errors.New("fx provider invalid response")
)

type Service struct {
	q         *sqlc.Queries
	client    *http.Client
	apiBase   string
	apiSource string
}

type latestRatesResponse struct {
	Result string             `json:"result"`
	Rates  map[string]float64 `json:"rates"`
}

func NewService(q *sqlc.Queries, apiBase string) *Service {
	base := strings.TrimSpace(apiBase)
	if base == "" {
		base = "https://open.er-api.com/v6"
	}
	base = strings.TrimRight(base, "/")
	return &Service{
		q:         q,
		client:    &http.Client{Timeout: 12 * time.Second},
		apiBase:   base,
		apiSource: "open.er-api.com",
	}
}

func (s *Service) RefreshRates(ctx context.Context) error {
	updatedAt := time.Now().UTC()
	for _, base := range supportedCurrencies {
		rates, err := s.fetchRates(ctx, base)
		if err != nil {
			return err
		}

		for _, target := range supportedCurrencies {
			rate := 1.0
			if base != target {
				val, ok := rates[target]
				if !ok || val <= 0 {
					return fmt.Errorf("rate %s->%s not found in provider response", base, target)
				}
				rate = val
			}
			if _, err := s.q.UpsertCurrencyRate(ctx, sqlc.UpsertCurrencyRateParams{
				FromCurrencyCode: base,
				ToCurrencyCode:   target,
				Rate:             rate,
				Source:           s.apiSource,
				UpdatedAt:        updatedAt,
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) fetchRates(ctx context.Context, base string) (map[string]float64, error) {
	url := fmt.Sprintf("%s/latest/%s", s.apiBase, base)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("%w", ErrFXTimeout)
		}
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return nil, fmt.Errorf("%w", ErrFXTimeout)
		}
		return nil, fmt.Errorf("%w", ErrFXUnavailable)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%w", ErrFXUnavailable)
	}

	var payload latestRatesResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&payload); err != nil {
		return nil, fmt.Errorf("%w", ErrFXInvalidResponse)
	}
	if payload.Result != "success" {
		return nil, fmt.Errorf("%w", ErrFXInvalidResponse)
	}
	if len(payload.Rates) == 0 {
		return nil, fmt.Errorf("%w", ErrFXInvalidResponse)
	}
	return payload.Rates, nil
}
