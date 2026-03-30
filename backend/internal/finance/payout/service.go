package payout

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"isbudget/backend/internal/db/sqlc"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Service struct {
	repo         Repository
	snapshotsDir string
}

func NewService(repo Repository, snapshotsDir ...string) *Service {
	dir := filepath.Join("storage", "payout_snapshots")
	if len(snapshotsDir) > 0 && strings.TrimSpace(snapshotsDir[0]) != "" {
		dir = snapshotsDir[0]
	}
	return &Service{repo: repo, snapshotsDir: dir}
}

var (
	ErrInvalidRuleLevel         = errors.New("уровень должен быть в диапазоне от 1 до 20")
	ErrInvalidRuleLevelSequence = errors.New("нельзя пропускать уровни: сначала создайте промежуточный уровень")
	ErrRulePercentOverflow      = errors.New("сумма процентов для уровня не может превышать 100")
	ErrRuleNextLevelBlocked     = errors.New("невозможно создать следующий уровень: на предыдущем уровне уже распределено 100%")
	ErrInvalidRuleCurrency      = errors.New("currency_code must be one of: EUR, USD, RUB")
	ErrInvalidRecipientUser     = errors.New("recipient_user_id must belong to a project member")
	ErrCurrencyRateUnavailable  = errors.New("currency rate unavailable")
	ErrInvalidSnapshotFileName  = errors.New("invalid snapshot file name")
	ErrSnapshotFileNotFound     = errors.New("snapshot file not found")
)

const MaxProfitShareLevel int32 = 20

type FinancialSummary struct {
	IncomeTotal         float64 `json:"income_total"`
	ExpensesTotal       float64 `json:"expenses_total"`
	PlannedTotal        float64 `json:"planned_total"`
	NetProfitToDate     float64 `json:"net_profit_to_date"`
	MandatoryTotal      float64 `json:"mandatory_total"`
	MandatoryPaid       float64 `json:"mandatory_paid"`
	MandatoryRemaining  float64 `json:"mandatory_remaining"`
	AvailableBeforeTax  float64 `json:"available_before_tax"`
	TaxPercentTotal     float64 `json:"tax_percent_total"`
	TaxTotal            float64 `json:"tax_total"`
	AvailableForSharing float64 `json:"available_for_sharing"`
}

type PreviewItem struct {
	RuleID                uuid.UUID  `json:"rule_id"`
	RecipientUserID       *uuid.UUID `json:"recipient_user_id"`
	RecipientName         string     `json:"recipient_name"`
	Responsibility        string     `json:"responsibility"`
	Level                 int        `json:"level"`
	Percent               float64    `json:"percent"`
	PayoutAmount          float64    `json:"payout_amount"`
	CurrencyCode          string     `json:"currency_code"`
	ConversionRate        float64    `json:"conversion_rate"`
	PayoutAmountConverted float64    `json:"payout_amount_converted"`
}

type PreviewResponse struct {
	Summary                  FinancialSummary `json:"summary"`
	ProjectCurrencyCode      string           `json:"project_currency_code"`
	SecondaryCurrencyCode    string           `json:"secondary_currency_code,omitempty"`
	TaxLevels                []TaxLevelItem   `json:"tax_levels"`
	DistributedAmount        float64          `json:"distributed_amount"`
	DistributedPercent       float64          `json:"distributed_percent"`
	RemainingAmount          float64          `json:"remaining_amount"`
	RemainingPercent         float64          `json:"remaining_percent"`
	RemainingCurrencyCode    string           `json:"remaining_currency_code"`
	RemainingConversionRate  float64          `json:"remaining_conversion_rate"`
	RemainingAmountConverted float64          `json:"remaining_amount_converted"`
	Items                    []PreviewItem    `json:"items"`
}

type TaxLevelItem struct {
	Level          int     `json:"level"`
	BaseAmount     float64 `json:"base_amount"`
	PercentTotal   float64 `json:"percent_total"`
	TaxAmount      float64 `json:"tax_amount"`
	RemainingAfter float64 `json:"remaining_after"`
}

type SnapshotFile struct {
	FileName    string    `json:"file_name"`
	SavedAt     time.Time `json:"saved_at"`
	SizeBytes   int64     `json:"size_bytes"`
	DownloadURL string    `json:"download_url,omitempty"`
}

type MyIncomeItem struct {
	RuleID               uuid.UUID `json:"rule_id"`
	RecipientName        string    `json:"recipient_name"`
	Responsibility       string    `json:"responsibility"`
	Level                int       `json:"level"`
	Percent              float64   `json:"percent"`
	ProjectAmount        float64   `json:"project_amount"`
	ProjectCurrencyCode  string    `json:"project_currency_code"`
	UserAmount           float64   `json:"user_amount"`
	UserCurrencyCode     string    `json:"user_currency_code"`
	ConversionRateToUser float64   `json:"conversion_rate_to_user"`
}

type MyIncomeForecastScenario struct {
	ProjectedIncomeTotal float64 `json:"projected_income_total"`
	ProjectAmount        float64 `json:"project_amount"`
	UserAmount           float64 `json:"user_amount"`
}

type MyIncomeForecastProjection struct {
	RunID     uuid.UUID                `json:"run_id"`
	CreatedAt time.Time                `json:"created_at"`
	P10       MyIncomeForecastScenario `json:"p10"`
	Yhat      MyIncomeForecastScenario `json:"yhat"`
	P90       MyIncomeForecastScenario `json:"p90"`
}

