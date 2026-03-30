package payout

import (
	"context"
	"time"

	"isbudget/backend/internal/db/sqlc"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	GetProjectByID(ctx context.Context, id uuid.UUID) (sqlc.Project, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (sqlc.User, error)
	GetProjectMemberByUser(ctx context.Context, req sqlc.GetProjectMemberByUserParams) (sqlc.ProjectMember, error)
	ListProjectMembers(ctx context.Context, projectID uuid.UUID) ([]sqlc.ProjectMemberWithUser, error)
	GetCurrencyRate(ctx context.Context, fromCurrencyCode, toCurrencyCode string) (sqlc.CurrencyRate, error)
	GetFinanceSummaryByProject(ctx context.Context, projectID uuid.UUID) (sqlc.FinanceSummary, error)
	ListProfitShareRulesByProject(ctx context.Context, projectID uuid.UUID) ([]sqlc.ProfitShareRule, error)
	ListTaxRulesByProject(ctx context.Context, projectID uuid.UUID) ([]sqlc.ProjectTaxRule, error)
	GetLatestForecastRunByProject(ctx context.Context, projectID uuid.UUID) (sqlc.ForecastRun, error)
	ListForecastPointsByRun(ctx context.Context, runID uuid.UUID) ([]sqlc.ForecastPoint, error)
	StorePayouts(ctx context.Context, projectID uuid.UUID, calculatedAt time.Time, items []PreviewItem) ([]sqlc.Payout, error)
	CreateProfitShareRule(ctx context.Context, req sqlc.CreateProfitShareRuleParams) (sqlc.ProfitShareRule, error)
	CreateProfitShareRulePrepend(ctx context.Context, req sqlc.CreateProfitShareRuleParams) (sqlc.ProfitShareRule, error)
	UpdateProfitShareRule(ctx context.Context, req sqlc.UpdateProfitShareRuleParams) (sqlc.ProfitShareRule, error)
	DeleteProfitShareRule(ctx context.Context, req sqlc.DeleteProfitShareRuleParams) error
}

type PGRepository struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
}

func NewPGRepository(q *sqlc.Queries, pool *pgxpool.Pool) *PGRepository {
	return &PGRepository{q: q, pool: pool}
}

func (r *PGRepository) GetFinanceSummaryByProject(ctx context.Context, projectID uuid.UUID) (sqlc.FinanceSummary, error) {
	return r.q.GetFinanceSummaryByProject(ctx, projectID)
}

func (r *PGRepository) GetProjectByID(ctx context.Context, id uuid.UUID) (sqlc.Project, error) {
	row, err := r.q.GetProjectByID(ctx, id)
	if err != nil {
		return sqlc.Project{}, err
	}
	return sqlc.Project{
		ID:                    row.ID,
		Name:                  row.Name,
		Description:           row.Description,
		CreatedBy:             row.CreatedBy,
		CreatedAt:             row.CreatedAt,
		UpdatedAt:             row.UpdatedAt,
		DurationDays:          row.DurationDays,
		Status:                row.Status,
		ActivatedAt:           row.ActivatedAt,
		CurrencyCode:          row.CurrencyCode,
		Category:              row.Category,
		SecondaryCurrencyCode: row.SecondaryCurrencyCode,
	}, nil
}

func (r *PGRepository) GetUserByID(ctx context.Context, id uuid.UUID) (sqlc.User, error) {
	row, err := r.q.GetUserByID(ctx, id)
	if err != nil {
		return sqlc.User{}, err
	}
	return sqlc.User{
		ID:                  row.ID,
		Email:               row.Email,
		PasswordHash:        row.PasswordHash,
		FullName:            row.FullName,
		CreatedAt:           row.CreatedAt,
		DefaultCurrencyCode: row.DefaultCurrencyCode,
		FirstName:           row.FirstName,
		LastName:            row.LastName,
		Nickname:            row.Nickname,
		AvatarPath:          row.AvatarPath,
	}, nil
}

func (r *PGRepository) GetProjectMemberByUser(ctx context.Context, req sqlc.GetProjectMemberByUserParams) (sqlc.ProjectMember, error) {
	return r.q.GetProjectMemberByUser(ctx, req)
}

func (r *PGRepository) ListProjectMembers(ctx context.Context, projectID uuid.UUID) ([]sqlc.ProjectMemberWithUser, error) {
	return r.q.ListProjectMembers(ctx, projectID)
}

func (r *PGRepository) GetCurrencyRate(ctx context.Context, fromCurrencyCode, toCurrencyCode string) (sqlc.CurrencyRate, error) {
	row, err := r.q.GetCurrencyRate(ctx, sqlc.GetCurrencyRateParams{
		FromCurrencyCode: fromCurrencyCode,
		ToCurrencyCode:   toCurrencyCode,
	})
	if err != nil {
		return sqlc.CurrencyRate{}, err
	}
	return sqlc.CurrencyRate(row), nil
}

