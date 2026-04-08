package snapshots

import (
	"context"
	"errors"
	"strings"
	"time"

	"isbudget/backend/internal/auth"
	"isbudget/backend/internal/db/sqlc"
	"isbudget/backend/internal/finance/pipeline"
	"isbudget/backend/internal/finance/teampreview"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrSnapshotNotFound          = errors.New("payout snapshot not found")
	ErrSnapshotProjectReadDenied = errors.New("project read access is required for payout snapshot")
	ErrSnapshotCreateDenied      = errors.New("project manager permissions are required to create payout snapshots")
	ErrSnapshotUpdateDenied      = errors.New("project manager permissions are required to update payout snapshots")
	ErrSnapshotSettlementStatus  = errors.New("payout snapshot settlement status is invalid")
	ErrSnapshotSettlementMove    = errors.New("payout snapshot settlement transition is invalid")
)

const (
	SettlementDraft              = "draft"
	SettlementApproved           = "approved"
	SettlementReadyForSettlement = "ready_for_settlement"
)

type TeamPayoutPreviewProvider interface {
	Preview(ctx context.Context, actorID, projectID uuid.UUID, grossOverride *float64) (teampreview.PreviewResponse, error)
}

type FinancePreviewProvider interface {
	Preview(ctx context.Context, projectID uuid.UUID, input pipeline.PreviewInput) (pipeline.PreviewResponse, error)
}

type Service struct {
	repo        Repository
	teamPreview TeamPayoutPreviewProvider
	finance     FinancePreviewProvider
}

type SnapshotSummary struct {
	ID                   uuid.UUID  `json:"id"`
	ProjectID            uuid.UUID  `json:"project_id"`
	CreatedBy            uuid.UUID  `json:"created_by"`
	CreatedByName        string     `json:"created_by_name"`
	CreatedAt            time.Time  `json:"created_at"`
	ProjectCurrencyCode  string     `json:"project_currency_code"`
	GrossAmount          float64    `json:"gross_amount"`
	GrossAmountSource    string     `json:"gross_amount_source"`
	FinalTeamPoolAmount  float64    `json:"final_team_pool_amount"`
	TotalPoints          int32      `json:"total_points"`
	EligibleMembersCount int32      `json:"eligible_members_count"`
	MemberCount          int32      `json:"member_count"`
	RoundingPolicy       string     `json:"rounding_policy"`
	SettlementStatus     string     `json:"settlement_status"`
	SettlementNote       *string    `json:"settlement_note,omitempty"`
	ApprovedBy           *uuid.UUID `json:"approved_by,omitempty"`
	ApprovedByName       *string    `json:"approved_by_name,omitempty"`
	ApprovedAt           *time.Time `json:"approved_at,omitempty"`
}

type SnapshotMember struct {
	ID                  uuid.UUID  `json:"id"`
	SnapshotID          uuid.UUID  `json:"snapshot_id"`
	UserID              *uuid.UUID `json:"user_id,omitempty"`
	UserName            string     `json:"user_name"`
	TotalPoints         int32      `json:"total_points"`
	PercentageShare     float64    `json:"percentage_share"`
	PayoutAmount        float64    `json:"payout_amount"`
	CompletedNodesCount int32      `json:"completed_nodes_count"`
	OrderIndex          int32      `json:"order_index"`
}

type SnapshotBlock struct {
	ID              uuid.UUID  `json:"id"`
	SnapshotID      uuid.UUID  `json:"snapshot_id"`
	SourceBlockID   *uuid.UUID `json:"source_block_id,omitempty"`
	Name            string     `json:"name"`
	Type            string     `json:"type"`
	LevelIndex      int32      `json:"level_index"`
	OrderIndex      int32      `json:"order_index"`
	ValueType       string     `json:"value_type"`
	Value           float64    `json:"value"`
	AppliesTo       string     `json:"applies_to"`
	InputAmount     float64    `json:"input_amount"`
	AllocatedAmount float64    `json:"allocated_amount"`
	RemainingAmount float64    `json:"remaining_amount"`
	IsTeamPool      bool       `json:"is_team_pool"`
}

