package kickstarter

import (
	"context"
	"errors"
	"testing"
	"time"

	"isbudget/backend/internal/auth"
	"isbudget/backend/internal/db/sqlc"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type fakeRepo struct {
	memberByUser map[uuid.UUID]sqlc.ProjectMember
	user         sqlc.GetUserByIDRow
	runs         []sqlc.ImportRun
	records      map[uuid.UUID][]sqlc.ImportRecord
	incomes      map[string]float64
}

func (f *fakeRepo) GetProjectMemberByUser(ctx context.Context, arg sqlc.GetProjectMemberByUserParams) (sqlc.ProjectMember, error) {
	member, ok := f.memberByUser[arg.UserID]
	if !ok || member.ProjectID != arg.ProjectID {
		return sqlc.ProjectMember{}, pgx.ErrNoRows
	}
	return member, nil
}

func (f *fakeRepo) GetUserByID(ctx context.Context, id uuid.UUID) (sqlc.GetUserByIDRow, error) {
	if f.user.ID != id {
		return sqlc.GetUserByIDRow{}, pgx.ErrNoRows
	}
	return f.user, nil
}

func (f *fakeRepo) CreateImportRun(ctx context.Context, arg sqlc.CreateImportRunParams) (sqlc.ImportRun, error) {
	row := sqlc.ImportRun{
		ID:             uuid.New(),
		ProjectID:      arg.ProjectID,
		UploadedBy:     arg.UploadedBy,
		UploadedByName: arg.UploadedByName,
		ImportType:     arg.ImportType,
		FileName:       arg.FileName,
		Status:         arg.Status,
		CreatedAt:      time.Now().UTC(),
	}
	f.runs = append(f.runs, row)
	return row, nil
}

func (f *fakeRepo) UpdateImportRunResult(ctx context.Context, arg sqlc.UpdateImportRunResultParams) (sqlc.ImportRun, error) {
	for i := range f.runs {
		if f.runs[i].ID == arg.ID {
			f.runs[i].Status = arg.Status
			f.runs[i].ErrorMessage = arg.ErrorMessage
			f.runs[i].RowsProcessed = arg.RowsProcessed
			f.runs[i].RowsSucceeded = arg.RowsSucceeded
			f.runs[i].RowsFailed = arg.RowsFailed
			f.runs[i].RowsSkipped = arg.RowsSkipped
			f.runs[i].TotalPledged = arg.TotalPledged
			f.runs[i].BackerCount = arg.BackerCount
			f.runs[i].RewardTiersCount = arg.RewardTiersCount
			f.runs[i].CompletedAt = arg.CompletedAt
			return f.runs[i], nil
		}
	}
	return sqlc.ImportRun{}, pgx.ErrNoRows
}

func (f *fakeRepo) CreateImportRecord(ctx context.Context, arg sqlc.CreateImportRecordParams) (sqlc.ImportRecord, error) {
	row := sqlc.ImportRecord{
		ID:               uuid.New(),
		RunID:            arg.RunID,
		RowNumber:        arg.RowNumber,
		Status:           arg.Status,
		ErrorMessage:     arg.ErrorMessage,
		PledgedAt:        arg.PledgedAt,
		PledgedAmount:    arg.PledgedAmount,
		BackerIdentifier: arg.BackerIdentifier,
		RewardTier:       arg.RewardTier,
		CreatedAt:        time.Now().UTC(),
	}
	if f.records == nil {
		f.records = map[uuid.UUID][]sqlc.ImportRecord{}
	}
	f.records[arg.RunID] = append(f.records[arg.RunID], row)
	return row, nil
}

func (f *fakeRepo) ListImportRunsByProject(ctx context.Context, projectID uuid.UUID) ([]sqlc.ImportRun, error) {
	items := make([]sqlc.ImportRun, 0)
	for _, run := range f.runs {
		if run.ProjectID == projectID {
			items = append(items, run)
		}
	}
	return items, nil
}

func (f *fakeRepo) GetImportRunByID(ctx context.Context, id uuid.UUID) (sqlc.ImportRun, error) {
	for _, run := range f.runs {
		if run.ID == id {
			return run, nil
		}
	}
	return sqlc.ImportRun{}, pgx.ErrNoRows
}

func (f *fakeRepo) ListImportRecordsByRun(ctx context.Context, runID uuid.UUID) ([]sqlc.ImportRecord, error) {
	return append([]sqlc.ImportRecord(nil), f.records[runID]...), nil
}

func (f *fakeRepo) ListIncomeDailyByProject(ctx context.Context, projectID uuid.UUID) ([]sqlc.ListIncomeDailyByProjectRow, error) {
	rows := make([]sqlc.ListIncomeDailyByProjectRow, 0, len(f.incomes))
	for key, amount := range f.incomes {
		date, _ := time.Parse("2006-01-02", key)
		rows = append(rows, sqlc.ListIncomeDailyByProjectRow{
			ID:         uuid.New(),
			ProjectID:  projectID,
			IncomeDate: date,
			Amount:     amount,
			CreatedAt:  time.Now().UTC(),
			UpdatedAt:  time.Now().UTC(),
		})
	}
	return rows, nil
}

func (f *fakeRepo) UpsertIncomeDaily(ctx context.Context, arg sqlc.UpsertIncomeDailyParams) (sqlc.UpsertIncomeDailyRow, error) {
	if f.incomes == nil {
		f.incomes = map[string]float64{}
	}
	key := normalizeDate(arg.IncomeDate).Format("2006-01-02")
	f.incomes[key] = arg.Amount
	return sqlc.UpsertIncomeDailyRow{
		ID:         uuid.New(),
		ProjectID:  arg.ProjectID,
		IncomeDate: arg.IncomeDate,
		Amount:     arg.Amount,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}, nil
}

func TestValidCSVParsingAndImport(t *testing.T) {
	projectID := uuid.New()
	userID := uuid.New()
	repo := newFakeRepo(projectID, userID, "PM")
	svc := NewService(repo)

	csv := "Pledge Amount,Pledged At (UTC),Pledged Status,Backer UID,Reward Title\n" +
		"€10.00,2026-02-01 10:00:00 UTC,collected,u1,Core Set\n" +
		"€15.50,2026-02-01 11:00:00 UTC,collected,u2,Core Set\n" +
		"€20.00,2026-02-02 09:00:00 UTC,collected,u3,Deluxe\n"

	detail, err := svc.ImportCSV(context.Background(), userID, projectID, "kickstarter.csv", []byte(csv), false)
	if err != nil {
		t.Fatalf("ImportCSV() error = %v", err)
	}
	if detail.Run.Status != StatusSuccess {
		t.Fatalf("expected success status, got %s", detail.Run.Status)
	}
	if got := repo.incomes["2026-02-01"]; got != 25.5 {
		t.Fatalf("expected 2026-02-01 amount 25.5, got %v", got)
	}
	if got := repo.incomes["2026-02-02"]; got != 20 {
		t.Fatalf("expected 2026-02-02 amount 20, got %v", got)
	}
	if detail.Run.BackerCount == nil || *detail.Run.BackerCount != 3 {
		t.Fatalf("expected backer count 3, got %+v", detail.Run.BackerCount)
	}
}

func TestInvalidCSVHandling(t *testing.T) {
	projectID := uuid.New()
	userID := uuid.New()
	repo := newFakeRepo(projectID, userID, "PM")
	svc := NewService(repo)

	_, err := svc.ImportCSV(context.Background(), userID, projectID, "bad.csv", []byte("foo,bar\n1,2\n"), false)
	if !errors.Is(err, ErrInvalidCSVStructure) {
		t.Fatalf("expected invalid csv structure, got %v", err)
	}
	if len(repo.runs) != 1 || repo.runs[0].Status != StatusFailed {
		t.Fatalf("expected failed import run to be stored, got %+v", repo.runs)
	}
}

func TestPartialFailuresStillImportValidRows(t *testing.T) {
	projectID := uuid.New()
	userID := uuid.New()
	repo := newFakeRepo(projectID, userID, "PM")
	svc := NewService(repo)

	csv := "Pledge Amount,Pledged At (UTC),Pledged Status\n" +
		"€10.00,2026-02-01 10:00:00 UTC,collected\n" +
		"oops,2026-02-01 11:00:00 UTC,collected\n" +
		"€20.00,2026-02-03 09:00:00 UTC,cancelled\n"

	detail, err := svc.ImportCSV(context.Background(), userID, projectID, "partial.csv", []byte(csv), false)
	if err != nil {
		t.Fatalf("expected partial import to succeed, got %v", err)
	}
	if detail.Run.RowsSucceeded != 1 || detail.Run.RowsFailed != 1 || detail.Run.RowsSkipped != 1 {
		t.Fatalf("unexpected row counters: %+v", detail.Run)
	}
	if got := repo.incomes["2026-02-01"]; got != 10 {
		t.Fatalf("expected valid row to import, got %v", got)
	}
}

func TestDuplicateImportsAreIdempotentWhenAmountsMatch(t *testing.T) {
	projectID := uuid.New()
	userID := uuid.New()
	repo := newFakeRepo(projectID, userID, "PM")
	svc := NewService(repo)
	csv := "Pledge Amount,Pledged At (UTC),Pledged Status\n€10.00,2026-02-01 10:00:00 UTC,collected\n"

	if _, err := svc.ImportCSV(context.Background(), userID, projectID, "a.csv", []byte(csv), false); err != nil {
		t.Fatalf("first import failed: %v", err)
	}
	if _, err := svc.ImportCSV(context.Background(), userID, projectID, "a.csv", []byte(csv), false); err != nil {
		t.Fatalf("second import should be idempotent, got %v", err)
	}
}

func TestConflictingImportRequiresReplaceExisting(t *testing.T) {
	projectID := uuid.New()
	userID := uuid.New()
	repo := newFakeRepo(projectID, userID, "PM")
	repo.incomes = map[string]float64{"2026-02-01": 12}
	svc := NewService(repo)
	csv := "Pledge Amount,Pledged At (UTC),Pledged Status\n€10.00,2026-02-01 10:00:00 UTC,collected\n"

	if _, err := svc.ImportCSV(context.Background(), userID, projectID, "conflict.csv", []byte(csv), false); !errors.Is(err, ErrImportConflictExistingIncome) {
		t.Fatalf("expected conflict, got %v", err)
	}
	if _, err := svc.ImportCSV(context.Background(), userID, projectID, "conflict.csv", []byte(csv), true); err != nil {
		t.Fatalf("expected replace_existing import to succeed, got %v", err)
	}
	if got := repo.incomes["2026-02-01"]; got != 10 {
		t.Fatalf("expected replace_existing to overwrite target date, got %v", got)
	}
}

func TestImportWritePermissions(t *testing.T) {
	projectID := uuid.New()
	userID := uuid.New()
	repo := newFakeRepo(projectID, userID, "TEAM")
	svc := NewService(repo)

	_, err := svc.ImportCSV(context.Background(), userID, projectID, "x.csv", []byte("Pledge Amount,Pledged At (UTC)\n€10,2026-02-01\n"), false)
	if !errors.Is(err, ErrImportWriteDenied) {
		t.Fatalf("expected write denied, got %v", err)
	}
}

func newFakeRepo(projectID, userID uuid.UUID, role string) *fakeRepo {
	return &fakeRepo{
		user: sqlc.GetUserByIDRow{
			ID:        userID,
			Email:     "pm@example.com",
			FullName:  "PM User",
			FirstName: "PM",
			LastName:  "User",
		},
		memberByUser: map[uuid.UUID]sqlc.ProjectMember{
			userID: {ProjectID: projectID, UserID: userID, Role: sqlc.ProjectRole(role)},
		},
		records: map[uuid.UUID][]sqlc.ImportRecord{},
		incomes: map[string]float64{},
	}
}

func TestRoleContractStillMatchesPMExpectation(t *testing.T) {
	if !auth.IsAllowed("PM", auth.PermProjectWrite) {
		t.Fatal("expected PM to retain project write access")
	}
}
