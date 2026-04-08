package lifecycle

import (
	"context"
	"errors"
	"strings"
	"time"

	"isbudget/backend/internal/auth"
	"isbudget/backend/internal/db/sqlc"
	"isbudget/backend/internal/finance/pipeline"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	StagePreLaunch    = "pre_launch"
	StageLiveCampaign = "live_campaign"
	StagePostCampaign = "post_campaign"
	StageClosed       = "closed"
)

var (
	ErrLifecycleReadDenied       = errors.New("campaign lifecycle read requires project membership")
	ErrLifecycleWriteDenied      = errors.New("campaign lifecycle updates require project manager permissions")
	ErrInvalidLifecycleStage     = errors.New("campaign lifecycle stage is invalid")
	ErrLifecycleSnapshotNotFound = errors.New("campaign stage snapshot not found")
)

type FinancePreviewProvider interface {
	Preview(ctx context.Context, projectID uuid.UUID, input pipeline.PreviewInput) (pipeline.PreviewResponse, error)
}

type Service struct {
	repo    Repository
	finance FinancePreviewProvider
}

func NewService(repo Repository, finance FinancePreviewProvider) *Service {
	return &Service{repo: repo, finance: finance}
}

func (s *Service) GetConfig(ctx context.Context, actorID, projectID uuid.UUID) (sqlc.CampaignConfig, error) {
	if err := s.assertProjectRead(ctx, actorID, projectID); err != nil {
		return sqlc.CampaignConfig{}, err
	}
	row, err := s.repo.GetCampaignConfigByProject(ctx, projectID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return defaultCampaignConfig(projectID), nil
		}
		return sqlc.CampaignConfig{}, err
	}
	return row, nil
}

func (s *Service) UpdateConfig(ctx context.Context, actorID, projectID uuid.UUID, stage string, launchAt *time.Time, latePledgesEnabled, pledgeOverTimeEnabled, pledgeManagerEnabled, digitalOnlyFulfillment bool) (sqlc.CampaignConfig, error) {
	if err := s.assertProjectWrite(ctx, actorID, projectID); err != nil {
		return sqlc.CampaignConfig{}, err
	}
	stage = strings.TrimSpace(stage)
	if !isValidStage(stage) {
		return sqlc.CampaignConfig{}, ErrInvalidLifecycleStage
	}
	return s.repo.UpsertCampaignConfig(ctx, sqlc.UpsertCampaignConfigParams{
		ProjectID:              projectID,
		CurrentStage:           stage,
		LaunchAt:               launchAt,
		LatePledgesEnabled:     latePledgesEnabled,
		PledgeOverTimeEnabled:  pledgeOverTimeEnabled,
		PledgeManagerEnabled:   pledgeManagerEnabled,
		DigitalOnlyFulfillment: digitalOnlyFulfillment,
	})
}

func (s *Service) CreateSnapshot(ctx context.Context, actorID, projectID uuid.UUID, stage string, followersCount *int32, notes *string, grossOverride *float64) (sqlc.CampaignStageSnapshot, error) {
	if err := s.assertProjectWrite(ctx, actorID, projectID); err != nil {
		return sqlc.CampaignStageSnapshot{}, err
	}
	stage = strings.TrimSpace(stage)
	if !isValidStage(stage) {
		return sqlc.CampaignStageSnapshot{}, ErrInvalidLifecycleStage
	}

	user, err := s.repo.GetUserByID(ctx, actorID)
	if err != nil {
		return sqlc.CampaignStageSnapshot{}, err
	}

	var grossAmountSnapshot *float64
	var teamPoolSnapshot *float64
	if s.finance != nil {
		preview, err := s.finance.Preview(ctx, projectID, pipeline.PreviewInput{GrossAmount: grossOverride})
		if err != nil {
			return sqlc.CampaignStageSnapshot{}, err
		}
		grossAmountSnapshot = &preview.GrossAmount
		teamPoolSnapshot = &preview.FinalTeamPoolAmount
	}

	var trimmedNotes *string
	if notes != nil {
		value := strings.TrimSpace(*notes)
		trimmedNotes = &value
	}

	return s.repo.CreateCampaignStageSnapshot(ctx, sqlc.CreateCampaignStageSnapshotParams{
		ProjectID:           projectID,
		Stage:               stage,
		CreatedBy:           actorID,
		CreatedByName:       formatDisplayName(user),
		CreatedAt:           time.Now().UTC(),
		FollowersCount:      followersCount,
		Notes:               trimmedNotes,
		GrossAmountSnapshot: grossAmountSnapshot,
		TeamPoolSnapshot:    teamPoolSnapshot,
	})
}

func (s *Service) ListSnapshots(ctx context.Context, actorID, projectID uuid.UUID) ([]sqlc.CampaignStageSnapshot, error) {
	if err := s.assertProjectRead(ctx, actorID, projectID); err != nil {
		return nil, err
	}
	return s.repo.ListCampaignStageSnapshotsByProject(ctx, projectID)
}

func (s *Service) GetSnapshot(ctx context.Context, actorID, snapshotID uuid.UUID) (sqlc.CampaignStageSnapshot, error) {
	row, err := s.repo.GetCampaignStageSnapshotByID(ctx, snapshotID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return sqlc.CampaignStageSnapshot{}, ErrLifecycleSnapshotNotFound
		}
		return sqlc.CampaignStageSnapshot{}, err
	}
	if err := s.assertProjectRead(ctx, actorID, row.ProjectID); err != nil {
		return sqlc.CampaignStageSnapshot{}, err
	}
	return row, nil
}

func (s *Service) assertProjectRead(ctx context.Context, actorID, projectID uuid.UUID) error {
	member, err := s.repo.GetProjectMemberByUser(ctx, sqlc.GetProjectMemberByUserParams{
		ProjectID: projectID,
		UserID:    actorID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrLifecycleReadDenied
		}
		return err
	}
	if !auth.IsAllowed(string(member.Role), auth.PermProjectRead) {
		return ErrLifecycleReadDenied
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
			return ErrLifecycleWriteDenied
		}
		return err
	}
	if !auth.IsAllowed(string(member.Role), auth.PermProjectWrite) {
		return ErrLifecycleWriteDenied
	}
	return nil
}

func defaultCampaignConfig(projectID uuid.UUID) sqlc.CampaignConfig {
	now := time.Now().UTC()
	return sqlc.CampaignConfig{
		ID:                     uuid.Nil,
		ProjectID:              projectID,
		CurrentStage:           StagePreLaunch,
		LaunchAt:               nil,
		LatePledgesEnabled:     false,
		PledgeOverTimeEnabled:  false,
		PledgeManagerEnabled:   false,
		DigitalOnlyFulfillment: false,
		CreatedAt:              now,
		UpdatedAt:              now,
	}
}

func isValidStage(stage string) bool {
	switch stage {
	case StagePreLaunch, StageLiveCampaign, StagePostCampaign, StageClosed:
		return true
	default:
		return false
	}
}

func formatDisplayName(user sqlc.User) string {
	if strings.TrimSpace(user.FullName) != "" {
		return strings.TrimSpace(user.FullName)
	}
	if strings.TrimSpace(user.Nickname) != "" {
		return strings.TrimSpace(user.Nickname)
	}
	return strings.TrimSpace(user.Email)
}