func (r *PGRepository) ListProfitShareRulesByProject(ctx context.Context, projectID uuid.UUID) ([]sqlc.ProfitShareRule, error) {
	rows, err := r.q.ListProfitShareRulesByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	items := make([]sqlc.ProfitShareRule, 0, len(rows))
	for _, row := range rows {
		items = append(items, sqlc.ProfitShareRule{
			ID:                    row.ID,
			ProjectID:             row.ProjectID,
			RecipientName:         row.RecipientName,
			Percent:               row.Percent,
			AppliesAfterMandatory: row.AppliesAfterMandatory,
			CreatedAt:             row.CreatedAt,
			UpdatedAt:             row.UpdatedAt,
			Responsibility:        row.Responsibility,
			Level:                 row.Level,
			CurrencyCode:          row.CurrencyCode,
			RecipientUserID:       row.RecipientUserID,
		})
	}
	return items, nil
}

func (r *PGRepository) ListTaxRulesByProject(ctx context.Context, projectID uuid.UUID) ([]sqlc.ProjectTaxRule, error) {
	rows, err := r.q.ListTaxRulesByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	items := make([]sqlc.ProjectTaxRule, 0, len(rows))
	for _, row := range rows {
		items = append(items, sqlc.ProjectTaxRule{
			ID:        row.ID,
			ProjectID: row.ProjectID,
			Name:      row.Name,
			Percent:   row.Percent,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
			Level:     row.Level,
		})
	}
	return items, nil
}

func (r *PGRepository) GetLatestForecastRunByProject(ctx context.Context, projectID uuid.UUID) (sqlc.ForecastRun, error) {
	return r.q.GetLatestForecastRunByProject(ctx, projectID)
}

func (r *PGRepository) ListForecastPointsByRun(ctx context.Context, runID uuid.UUID) ([]sqlc.ForecastPoint, error) {
	rows, err := r.q.ListForecastPointsByRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	items := make([]sqlc.ForecastPoint, 0, len(rows))
	for _, row := range rows {
		items = append(items, sqlc.ForecastPoint{
			ID:           row.ID,
			RunID:        row.RunID,
			ForecastDate: row.ForecastDate,
			Yhat:         row.Yhat,
			P10:          row.P10,
			P90:          row.P90,
		})
	}
	return items, nil
}