type SnapshotDetail struct {
	SnapshotSummary
	Members []SnapshotMember `json:"members"`
	Blocks  []SnapshotBlock  `json:"blocks"`
}

func NewService(repo Repository, teamPreview TeamPayoutPreviewProvider, finance FinancePreviewProvider) *Service {
	return &Service{repo: repo, teamPreview: teamPreview, finance: finance}
}

func (s *Service) Create(ctx context.Context, actorID, projectID uuid.UUID, grossOverride *float64) (SnapshotDetail, error) {
	if err := s.assertProjectManager(ctx, projectID, actorID); err != nil {
		return SnapshotDetail{}, err
	}

	user, err := s.repo.GetUserByID(ctx, actorID)
	if err != nil {
		return SnapshotDetail{}, err
	}
	financePreview, err := s.finance.Preview(ctx, projectID, pipeline.PreviewInput{GrossAmount: grossOverride})
	if err != nil {
		return SnapshotDetail{}, err
	}
	teamPreview, err := s.teamPreview.Preview(ctx, actorID, projectID, grossOverride)
	if err != nil {
		return SnapshotDetail{}, err
	}

	var detail SnapshotDetail
	err = s.repo.WithTx(ctx, func(repo Repository) error {
		created, err := repo.CreatePayoutSnapshot(ctx, sqlc.CreatePayoutSnapshotParams{
			ProjectID:            projectID,
			CreatedBy:            actorID,
			CreatedByName:        formatDisplayName(user),
			CreatedAt:            time.Now().UTC(),
			ProjectCurrencyCode:  financePreview.ProjectCurrencyCode,
			GrossAmount:          financePreview.GrossAmount,
			GrossAmountSource:    financePreview.GrossAmountSource,
			FinalTeamPoolAmount:  financePreview.FinalTeamPoolAmount,
			TotalPoints:          teamPreview.TotalPointsInPool,
			EligibleMembersCount: teamPreview.NumberOfEligibleMembers,
			MemberCount:          int32(len(teamPreview.Members)),
			RoundingPolicy:       teamPreview.RoundingPolicy,
			SettlementStatus:     SettlementDraft,
		})
		if err != nil {
			return err
		}

		members := make([]SnapshotMember, 0, len(teamPreview.Members))
		for index, member := range teamPreview.Members {
			row, err := repo.CreatePayoutSnapshotMember(ctx, sqlc.CreatePayoutSnapshotMemberParams{
				SnapshotID:          created.ID,
				UserID:              &member.UserID,
				UserName:            member.UserName,
				TotalPoints:         member.TotalPoints,
				PercentageShare:     member.PercentageShare,
				PayoutAmount:        member.PayoutAmount,
				CompletedNodesCount: member.CompletedNodesCount,
				OrderIndex:          int32(index + 1),
			})
			if err != nil {
				return err
			}
			members = append(members, mapSnapshotMember(row))
		}

		blocks := make([]SnapshotBlock, 0, len(financePreview.Steps))
		for _, step := range financePreview.Steps {
			row, err := repo.CreatePayoutSnapshotBlock(ctx, sqlc.CreatePayoutSnapshotBlockParams{
				SnapshotID:      created.ID,
				SourceBlockID:   &step.BlockID,
				Name:            step.Name,
				Type:            step.Type,
				LevelIndex:      step.LevelIndex,
				OrderIndex:      step.OrderIndex,
				ValueType:       step.ValueType,
				Value:           step.Value,
				AppliesTo:       step.AppliesTo,
				InputAmount:     step.InputAmount,
				AllocatedAmount: step.AllocatedAmount,
				RemainingAmount: step.RemainingAmount,
				IsTeamPool:      step.IsTeamPool,
			})
			if err != nil {
				return err
			}
			blocks = append(blocks, mapSnapshotBlock(row))
		}

		detail = SnapshotDetail{
			SnapshotSummary: mapSnapshotSummary(created),
			Members:         members,
			Blocks:          blocks,
		}
		return nil
	})
	if err != nil {
		return SnapshotDetail{}, err
	}
	return detail, nil
}

