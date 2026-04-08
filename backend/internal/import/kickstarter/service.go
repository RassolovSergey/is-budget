package kickstarter

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"isbudget/backend/internal/auth"
	"isbudget/backend/internal/db/sqlc"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	ImportTypeKickstarterCSV = "kickstarter_csv"
	StatusPending            = "pending"
	StatusSuccess            = "success"
	StatusFailed             = "failed"
)

var (
	ErrImportReadDenied            = errors.New("import history requires project membership")
	ErrImportWriteDenied           = errors.New("kickstarter import requires project manager permissions")
	ErrImportRunNotFound           = errors.New("import run not found")
	ErrImportConflictExistingIncome = errors.New("existing income records would be overwritten; retry with replace_existing")
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ImportCSV(ctx context.Context, actorID, projectID uuid.UUID, fileName string, content []byte, replaceExisting bool) (ImportRunDetail, error) {
	if err := s.assertProjectWrite(ctx, actorID, projectID); err != nil {
		return ImportRunDetail{}, err
	}
	user, err := s.repo.GetUserByID(ctx, actorID)
	if err != nil {
		return ImportRunDetail{}, err
	}
	run, err := s.repo.CreateImportRun(ctx, sqlc.CreateImportRunParams{
		ProjectID:      projectID,
		UploadedBy:     actorID,
		UploadedByName: formatUserDisplayName(user),
		ImportType:     ImportTypeKickstarterCSV,
		FileName:       strings.TrimSpace(fileName),
		Status:         StatusPending,
	})
	if err != nil {
		return ImportRunDetail{}, err
	}

	result, parseErr := parseKickstarterCSV(content)
	for _, row := range result.Rows {
		_, _ = s.repo.CreateImportRecord(ctx, sqlc.CreateImportRecordParams{
			RunID:             run.ID,
			RowNumber:         row.RowNumber,
			Status:            row.Status,
			ErrorMessage:      row.ErrorMessage,
			PledgedAt:         row.PledgedAt,
			PledgedAmount:     row.PledgedAmount,
			BackerIdentifier:  row.BackerIdentifier,
			RewardTier:        row.RewardTier,
		})
	}
	if parseErr != nil {
		updated, updateErr := s.repo.UpdateImportRunResult(ctx, sqlc.UpdateImportRunResultParams{
			ID:               run.ID,
			Status:           StatusFailed,
			ErrorMessage:     ptrString(parseErr.Error()),
			RowsProcessed:    result.RowsProcessed,
			RowsSucceeded:    result.RowsSucceeded,
			RowsFailed:       result.RowsFailed,
			RowsSkipped:      result.RowsSkipped,
			TotalPledged:     result.TotalPledged,
			BackerCount:      result.BackerCount,
			RewardTiersCount: result.RewardTiersCount,
			CompletedAt:      ptrTime(time.Now().UTC()),
		})
		if updateErr != nil {
			return ImportRunDetail{}, updateErr
		}
		detail, detailErr := s.buildDetail(ctx, updated)
		if detailErr != nil {
			return ImportRunDetail{}, detailErr
		}
		return detail, parseErr
	}

	if err := s.assertNoConflicts(ctx, projectID, result.DailyAmounts, replaceExisting); err != nil {
		updated, updateErr := s.repo.UpdateImportRunResult(ctx, sqlc.UpdateImportRunResultParams{
			ID:               run.ID,
			Status:           StatusFailed,
			ErrorMessage:     ptrString(err.Error()),
			RowsProcessed:    result.RowsProcessed,
			RowsSucceeded:    result.RowsSucceeded,
			RowsFailed:       result.RowsFailed,
			RowsSkipped:      result.RowsSkipped,
			TotalPledged:     result.TotalPledged,
			BackerCount:      result.BackerCount,
			RewardTiersCount: result.RewardTiersCount,
			CompletedAt:      ptrTime(time.Now().UTC()),
		})
		if updateErr != nil {
			return ImportRunDetail{}, updateErr
		}
		detail, detailErr := s.buildDetail(ctx, updated)
		if detailErr != nil {
			return ImportRunDetail{}, detailErr
		}
		return detail, err
	}

	for _, date := range sortedImportDates(result.DailyAmounts) {
		if _, err := s.repo.UpsertIncomeDaily(ctx, sqlc.UpsertIncomeDailyParams{
			ProjectID:  projectID,
			IncomeDate: date,
			Amount:     result.DailyAmounts[date],
		}); err != nil {
			return ImportRunDetail{}, err
		}
	}

	updated, err := s.repo.UpdateImportRunResult(ctx, sqlc.UpdateImportRunResultParams{
		ID:               run.ID,
		Status:           StatusSuccess,
		ErrorMessage:     nil,
		RowsProcessed:    result.RowsProcessed,
		RowsSucceeded:    result.RowsSucceeded,
		RowsFailed:       result.RowsFailed,
		RowsSkipped:      result.RowsSkipped,
		TotalPledged:     result.TotalPledged,
		BackerCount:      result.BackerCount,
		RewardTiersCount: result.RewardTiersCount,
		CompletedAt:      ptrTime(time.Now().UTC()),
	})
	if err != nil {
		return ImportRunDetail{}, err
	}
	return s.buildDetail(ctx, updated)
}

func (s *Service) ListRuns(ctx context.Context, actorID, projectID uuid.UUID) ([]ImportRunSummary, error) {
	if err := s.assertProjectRead(ctx, actorID, projectID); err != nil {
		return nil, err
	}
	rows, err := s.repo.ListImportRunsByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	items := make([]ImportRunSummary, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapImportRun(row))
	}
	return items, nil
}

