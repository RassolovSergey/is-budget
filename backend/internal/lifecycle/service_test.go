package lifecycle

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"isbudget/backend/internal/auth"
	"isbudget/backend/internal/db/sqlc"
	"isbudget/backend/internal/finance/pipeline"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type fakeRepo struct {
	members   map[uuid.UUID]sqlc.ProjectMember
	user      sqlc.User
	config    *sqlc.CampaignConfig
	snapshots []sqlc.CampaignStageSnapshot
}

func (f *fakeRepo) GetProjectMemberByUser(ctx context.Context, arg sqlc.GetProjectMemberByUserParams) (sqlc.ProjectMember, error) {
	member, ok := f.members[arg.UserID]
	if !ok || member.ProjectID != arg.ProjectID {
		return sqlc.ProjectMember{}, pgx.ErrNoRows
	}
	return member, nil
}

func (f *fakeRepo) GetUserByID(ctx context.Context, id uuid.UUID) (sqlc.User, error) {
	if f.user.ID != id {
		return sqlc.User{}, pgx.ErrNoRows
	}
	return f.user, nil
}

func (f *fakeRepo) GetCampaignConfigByProject(ctx context.Context, projectID uuid.UUID) (sqlc.CampaignConfig, error) {
	if f.config == nil || f.config.ProjectID != projectID {
		return sqlc.CampaignConfig{}, pgx.ErrNoRows
	}
	return *f.config, nil
}

func (f *fakeRepo) UpsertCampaignConfig(ctx context.Context, arg sqlc.UpsertCampaignConfigParams) (sqlc.CampaignConfig, error) {
	now := time.Now().UTC()
	row := sqlc.CampaignConfig{
		ID:                     uuid.New(),
		ProjectID:              arg.ProjectID,
		CurrentStage:           arg.CurrentStage,
		LaunchAt:               arg.LaunchAt,
		LatePledgesEnabled:     arg.LatePledgesEnabled,
		PledgeOverTimeEnabled:  arg.PledgeOverTimeEnabled,
		PledgeManagerEnabled:   arg.PledgeManagerEnabled,
		DigitalOnlyFulfillment: arg.DigitalOnlyFulfillment,
		CreatedAt:              now,
		UpdatedAt:              now,
	}
	f.config = &row
	return row, nil
}

func (f *fakeRepo) CreateCampaignStageSnapshot(ctx context.Context, arg sqlc.CreateCampaignStageSnapshotParams) (sqlc.CampaignStageSnapshot, error) {
	row := sqlc.CampaignStageSnapshot{
		ID:                  uuid.New(),
		ProjectID:           arg.ProjectID,
		Stage:               arg.Stage,
		CreatedBy:           arg.CreatedBy,
		CreatedByName:       arg.CreatedByName,
		CreatedAt:           arg.CreatedAt,
		FollowersCount:      arg.FollowersCount,
		Notes:               arg.Notes,
		GrossAmountSnapshot: arg.GrossAmountSnapshot,
		TeamPoolSnapshot:    arg.TeamPoolSnapshot,
	}
	f.snapshots = append(f.snapshots, row)
	return row, nil
}

func (f *fakeRepo) ListCampaignStageSnapshotsByProject(ctx context.Context, projectID uuid.UUID) ([]sqlc.CampaignStageSnapshot, error) {
	items := make([]sqlc.CampaignStageSnapshot, 0)
	for _, snapshot := range f.snapshots {
		if snapshot.ProjectID == projectID {
			items = append(items, snapshot)
		}
	}
	slices.SortFunc(items, func(a, b sqlc.CampaignStageSnapshot) int { return b.CreatedAt.Compare(a.CreatedAt) })
	return items, nil
}

func (f *fakeRepo) GetCampaignStageSnapshotByID(ctx context.Context, id uuid.UUID) (sqlc.CampaignStageSnapshot, error) {
	for _, snapshot := range f.snapshots {
		if snapshot.ID == id {
			return snapshot, nil
		}
	}
	return sqlc.CampaignStageSnapshot{}, pgx.ErrNoRows
}

type fakeFinancePreview struct{ response pipeline.PreviewResponse }

func (f *fakeFinancePreview) Preview(ctx context.Context, projectID uuid.UUID, input pipeline.PreviewInput) (pipeline.PreviewResponse, error) {
	return f.response, nil
}

func TestValidAndInvalidStageUpdates(t *testing.T) {
	projectID := uuid.New()
	actorID := uuid.New()
	repo := newFakeRepo(projectID, actorID, "PM")
	svc := NewService(repo, nil)

	if _, err := svc.UpdateConfig(context.Background(), actorID, projectID, StageLiveCampaign, nil, true, false, false, true); err != nil {
		t.Fatalf("expected valid stage update, got %v", err)
	}
	if _, err := svc.UpdateConfig(context.Background(), actorID, projectID, "bad_stage", nil, false, false, false, false); !errors.Is(err, ErrInvalidLifecycleStage) {
		t.Fatalf("expected invalid stage error, got %v", err)
	}
}