func (s *Service) UpdateSettlement(ctx context.Context, actorID, snapshotID uuid.UUID, status string, note *string) (SnapshotDetail, error) {
	row, err := s.repo.GetPayoutSnapshotByID(ctx, snapshotID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SnapshotDetail{}, ErrSnapshotNotFound
		}
		return SnapshotDetail{}, err
	}
	if err := s.assertProjectManager(ctx, row.ProjectID, actorID); err != nil {
		if errors.Is(err, ErrSnapshotCreateDenied) {
			return SnapshotDetail{}, ErrSnapshotUpdateDenied
		}
		return SnapshotDetail{}, err
	}
	if !isValidSettlementStatus(status) {
		return SnapshotDetail{}, ErrSnapshotSettlementStatus
	}
	if !isAllowedSettlementTransition(row.SettlementStatus, status) {
		return SnapshotDetail{}, ErrSnapshotSettlementMove
	}

	var normalizedNote *string
	if note != nil {
		trimmed := strings.TrimSpace(*note)
		if trimmed != "" {
			normalizedNote = &trimmed
		}
	}

	var approvedBy *uuid.UUID
	var approvedByName *string
	var approvedAt *time.Time
	if status == SettlementApproved || status == SettlementReadyForSettlement {
		if row.ApprovedBy != nil && row.ApprovedAt != nil && row.ApprovedByName != nil {
			approvedBy = row.ApprovedBy
			approvedByName = row.ApprovedByName
			approvedAt = row.ApprovedAt
		} else {
			user, err := s.repo.GetUserByID(ctx, actorID)
			if err != nil {
				return SnapshotDetail{}, err
			}
			name := formatDisplayName(user)
			now := time.Now().UTC()
			approvedBy = &actorID
			approvedByName = &name
			approvedAt = &now
		}
	}

	if status == SettlementDraft {
		approvedBy = nil
		approvedByName = nil
		approvedAt = nil
	}

	if _, err := s.repo.UpdatePayoutSnapshotSettlement(ctx, sqlc.UpdatePayoutSnapshotSettlementParams{
		ID:               snapshotID,
		SettlementStatus: status,
		SettlementNote:   normalizedNote,
		ApprovedBy:       approvedBy,
		ApprovedByName:   approvedByName,
		ApprovedAt:       approvedAt,
	}); err != nil {
		return SnapshotDetail{}, err
	}

	return s.Get(ctx, actorID, snapshotID)
}

func (s *Service) List(ctx context.Context, actorID, projectID uuid.UUID) ([]SnapshotSummary, error) {
	if err := s.assertProjectReader(ctx, projectID, actorID); err != nil {
		return nil, err
	}
	rows, err := s.repo.ListPayoutSnapshotsByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	items := make([]SnapshotSummary, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapSnapshotSummary(row))
	}
	return items, nil
}

func (s *Service) Get(ctx context.Context, actorID, snapshotID uuid.UUID) (SnapshotDetail, error) {
	row, err := s.repo.GetPayoutSnapshotByID(ctx, snapshotID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SnapshotDetail{}, ErrSnapshotNotFound
		}
		return SnapshotDetail{}, err
	}
	if err := s.assertProjectReader(ctx, row.ProjectID, actorID); err != nil {
		return SnapshotDetail{}, err
	}
	membersRows, err := s.repo.ListPayoutSnapshotMembersBySnapshot(ctx, snapshotID)
	if err != nil {
		return SnapshotDetail{}, err
	}
	blockRows, err := s.repo.ListPayoutSnapshotBlocksBySnapshot(ctx, snapshotID)
	if err != nil {
		return SnapshotDetail{}, err
	}
	members := make([]SnapshotMember, 0, len(membersRows))
	for _, member := range membersRows {
		members = append(members, mapSnapshotMember(member))
	}
	blocks := make([]SnapshotBlock, 0, len(blockRows))
	for _, block := range blockRows {
		blocks = append(blocks, mapSnapshotBlock(block))
	}
	return SnapshotDetail{
		SnapshotSummary: mapSnapshotSummary(row),
		Members:         members,
		Blocks:          blocks,
	}, nil
}