type MyIncomeResponse struct {
	ProjectID           uuid.UUID                   `json:"project_id"`
	ProjectCurrencyCode string                      `json:"project_currency_code"`
	UserCurrencyCode    string                      `json:"user_currency_code"`
	IncomeTotal         float64                     `json:"income_total"`
	ExpensesTotal       float64                     `json:"expenses_total"`
	NetProfitToDate     float64                     `json:"net_profit_to_date"`
	TaxTotal            float64                     `json:"tax_total"`
	AvailableForSharing float64                     `json:"available_for_sharing"`
	DistributedAmount   float64                     `json:"distributed_amount"`
	DistributedToOthers float64                     `json:"distributed_to_others"`
	TotalProjectAmount  float64                     `json:"total_project_amount"`
	TotalUserAmount     float64                     `json:"total_user_amount"`
	TotalConversionRate float64                     `json:"total_conversion_rate"`
	MatchingStrategy    string                      `json:"matching_strategy"`
	ForecastProjection  *MyIncomeForecastProjection `json:"forecast_projection,omitempty"`
	Items               []MyIncomeItem              `json:"items"`
}

type myIncomeComputation struct {
	Items               []MyIncomeItem
	UserCurrencyCode    string
	TotalProjectAmount  float64
	TotalUserAmount     float64
	TotalConversionRate float64
	DistributedToOthers float64
}

func (s *Service) ListRules(ctx context.Context, projectID uuid.UUID) ([]sqlc.ProfitShareRule, error) {
	return s.normalizeRuleLevels(ctx, projectID)
}

func (s *Service) CreateRule(ctx context.Context, req sqlc.CreateProfitShareRuleParams) (sqlc.ProfitShareRule, error) {
	if _, err := s.normalizeRuleLevels(ctx, req.ProjectID); err != nil {
		return sqlc.ProfitShareRule{}, err
	}
	project, err := s.repo.GetProjectByID(ctx, req.ProjectID)
	if err != nil {
		return sqlc.ProfitShareRule{}, err
	}
	req.Level = normalizeRuleLevel(req.Level, req.AppliesAfterMandatory)
	if req.Level < 1 || req.Level > MaxProfitShareLevel {
		return sqlc.ProfitShareRule{}, ErrInvalidRuleLevel
	}
	normalizedCurrency, err := normalizePayoutCurrencyCode(req.CurrencyCode, project.CurrencyCode)
	if err != nil {
		return sqlc.ProfitShareRule{}, err
	}
	req.CurrencyCode = normalizedCurrency
	if err := s.validateRecipientUser(ctx, req.ProjectID, req.RecipientUserID); err != nil {
		return sqlc.ProfitShareRule{}, err
	}
	req.AppliesAfterMandatory = req.Level > 1
	if err := s.validateLevelSequence(ctx, req.ProjectID, req.Level, nil); err != nil {
		return sqlc.ProfitShareRule{}, err
	}
	if err := s.validateNoConsumedPreviousLevel(ctx, req.ProjectID, req.Level, nil); err != nil {
		return sqlc.ProfitShareRule{}, err
	}
	if err := s.validatePercentBudget(ctx, req.ProjectID, req.Level, req.Percent, nil); err != nil {
		return sqlc.ProfitShareRule{}, err
	}
	created, err := s.repo.CreateProfitShareRule(ctx, req)
	if err != nil {
		return sqlc.ProfitShareRule{}, err
	}
	rules, err := s.normalizeRuleLevels(ctx, req.ProjectID)
	if err != nil {
		return sqlc.ProfitShareRule{}, err
	}
	for _, rule := range rules {
		if rule.ID == created.ID {
			return rule, nil
		}
	}
	return created, nil
}

func (s *Service) CreateRulePrepend(ctx context.Context, req sqlc.CreateProfitShareRuleParams) (sqlc.ProfitShareRule, error) {
	if _, err := s.normalizeRuleLevels(ctx, req.ProjectID); err != nil {
		return sqlc.ProfitShareRule{}, err
	}
	project, err := s.repo.GetProjectByID(ctx, req.ProjectID)
	if err != nil {
		return sqlc.ProfitShareRule{}, err
	}
	req.Level = 1
	req.AppliesAfterMandatory = false
	normalizedCurrency, err := normalizePayoutCurrencyCode(req.CurrencyCode, project.CurrencyCode)
	if err != nil {
		return sqlc.ProfitShareRule{}, err
	}
	req.CurrencyCode = normalizedCurrency
	if err := s.validateRecipientUser(ctx, req.ProjectID, req.RecipientUserID); err != nil {
		return sqlc.ProfitShareRule{}, err
	}
	created, err := s.repo.CreateProfitShareRulePrepend(ctx, req)
	if err != nil {
		return sqlc.ProfitShareRule{}, err
	}
	rules, err := s.normalizeRuleLevels(ctx, req.ProjectID)
	if err != nil {
		return sqlc.ProfitShareRule{}, err
	}
	for _, rule := range rules {
		if rule.ID == created.ID {
			return rule, nil
		}
	}
	return created, nil
}