func (s *Service) GetRun(ctx context.Context, actorID, runID uuid.UUID) (ImportRunDetail, error) {
	run, err := s.repo.GetImportRunByID(ctx, runID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ImportRunDetail{}, ErrImportRunNotFound
		}
		return ImportRunDetail{}, err
	}
	if err := s.assertProjectRead(ctx, actorID, run.ProjectID); err != nil {
		return ImportRunDetail{}, err
	}
	return s.buildDetail(ctx, run)
}

func (s *Service) buildDetail(ctx context.Context, run sqlc.ImportRun) (ImportRunDetail, error) {
	rows, err := s.repo.ListImportRecordsByRun(ctx, run.ID)
	if err != nil {
		return ImportRunDetail{}, err
	}
	records := make([]ImportRecordResponse, 0, len(rows))
	for _, row := range rows {
		records = append(records, ImportRecordResponse{
			ID:               row.ID.String(),
			RunID:            row.RunID.String(),
			RowNumber:        row.RowNumber,
			Status:           row.Status,
			ErrorMessage:     row.ErrorMessage,
			PledgedAt:        row.PledgedAt,
			PledgedAmount:    row.PledgedAmount,
			BackerIdentifier: row.BackerIdentifier,
			RewardTier:       row.RewardTier,
			CreatedAt:        row.CreatedAt,
		})
	}
	return ImportRunDetail{Run: mapImportRun(run), Records: records}, nil
}

func (s *Service) assertNoConflicts(ctx context.Context, projectID uuid.UUID, imported map[time.Time]float64, replaceExisting bool) error {
	if replaceExisting || len(imported) == 0 {
		return nil
	}
	existingRows, err := s.repo.ListIncomeDailyByProject(ctx, projectID)
	if err != nil {
		return err
	}
	existing := make(map[string]float64, len(existingRows))
	for _, row := range existingRows {
		existing[normalizeDate(row.IncomeDate).Format("2006-01-02")] = round2(row.Amount)
	}
	for date, amount := range imported {
		key := normalizeDate(date).Format("2006-01-02")
		if current, ok := existing[key]; ok && round2(current) != round2(amount) {
			return fmt.Errorf("%w: %s", ErrImportConflictExistingIncome, key)
		}
	}
	return nil
}

func (s *Service) assertProjectRead(ctx context.Context, actorID, projectID uuid.UUID) error {
	member, err := s.repo.GetProjectMemberByUser(ctx, sqlc.GetProjectMemberByUserParams{
		ProjectID: projectID,
		UserID:    actorID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrImportReadDenied
		}
		return err
	}
	if !auth.IsAllowed(string(member.Role), auth.PermProjectRead) {
		return ErrImportReadDenied
	}
	return nil
}

func (s *Service) assertProjectWrite(ctx context.Context, actorID, projectID uuid.UUID) error {
	member, err := s.repo.GetProjectMemberByUser(ctx, sqlc.GetProjectMemberByUserParams{
		ProjectID: projectID,
		UserID:    actorID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrImportWriteDenied
		}
		return err
	}
	if !auth.IsAllowed(string(member.Role), auth.PermProjectWrite) {
		return ErrImportWriteDenied
	}
	return nil
}

func mapImportRun(row sqlc.ImportRun) ImportRunSummary {
	return ImportRunSummary{
		ID:               row.ID.String(),
		ProjectID:        row.ProjectID.String(),
		UploadedBy:       row.UploadedBy.String(),
		UploadedByName:   row.UploadedByName,
		ImportType:       row.ImportType,
		FileName:         row.FileName,
		Status:           row.Status,
		ErrorMessage:     row.ErrorMessage,
		RowsProcessed:    row.RowsProcessed,
		RowsSucceeded:    row.RowsSucceeded,
		RowsFailed:       row.RowsFailed,
		RowsSkipped:      row.RowsSkipped,
		TotalPledged:     row.TotalPledged,
		BackerCount:      row.BackerCount,
		RewardTiersCount: row.RewardTiersCount,
		CreatedAt:        row.CreatedAt,
		CompletedAt:      row.CompletedAt,
	}
}

func formatUserDisplayName(user sqlc.GetUserByIDRow) string {
	full := strings.TrimSpace(user.FullName)
	if full != "" {
		return full
	}
	name := strings.TrimSpace(strings.TrimSpace(user.FirstName) + " " + strings.TrimSpace(user.LastName))
	if name != "" {
		return name
	}
	if strings.TrimSpace(user.Nickname) != "" {
		return strings.TrimSpace(user.Nickname)
	}
	return strings.TrimSpace(user.Email)
}

func ptrString(value string) *string { return &value }
func ptrTime(value time.Time) *time.Time { return &value }