func (s *Service) assertProjectReader(ctx context.Context, projectID, actorID uuid.UUID) error {
	member, err := s.repo.GetProjectMemberByUser(ctx, sqlc.GetProjectMemberByUserParams{
		ProjectID: projectID,
		UserID:    actorID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrSnapshotProjectReadDenied
		}
		return err
	}
	if !auth.IsAllowed(string(member.Role), auth.PermProjectRead) {
		return ErrSnapshotProjectReadDenied
	}
	return nil
}

func (s *Service) assertProjectManager(ctx context.Context, projectID, actorID uuid.UUID) error {
	member, err := s.repo.GetProjectMemberByUser(ctx, sqlc.GetProjectMemberByUserParams{
		ProjectID: projectID,
		UserID:    actorID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrSnapshotCreateDenied
		}
		return err
	}
	if !auth.IsAllowed(string(member.Role), auth.PermProjectWrite) {
		return ErrSnapshotCreateDenied
	}
	return nil
}

func isValidSettlementStatus(status string) bool {
	switch status {
	case SettlementDraft, SettlementApproved, SettlementReadyForSettlement:
		return true
	default:
		return false
	}
}

func isAllowedSettlementTransition(from, to string) bool {
	if from == "" {
		from = SettlementDraft
	}
	if from == to {
		return true
	}
	switch from {
	case SettlementDraft:
		return to == SettlementApproved
	case SettlementApproved:
		return to == SettlementDraft || to == SettlementReadyForSettlement
	case SettlementReadyForSettlement:
		return to == SettlementApproved
	default:
		return false
	}
}

func mapSnapshotSummary(row sqlc.PayoutSnapshot) SnapshotSummary {
	return SnapshotSummary{
		ID:                   row.ID,
		ProjectID:            row.ProjectID,
		CreatedBy:            row.CreatedBy,
		CreatedByName:        row.CreatedByName,
		CreatedAt:            row.CreatedAt,
		ProjectCurrencyCode:  row.ProjectCurrencyCode,
		GrossAmount:          row.GrossAmount,
		GrossAmountSource:    row.GrossAmountSource,
		FinalTeamPoolAmount:  row.FinalTeamPoolAmount,
		TotalPoints:          row.TotalPoints,
		EligibleMembersCount: row.EligibleMembersCount,
		MemberCount:          row.MemberCount,
		RoundingPolicy:       row.RoundingPolicy,
		SettlementStatus:     row.SettlementStatus,
		SettlementNote:       row.SettlementNote,
		ApprovedBy:           row.ApprovedBy,
		ApprovedByName:       row.ApprovedByName,
		ApprovedAt:           row.ApprovedAt,
	}
}

func mapSnapshotMember(row sqlc.PayoutSnapshotMember) SnapshotMember {
	return SnapshotMember{
		ID:                  row.ID,
		SnapshotID:          row.SnapshotID,
		UserID:              row.UserID,
		UserName:            row.UserName,
		TotalPoints:         row.TotalPoints,
		PercentageShare:     row.PercentageShare,
		PayoutAmount:        row.PayoutAmount,
		CompletedNodesCount: row.CompletedNodesCount,
		OrderIndex:          row.OrderIndex,
	}
}

func mapSnapshotBlock(row sqlc.PayoutSnapshotBlock) SnapshotBlock {
	return SnapshotBlock{
		ID:              row.ID,
		SnapshotID:      row.SnapshotID,
		SourceBlockID:   row.SourceBlockID,
		Name:            row.Name,
		Type:            row.Type,
		LevelIndex:      row.LevelIndex,
		OrderIndex:      row.OrderIndex,
		ValueType:       row.ValueType,
		Value:           row.Value,
		AppliesTo:       row.AppliesTo,
		InputAmount:     row.InputAmount,
		AllocatedAmount: row.AllocatedAmount,
		RemainingAmount: row.RemainingAmount,
		IsTeamPool:      row.IsTeamPool,
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
