package payout

import (
	"context"
	"errors"
	"testing"
	"time"

	"isbudget/backend/internal/db/sqlc"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type fakeRepo struct {
	summary        sqlc.FinanceSummary
	rules          []sqlc.ProfitShareRule
	taxRules       []sqlc.ProjectTaxRule
	project        sqlc.Project
	user           sqlc.User
	users          map[uuid.UUID]sqlc.User
	members        []sqlc.ProjectMemberWithUser
	currencyRate   map[string]sqlc.CurrencyRate
	forecastRun    sqlc.ForecastRun
	forecastPoints []sqlc.ForecastPoint
	lastCreate     sqlc.CreateProfitShareRuleParams
	createCalled   bool
	prependCalled  bool
}

func (f *fakeRepo) GetFinanceSummaryByProject(ctx context.Context, projectID uuid.UUID) (sqlc.FinanceSummary, error) {
	return f.summary, nil
}

func (f *fakeRepo) GetProjectByID(ctx context.Context, id uuid.UUID) (sqlc.Project, error) {
	if f.project.ID == uuid.Nil {
		f.project = sqlc.Project{
			ID:           id,
			Name:         "Test project",
			CurrencyCode: "EUR",
		}
	}
	return f.project, nil
}

func (f *fakeRepo) GetUserByID(ctx context.Context, id uuid.UUID) (sqlc.User, error) {
	if f.users != nil {
		if user, ok := f.users[id]; ok {
			return user, nil
		}
	}
	if f.user.ID == uuid.Nil {
		return sqlc.User{
			ID:                  id,
			Email:               "demo@example.com",
			FullName:            "Demo User",
			DefaultCurrencyCode: "EUR",
		}, nil
	}
	return f.user, nil
}

func (f *fakeRepo) GetCurrencyRate(ctx context.Context, fromCurrencyCode, toCurrencyCode string) (sqlc.CurrencyRate, error) {
	if fromCurrencyCode == toCurrencyCode {
		return sqlc.CurrencyRate{
			FromCurrencyCode: fromCurrencyCode,
			ToCurrencyCode:   toCurrencyCode,
			Rate:             1,
		}, nil
	}
	key := fromCurrencyCode + "_" + toCurrencyCode
	if f.currencyRate != nil {
		if rate, ok := f.currencyRate[key]; ok {
			return rate, nil
		}
	}
	return sqlc.CurrencyRate{}, pgx.ErrNoRows
}

func (f *fakeRepo) GetProjectMemberByUser(ctx context.Context, req sqlc.GetProjectMemberByUserParams) (sqlc.ProjectMember, error) {
	return sqlc.ProjectMember{
		ID:        uuid.New(),
		ProjectID: req.ProjectID,
		UserID:    req.UserID,
		Role:      "TEAM",
	}, nil
}

func (f *fakeRepo) ListProjectMembers(ctx context.Context, projectID uuid.UUID) ([]sqlc.ProjectMemberWithUser, error) {
	if len(f.members) > 0 {
		return f.members, nil
	}
	return []sqlc.ProjectMemberWithUser{}, nil
}

func (f *fakeRepo) ListProfitShareRulesByProject(ctx context.Context, projectID uuid.UUID) ([]sqlc.ProfitShareRule, error) {
	return f.rules, nil
}

func (f *fakeRepo) ListTaxRulesByProject(ctx context.Context, projectID uuid.UUID) ([]sqlc.ProjectTaxRule, error) {
	return f.taxRules, nil
}

func (f *fakeRepo) GetLatestForecastRunByProject(ctx context.Context, projectID uuid.UUID) (sqlc.ForecastRun, error) {
	if f.forecastRun.ID == uuid.Nil {
		return sqlc.ForecastRun{}, pgx.ErrNoRows
	}
	return f.forecastRun, nil
}

func (f *fakeRepo) ListForecastPointsByRun(ctx context.Context, runID uuid.UUID) ([]sqlc.ForecastPoint, error) {
	return append([]sqlc.ForecastPoint(nil), f.forecastPoints...), nil
}

func (f *fakeRepo) StorePayouts(ctx context.Context, projectID uuid.UUID, calculatedAt time.Time, items []PreviewItem) ([]sqlc.Payout, error) {
	out := make([]sqlc.Payout, 0, len(items))
	for _, item := range items {
		var ruleID *uuid.UUID
		if item.RuleID != uuid.Nil {
			value := item.RuleID
			ruleID = &value
		}
		out = append(out, sqlc.Payout{
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
	}
	return out, nil
}

func (f *fakeRepo) CreateProfitShareRule(ctx context.Context, req sqlc.CreateProfitShareRuleParams) (sqlc.ProfitShareRule, error) {
	f.lastCreate = req
	f.createCalled = true
	return sqlc.ProfitShareRule{
		ID:                    uuid.New(),
		ProjectID:             req.ProjectID,
		RecipientName:         req.RecipientName,
		Responsibility:        req.Responsibility,
		CurrencyCode:          req.CurrencyCode,
		Percent:               req.Percent,
		Level:                 req.Level,
		AppliesAfterMandatory: req.AppliesAfterMandatory,
	}, nil
}

func (f *fakeRepo) CreateProfitShareRulePrepend(ctx context.Context, req sqlc.CreateProfitShareRuleParams) (sqlc.ProfitShareRule, error) {
	f.lastCreate = req
	f.createCalled = true
	f.prependCalled = true
	shifted := shiftProfitShareRulesForPrepend(f.rules)
	f.rules = append([]sqlc.ProfitShareRule(nil), shifted...)
	created := sqlc.ProfitShareRule{
		ID:                    uuid.New(),
		ProjectID:             req.ProjectID,
		RecipientUserID:       req.RecipientUserID,
		RecipientName:         req.RecipientName,
		Responsibility:        req.Responsibility,
		CurrencyCode:          req.CurrencyCode,
		Percent:               req.Percent,
		Level:                 1,
		AppliesAfterMandatory: false,
	}
	f.rules = append(f.rules, created)
	return created, nil
}

func (f *fakeRepo) UpdateProfitShareRule(ctx context.Context, req sqlc.UpdateProfitShareRuleParams) (sqlc.ProfitShareRule, error) {
	for i := range f.rules {
		if f.rules[i].ProjectID == req.ProjectID && f.rules[i].ID == req.ID {
			f.rules[i].RecipientUserID = req.RecipientUserID
			f.rules[i].RecipientName = req.RecipientName
			f.rules[i].Responsibility = req.Responsibility
			f.rules[i].CurrencyCode = req.CurrencyCode
			f.rules[i].Percent = req.Percent
			f.rules[i].Level = req.Level
			f.rules[i].AppliesAfterMandatory = req.AppliesAfterMandatory
			return f.rules[i], nil
		}
	}
	return sqlc.ProfitShareRule{}, pgx.ErrNoRows
}

func (f *fakeRepo) DeleteProfitShareRule(ctx context.Context, req sqlc.DeleteProfitShareRuleParams) error {
	return nil
}

func TestComputeSummaryMandatoryRemaining(t *testing.T) {
	totals := sqlc.FinanceSummary{
		IncomeTotal:    1000,
		ExpensesTotal:  300,
		PlannedTotal:   1200,
		MandatoryTotal: 800,
		MandatoryPaid:  200,
	}
	got := ComputeSummary(totals, nil)

	if got.NetProfitToDate != 700 {
		t.Fatalf("expected net profit 700, got %.2f", got.NetProfitToDate)
	}
	if got.MandatoryRemaining != 600 {
		t.Fatalf("expected mandatory remaining 600, got %.2f", got.MandatoryRemaining)
	}
	if got.AvailableForSharing != 100 {
		t.Fatalf("expected available_for_sharing 100, got %.2f", got.AvailableForSharing)
	}
}

func TestComputeSummaryWithTaxRules(t *testing.T) {
	totals := sqlc.FinanceSummary{
		IncomeTotal:    1500,
		ExpensesTotal:  300,
		MandatoryTotal: 0,
		MandatoryPaid:  0,
	}
	got := ComputeSummary(totals, []sqlc.ProjectTaxRule{
		{Percent: 20, Level: 1},
		{Percent: 10, Level: 2},
	})

	if got.AvailableBeforeTax != 1200 {
		t.Fatalf("expected available_before_tax 1200, got %.2f", got.AvailableBeforeTax)
	}
	// 1200*20% = 240, remaining 960; 960*10% = 96; total = 336
	if got.TaxTotal != 336 {
		t.Fatalf("expected tax_total 336, got %.2f", got.TaxTotal)
	}
	if got.TaxPercentTotal != 28 {
		t.Fatalf("expected effective tax_percent_total 28, got %.2f", got.TaxPercentTotal)
	}
	if got.AvailableForSharing != 864 {
		t.Fatalf("expected available_for_sharing 864, got %.2f", got.AvailableForSharing)
	}
}

func TestComputeSummaryTaxPercentTotalIsEffectiveSequentialRate(t *testing.T) {
	totals := sqlc.FinanceSummary{
		IncomeTotal:    1000,
		ExpensesTotal:  0,
		MandatoryTotal: 0,
		MandatoryPaid:  0,
	}

	got := ComputeSummary(totals, []sqlc.ProjectTaxRule{
		{Percent: 25, Level: 1},
		{Percent: 12, Level: 2},
		{Percent: 15, Level: 3},
	})

	// 1000 -> 750 (25%) -> 660 (12%) -> 561 (15%).
	// Total tax = 439, effective rate = 43.9% of initial base.
	if got.TaxTotal != 439 {
		t.Fatalf("expected tax_total 439, got %.2f", got.TaxTotal)
	}
	if got.TaxPercentTotal != 43.9 {
		t.Fatalf("expected effective tax_percent_total 43.9, got %.2f", got.TaxPercentTotal)
	}
	if got.AvailableForSharing != 561 {
		t.Fatalf("expected available_for_sharing 561, got %.2f", got.AvailableForSharing)
	}
}

func TestComputeSummaryWithMultipleTaxesOnSameLevel(t *testing.T) {
	totals := sqlc.FinanceSummary{
		IncomeTotal:    1000,
		ExpensesTotal:  0,
		MandatoryTotal: 0,
		MandatoryPaid:  0,
	}

	got := ComputeSummary(totals, []sqlc.ProjectTaxRule{
		{Percent: 10, Level: 1},
		{Percent: 10, Level: 1},
		{Percent: 10, Level: 2},
	})

	// Level 1 from 1000: 100 + 100 = 200, remaining 800.
	// Level 2 from 800: 80, remaining 720.
	if got.TaxTotal != 280 {
		t.Fatalf("expected tax_total 280, got %.2f", got.TaxTotal)
	}
	if got.TaxPercentTotal != 28 {
		t.Fatalf("expected effective tax_percent_total 28, got %.2f", got.TaxPercentTotal)
	}
	if got.AvailableForSharing != 720 {
		t.Fatalf("expected available_for_sharing 720, got %.2f", got.AvailableForSharing)
	}
}

func TestPreviewReturnsTaxLevelBreakdown(t *testing.T) {
	projectID := uuid.New()
	repo := &fakeRepo{
		summary: sqlc.FinanceSummary{
			IncomeTotal:   1000,
			ExpensesTotal: 0,
		},
		project: sqlc.Project{
			ID:           projectID,
			CurrencyCode: "EUR",
		},
		taxRules: []sqlc.ProjectTaxRule{
			{Percent: 20, Level: 1},
			{Percent: 10, Level: 2},
		},
	}

	svc := NewService(repo)
	preview, err := svc.Preview(context.Background(), projectID)
	if err != nil {
		t.Fatalf("preview failed: %v", err)
	}
	if len(preview.TaxLevels) != 2 {
		t.Fatalf("expected 2 tax levels, got %d", len(preview.TaxLevels))
	}

	level1 := preview.TaxLevels[0]
	if level1.Level != 1 || level1.BaseAmount != 1000 || level1.PercentTotal != 20 || level1.TaxAmount != 200 || level1.RemainingAfter != 800 {
		t.Fatalf("unexpected level1 breakdown: %+v", level1)
	}

	level2 := preview.TaxLevels[1]
	if level2.Level != 2 || level2.BaseAmount != 800 || level2.PercentTotal != 10 || level2.TaxAmount != 80 || level2.RemainingAfter != 720 {
		t.Fatalf("unexpected level2 breakdown: %+v", level2)
	}
}

func TestPreviewCalculatesPayouts(t *testing.T) {
	repo := &fakeRepo{
		summary: sqlc.FinanceSummary{
			IncomeTotal:    3000,
			ExpensesTotal:  1000,
			PlannedTotal:   2000,
			MandatoryTotal: 900,
			MandatoryPaid:  900,
		},
		rules: []sqlc.ProfitShareRule{
			{ID: uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"), RecipientName: "3D Artist", Percent: 20},
			{ID: uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"), RecipientName: "PM", Percent: 10},
		},
	}
	svc := NewService(repo)
	preview, err := svc.Preview(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("preview failed: %v", err)
	}

	if preview.Summary.AvailableForSharing != 2000 {
		t.Fatalf("expected available_for_sharing 2000, got %.2f", preview.Summary.AvailableForSharing)
	}
	if len(preview.Items) != 2 {
		t.Fatalf("expected 2 preview items, got %d", len(preview.Items))
	}
	if preview.Items[0].PayoutAmount != 400 {
		t.Fatalf("expected first payout 400, got %.2f", preview.Items[0].PayoutAmount)
	}
	if preview.Items[1].PayoutAmount != 200 {
		t.Fatalf("expected second payout 200, got %.2f", preview.Items[1].PayoutAmount)
	}
}

func TestBuildPreviewItemsByLevels(t *testing.T) {
	rules := []sqlc.ProfitShareRule{
		{
			ID:                    uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
			RecipientName:         "Level1 A",
			Responsibility:        "Partner",
			Percent:               25,
			AppliesAfterMandatory: false,
		},
		{
			ID:                    uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
			RecipientName:         "Level1 B",
			Responsibility:        "Commission",
			Percent:               10,
			AppliesAfterMandatory: false,
		},
		{
			ID:                    uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc"),
			RecipientName:         "Level2 Team",
			Responsibility:        "Team",
			Percent:               50,
			AppliesAfterMandatory: true,
		},
	}

	items := BuildPreviewItems(1000, rules, "EUR")
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	if items[0].Level != 1 || items[0].PayoutAmount != 250 {
		t.Fatalf("expected first item level=1 payout=250, got level=%d payout=%.2f", items[0].Level, items[0].PayoutAmount)
	}
	if items[1].Level != 1 || items[1].PayoutAmount != 100 {
		t.Fatalf("expected second item level=1 payout=100, got level=%d payout=%.2f", items[1].Level, items[1].PayoutAmount)
	}
	if items[2].Level != 2 || items[2].PayoutAmount != 325 {
		t.Fatalf("expected third item level=2 payout=325, got level=%d payout=%.2f", items[2].Level, items[2].PayoutAmount)
	}
}

func TestCreateRuleRejectsPercentOverflowInSameLevel(t *testing.T) {
	projectID := uuid.New()
	memberID := uuid.New()
	repo := &fakeRepo{
		rules: []sqlc.ProfitShareRule{
			{
				ID:                    uuid.New(),
				ProjectID:             projectID,
				RecipientName:         "Existing",
				Responsibility:        "Role",
				Percent:               70,
				AppliesAfterMandatory: false,
			},
		},
		members: []sqlc.ProjectMemberWithUser{
			{UserID: memberID, FullName: "Overflow User"},
		},
	}
	svc := NewService(repo)

	_, err := svc.CreateRule(context.Background(), sqlc.CreateProfitShareRuleParams{
		ProjectID:             projectID,
		RecipientUserID:       &memberID,
		Responsibility:        "Role",
		Percent:               40,
		AppliesAfterMandatory: false,
	})
	if err == nil {
		t.Fatal("expected overflow error, got nil")
	}
	if !errors.Is(err, ErrRulePercentOverflow) {
		t.Fatalf("expected ErrRulePercentOverflow, got %v", err)
	}
	if repo.createCalled {
		t.Fatal("create should not be called when percent budget is exceeded")
	}
}

func TestCreateRuleAllowsDifferentLevelBudget(t *testing.T) {
	projectID := uuid.New()
	memberID := uuid.New()
	repo := &fakeRepo{
		rules: []sqlc.ProfitShareRule{
			{
				ID:                    uuid.New(),
				ProjectID:             projectID,
				RecipientName:         "Level1 Existing",
				Responsibility:        "Role",
				Percent:               90,
				AppliesAfterMandatory: false,
			},
		},
		members: []sqlc.ProjectMemberWithUser{
			{UserID: memberID, FullName: "Allowed User"},
		},
	}
	svc := NewService(repo)

	_, err := svc.CreateRule(context.Background(), sqlc.CreateProfitShareRuleParams{
		ProjectID:             projectID,
		RecipientUserID:       &memberID,
		Responsibility:        "Role",
		Percent:               95,
		Level:                 2,
		AppliesAfterMandatory: true,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !repo.createCalled {
		t.Fatal("expected create to be called for valid rule")
	}
}

func TestCreateRulePrependShiftsExistingLevels(t *testing.T) {
	projectID := uuid.New()
	memberID := uuid.New()
	repo := &fakeRepo{
		rules: []sqlc.ProfitShareRule{
			{ID: uuid.New(), ProjectID: projectID, RecipientName: "Level1", Responsibility: "Lead", Percent: 40, Level: 1},
			{ID: uuid.New(), ProjectID: projectID, RecipientName: "Level2", Responsibility: "Artist", Percent: 30, Level: 2, AppliesAfterMandatory: true},
			{ID: uuid.New(), ProjectID: projectID, RecipientName: "Level3", Responsibility: "QA", Percent: 20, Level: 3, AppliesAfterMandatory: true},
		},
		project: sqlc.Project{ID: projectID, CurrencyCode: "EUR"},
		members: []sqlc.ProjectMemberWithUser{
			{UserID: memberID, FullName: "Prepended User"},
		},
	}
	svc := NewService(repo)

	created, err := svc.CreateRulePrepend(context.Background(), sqlc.CreateProfitShareRuleParams{
		ProjectID:       projectID,
		RecipientUserID: &memberID,
		Responsibility:  "Producer",
		Percent:         15,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !repo.prependCalled {
		t.Fatal("expected prepend create to be called")
	}
	if created.Level != 1 {
		t.Fatalf("expected created rule level 1, got %d", created.Level)
	}
	if len(repo.rules) != 4 {
		t.Fatalf("expected 4 rules after prepend, got %d", len(repo.rules))
	}
	if repo.rules[0].Level != 2 || repo.rules[1].Level != 3 || repo.rules[2].Level != 4 {
		t.Fatalf("expected existing levels to shift to 2,3,4, got %d,%d,%d", repo.rules[0].Level, repo.rules[1].Level, repo.rules[2].Level)
	}
	if repo.lastCreate.Level != 1 {
		t.Fatalf("expected prepend request to create level 1, got %d", repo.lastCreate.Level)
	}
}

func TestCreateRuleRejectsSkippedLevels(t *testing.T) {
	projectID := uuid.New()
	memberID := uuid.New()
	repo := &fakeRepo{
		rules: []sqlc.ProfitShareRule{
			{
				ID:             uuid.New(),
				ProjectID:      projectID,
				RecipientName:  "Level1",
				Responsibility: "Role",
				Percent:        40,
				Level:          1,
			},
		},
		members: []sqlc.ProjectMemberWithUser{
			{UserID: memberID, FullName: "Skip User"},
		},
	}
	svc := NewService(repo)

	_, err := svc.CreateRule(context.Background(), sqlc.CreateProfitShareRuleParams{
		ProjectID:       projectID,
		RecipientUserID: &memberID,
		Responsibility:  "Role",
		Percent:         10,
		Level:           3,
	})
	if err == nil {
		t.Fatal("expected skipped level error, got nil")
	}
	if !errors.Is(err, ErrInvalidRuleLevelSequence) {
		t.Fatalf("expected ErrInvalidRuleLevelSequence, got %v", err)
	}
}

func TestListRulesNormalizesLevelsToContiguousSequence(t *testing.T) {
	projectID := uuid.New()
	repo := &fakeRepo{
		rules: []sqlc.ProfitShareRule{
			{
				ID:             uuid.New(),
				ProjectID:      projectID,
				RecipientName:  "Rule A",
				Responsibility: "Role",
				Percent:        10,
				Level:          5,
			},
			{
				ID:             uuid.New(),
				ProjectID:      projectID,
				RecipientName:  "Rule B",
				Responsibility: "Role",
				Percent:        20,
				Level:          16,
			},
		},
	}
	svc := NewService(repo)

	rules, err := svc.ListRules(context.Background(), projectID)
	if err != nil {
		t.Fatalf("list rules failed: %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(rules))
	}
	if rules[0].Level != 1 {
		t.Fatalf("expected first normalized level 1, got %d", rules[0].Level)
	}
	if rules[1].Level != 2 {
		t.Fatalf("expected second normalized level 2, got %d", rules[1].Level)
	}
}

func TestCreateRuleRejectsHigherLevelWhenPreviousIsFull(t *testing.T) {
	projectID := uuid.New()
	memberID := uuid.New()
	repo := &fakeRepo{
		rules: []sqlc.ProfitShareRule{
			{
				ID:             uuid.New(),
				ProjectID:      projectID,
				RecipientName:  "Full level one",
				Responsibility: "Role",
				Percent:        100,
				Level:          1,
			},
		},
		members: []sqlc.ProjectMemberWithUser{
			{UserID: memberID, FullName: "Blocked User"},
		},
	}
	svc := NewService(repo)

	_, err := svc.CreateRule(context.Background(), sqlc.CreateProfitShareRuleParams{
		ProjectID:       projectID,
		RecipientUserID: &memberID,
		Responsibility:  "Role",
		Percent:         10,
		Level:           2,
	})
	if err == nil {
		t.Fatal("expected next level blocked error, got nil")
	}
	if !errors.Is(err, ErrRuleNextLevelBlocked) {
		t.Fatalf("expected ErrRuleNextLevelBlocked, got %v", err)
	}
}

func TestCreateRuleRejectsLevelAboveMax(t *testing.T) {
	projectID := uuid.New()
	memberID := uuid.New()
	repo := &fakeRepo{
		members: []sqlc.ProjectMemberWithUser{
			{UserID: memberID, FullName: "High Level User"},
		},
	}
	svc := NewService(repo)

	_, err := svc.CreateRule(context.Background(), sqlc.CreateProfitShareRuleParams{
		ProjectID:       projectID,
		RecipientUserID: &memberID,
		Responsibility:  "Role",
		Percent:         10,
		Level:           MaxProfitShareLevel + 1,
	})
	if err == nil {
		t.Fatal("expected invalid level error, got nil")
	}
	if !errors.Is(err, ErrInvalidRuleLevel) {
		t.Fatalf("expected ErrInvalidRuleLevel, got %v", err)
	}
}

func TestPreviewConvertsRecipientCurrency(t *testing.T) {
	projectID := uuid.New()
	repo := &fakeRepo{
		summary: sqlc.FinanceSummary{
			IncomeTotal:    1000,
			ExpensesTotal:  0,
			MandatoryTotal: 0,
			MandatoryPaid:  0,
		},
		project: sqlc.Project{
			ID:                    projectID,
			Name:                  "Demo",
			CurrencyCode:          "EUR",
			SecondaryCurrencyCode: strPtr("RUB"),
		},
		rules: []sqlc.ProfitShareRule{
			{
				ID:             uuid.New(),
				ProjectID:      projectID,
				RecipientName:  "Alice",
				Responsibility: "PM",
				CurrencyCode:   "USD",
				Percent:        10,
				Level:          1,
			},
		},
		currencyRate: map[string]sqlc.CurrencyRate{
			"EUR_USD": {FromCurrencyCode: "EUR", ToCurrencyCode: "USD", Rate: 1.1},
			"EUR_RUB": {FromCurrencyCode: "EUR", ToCurrencyCode: "RUB", Rate: 100},
		},
	}

	svc := NewService(repo)
	preview, err := svc.Preview(context.Background(), projectID)
	if err != nil {
		t.Fatalf("preview failed: %v", err)
	}
	if len(preview.Items) != 1 {
		t.Fatalf("expected 1 preview item, got %d", len(preview.Items))
	}
	if preview.Items[0].PayoutAmount != 100 {
		t.Fatalf("expected base payout 100 EUR, got %.2f", preview.Items[0].PayoutAmount)
	}
	if preview.Items[0].PayoutAmountConverted != 110 {
		t.Fatalf("expected converted payout 110 USD, got %.2f", preview.Items[0].PayoutAmountConverted)
	}
	if preview.RemainingCurrencyCode != "RUB" {
		t.Fatalf("expected remaining currency RUB, got %s", preview.RemainingCurrencyCode)
	}
	if preview.RemainingAmountConverted != 90000 {
		t.Fatalf("expected remaining converted 90000 RUB, got %.2f", preview.RemainingAmountConverted)
	}
}

func TestPreviewAssignsResidualToProjectAdministrator(t *testing.T) {
	projectID := uuid.New()
	adminID := uuid.New()
	repo := &fakeRepo{
		summary: sqlc.FinanceSummary{
			IncomeTotal:   1000,
			ExpensesTotal: 0,
		},
		project: sqlc.Project{
			ID:           projectID,
			CreatedBy:    adminID,
			CurrencyCode: "EUR",
		},
		user: sqlc.User{
			ID:                  adminID,
			Email:               "admin@example.com",
			FullName:            "Project Admin",
			DefaultCurrencyCode: "EUR",
		},
		rules: []sqlc.ProfitShareRule{
			{
				ID:              uuid.New(),
				ProjectID:       projectID,
				RecipientName:   "Member",
				Responsibility:  "Role",
				RecipientUserID: ptrUUID(uuid.New()),
				Percent:         10,
				Level:           1,
				CurrencyCode:    "EUR",
			},
		},
	}

	svc := NewService(repo)
	preview, err := svc.Preview(context.Background(), projectID)
	if err != nil {
		t.Fatalf("preview failed: %v", err)
	}

	if preview.RemainingAmount != 0 {
		t.Fatalf("expected zero remaining after admin residual assignment, got %.2f", preview.RemainingAmount)
	}
	if len(preview.Items) != 2 {
		t.Fatalf("expected 2 payout items including admin residual, got %d", len(preview.Items))
	}
	adminItem := preview.Items[1]
	if adminItem.RecipientUserID == nil || *adminItem.RecipientUserID != adminID {
		t.Fatalf("expected admin residual recipient %s, got %+v", adminID, adminItem.RecipientUserID)
	}
	if adminItem.PayoutAmount != 900 {
		t.Fatalf("expected admin residual payout 900, got %.2f", adminItem.PayoutAmount)
	}
	if preview.DistributedAmount != 1000 {
		t.Fatalf("expected distributed amount 1000, got %.2f", preview.DistributedAmount)
	}
}

func TestCreateRuleDefaultsCurrencyToProjectBase(t *testing.T) {
	projectID := uuid.New()
	memberID := uuid.New()
	repo := &fakeRepo{
		project: sqlc.Project{
			ID:           projectID,
			Name:         "Demo",
			CurrencyCode: "EUR",
		},
		members: []sqlc.ProjectMemberWithUser{
			{UserID: memberID, FullName: "Bob"},
		},
	}
	svc := NewService(repo)

	_, err := svc.CreateRule(context.Background(), sqlc.CreateProfitShareRuleParams{
		ProjectID:       projectID,
		RecipientUserID: &memberID,
		Responsibility:  "Artist",
		Percent:         10,
		Level:           1,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repo.lastCreate.CurrencyCode != "EUR" {
		t.Fatalf("expected default currency EUR, got %s", repo.lastCreate.CurrencyCode)
	}
}

func TestCreateRuleRejectsMissingRecipientUser(t *testing.T) {
	projectID := uuid.New()
	svc := NewService(&fakeRepo{
		project: sqlc.Project{ID: projectID, CurrencyCode: "EUR"},
	})

	_, err := svc.CreateRule(context.Background(), sqlc.CreateProfitShareRuleParams{
		ProjectID:      projectID,
		Responsibility: "Producer",
		Percent:        10,
		Level:          1,
	})
	if !errors.Is(err, ErrRecipientUserRequired) {
		t.Fatalf("expected ErrRecipientUserRequired, got %v", err)
	}
}

func TestCreateRuleDerivesRecipientNameFromLinkedMember(t *testing.T) {
	projectID := uuid.New()
	memberID := uuid.New()
	repo := &fakeRepo{
		project: sqlc.Project{ID: projectID, CurrencyCode: "EUR"},
		members: []sqlc.ProjectMemberWithUser{
			{
				UserID:    memberID,
				FirstName: "Alice",
				LastName:  "Brown",
				FullName:  "Ignored Full Name",
				Nickname:  "aliceb",
				Email:     "alice@example.com",
			},
		},
	}
	svc := NewService(repo)

	_, err := svc.CreateRule(context.Background(), sqlc.CreateProfitShareRuleParams{
		ProjectID:       projectID,
		RecipientUserID: &memberID,
		Responsibility:  "Artist",
		Percent:         15,
		Level:           1,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repo.lastCreate.RecipientName != "Alice Brown" {
		t.Fatalf("expected derived recipient name 'Alice Brown', got %q", repo.lastCreate.RecipientName)
	}
}

func TestPreviewForUserIncludesViewerCurrencyEquivalent(t *testing.T) {
	projectID := uuid.New()
	viewerID := uuid.New()
	recipientID := uuid.New()
	repo := &fakeRepo{
		summary: sqlc.FinanceSummary{
			IncomeTotal: 1000,
		},
		project: sqlc.Project{
			ID:           projectID,
			CurrencyCode: "EUR",
		},
		rules: []sqlc.ProfitShareRule{
			{
				ID:              uuid.New(),
				ProjectID:       projectID,
				RecipientUserID: &recipientID,
				RecipientName:   "Alice Brown",
				Responsibility:  "Artist",
				CurrencyCode:    "EUR",
				Percent:         10,
				Level:           1,
			},
		},
		users: map[uuid.UUID]sqlc.User{
			viewerID:    {ID: viewerID, DefaultCurrencyCode: "USD"},
			recipientID: {ID: recipientID, DefaultCurrencyCode: "EUR"},
		},
		currencyRate: map[string]sqlc.CurrencyRate{
			"EUR_USD": {FromCurrencyCode: "EUR", ToCurrencyCode: "USD", Rate: 1.2},
		},
	}

	svc := NewService(repo)
	preview, err := svc.PreviewForUser(context.Background(), projectID, viewerID)
	if err != nil {
		t.Fatalf("preview failed: %v", err)
	}
	if preview.ViewerCurrencyCode != "USD" {
		t.Fatalf("expected viewer currency USD, got %q", preview.ViewerCurrencyCode)
	}
	if len(preview.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(preview.Items))
	}
	if preview.Items[0].ViewerCurrencyCode != "USD" {
		t.Fatalf("expected item viewer currency USD, got %q", preview.Items[0].ViewerCurrencyCode)
	}
	if preview.Items[0].ViewerAmountConverted == nil || *preview.Items[0].ViewerAmountConverted != 120 {
		t.Fatalf("expected viewer converted amount 120, got %+v", preview.Items[0].ViewerAmountConverted)
	}
}

func TestPreviewForUserGracefullyHandlesMissingViewerConversion(t *testing.T) {
	projectID := uuid.New()
	viewerID := uuid.New()
	recipientID := uuid.New()
	repo := &fakeRepo{
		summary: sqlc.FinanceSummary{
			IncomeTotal: 1000,
		},
		project: sqlc.Project{
			ID:           projectID,
			CurrencyCode: "EUR",
		},
		rules: []sqlc.ProfitShareRule{
			{
				ID:              uuid.New(),
				ProjectID:       projectID,
				RecipientUserID: &recipientID,
				RecipientName:   "Alice Brown",
				Responsibility:  "Artist",
				CurrencyCode:    "EUR",
				Percent:         10,
				Level:           1,
			},
		},
		users: map[uuid.UUID]sqlc.User{
			viewerID:    {ID: viewerID, DefaultCurrencyCode: "USD"},
			recipientID: {ID: recipientID, DefaultCurrencyCode: "EUR"},
		},
	}

	svc := NewService(repo)
	preview, err := svc.PreviewForUser(context.Background(), projectID, viewerID)
	if err != nil {
		t.Fatalf("preview should not fail without viewer conversion rate, got %v", err)
	}
	if preview.ViewerCurrencyCode != "USD" {
		t.Fatalf("expected viewer currency USD, got %q", preview.ViewerCurrencyCode)
	}
	if preview.Items[0].ViewerAmountConverted != nil {
		t.Fatalf("expected no viewer converted amount when rate is missing, got %+v", *preview.Items[0].ViewerAmountConverted)
	}
}

func strPtr(v string) *string {
	return &v
}

func ptrUUID(v uuid.UUID) *uuid.UUID {
	return &v
}

func TestMyIncomeMatchesByEmailAndConvertsToUserCurrency(t *testing.T) {
	projectID := uuid.New()
	userID := uuid.New()
	repo := &fakeRepo{
		summary: sqlc.FinanceSummary{
			IncomeTotal:   1000,
			ExpensesTotal: 0,
		},
		project: sqlc.Project{
			ID:           projectID,
			CurrencyCode: "EUR",
		},
		user: sqlc.User{
			ID:                  userID,
			Email:               "teammate@example.com",
			FullName:            "Teammate",
			DefaultCurrencyCode: "RUB",
		},
		rules: []sqlc.ProfitShareRule{
			{
				ID:              uuid.New(),
				ProjectID:       projectID,
				RecipientUserID: &userID,
				RecipientName:   "teammate@example.com",
				Responsibility:  "Artist",
				CurrencyCode:    "EUR",
				Percent:         10,
				Level:           1,
			},
		},
		currencyRate: map[string]sqlc.CurrencyRate{
			"EUR_RUB": {FromCurrencyCode: "EUR", ToCurrencyCode: "RUB", Rate: 100},
		},
	}

	svc := NewService(repo)
	got, err := svc.MyIncome(context.Background(), projectID, userID)
	if err != nil {
		t.Fatalf("MyIncome returned error: %v", err)
	}
	if len(got.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(got.Items))
	}
	if got.TotalProjectAmount != 100 {
		t.Fatalf("expected total project amount 100 EUR, got %.2f", got.TotalProjectAmount)
	}
	if got.TotalUserAmount != 10000 {
		t.Fatalf("expected total user amount 10000 RUB, got %.2f", got.TotalUserAmount)
	}
	if got.ForecastProjection != nil {
		t.Fatalf("expected no forecast projection without forecast run, got %+v", got.ForecastProjection)
	}
}

func TestMyIncomeAddsForecastProjectionWithoutChangingCurrentIncome(t *testing.T) {
	projectID := uuid.New()
	userID := uuid.New()
	runID := uuid.New()
	repo := &fakeRepo{
		summary: sqlc.FinanceSummary{
			IncomeTotal:   1000,
			ExpensesTotal: 100,
		},
		project: sqlc.Project{
			ID:           projectID,
			CurrencyCode: "EUR",
		},
		user: sqlc.User{
			ID:                  userID,
			Email:               "pm@example.com",
			FullName:            "PM",
			DefaultCurrencyCode: "EUR",
		},
		rules: []sqlc.ProfitShareRule{
			{
				ID:              uuid.New(),
				ProjectID:       projectID,
				RecipientUserID: &userID,
				RecipientName:   "pm@example.com",
				Responsibility:  "Administrator residual",
				CurrencyCode:    "EUR",
				Percent:         100,
				Level:           1,
			},
		},
		taxRules: []sqlc.ProjectTaxRule{
			{ID: uuid.New(), ProjectID: projectID, Percent: 10, Level: 1},
		},
		forecastRun: sqlc.ForecastRun{
			ID:        runID,
			ProjectID: projectID,
			CreatedAt: time.Now().UTC(),
		},
		forecastPoints: []sqlc.ForecastPoint{
			{ID: uuid.New(), RunID: runID, P10: 100, Yhat: 200, P90: 400},
			{ID: uuid.New(), RunID: runID, P10: 50, Yhat: 100, P90: 200},
		},
	}

	svc := NewService(repo)
	got, err := svc.MyIncome(context.Background(), projectID, userID)
	if err != nil {
		t.Fatalf("MyIncome returned error: %v", err)
	}

	if got.TotalProjectAmount != 810 {
		t.Fatalf("expected current total project amount 810, got %.2f", got.TotalProjectAmount)
	}
	if got.ForecastProjection == nil {
		t.Fatalf("expected forecast projection, got nil")
	}
	if got.ForecastProjection.P10.ProjectedIncomeTotal != 1150 {
		t.Fatalf("expected projected income p10 1150, got %.2f", got.ForecastProjection.P10.ProjectedIncomeTotal)
	}
	if got.ForecastProjection.Yhat.ProjectedIncomeTotal != 1300 {
		t.Fatalf("expected projected income yhat 1300, got %.2f", got.ForecastProjection.Yhat.ProjectedIncomeTotal)
	}
	if got.ForecastProjection.P90.ProjectedIncomeTotal != 1600 {
		t.Fatalf("expected projected income p90 1600, got %.2f", got.ForecastProjection.P90.ProjectedIncomeTotal)
	}
	if got.ForecastProjection.P10.UserAmount != 945 {
		t.Fatalf("expected p10 user amount 945, got %.2f", got.ForecastProjection.P10.UserAmount)
	}
	if got.ForecastProjection.Yhat.UserAmount != 1080 {
		t.Fatalf("expected yhat user amount 1080, got %.2f", got.ForecastProjection.Yhat.UserAmount)
	}
	if got.ForecastProjection.P90.UserAmount != 1350 {
		t.Fatalf("expected p90 user amount 1350, got %.2f", got.ForecastProjection.P90.UserAmount)
	}
	if got.TotalUserAmount != got.TotalProjectAmount {
		t.Fatalf("expected current income conversion to stay unchanged, got project %.2f user %.2f", got.TotalProjectAmount, got.TotalUserAmount)
	}
}

func TestMyIncomeForecastProjectionRespectsProfitShareRules(t *testing.T) {
	projectID := uuid.New()
	userID := uuid.New()
	otherUserID := uuid.New()
	runID := uuid.New()
	repo := &fakeRepo{
		summary: sqlc.FinanceSummary{
			IncomeTotal:   1000,
			ExpensesTotal: 100,
		},
		project: sqlc.Project{
			ID:           projectID,
			CurrencyCode: "EUR",
		},
		user: sqlc.User{
			ID:                  userID,
			Email:               "artist@example.com",
			FullName:            "Artist",
			DefaultCurrencyCode: "EUR",
		},
		rules: []sqlc.ProfitShareRule{
			{
				ID:              uuid.New(),
				ProjectID:       projectID,
				RecipientUserID: &userID,
				RecipientName:   "artist@example.com",
				Responsibility:  "Artist",
				CurrencyCode:    "EUR",
				Percent:         50,
				Level:           1,
			},
			{
				ID:              uuid.New(),
				ProjectID:       projectID,
				RecipientUserID: &otherUserID,
				RecipientName:   "other@example.com",
				Responsibility:  "Support",
				CurrencyCode:    "EUR",
				Percent:         50,
				Level:           1,
			},
		},
		taxRules: []sqlc.ProjectTaxRule{
			{ID: uuid.New(), ProjectID: projectID, Percent: 20, Level: 1},
		},
		forecastRun: sqlc.ForecastRun{
			ID:        runID,
			ProjectID: projectID,
			CreatedAt: time.Now().UTC(),
		},
		forecastPoints: []sqlc.ForecastPoint{
			{ID: uuid.New(), RunID: runID, P10: 100, Yhat: 200, P90: 300},
		},
	}

	svc := NewService(repo)
	got, err := svc.MyIncome(context.Background(), projectID, userID)
	if err != nil {
		t.Fatalf("MyIncome returned error: %v", err)
	}
	if got.ForecastProjection == nil {
		t.Fatalf("expected forecast projection, got nil")
	}
	if got.ForecastProjection.Yhat.UserAmount != 440 {
		t.Fatalf("expected yhat user amount 440 with 20%% tax and 50%% share, got %.2f", got.ForecastProjection.Yhat.UserAmount)
	}
}

func TestTeamIncomeForecastIncludesCurrentAndScenarioAmounts(t *testing.T) {
	projectID := uuid.New()
	pmID := uuid.New()
	teamID := uuid.New()
	runID := uuid.New()
	createdAt := time.Now().UTC()

	repo := &fakeRepo{
		summary: sqlc.FinanceSummary{
			IncomeTotal:   1000,
			ExpensesTotal: 100,
		},
		project: sqlc.Project{
			ID:           projectID,
			CurrencyCode: "EUR",
		},
		members: []sqlc.ProjectMemberWithUser{
			{
				UserID:              pmID,
				Role:                "PM",
				FullName:            "Admin One",
				DefaultCurrencyCode: "RUB",
				Email:               "pm@example.com",
			},
			{
				UserID:              teamID,
				Role:                "TEAM",
				FullName:            "Artist Two",
				DefaultCurrencyCode: "USD",
				Email:               "team@example.com",
			},
		},
		rules: []sqlc.ProfitShareRule{
			{
				ID:              uuid.New(),
				ProjectID:       projectID,
				RecipientUserID: &teamID,
				RecipientName:   "Artist Two",
				Responsibility:  "Artist",
				CurrencyCode:    "EUR",
				Percent:         50,
				Level:           1,
			},
		},
		forecastRun: sqlc.ForecastRun{
			ID:        runID,
			CreatedAt: createdAt,
		},
		forecastPoints: []sqlc.ForecastPoint{
			{P10: 100, Yhat: 300, P90: 500},
		},
		currencyRate: map[string]sqlc.CurrencyRate{
			"EUR_USD": {FromCurrencyCode: "EUR", ToCurrencyCode: "USD", Rate: 2},
			"EUR_RUB": {FromCurrencyCode: "EUR", ToCurrencyCode: "RUB", Rate: 100},
		},
	}

	svc := NewService(repo)
	got, err := svc.TeamIncomeForecast(context.Background(), projectID)
	if err != nil {
		t.Fatalf("TeamIncomeForecast returned error: %v", err)
	}

	if !got.ForecastAvailable {
		t.Fatal("expected forecast to be available")
	}
	if got.LatestForecastRunID == nil || *got.LatestForecastRunID != runID {
		t.Fatalf("expected latest forecast run id %s, got %+v", runID, got.LatestForecastRunID)
	}
	if len(got.Members) != 2 {
		t.Fatalf("expected 2 member rows, got %d", len(got.Members))
	}

	var teamRow *TeamIncomeForecastMember
	for i := range got.Members {
		if got.Members[i].UserID == teamID {
			teamRow = &got.Members[i]
			break
		}
	}
	if teamRow == nil {
		t.Fatal("expected team member row to be present")
	}
	if teamRow.CurrentEarned.ProjectAmount != 450 {
		t.Fatalf("expected current project amount 450, got %.2f", teamRow.CurrentEarned.ProjectAmount)
	}
	if teamRow.CurrentEarned.UserAmount != 900 {
		t.Fatalf("expected current user amount 900 USD, got %.2f", teamRow.CurrentEarned.UserAmount)
	}
	if teamRow.PoorScenario == nil || teamRow.PoorScenario.ProjectAmount != 500 {
		t.Fatalf("expected poor scenario project amount 500, got %+v", teamRow.PoorScenario)
	}
	if teamRow.AverageScenario == nil || teamRow.AverageScenario.ProjectAmount != 600 {
		t.Fatalf("expected average scenario project amount 600, got %+v", teamRow.AverageScenario)
	}
	if teamRow.GoodScenario == nil || teamRow.GoodScenario.ProjectAmount != 700 {
		t.Fatalf("expected good scenario project amount 700, got %+v", teamRow.GoodScenario)
	}
	if got.ScenarioModel.Poor.ForecastKey != "p10" || got.ScenarioModel.Average.ForecastKey != "yhat" || got.ScenarioModel.Good.ForecastKey != "p90" {
		t.Fatalf("unexpected scenario model: %+v", got.ScenarioModel)
	}
}

func TestTeamIncomeForecastReturnsCurrentAmountsWithoutForecastRun(t *testing.T) {
	projectID := uuid.New()
	memberID := uuid.New()

	repo := &fakeRepo{
		summary: sqlc.FinanceSummary{
			IncomeTotal:   1000,
			ExpensesTotal: 0,
		},
		project: sqlc.Project{
			ID:           projectID,
			CurrencyCode: "EUR",
		},
		members: []sqlc.ProjectMemberWithUser{
			{
				UserID:              memberID,
				Role:                "TEAM",
				FullName:            "Only Member",
				DefaultCurrencyCode: "EUR",
				Email:               "member@example.com",
			},
		},
		rules: []sqlc.ProfitShareRule{
			{
				ID:              uuid.New(),
				ProjectID:       projectID,
				RecipientUserID: &memberID,
				RecipientName:   "Only Member",
				Responsibility:  "Artist",
				CurrencyCode:    "EUR",
				Percent:         10,
				Level:           1,
			},
		},
	}

	svc := NewService(repo)
	got, err := svc.TeamIncomeForecast(context.Background(), projectID)
	if err != nil {
		t.Fatalf("TeamIncomeForecast returned error: %v", err)
	}
	if got.ForecastAvailable {
		t.Fatal("expected forecast to be unavailable without latest run")
	}
	if len(got.Members) != 1 {
		t.Fatalf("expected 1 member row, got %d", len(got.Members))
	}
	if got.Members[0].CurrentEarned.ProjectAmount != 100 {
		t.Fatalf("expected current project amount 100, got %.2f", got.Members[0].CurrentEarned.ProjectAmount)
	}
	if got.Members[0].PoorScenario != nil || got.Members[0].AverageScenario != nil || got.Members[0].GoodScenario != nil {
		t.Fatalf("expected no forecast scenarios without run, got %+v", got.Members[0])
	}
}