func (s *Service) UpdateRule(ctx context.Context, req sqlc.UpdateProfitShareRuleParams) (sqlc.ProfitShareRule, error) {
	if _, err := s.normalizeRuleLevels(ctx, req.ProjectID); err != nil {
		return sqlc.ProfitShareRule{}, err
	}
	project, err := s.repo.GetProjectByID(ctx, req.ProjectID)
	if err != nil {
		return sqlc.ProfitShareRule{}, err
	}
	req.Level = normalizeRuleLevel(req.Level, req.AppliesAfterMandatory)
	if req.Level < 1 || req.Level > MaxProfitShareLevel {
		return sqlc.ProfitShareRule{}, ErrInvalidRuleLevel
	}
	normalizedCurrency, err := normalizePayoutCurrencyCode(req.CurrencyCode, project.CurrencyCode)
	if err != nil {
		return sqlc.ProfitShareRule{}, err
	}
	req.CurrencyCode = normalizedCurrency
	if err := s.validateRecipientUser(ctx, req.ProjectID, req.RecipientUserID); err != nil {
		return sqlc.ProfitShareRule{}, err
	}
	req.AppliesAfterMandatory = req.Level > 1
	if err := s.validateLevelSequence(ctx, req.ProjectID, req.Level, &req.ID); err != nil {
		return sqlc.ProfitShareRule{}, err
	}
	if err := s.validateNoConsumedPreviousLevel(ctx, req.ProjectID, req.Level, &req.ID); err != nil {
		return sqlc.ProfitShareRule{}, err
	}
	if err := s.validatePercentBudget(ctx, req.ProjectID, req.Level, req.Percent, &req.ID); err != nil {
		return sqlc.ProfitShareRule{}, err
	}
	updated, err := s.repo.UpdateProfitShareRule(ctx, req)
	if err != nil {
		return sqlc.ProfitShareRule{}, err
	}
	rules, err := s.normalizeRuleLevels(ctx, req.ProjectID)
	if err != nil {
		return sqlc.ProfitShareRule{}, err
	}
	for _, rule := range rules {
		if rule.ID == updated.ID {
			return rule, nil
		}
	}
	return updated, nil
}

func (s *Service) DeleteRule(ctx context.Context, req sqlc.DeleteProfitShareRuleParams) error {
	if err := s.repo.DeleteProfitShareRule(ctx, req); err != nil {
		return err
	}
	_, err := s.normalizeRuleLevels(ctx, req.ProjectID)
	return err
}

func (s *Service) Summary(ctx context.Context, projectID uuid.UUID) (FinancialSummary, error) {
	totals, err := s.repo.GetFinanceSummaryByProject(ctx, projectID)
	if err != nil {
		return FinancialSummary{}, err
	}
	taxRules, err := s.repo.ListTaxRulesByProject(ctx, projectID)
	if err != nil {
		return FinancialSummary{}, err
	}
	return ComputeSummary(totals, taxRules), nil
}

func (s *Service) Preview(ctx context.Context, projectID uuid.UUID) (PreviewResponse, error) {
	project, totals, taxRules, rules, err := s.loadPreviewInputs(ctx, projectID)
	if err != nil {
		return PreviewResponse{}, err
	}
	return s.buildPreviewResponse(ctx, project, totals, taxRules, rules)
}

func (s *Service) loadPreviewInputs(ctx context.Context, projectID uuid.UUID) (sqlc.Project, sqlc.FinanceSummary, []sqlc.ProjectTaxRule, []sqlc.ProfitShareRule, error) {
	project, err := s.repo.GetProjectByID(ctx, projectID)
	if err != nil {
		return sqlc.Project{}, sqlc.FinanceSummary{}, nil, nil, err
	}
	totals, err := s.repo.GetFinanceSummaryByProject(ctx, projectID)
	if err != nil {
		return sqlc.Project{}, sqlc.FinanceSummary{}, nil, nil, err
	}
	taxRules, err := s.repo.ListTaxRulesByProject(ctx, projectID)
	if err != nil {
		return sqlc.Project{}, sqlc.FinanceSummary{}, nil, nil, err
	}
	rules, err := s.repo.ListProfitShareRulesByProject(ctx, projectID)
	if err != nil {
		return sqlc.Project{}, sqlc.FinanceSummary{}, nil, nil, err
	}
	return project, totals, taxRules, rules, nil
}

