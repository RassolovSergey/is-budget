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
	summary      sqlc.FinanceSummary
	rules        []sqlc.ProfitShareRule
	taxRules     []sqlc.ProjectTaxRule
	project      sqlc.Project
	user         sqlc.User
	members      []sqlc.ProjectMemberWithUser
	currencyRate map[string]sqlc.CurrencyRate
	lastCreate   sqlc.CreateProfitShareRuleParams
	createCalled bool
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
	}
	svc := NewService(repo)

	_, err := svc.CreateRule(context.Background(), sqlc.CreateProfitShareRuleParams{
		ProjectID:             projectID,
		RecipientName:         "Overflow",
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
	}
	svc := NewService(repo)

	_, err := svc.CreateRule(context.Background(), sqlc.CreateProfitShareRuleParams{
		ProjectID:             projectID,
		RecipientName:         "Level2 Allowed",
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

func TestCreateRuleRejectsSkippedLevels(t *testing.T) {
	projectID := uuid.New()
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
	}
	svc := NewService(repo)

	_, err := svc.CreateRule(context.Background(), sqlc.CreateProfitShareRuleParams{
		ProjectID:      projectID,
		RecipientName:  "Skip to level 3",
		Responsibility: "Role",
		Percent:        10,
		Level:          3,
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
	}
	svc := NewService(repo)

	_, err := svc.CreateRule(context.Background(), sqlc.CreateProfitShareRuleParams{
		ProjectID:      projectID,
		RecipientName:  "Blocked",
		Responsibility: "Role",
		Percent:        10,
		Level:          2,
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
	repo := &fakeRepo{}
	svc := NewService(repo)

	_, err := svc.CreateRule(context.Background(), sqlc.CreateProfitShareRuleParams{
		ProjectID:      projectID,
		RecipientName:  "Too high",
		Responsibility: "Role",
		Percent:        10,
		Level:          MaxProfitShareLevel + 1,
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
	repo := &fakeRepo{
		project: sqlc.Project{
			ID:           projectID,
			Name:         "Demo",
			CurrencyCode: "EUR",
		},
	}
	svc := NewService(repo)

	_, err := svc.CreateRule(context.Background(), sqlc.CreateProfitShareRuleParams{
		ProjectID:      projectID,
		RecipientName:  "Bob",
		Responsibility: "Artist",
		Percent:        10,
		Level:          1,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repo.lastCreate.CurrencyCode != "EUR" {
		t.Fatalf("expected default currency EUR, got %s", repo.lastCreate.CurrencyCode)
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
}
