package snapshots

import (
	"context"
	"errors"
	"slices"
	"testing"

	"isbudget/backend/internal/auth"
	"isbudget/backend/internal/db/sqlc"
	"isbudget/backend/internal/finance/pipeline"
	"isbudget/backend/internal/finance/teampreview"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type fakeRepo struct {
	user      sqlc.User
	members   map[uuid.UUID]sqlc.ProjectMember
	snapshots []sqlc.PayoutSnapshot
	sm        []sqlc.PayoutSnapshotMember
	sb        []sqlc.PayoutSnapshotBlock
}

func (f *fakeRepo) WithTx(ctx context.Context, fn func(repo Repository) error) error {
	return fn(f)
}

func (f *fakeRepo) GetUserByID(ctx context.Context, id uuid.UUID) (sqlc.User, error) {
	if f.user.ID == id {
		return f.user, nil
	}
	return sqlc.User{}, pgx.ErrNoRows
}

func (f *fakeRepo) GetProjectMemberByUser(ctx context.Context, arg sqlc.GetProjectMemberByUserParams) (sqlc.ProjectMember, error) {
	member, ok := f.members[arg.UserID]
	if !ok || member.ProjectID != arg.ProjectID {
		return sqlc.ProjectMember{}, pgx.ErrNoRows
	}
	return member, nil
}

func (f *fakeRepo) CreatePayoutSnapshot(ctx context.Context, arg sqlc.CreatePayoutSnapshotParams) (sqlc.PayoutSnapshot, error) {
	row := sqlc.PayoutSnapshot{
		ID:                   uuid.New(),
		ProjectID:            arg.ProjectID,
		CreatedBy:            arg.CreatedBy,
		CreatedByName:        arg.CreatedByName,
		CreatedAt:            arg.CreatedAt,
		ProjectCurrencyCode:  arg.ProjectCurrencyCode,
		GrossAmount:          arg.GrossAmount,
		GrossAmountSource:    arg.GrossAmountSource,
		FinalTeamPoolAmount:  arg.FinalTeamPoolAmount,
		TotalPoints:          arg.TotalPoints,
		EligibleMembersCount: arg.EligibleMembersCount,
		MemberCount:          arg.MemberCount,
		RoundingPolicy:       arg.RoundingPolicy,
		SettlementStatus:     arg.SettlementStatus,
		SettlementNote:       arg.SettlementNote,
		ApprovedBy:           arg.ApprovedBy,
		ApprovedByName:       arg.ApprovedByName,
		ApprovedAt:           arg.ApprovedAt,
	}
	f.snapshots = append(f.snapshots, row)
	return row, nil
}

func (f *fakeRepo) UpdatePayoutSnapshotSettlement(ctx context.Context, arg sqlc.UpdatePayoutSnapshotSettlementParams) (sqlc.PayoutSnapshot, error) {
	for i := range f.snapshots {
		if f.snapshots[i].ID == arg.ID {
			f.snapshots[i].SettlementStatus = arg.SettlementStatus
			f.snapshots[i].SettlementNote = arg.SettlementNote
			f.snapshots[i].ApprovedBy = arg.ApprovedBy
			f.snapshots[i].ApprovedByName = arg.ApprovedByName
			f.snapshots[i].ApprovedAt = arg.ApprovedAt
			return f.snapshots[i], nil
		}
	}
	return sqlc.PayoutSnapshot{}, pgx.ErrNoRows
}

func (f *fakeRepo) CreatePayoutSnapshotMember(ctx context.Context, arg sqlc.CreatePayoutSnapshotMemberParams) (sqlc.PayoutSnapshotMember, error) {
	row := sqlc.PayoutSnapshotMember{
		ID:                  uuid.New(),
		SnapshotID:          arg.SnapshotID,
		UserID:              arg.UserID,
		UserName:            arg.UserName,
		TotalPoints:         arg.TotalPoints,
		PercentageShare:     arg.PercentageShare,
		PayoutAmount:        arg.PayoutAmount,
		CompletedNodesCount: arg.CompletedNodesCount,
		OrderIndex:          arg.OrderIndex,
	}
	f.sm = append(f.sm, row)
	return row, nil
}

func (f *fakeRepo) CreatePayoutSnapshotBlock(ctx context.Context, arg sqlc.CreatePayoutSnapshotBlockParams) (sqlc.PayoutSnapshotBlock, error) {
	row := sqlc.PayoutSnapshotBlock{
		ID:              uuid.New(),
		SnapshotID:      arg.SnapshotID,
		SourceBlockID:   arg.SourceBlockID,
		Name:            arg.Name,
		Type:            arg.Type,
		OrderIndex:      arg.OrderIndex,
		ValueType:       arg.ValueType,
		Value:           arg.Value,
		AppliesTo:       arg.AppliesTo,
		InputAmount:     arg.InputAmount,
		AllocatedAmount: arg.AllocatedAmount,
		RemainingAmount: arg.RemainingAmount,
		IsTeamPool:      arg.IsTeamPool,
	}
	f.sb = append(f.sb, row)
	return row, nil
}

func (f *fakeRepo) ListPayoutSnapshotsByProject(ctx context.Context, projectID uuid.UUID) ([]sqlc.PayoutSnapshot, error) {
	items := make([]sqlc.PayoutSnapshot, 0)
	for _, snapshot := range f.snapshots {
		if snapshot.ProjectID == projectID {
			items = append(items, snapshot)
		}
	}
	slices.SortFunc(items, func(a, b sqlc.PayoutSnapshot) int { return b.CreatedAt.Compare(a.CreatedAt) })
	return items, nil
}

func (f *fakeRepo) GetPayoutSnapshotByID(ctx context.Context, id uuid.UUID) (sqlc.PayoutSnapshot, error) {
	for _, snapshot := range f.snapshots {
		if snapshot.ID == id {
			return snapshot, nil
		}
	}
	return sqlc.PayoutSnapshot{}, pgx.ErrNoRows
}

func (f *fakeRepo) ListPayoutSnapshotMembersBySnapshot(ctx context.Context, snapshotID uuid.UUID) ([]sqlc.PayoutSnapshotMember, error) {
	items := make([]sqlc.PayoutSnapshotMember, 0)
	for _, member := range f.sm {
		if member.SnapshotID == snapshotID {
			items = append(items, member)
		}
	}
	slices.SortFunc(items, func(a, b sqlc.PayoutSnapshotMember) int {
		if a.OrderIndex < b.OrderIndex {
			return -1
		}
		if a.OrderIndex > b.OrderIndex {
			return 1
		}
		return 0
	})
	return items, nil
}

func (f *fakeRepo) ListPayoutSnapshotBlocksBySnapshot(ctx context.Context, snapshotID uuid.UUID) ([]sqlc.PayoutSnapshotBlock, error) {
	items := make([]sqlc.PayoutSnapshotBlock, 0)
	for _, block := range f.sb {
		if block.SnapshotID == snapshotID {
			items = append(items, block)
		}
	}
	slices.SortFunc(items, func(a, b sqlc.PayoutSnapshotBlock) int {
		if a.OrderIndex < b.OrderIndex {
			return -1
		}
		if a.OrderIndex > b.OrderIndex {
			return 1
		}
		return 0
	})
	return items, nil
}

type fakeTeamPreview struct{ response teampreview.PreviewResponse }

func (f *fakeTeamPreview) Preview(ctx context.Context, actorID, projectID uuid.UUID, grossOverride *float64) (teampreview.PreviewResponse, error) {
	return f.response, nil
}

type fakeFinancePreview struct{ response pipeline.PreviewResponse }

func (f *fakeFinancePreview) Preview(ctx context.Context, projectID uuid.UUID, input pipeline.PreviewInput) (pipeline.PreviewResponse, error) {
	return f.response, nil
}

func TestSnapshotStoresFrozenValues(t *testing.T) {
	projectID := uuid.New()
	actorID := uuid.New()
	repo := newFakeRepo(projectID, actorID, "PM")
	teamProvider := &fakeTeamPreview{
		response: teampreview.PreviewResponse{
			ProjectID:               projectID,
			ProjectCurrencyCode:     "EUR",
			TeamPoolAmount:          900,
			TotalPointsInPool:       10,
			NumberOfEligibleMembers: 1,
			RoundingPolicy:          "rounding-policy",
			Members: []teampreview.MemberPreview{
				{UserID: uuid.New(), UserName: "Alice", TotalPoints: 10, PercentageShare: 100, PayoutAmount: 900, CompletedNodesCount: 2},
			},
		},
	}
	financeProvider := &fakeFinancePreview{
		response: pipeline.PreviewResponse{
			ProjectID:           projectID,
			ProjectCurrencyCode: "EUR",
			GrossAmount:         1200,
			GrossAmountSource:   "manual_override",
			FinalTeamPoolAmount: 900,
			Steps: []pipeline.PreviewStep{
				{BlockID: uuid.New(), Name: "Team pool", Type: "team_pool_remainder", OrderIndex: 1, ValueType: "fixed", Value: 0, AppliesTo: "current_balance", InputAmount: 900, AllocatedAmount: 900, RemainingAmount: 0, IsTeamPool: true},
			},
		},
	}
	svc := NewService(repo, teamProvider, financeProvider)

	snapshot, err := svc.Create(context.Background(), actorID, projectID, nil)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if snapshot.GrossAmount != 1200 || snapshot.FinalTeamPoolAmount != 900 {
		t.Fatalf("expected frozen top-level values, got %+v", snapshot.SnapshotSummary)
	}
	if len(snapshot.Members) != 1 || snapshot.Members[0].PayoutAmount != 900 {
		t.Fatalf("expected frozen member payout, got %+v", snapshot.Members)
	}
	if len(snapshot.Blocks) != 1 || snapshot.Blocks[0].AllocatedAmount != 900 {
		t.Fatalf("expected frozen block step, got %+v", snapshot.Blocks)
	}
}

func TestLaterChangesDoNotAffectExistingSnapshot(t *testing.T) {
	projectID := uuid.New()
	actorID := uuid.New()
	repo := newFakeRepo(projectID, actorID, "PM")
	teamProvider := &fakeTeamPreview{
		response: teampreview.PreviewResponse{
			ProjectID:               projectID,
			ProjectCurrencyCode:     "EUR",
			TeamPoolAmount:          300,
			TotalPointsInPool:       3,
			NumberOfEligibleMembers: 1,
			RoundingPolicy:          "policy",
			Members: []teampreview.MemberPreview{
				{UserID: uuid.New(), UserName: "Alice", TotalPoints: 3, PercentageShare: 100, PayoutAmount: 300},
			},
		},
	}
	financeProvider := &fakeFinancePreview{
		response: pipeline.PreviewResponse{
			ProjectID:           projectID,
			ProjectCurrencyCode: "EUR",
			GrossAmount:         500,
			GrossAmountSource:   "project_income_total",
			FinalTeamPoolAmount: 300,
		},
	}
	svc := NewService(repo, teamProvider, financeProvider)

	first, err := svc.Create(context.Background(), actorID, projectID, nil)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	teamProvider.response.TeamPoolAmount = 999
	teamProvider.response.Members[0].PayoutAmount = 999
	financeProvider.response.FinalTeamPoolAmount = 999

	stored, err := svc.Get(context.Background(), actorID, first.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if stored.FinalTeamPoolAmount != 300 || stored.Members[0].PayoutAmount != 300 {
		t.Fatalf("expected original frozen values, got %+v", stored)
	}
}

func TestSnapshotDetailReturnsCorrectMemberAmounts(t *testing.T) {
	projectID := uuid.New()
	actorID := uuid.New()
	repo := newFakeRepo(projectID, actorID, "PM")
	teamProvider := &fakeTeamPreview{
		response: teampreview.PreviewResponse{
			ProjectID:               projectID,
			ProjectCurrencyCode:     "EUR",
			TeamPoolAmount:          100,
			TotalPointsInPool:       4,
			NumberOfEligibleMembers: 2,
			RoundingPolicy:          "policy",
			Members: []teampreview.MemberPreview{
				{UserID: uuid.New(), UserName: "Alice", TotalPoints: 3, PercentageShare: 75, PayoutAmount: 75},
				{UserID: uuid.New(), UserName: "Bob", TotalPoints: 1, PercentageShare: 25, PayoutAmount: 25},
			},
		},
	}
	financeProvider := &fakeFinancePreview{
		response: pipeline.PreviewResponse{ProjectID: projectID, ProjectCurrencyCode: "EUR", GrossAmount: 100, GrossAmountSource: "project_income_total", FinalTeamPoolAmount: 100},
	}
	svc := NewService(repo, teamProvider, financeProvider)
	created, err := svc.Create(context.Background(), actorID, projectID, nil)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	detail, err := svc.Get(context.Background(), actorID, created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if len(detail.Members) != 2 || detail.Members[0].PayoutAmount != 75 || detail.Members[1].PayoutAmount != 25 {
		t.Fatalf("unexpected snapshot detail members: %+v", detail.Members)
	}
}

func TestZeroPointCaseStillSavesCorrectly(t *testing.T) {
	projectID := uuid.New()
	actorID := uuid.New()
	repo := newFakeRepo(projectID, actorID, "PM")
	teamProvider := &fakeTeamPreview{
		response: teampreview.PreviewResponse{
			ProjectID:               projectID,
			ProjectCurrencyCode:     "EUR",
			TeamPoolAmount:          200,
			TotalPointsInPool:       0,
			NumberOfEligibleMembers: 0,
			RoundingPolicy:          "policy",
			Members: []teampreview.MemberPreview{
				{UserID: uuid.New(), UserName: "Alice", TotalPoints: 0, PercentageShare: 0, PayoutAmount: 0},
			},
		},
	}
	financeProvider := &fakeFinancePreview{
		response: pipeline.PreviewResponse{ProjectID: projectID, ProjectCurrencyCode: "EUR", GrossAmount: 300, GrossAmountSource: "project_income_total", FinalTeamPoolAmount: 200},
	}
	svc := NewService(repo, teamProvider, financeProvider)
	snapshot, err := svc.Create(context.Background(), actorID, projectID, nil)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if snapshot.TotalPoints != 0 || snapshot.Members[0].PayoutAmount != 0 {
		t.Fatalf("expected zero-point snapshot to persist zero payouts, got %+v", snapshot)
	}
}

func TestGrossOverrideIsFrozen(t *testing.T) {
	projectID := uuid.New()
	actorID := uuid.New()
	override := 1777.77
	repo := newFakeRepo(projectID, actorID, "PM")
	teamProvider := &fakeTeamPreview{
		response: teampreview.PreviewResponse{
			ProjectID:               projectID,
			ProjectCurrencyCode:     "EUR",
			TeamPoolAmount:          444.44,
			TotalPointsInPool:       1,
			NumberOfEligibleMembers: 1,
			RoundingPolicy:          "policy",
			Members: []teampreview.MemberPreview{
				{UserID: uuid.New(), UserName: "Alice", TotalPoints: 1, PercentageShare: 100, PayoutAmount: 444.44},
			},
		},
	}
	financeProvider := &fakeFinancePreview{
		response: pipeline.PreviewResponse{ProjectID: projectID, ProjectCurrencyCode: "EUR", GrossAmount: override, GrossAmountSource: "manual_override", FinalTeamPoolAmount: 444.44},
	}
	svc := NewService(repo, teamProvider, financeProvider)
	snapshot, err := svc.Create(context.Background(), actorID, projectID, &override)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if snapshot.GrossAmount != override || snapshot.GrossAmountSource != "manual_override" {
		t.Fatalf("expected gross override to be frozen, got %+v", snapshot.SnapshotSummary)
	}
}

func TestUpdateSettlementStoresApprovalMetadata(t *testing.T) {
	projectID := uuid.New()
	actorID := uuid.New()
	repo := newFakeRepo(projectID, actorID, "PM")
	teamProvider := &fakeTeamPreview{
		response: teampreview.PreviewResponse{
			ProjectID:               projectID,
			ProjectCurrencyCode:     "EUR",
			TeamPoolAmount:          200,
			TotalPointsInPool:       2,
			NumberOfEligibleMembers: 1,
			RoundingPolicy:          "policy",
			Members: []teampreview.MemberPreview{
				{UserID: uuid.New(), UserName: "Alice", TotalPoints: 2, PercentageShare: 100, PayoutAmount: 200},
			},
		},
	}
	financeProvider := &fakeFinancePreview{
		response: pipeline.PreviewResponse{ProjectID: projectID, ProjectCurrencyCode: "EUR", GrossAmount: 300, GrossAmountSource: "project_income_total", FinalTeamPoolAmount: 200},
	}
	svc := NewService(repo, teamProvider, financeProvider)
	created, err := svc.Create(context.Background(), actorID, projectID, nil)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	note := "Reviewed and approved"
	updated, err := svc.UpdateSettlement(context.Background(), actorID, created.ID, SettlementApproved, &note)
	if err != nil {
		t.Fatalf("UpdateSettlement() error = %v", err)
	}
	if updated.SettlementStatus != SettlementApproved {
		t.Fatalf("expected approved status, got %s", updated.SettlementStatus)
	}
	if updated.SettlementNote == nil || *updated.SettlementNote != note {
		t.Fatalf("expected settlement note to persist, got %+v", updated.SettlementNote)
	}
	if updated.ApprovedBy == nil || *updated.ApprovedBy != actorID || updated.ApprovedAt == nil {
		t.Fatalf("expected approval metadata, got %+v", updated.SnapshotSummary)
	}
}

func TestSettlementTransitionRequiresApprovalBeforeReady(t *testing.T) {
	projectID := uuid.New()
	actorID := uuid.New()
	repo := newFakeRepo(projectID, actorID, "PM")
	svc := NewService(repo, &fakeTeamPreview{
		response: teampreview.PreviewResponse{
			ProjectID: projectID, ProjectCurrencyCode: "EUR", RoundingPolicy: "policy",
			Members: []teampreview.MemberPreview{{UserID: uuid.New(), UserName: "Alice"}},
		},
	}, &fakeFinancePreview{
		response: pipeline.PreviewResponse{ProjectID: projectID, ProjectCurrencyCode: "EUR"},
	})
	created, err := svc.Create(context.Background(), actorID, projectID, nil)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	_, err = svc.UpdateSettlement(context.Background(), actorID, created.ID, SettlementReadyForSettlement, nil)
	if !errors.Is(err, ErrSnapshotSettlementMove) {
		t.Fatalf("expected invalid transition, got %v", err)
	}
}

func TestSettlementUpdateRequiresManager(t *testing.T) {
	projectID := uuid.New()
	actorID := uuid.New()
	repo := newFakeRepo(projectID, actorID, "TEAM")
	repo.snapshots = append(repo.snapshots, sqlc.PayoutSnapshot{
		ID:               uuid.New(),
		ProjectID:        projectID,
		CreatedBy:        actorID,
		CreatedByName:    "Project PM",
		SettlementStatus: SettlementDraft,
	})
	svc := NewService(repo, &fakeTeamPreview{}, &fakeFinancePreview{})
	_, err := svc.UpdateSettlement(context.Background(), actorID, repo.snapshots[0].ID, SettlementApproved, nil)
	if !errors.Is(err, ErrSnapshotUpdateDenied) {
		t.Fatalf("expected update denied, got %v", err)
	}
}

func TestSettlementCanAdvanceFromApprovedToReady(t *testing.T) {
	projectID := uuid.New()
	actorID := uuid.New()
	repo := newFakeRepo(projectID, actorID, "PM")
	svc := NewService(repo, &fakeTeamPreview{
		response: teampreview.PreviewResponse{
			ProjectID: projectID, ProjectCurrencyCode: "EUR", RoundingPolicy: "policy",
			Members: []teampreview.MemberPreview{{UserID: uuid.New(), UserName: "Alice"}},
		},
	}, &fakeFinancePreview{
		response: pipeline.PreviewResponse{ProjectID: projectID, ProjectCurrencyCode: "EUR"},
	})
	created, err := svc.Create(context.Background(), actorID, projectID, nil)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	approved, err := svc.UpdateSettlement(context.Background(), actorID, created.ID, SettlementApproved, nil)
	if err != nil {
		t.Fatalf("approve error = %v", err)
	}
	ready, err := svc.UpdateSettlement(context.Background(), actorID, created.ID, SettlementReadyForSettlement, nil)
	if err != nil {
		t.Fatalf("ready error = %v", err)
	}
	if ready.SettlementStatus != SettlementReadyForSettlement {
		t.Fatalf("expected ready status, got %s", ready.SettlementStatus)
	}
	if approved.ApprovedAt == nil || ready.ApprovedAt == nil || !approved.ApprovedAt.Equal(*ready.ApprovedAt) {
		t.Fatalf("expected approval timestamp to persist, approved=%+v ready=%+v", approved.ApprovedAt, ready.ApprovedAt)
	}
}

func newFakeRepo(projectID, actorID uuid.UUID, role string) *fakeRepo {
	return &fakeRepo{
		user: sqlc.User{ID: actorID, FullName: "Project PM", Email: "pm@example.com"},
		members: map[uuid.UUID]sqlc.ProjectMember{
			actorID: {ProjectID: projectID, UserID: actorID, Role: sqlc.ProjectRole(role)},
		},
	}
}

func TestCreateRequiresManager(t *testing.T) {
	projectID := uuid.New()
	actorID := uuid.New()
	repo := newFakeRepo(projectID, actorID, "TEAM")
	svc := NewService(repo, &fakeTeamPreview{}, &fakeFinancePreview{})
	_, err := svc.Create(context.Background(), actorID, projectID, nil)
	if !errors.Is(err, ErrSnapshotCreateDenied) {
		t.Fatalf("expected create denied, got %v", err)
	}
}

func TestListRequiresReadMembership(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, &fakeTeamPreview{}, &fakeFinancePreview{})
	_, err := svc.List(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, ErrSnapshotProjectReadDenied) {
		t.Fatalf("expected read denied, got %v", err)
	}
}

func TestRoleCompatibility(t *testing.T) {
	if !auth.IsAllowed("PM", auth.PermProjectWrite) {
		t.Fatal("expected PM to have project write")
	}
}