func (s *Service) buildPreviewResponse(
	ctx context.Context,
	project sqlc.Project,
	totals sqlc.FinanceSummary,
	taxRules []sqlc.ProjectTaxRule,
	rules []sqlc.ProfitShareRule,
) (PreviewResponse, error) {
	summary := ComputeSummary(totals, taxRules)
	taxLevels := buildTaxLevelItems(summary.AvailableBeforeTax, taxRules)
	rules = normalizedRulesInMemory(rules)
	items := BuildPreviewItems(summary.AvailableForSharing, rules, project.CurrencyCode)
	distributedAmount, distributedPercent, remainingAmount, remainingPercent := calculateDistributionStats(summary.AvailableForSharing, items)
	userCurrencyCache := make(map[uuid.UUID]string)

	if remainingAmount > 0 && project.CreatedBy != uuid.Nil {
		adminUser, err := s.repo.GetUserByID(ctx, project.CreatedBy)
		if err != nil {
			return PreviewResponse{}, err
		}
		adminName := strings.TrimSpace(adminUser.FullName)
		if adminName == "" {
			adminName = "Project Administrator"
		}
		maxLevel := 0
		for _, rule := range rules {
			level := int(profitRuleLevel(rule))
			if level > maxLevel {
				maxLevel = level
			}
		}
		if maxLevel < 1 {
			maxLevel = 1
		} else {
			maxLevel++
		}
		adminCurrency, err := normalizePayoutCurrencyCode(adminUser.DefaultCurrencyCode, project.CurrencyCode)
		if err != nil {
			adminCurrency = project.CurrencyCode
		}
		userCurrencyCache[project.CreatedBy] = adminCurrency

		adminUserID := project.CreatedBy
		items = append(items, PreviewItem{
			RuleID:                uuid.Nil,
			RecipientUserID:       &adminUserID,
			RecipientName:         adminName,
			Responsibility:        "Administrator",
			Level:                 maxLevel,
			Percent:               remainingPercent,
			PayoutAmount:          remainingAmount,
			CurrencyCode:          adminCurrency,
			ConversionRate:        1,
			PayoutAmountConverted: remainingAmount,
		})
	}

	for idx := range items {
		targetCurrency := items[idx].CurrencyCode
		if items[idx].RecipientUserID != nil {
			recipientUserID := *items[idx].RecipientUserID
			if cachedCurrency, ok := userCurrencyCache[recipientUserID]; ok {
				targetCurrency = cachedCurrency
			} else {
				recipientUser, err := s.repo.GetUserByID(ctx, recipientUserID)
				if err != nil {
					return PreviewResponse{}, err
				}
				resolvedCurrency, err := normalizePayoutCurrencyCode(recipientUser.DefaultCurrencyCode, project.CurrencyCode)
				if err != nil {
					resolvedCurrency = project.CurrencyCode
				}
				userCurrencyCache[recipientUserID] = resolvedCurrency
				targetCurrency = resolvedCurrency
			}
		}
		items[idx].CurrencyCode = targetCurrency
		convertedAmount, rate, err := s.convertAmount(ctx, items[idx].PayoutAmount, project.CurrencyCode, targetCurrency)
		if err != nil {
			return PreviewResponse{}, err
		}
		items[idx].PayoutAmountConverted = convertedAmount
		items[idx].ConversionRate = rate
	}

	distributedAmount, distributedPercent, remainingAmount, remainingPercent = calculateDistributionStats(summary.AvailableForSharing, items)
	remainingCurrency := project.CurrencyCode
	if project.SecondaryCurrencyCode != nil && strings.TrimSpace(*project.SecondaryCurrencyCode) != "" {
		remainingCurrency = strings.ToUpper(strings.TrimSpace(*project.SecondaryCurrencyCode))
	}
	remainingAmountConverted, remainingConversionRate, err := s.convertAmount(ctx, remainingAmount, project.CurrencyCode, remainingCurrency)
	if err != nil {
		return PreviewResponse{}, err
	}

	secondaryCurrencyCode := ""
	if project.SecondaryCurrencyCode != nil {
		secondaryCurrencyCode = strings.ToUpper(strings.TrimSpace(*project.SecondaryCurrencyCode))
	}

	return PreviewResponse{
		Summary:                  summary,
		ProjectCurrencyCode:      project.CurrencyCode,
		SecondaryCurrencyCode:    secondaryCurrencyCode,
		TaxLevels:                taxLevels,
		DistributedAmount:        distributedAmount,
		DistributedPercent:       distributedPercent,
		RemainingAmount:          remainingAmount,
		RemainingPercent:         remainingPercent,
		RemainingCurrencyCode:    remainingCurrency,
		RemainingConversionRate:  remainingConversionRate,
		RemainingAmountConverted: remainingAmountConverted,
		Items:                    items,
	}, nil
}

func (s *Service) normalizeRuleLevels(ctx context.Context, projectID uuid.UUID) ([]sqlc.ProfitShareRule, error) {
	rules, err := s.repo.ListProfitShareRulesByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if len(rules) == 0 {
		return rules, nil
	}

	normalized := normalizedRulesInMemory(rules)
	for _, rule := range normalized {
		var current *sqlc.ProfitShareRule
		for i := range rules {
			if rules[i].ID == rule.ID {
				current = &rules[i]
				break
			}
		}
		if current == nil {
			continue
		}

		changed := current.Level != rule.Level || current.AppliesAfterMandatory != rule.AppliesAfterMandatory
		if !changed {
			continue
		}

		_, err := s.repo.UpdateProfitShareRule(ctx, sqlc.UpdateProfitShareRuleParams{
			ProjectID:             projectID,
			ID:                    rule.ID,
			RecipientUserID:       rule.RecipientUserID,
			RecipientName:         rule.RecipientName,
			Responsibility:        rule.Responsibility,
			CurrencyCode:          rule.CurrencyCode,
			Percent:               rule.Percent,
			Level:                 rule.Level,
			AppliesAfterMandatory: rule.AppliesAfterMandatory,
		})
		if err != nil {
			return nil, err
		}
	}

	return normalized, nil
}