func (r *PGRepository) StorePayouts(ctx context.Context, projectID uuid.UUID, calculatedAt time.Time, items []PreviewItem) ([]sqlc.Payout, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	qtx := r.q.WithTx(tx)

	created := make([]sqlc.Payout, 0, len(items))
	for _, item := range items {
		var ruleID *uuid.UUID
		if item.RuleID != uuid.Nil {
			value := item.RuleID
			ruleID = &value
		}
		payout, err := qtx.CreatePayout(ctx, sqlc.CreatePayoutParams{
			ProjectID:       projectID,
			RuleID:          ruleID,
			CalculatedAt:    calculatedAt,
			RecipientName:   item.RecipientName,
			Amount:          item.PayoutAmount,
			CurrencyCode:    item.CurrencyCode,
			AmountConverted: item.PayoutAmountConverted,
			ConversionRate:  item.ConversionRate,
			Status:          "calculated",
		})
		if err != nil {
			return nil, err
		}
		created = append(created, sqlc.Payout{
			ID:              payout.ID,
			ProjectID:       payout.ProjectID,
			RuleID:          payout.RuleID,
			CalculatedAt:    payout.CalculatedAt,
			RecipientName:   payout.RecipientName,
			Amount:          payout.Amount,
			Status:          payout.Status,
			CurrencyCode:    payout.CurrencyCode,
			AmountConverted: payout.AmountConverted,
			ConversionRate:  payout.ConversionRate,
		})
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return created, nil
}

func (r *PGRepository) CreateProfitShareRule(ctx context.Context, req sqlc.CreateProfitShareRuleParams) (sqlc.ProfitShareRule, error) {
	row, err := r.q.CreateProfitShareRule(ctx, req)
	if err != nil {
		return sqlc.ProfitShareRule{}, err
	}
	return sqlc.ProfitShareRule{
		ID:                    row.ID,
		ProjectID:             row.ProjectID,
		RecipientName:         row.RecipientName,
		Percent:               row.Percent,
		AppliesAfterMandatory: row.AppliesAfterMandatory,
		CreatedAt:             row.CreatedAt,
		UpdatedAt:             row.UpdatedAt,
		Responsibility:        row.Responsibility,
		Level:                 row.Level,
		CurrencyCode:          row.CurrencyCode,
		RecipientUserID:       row.RecipientUserID,
	}, nil
}

func (r *PGRepository) CreateProfitShareRulePrepend(ctx context.Context, req sqlc.CreateProfitShareRuleParams) (sqlc.ProfitShareRule, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return sqlc.ProfitShareRule{}, err
	}
	defer tx.Rollback(ctx)

	qtx := r.q.WithTx(tx)
	rows, err := qtx.ListProfitShareRulesByProject(ctx, req.ProjectID)
	if err != nil {
		return sqlc.ProfitShareRule{}, err
	}
	rules := make([]sqlc.ProfitShareRule, 0, len(rows))
	for _, row := range rows {
		rules = append(rules, sqlc.ProfitShareRule{
			ID:                    row.ID,
			ProjectID:             row.ProjectID,
			RecipientName:         row.RecipientName,
			Percent:               row.Percent,
			AppliesAfterMandatory: row.AppliesAfterMandatory,
			CreatedAt:             row.CreatedAt,
			UpdatedAt:             row.UpdatedAt,
			Responsibility:        row.Responsibility,
			Level:                 row.Level,
			CurrencyCode:          row.CurrencyCode,
			RecipientUserID:       row.RecipientUserID,
		})
	}

	shifted := shiftProfitShareRulesForPrepend(rules)
	tempBase := int32(len(shifted) + 1000)
	for idx, rule := range shifted {
		if _, err := qtx.UpdateProfitShareRule(ctx, sqlc.UpdateProfitShareRuleParams{
			ProjectID:             req.ProjectID,
			ID:                    rule.ID,
			RecipientUserID:       rule.RecipientUserID,
			RecipientName:         rule.RecipientName,
			Responsibility:        rule.Responsibility,
			CurrencyCode:          rule.CurrencyCode,
			Percent:               rule.Percent,
			Level:                 tempBase + int32(idx),
			AppliesAfterMandatory: true,
		}); err != nil {
			return sqlc.ProfitShareRule{}, err
		}
	}
	for _, rule := range shifted {
		if _, err := qtx.UpdateProfitShareRule(ctx, sqlc.UpdateProfitShareRuleParams{
			ProjectID:             req.ProjectID,
			ID:                    rule.ID,
			RecipientUserID:       rule.RecipientUserID,
			RecipientName:         rule.RecipientName,
			Responsibility:        rule.Responsibility,
			CurrencyCode:          rule.CurrencyCode,
			Percent:               rule.Percent,
			Level:                 rule.Level,
			AppliesAfterMandatory: rule.Level > 1,
		}); err != nil {
			return sqlc.ProfitShareRule{}, err
		}
	}

	req.Level = 1
	req.AppliesAfterMandatory = false
	row, err := qtx.CreateProfitShareRule(ctx, req)
	if err != nil {
		return sqlc.ProfitShareRule{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return sqlc.ProfitShareRule{}, err
	}
	return sqlc.ProfitShareRule{
		ID:                    row.ID,
		ProjectID:             row.ProjectID,
		RecipientName:         row.RecipientName,
		Percent:               row.Percent,
		AppliesAfterMandatory: row.AppliesAfterMandatory,
		CreatedAt:             row.CreatedAt,
		UpdatedAt:             row.UpdatedAt,
		Responsibility:        row.Responsibility,
		Level:                 row.Level,
		CurrencyCode:          row.CurrencyCode,
		RecipientUserID:       row.RecipientUserID,
	}, nil
}

func (r *PGRepository) UpdateProfitShareRule(ctx context.Context, req sqlc.UpdateProfitShareRuleParams) (sqlc.ProfitShareRule, error) {
	row, err := r.q.UpdateProfitShareRule(ctx, req)
	if err != nil {
		return sqlc.ProfitShareRule{}, err
	}
	return sqlc.ProfitShareRule{
		ID:                    row.ID,
		ProjectID:             row.ProjectID,
		RecipientName:         row.RecipientName,
		Percent:               row.Percent,
		AppliesAfterMandatory: row.AppliesAfterMandatory,
		CreatedAt:             row.CreatedAt,
		UpdatedAt:             row.UpdatedAt,
		Responsibility:        row.Responsibility,
		Level:                 row.Level,
		CurrencyCode:          row.CurrencyCode,
		RecipientUserID:       row.RecipientUserID,
	}, nil
}

func (r *PGRepository) DeleteProfitShareRule(ctx context.Context, req sqlc.DeleteProfitShareRuleParams) error {
	return r.q.DeleteProfitShareRule(ctx, req)
}