func TestLaunchMilestonePersistence(t *testing.T) {
	projectID := uuid.New()
	actorID := uuid.New()
	repo := newFakeRepo(projectID, actorID, "PM")
	svc := NewService(repo, nil)
	launchAt := time.Date(2026, 7, 2, 15, 30, 0, 0, time.UTC)

	config, err := svc.UpdateConfig(context.Background(), actorID, projectID, StagePreLaunch, &launchAt, false, true, false, true)
	if err != nil {
		t.Fatalf("UpdateConfig() error = %v", err)
	}
	if config.LaunchAt == nil || !config.LaunchAt.Equal(launchAt) {
		t.Fatalf("expected launch milestone to persist, got %+v", config.LaunchAt)
	}
}

func TestSnapshotFreezing(t *testing.T) {
	projectID := uuid.New()
	actorID := uuid.New()
	repo := newFakeRepo(projectID, actorID, "PM")
	finance := &fakeFinancePreview{response: pipeline.PreviewResponse{GrossAmount: 1200, FinalTeamPoolAmount: 500}}
	svc := NewService(repo, finance)
	followers := int32(321)
	note := "Before launch"

	snapshot, err := svc.CreateSnapshot(context.Background(), actorID, projectID, StagePreLaunch, &followers, &note, nil)
	if err != nil {
		t.Fatalf("CreateSnapshot() error = %v", err)
	}
	if snapshot.GrossAmountSnapshot == nil || *snapshot.GrossAmountSnapshot != 1200 {
		t.Fatalf("expected frozen gross amount, got %+v", snapshot.GrossAmountSnapshot)
	}
	if snapshot.TeamPoolSnapshot == nil || *snapshot.TeamPoolSnapshot != 500 {
		t.Fatalf("expected frozen team pool, got %+v", snapshot.TeamPoolSnapshot)
	}
}

func TestPreLaunchFollowerSnapshot(t *testing.T) {
	projectID := uuid.New()
	actorID := uuid.New()
	repo := newFakeRepo(projectID, actorID, "PM")
	svc := NewService(repo, nil)
	followers := int32(987)

	snapshot, err := svc.CreateSnapshot(context.Background(), actorID, projectID, StagePreLaunch, &followers, nil, nil)
	if err != nil {
		t.Fatalf("CreateSnapshot() error = %v", err)
	}
	if snapshot.FollowersCount == nil || *snapshot.FollowersCount != followers {
		t.Fatalf("expected followers to be frozen, got %+v", snapshot.FollowersCount)
	}
}

func TestPermissionsForLifecycleWrites(t *testing.T) {
	projectID := uuid.New()
	actorID := uuid.New()
	repo := newFakeRepo(projectID, actorID, "TEAM")
	svc := NewService(repo, nil)

	if _, err := svc.UpdateConfig(context.Background(), actorID, projectID, StagePreLaunch, nil, false, false, false, false); !errors.Is(err, ErrLifecycleWriteDenied) {
		t.Fatalf("expected write denied for non-PM, got %v", err)
	}
	if _, err := svc.CreateSnapshot(context.Background(), actorID, projectID, StagePreLaunch, nil, nil, nil); !errors.Is(err, ErrLifecycleWriteDenied) {
		t.Fatalf("expected snapshot write denied for non-PM, got %v", err)
	}
}

func TestSnapshotHistoryRetrieval(t *testing.T) {
	projectID := uuid.New()
	actorID := uuid.New()
	repo := newFakeRepo(projectID, actorID, "PM")
	svc := NewService(repo, nil)

	first, _ := svc.CreateSnapshot(context.Background(), actorID, projectID, StagePreLaunch, nil, nil, nil)
	time.Sleep(2 * time.Millisecond)
	second, _ := svc.CreateSnapshot(context.Background(), actorID, projectID, StageLiveCampaign, nil, nil, nil)

	items, err := svc.ListSnapshots(context.Background(), actorID, projectID)
	if err != nil {
		t.Fatalf("ListSnapshots() error = %v", err)
	}
	if len(items) != 2 || items[0].ID != second.ID || items[1].ID != first.ID {
		t.Fatalf("expected latest-first snapshot history, got %+v", items)
	}
}

func TestGetSnapshotDetailRequiresReadMembership(t *testing.T) {
	projectID := uuid.New()
	actorID := uuid.New()
	repo := newFakeRepo(projectID, actorID, "PM")
	svc := NewService(repo, nil)
	snapshot, _ := svc.CreateSnapshot(context.Background(), actorID, projectID, StagePreLaunch, nil, nil, nil)

	otherRepo := newFakeRepo(projectID, uuid.New(), "TEAM")
	otherRepo.snapshots = repo.snapshots
	otherSvc := NewService(otherRepo, nil)
	_, err := otherSvc.GetSnapshot(context.Background(), uuid.New(), snapshot.ID)
	if !errors.Is(err, ErrLifecycleReadDenied) {
		t.Fatalf("expected read denied, got %v", err)
	}
}

func newFakeRepo(projectID, actorID uuid.UUID, role string) *fakeRepo {
	return &fakeRepo{
		user: sqlc.User{ID: actorID, FullName: "PM User", Email: "pm@example.com"},
		members: map[uuid.UUID]sqlc.ProjectMember{
			actorID: {ProjectID: projectID, UserID: actorID, Role: sqlc.ProjectRole(role)},
		},
	}
}

func TestRoleContractStillMatchesExpectations(t *testing.T) {
	if !auth.IsAllowed("PM", auth.PermProjectWrite) {
		t.Fatal("expected PM to retain project write access")
	}
}