func normalizedRulesInMemory(rules []sqlc.ProfitShareRule) []sqlc.ProfitShareRule {
	if len(rules) == 0 {
		return rules
	}

	out := append([]sqlc.ProfitShareRule(nil), rules...)
	uniqueLevels := make(map[int32]struct{}, len(out))
	for _, rule := range out {
		uniqueLevels[profitRuleLevel(rule)] = struct{}{}
	}

	levels := make([]int32, 0, len(uniqueLevels))
	for level := range uniqueLevels {
		levels = append(levels, level)
	}
	sort.Slice(levels, func(i, j int) bool { return levels[i] < levels[j] })

	levelMap := make(map[int32]int32, len(levels))
	for idx, oldLevel := range levels {
		levelMap[oldLevel] = int32(idx + 1)
	}

	for i := range out {
		oldLevel := profitRuleLevel(out[i])
		newLevel := levelMap[oldLevel]
		out[i].Level = newLevel
		out[i].AppliesAfterMandatory = newLevel > 1
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Level == out[j].Level {
			return out[i].CreatedAt.Before(out[j].CreatedAt)
		}
		return out[i].Level < out[j].Level
	})
	return out
}

func (s *Service) MyIncome(ctx context.Context, projectID, userID uuid.UUID) (MyIncomeResponse, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return MyIncomeResponse{}, err
	}

	project, totals, taxRules, rules, err := s.loadPreviewInputs(ctx, projectID)
	if err != nil {
		return MyIncomeResponse{}, err
	}

	preview, err := s.buildPreviewResponse(ctx, project, totals, taxRules, rules)
	if err != nil {
		return MyIncomeResponse{}, err
	}

	computation, err := s.computeMyIncomeFromPreview(ctx, user, preview)
	if err != nil {
		return MyIncomeResponse{}, err
	}

	forecastProjection, err := s.buildForecastProjection(ctx, project, user, totals, taxRules, rules)
	if err != nil {
		return MyIncomeResponse{}, err
	}

	return MyIncomeResponse{
		ProjectID:           projectID,
		ProjectCurrencyCode: preview.ProjectCurrencyCode,
		UserCurrencyCode:    computation.UserCurrencyCode,
		IncomeTotal:         preview.Summary.IncomeTotal,
		ExpensesTotal:       preview.Summary.ExpensesTotal,
		NetProfitToDate:     preview.Summary.NetProfitToDate,
		TaxTotal:            preview.Summary.TaxTotal,
		AvailableForSharing: preview.Summary.AvailableForSharing,
		DistributedAmount:   preview.DistributedAmount,
		DistributedToOthers: computation.DistributedToOthers,
		TotalProjectAmount:  computation.TotalProjectAmount,
		TotalUserAmount:     computation.TotalUserAmount,
		TotalConversionRate: computation.TotalConversionRate,
		MatchingStrategy:    "strict recipient_user_id == current user id",
		ForecastProjection:  forecastProjection,
		Items:               computation.Items,
	}, nil
}

func (s *Service) computeMyIncomeFromPreview(ctx context.Context, user sqlc.User, preview PreviewResponse) (myIncomeComputation, error) {
	userCurrency, err := normalizePayoutCurrencyCode(user.DefaultCurrencyCode, preview.ProjectCurrencyCode)
	if err != nil {
		userCurrency = preview.ProjectCurrencyCode
	}

	items := make([]MyIncomeItem, 0, len(preview.Items))
	totalProjectAmount := 0.0
	totalUserAmount := 0.0

	for _, item := range preview.Items {
		if item.RecipientUserID == nil || *item.RecipientUserID != user.ID {
			continue
		}

		userAmount, rate, err := s.convertAmount(ctx, item.PayoutAmount, preview.ProjectCurrencyCode, userCurrency)
		if err != nil {
			return myIncomeComputation{}, err
		}

		items = append(items, MyIncomeItem{
			RuleID:               item.RuleID,
			RecipientName:        item.RecipientName,
			Responsibility:       item.Responsibility,
			Level:                item.Level,
			Percent:              item.Percent,
			ProjectAmount:        item.PayoutAmount,
			ProjectCurrencyCode:  preview.ProjectCurrencyCode,
			UserAmount:           userAmount,
			UserCurrencyCode:     userCurrency,
			ConversionRateToUser: rate,
		})
		totalProjectAmount = round2(totalProjectAmount + item.PayoutAmount)
		totalUserAmount = round2(totalUserAmount + userAmount)
	}

	totalRate := 0.0
	if totalProjectAmount > 0 {
		totalRate = round6(totalUserAmount / totalProjectAmount)
	}

	return myIncomeComputation{
		Items:               items,
		UserCurrencyCode:    userCurrency,
		TotalProjectAmount:  totalProjectAmount,
		TotalUserAmount:     totalUserAmount,
		TotalConversionRate: totalRate,
		DistributedToOthers: round2(math.Max(preview.DistributedAmount-totalProjectAmount, 0)),
	}, nil
}

func (s *Service) buildForecastProjection(
	ctx context.Context,
	project sqlc.Project,
	user sqlc.User,
	totals sqlc.FinanceSummary,
	taxRules []sqlc.ProjectTaxRule,
	rules []sqlc.ProfitShareRule,
) (*MyIncomeForecastProjection, error) {
	run, err := s.repo.GetLatestForecastRunByProject(ctx, project.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	points, err := s.repo.ListForecastPointsByRun(ctx, run.ID)
	if err != nil {
		return nil, err
	}

	currentIncome := round2(totals.IncomeTotal)
	sums := sumForecastScenarios(points)

	p10, err := s.buildForecastScenario(ctx, project, user, totals, taxRules, rules, currentIncome+sums.P10)
	if err != nil {
		return nil, err
	}
	yhat, err := s.buildForecastScenario(ctx, project, user, totals, taxRules, rules, currentIncome+sums.Yhat)
	if err != nil {
		return nil, err
	}
	p90, err := s.buildForecastScenario(ctx, project, user, totals, taxRules, rules, currentIncome+sums.P90)
	if err != nil {
		return nil, err
	}

	return &MyIncomeForecastProjection{
		RunID:     run.ID,
		CreatedAt: run.CreatedAt,
		P10:       p10,
		Yhat:      yhat,
		P90:       p90,
	}, nil
}

func (s *Service) buildForecastScenario(
	ctx context.Context,
	project sqlc.Project,
	user sqlc.User,
	totals sqlc.FinanceSummary,
	taxRules []sqlc.ProjectTaxRule,
	rules []sqlc.ProfitShareRule,
	projectedIncomeTotal float64,
) (MyIncomeForecastScenario, error) {
	projectedTotals := totals
	projectedTotals.IncomeTotal = round2(projectedIncomeTotal)

	preview, err := s.buildPreviewResponse(ctx, project, projectedTotals, taxRules, rules)
	if err != nil {
		return MyIncomeForecastScenario{}, err
	}
	computation, err := s.computeMyIncomeFromPreview(ctx, user, preview)
	if err != nil {
		return MyIncomeForecastScenario{}, err
	}

	return MyIncomeForecastScenario{
		ProjectedIncomeTotal: preview.Summary.IncomeTotal,
		ProjectAmount:        computation.TotalProjectAmount,
		UserAmount:           computation.TotalUserAmount,
	}, nil
}

type forecastScenarioSums struct {
	P10  float64
	Yhat float64
	P90  float64
}

func sumForecastScenarios(points []sqlc.ForecastPoint) forecastScenarioSums {
	out := forecastScenarioSums{}
	for _, point := range points {
		out.P10 = round2(out.P10 + point.P10)
		out.Yhat = round2(out.Yhat + point.Yhat)
		out.P90 = round2(out.P90 + point.P90)
	}
	return out
}

func (s *Service) CalculateAndPersist(ctx context.Context, projectID uuid.UUID) ([]sqlc.Payout, PreviewResponse, SnapshotFile, error) {
	preview, err := s.Preview(ctx, projectID)
	if err != nil {
		return nil, PreviewResponse{}, SnapshotFile{}, err
	}
	calculatedAt := time.Now().UTC()
	payouts, err := s.repo.StorePayouts(ctx, projectID, calculatedAt, preview.Items)
	if err != nil {
		return nil, PreviewResponse{}, SnapshotFile{}, err
	}
	snapshot, err := s.saveSnapshotPDF(ctx, projectID, calculatedAt, preview)
	if err != nil {
		return nil, PreviewResponse{}, SnapshotFile{}, err
	}
	return payouts, preview, snapshot, nil
}

func (s *Service) GetSnapshotPath(projectID uuid.UUID, fileName string) (string, error) {
	if fileName == "" || fileName != filepath.Base(fileName) || strings.Contains(fileName, "..") || !strings.HasSuffix(strings.ToLower(fileName), ".pdf") {
		return "", ErrInvalidSnapshotFileName
	}
	path := filepath.Join(s.snapshotsDir, projectID.String(), fileName)
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", ErrSnapshotFileNotFound
		}
		return "", err
	}
	return path, nil
}

func ComputeSummary(totals sqlc.FinanceSummary, taxRules []sqlc.ProjectTaxRule) FinancialSummary {
	netProfit := totals.IncomeTotal - totals.ExpensesTotal
	mandatoryRemaining := totals.MandatoryTotal - totals.MandatoryPaid
	if mandatoryRemaining < 0 {
		mandatoryRemaining = 0
	}

	// Tax rules are applied sequentially by level:
	// each next level is calculated from the remaining amount after previous level.
	availableBeforeTax := netProfit - mandatoryRemaining
	if availableBeforeTax < 0 {
		availableBeforeTax = 0
	}
	taxTotal, available := calculateSequentialTax(availableBeforeTax, taxRules)
	taxPercentTotal := effectiveTaxPercent(availableBeforeTax, taxTotal)

	return FinancialSummary{
		IncomeTotal:         round2(totals.IncomeTotal),
		ExpensesTotal:       round2(totals.ExpensesTotal),
		PlannedTotal:        round2(totals.PlannedTotal),
		NetProfitToDate:     round2(netProfit),
		MandatoryTotal:      round2(totals.MandatoryTotal),
		MandatoryPaid:       round2(totals.MandatoryPaid),
		MandatoryRemaining:  round2(mandatoryRemaining),
		AvailableBeforeTax:  round2(availableBeforeTax),
		TaxPercentTotal:     round2(taxPercentTotal),
		TaxTotal:            round2(taxTotal),
		AvailableForSharing: round2(available),
	}
}

func BuildPreviewItems(availableForSharing float64, rules []sqlc.ProfitShareRule, projectCurrencyCode string) []PreviewItem {
	items := make([]PreviewItem, 0, len(rules))

	manualByLevel := make(map[int32][]sqlc.ProfitShareRule)
	levels := make(map[int32]struct{})
	for _, rule := range rules {
		level := profitRuleLevel(rule)
		manualByLevel[level] = append(manualByLevel[level], rule)
		levels[level] = struct{}{}
	}
	if len(levels) == 0 {
		return items
	}

	levelOrder := make([]int32, 0, len(levels))
	for level := range levels {
		levelOrder = append(levelOrder, level)
	}
	sort.Slice(levelOrder, func(i, j int) bool { return levelOrder[i] < levelOrder[j] })

	for level := range manualByLevel {
		sort.SliceStable(manualByLevel[level], func(i, j int) bool {
			return manualByLevel[level][i].CreatedAt.Before(manualByLevel[level][j].CreatedAt)
		})
	}

	remaining := round2(availableForSharing)
	for _, level := range levelOrder {
		if remaining <= 0 {
			break
		}
		levelBase := remaining

		levelSpent := 0.0
		for _, rule := range manualByLevel[level] {
			payoutAmount := round2(levelBase * (rule.Percent / 100.0))
			ruleCurrencyCode, err := normalizePayoutCurrencyCode(rule.CurrencyCode, projectCurrencyCode)
			if err != nil {
				ruleCurrencyCode = projectCurrencyCode
			}
			items = append(items, PreviewItem{
				RuleID:                rule.ID,
				RecipientUserID:       rule.RecipientUserID,
				RecipientName:         rule.RecipientName,
				Responsibility:        rule.Responsibility,
				Level:                 int(level),
				Percent:               round2(rule.Percent),
				PayoutAmount:          payoutAmount,
				CurrencyCode:          ruleCurrencyCode,
				ConversionRate:        1,
				PayoutAmountConverted: payoutAmount,
			})
			levelSpent = round2(levelSpent + payoutAmount)
		}

		if levelSpent > remaining {
			levelSpent = remaining
		}
		remaining = round2(remaining - levelSpent)
		if remaining < 0 {
			remaining = 0
		}
	}
	return items
}

func normalizePayoutCurrencyCode(code, fallback string) (string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(code))
	if normalized == "" {
		normalized = strings.ToUpper(strings.TrimSpace(fallback))
	}
	switch normalized {
	case "EUR", "USD", "RUB":
		return normalized, nil
	default:
		return "", ErrInvalidRuleCurrency
	}
}

func calculateDistributionStats(availableForSharing float64, items []PreviewItem) (distributedAmount, distributedPercent, remainingAmount, remainingPercent float64) {
	distributedAmount = 0
	for _, item := range items {
		distributedAmount = round2(distributedAmount + item.PayoutAmount)
	}
	remainingAmount = round2(availableForSharing - distributedAmount)
	if remainingAmount < 0 {
		remainingAmount = 0
	}
	if availableForSharing > 0 {
		distributedPercent = round2((distributedAmount / availableForSharing) * 100)
		remainingPercent = round2((remainingAmount / availableForSharing) * 100)
	}
	return distributedAmount, distributedPercent, remainingAmount, remainingPercent
}

func (s *Service) convertAmount(ctx context.Context, amount float64, fromCurrencyCode, toCurrencyCode string) (float64, float64, error) {
	if round2(amount) == 0 {
		return 0, 1, nil
	}

	from := strings.ToUpper(strings.TrimSpace(fromCurrencyCode))
	to := strings.ToUpper(strings.TrimSpace(toCurrencyCode))
	if from == "" || to == "" {
		return 0, 0, ErrInvalidRuleCurrency
	}
	if from == to {
		return round2(amount), 1, nil
	}

	rate, err := s.repo.GetCurrencyRate(ctx, from, to)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, 0, fmt.Errorf("%w: %s/%s", ErrCurrencyRateUnavailable, from, to)
		}
		return 0, 0, err
	}
	converted := round2(amount * rate.Rate)
	return converted, round6(rate.Rate), nil
}

func normalizeRuleLevel(level int32, appliesAfterMandatory bool) int32 {
	if level >= 1 {
		return level
	}
	if appliesAfterMandatory {
		return 2
	}
	return 1
}

func shiftProfitShareRulesForPrepend(rules []sqlc.ProfitShareRule) []sqlc.ProfitShareRule {
	shifted := make([]sqlc.ProfitShareRule, 0, len(rules))
	for _, rule := range rules {
		next := rule
		next.Level = rule.Level + 1
		next.AppliesAfterMandatory = true
		shifted = append(shifted, next)
	}
	return shifted
}

func profitRuleLevel(rule sqlc.ProfitShareRule) int32 {
	if rule.Level >= 1 {
		return rule.Level
	}
	if rule.AppliesAfterMandatory {
		return 2
	}
	return 1
}

func (s *Service) validateRecipientUser(ctx context.Context, projectID uuid.UUID, recipientUserID *uuid.UUID) error {
	if recipientUserID == nil {
		return nil
	}
	_, err := s.repo.GetProjectMemberByUser(ctx, sqlc.GetProjectMemberByUserParams{
		ProjectID: projectID,
		UserID:    *recipientUserID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrInvalidRecipientUser
		}
		return err
	}
	return nil
}

func (s *Service) validateLevelSequence(ctx context.Context, projectID uuid.UUID, level int32, excludeID *uuid.UUID) error {
	rules, err := s.repo.ListProfitShareRulesByProject(ctx, projectID)
	if err != nil {
		return err
	}

	maxLevel := int32(0)
	for _, rule := range rules {
		if excludeID != nil && rule.ID == *excludeID {
			continue
		}
		ruleLevel := profitRuleLevel(rule)
		if ruleLevel > maxLevel {
			maxLevel = ruleLevel
		}
	}

	if level > maxLevel+1 {
		return ErrInvalidRuleLevelSequence
	}
	return nil
}

func (s *Service) validateNoConsumedPreviousLevel(ctx context.Context, projectID uuid.UUID, level int32, excludeID *uuid.UUID) error {
	if level <= 1 {
		return nil
	}
	rules, err := s.repo.ListProfitShareRulesByProject(ctx, projectID)
	if err != nil {
		return err
	}

	percentsByLevel := make(map[int32]float64, len(rules))
	for _, rule := range rules {
		if excludeID != nil && rule.ID == *excludeID {
			continue
		}
		ruleLevel := profitRuleLevel(rule)
		percentsByLevel[ruleLevel] += rule.Percent
	}

	for previous := int32(1); previous < level; previous++ {
		if percentsByLevel[previous] >= 100.0-0.000001 {
			return fmt.Errorf("%w (уровень %d)", ErrRuleNextLevelBlocked, previous)
		}
	}
	return nil
}

func (s *Service) validatePercentBudget(ctx context.Context, projectID uuid.UUID, level int32, percent float64, excludeID *uuid.UUID) error {
	rules, err := s.repo.ListProfitShareRulesByProject(ctx, projectID)
	if err != nil {
		return err
	}

	sum := 0.0
	for _, rule := range rules {
		if excludeID != nil && rule.ID == *excludeID {
			continue
		}
		if profitRuleLevel(rule) == level {
			sum += rule.Percent
		}
	}

	if sum+percent > 100.000001 {
		return fmt.Errorf("%w (уровень %d)", ErrRulePercentOverflow, level)
	}
	return nil
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func round6(v float64) float64 {
	return math.Round(v*1_000_000) / 1_000_000
}

func calculateSequentialTax(baseAmount float64, rules []sqlc.ProjectTaxRule) (float64, float64) {
	remaining := round2(baseAmount)
	totalTax := 0.0

	if len(rules) == 0 || remaining <= 0 {
		return totalTax, remaining
	}

	sortedRules := append([]sqlc.ProjectTaxRule(nil), rules...)
	sort.SliceStable(sortedRules, func(i, j int) bool {
		if sortedRules[i].Level == sortedRules[j].Level {
			return sortedRules[i].CreatedAt.Before(sortedRules[j].CreatedAt)
		}
		return sortedRules[i].Level < sortedRules[j].Level
	})

	for i := 0; i < len(sortedRules); {
		if remaining <= 0 {
			break
		}

		level := sortedRules[i].Level
		levelBase := remaining
		levelTax := 0.0

		for i < len(sortedRules) && sortedRules[i].Level == level {
			taxAmount := round2(levelBase * (sortedRules[i].Percent / 100.0))
			levelTax = round2(levelTax + taxAmount)
			i++
		}

		if levelTax > remaining {
			levelTax = remaining
		}

		totalTax = round2(totalTax + levelTax)
		remaining = round2(remaining - levelTax)
		if remaining < 0 {
			remaining = 0
		}
	}

	return totalTax, remaining
}

func buildTaxLevelItems(baseAmount float64, rules []sqlc.ProjectTaxRule) []TaxLevelItem {
	remaining := round2(baseAmount)
	if len(rules) == 0 || remaining <= 0 {
		return []TaxLevelItem{}
	}

	sortedRules := append([]sqlc.ProjectTaxRule(nil), rules...)
	sort.SliceStable(sortedRules, func(i, j int) bool {
		leftLevel := sortedRules[i].Level
		if leftLevel <= 0 {
			leftLevel = 1
		}
		rightLevel := sortedRules[j].Level
		if rightLevel <= 0 {
			rightLevel = 1
		}
		if leftLevel == rightLevel {
			return sortedRules[i].CreatedAt.Before(sortedRules[j].CreatedAt)
		}
		return leftLevel < rightLevel
	})

	out := make([]TaxLevelItem, 0, len(sortedRules))
	for i := 0; i < len(sortedRules); {
		level := sortedRules[i].Level
		if level <= 0 {
			level = 1
		}
		levelPercent := 0.0
		for i < len(sortedRules) {
			nextLevel := sortedRules[i].Level
			if nextLevel <= 0 {
				nextLevel = 1
			}
			if nextLevel != level {
				break
			}
			levelPercent += sortedRules[i].Percent
			i++
		}

		levelBase := remaining
		levelTax := round2(levelBase * (levelPercent / 100.0))
		if levelTax > remaining {
			levelTax = remaining
		}
		remaining = round2(remaining - levelTax)
		if remaining < 0 {
			remaining = 0
		}

		out = append(out, TaxLevelItem{
			Level:          int(level),
			BaseAmount:     round2(levelBase),
			PercentTotal:   round2(levelPercent),
			TaxAmount:      round2(levelTax),
			RemainingAfter: round2(remaining),
		})
		if remaining <= 0 {
			break
		}
	}
	return out
}

func effectiveTaxPercent(baseAmount, taxTotal float64) float64 {
	if baseAmount <= 0 || taxTotal <= 0 {
		return 0
	}
	return round2((taxTotal / baseAmount) * 100)
}
