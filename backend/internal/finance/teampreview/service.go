package teampreview

import (
	"context"
	"math"
	"strings"

	"isbudget/backend/internal/finance/pipeline"
	"isbudget/backend/internal/workflow"

	"github.com/google/uuid"
)

const roundingPolicy = "half_up_2dp_with_residual_to_last_eligible_member"

type FinancePreviewProvider interface {
	Preview(ctx context.Context, projectID uuid.UUID, input pipeline.PreviewInput) (pipeline.PreviewResponse, error)
}

type WorkflowSummaryProvider interface {
	GetSummary(ctx context.Context, actorID, projectID uuid.UUID) (workflow.ProjectContributionSummary, error)
}

type Service struct {
	finance  FinancePreviewProvider
	workflow WorkflowSummaryProvider
}

type MemberPreview struct {
	UserID              uuid.UUID `json:"user_id"`
	UserName            string    `json:"user_name"`
	TotalPoints         int32     `json:"total_points"`
	PercentageShare     float64   `json:"percentage_share"`
	PayoutAmount        float64   `json:"payout_amount"`
	CompletedNodesCount int32     `json:"completed_nodes_count"`
}

type PreviewResponse struct {
	ProjectID               uuid.UUID       `json:"project_id"`
	ProjectCurrencyCode     string          `json:"project_currency_code"`
	TeamPoolAmount          float64         `json:"team_pool_amount"`
	TotalPointsInPool       int32           `json:"total_points_in_pool"`
	NumberOfEligibleMembers int32           `json:"number_of_eligible_members"`
	RoundingPolicy          string          `json:"rounding_policy"`
	Members                 []MemberPreview `json:"members"`
}

func NewService(finance FinancePreviewProvider, workflow WorkflowSummaryProvider) *Service {
	return &Service{finance: finance, workflow: workflow}
}

func (s *Service) Preview(ctx context.Context, actorID, projectID uuid.UUID, grossOverride *float64) (PreviewResponse, error) {
	financePreview, err := s.finance.Preview(ctx, projectID, pipeline.PreviewInput{GrossAmount: grossOverride})
	if err != nil {
		return PreviewResponse{}, err
	}
	contributionSummary, err := s.workflow.GetSummary(ctx, actorID, projectID)
	if err != nil {
		return PreviewResponse{}, err
	}
	return ComposePreview(financePreview, contributionSummary), nil
}

func ComposePreview(financePreview pipeline.PreviewResponse, contributionSummary workflow.ProjectContributionSummary) PreviewResponse {
	teamPoolAmount := roundMoney(financePreview.FinalTeamPoolAmount)
	totalPoints := contributionSummary.CompletedPoints
	eligibleMembers := int32(0)
	for _, member := range contributionSummary.Users {
		if member.EarnedPoints > 0 {
			eligibleMembers++
		}
	}

	members := make([]MemberPreview, 0, len(contributionSummary.Users))
	if totalPoints <= 0 || teamPoolAmount <= 0 {
		for _, member := range contributionSummary.Users {
			members = append(members, MemberPreview{
				UserID:              member.UserID,
				UserName:            formatUserName(member),
				TotalPoints:         member.EarnedPoints,
				PercentageShare:     0,
				PayoutAmount:        0,
				CompletedNodesCount: member.CompletedNodesCount,
			})
		}
		return PreviewResponse{
			ProjectID:               financePreview.ProjectID,
			ProjectCurrencyCode:     financePreview.ProjectCurrencyCode,
			TeamPoolAmount:          teamPoolAmount,
			TotalPointsInPool:       totalPoints,
			NumberOfEligibleMembers: eligibleMembers,
			RoundingPolicy:          roundingPolicy,
			Members:                 members,
		}
	}

	lastEligibleIndex := -1
	for i, member := range contributionSummary.Users {
		if member.EarnedPoints > 0 {
			lastEligibleIndex = i
		}
	}

	allocatedSoFar := 0.0
	for i, member := range contributionSummary.Users {
		shareRatio := 0.0
		if member.EarnedPoints > 0 {
			shareRatio = float64(member.EarnedPoints) / float64(totalPoints)
		}
		sharePercent := roundShare(shareRatio * 100)
		payoutAmount := 0.0
		if member.EarnedPoints > 0 {
			if i == lastEligibleIndex {
				payoutAmount = roundMoney(teamPoolAmount - allocatedSoFar)
			} else {
				payoutAmount = roundMoney(teamPoolAmount * shareRatio)
				allocatedSoFar = roundMoney(allocatedSoFar + payoutAmount)
			}
		}
		members = append(members, MemberPreview{
			UserID:              member.UserID,
			UserName:            formatUserName(member),
			TotalPoints:         member.EarnedPoints,
			PercentageShare:     sharePercent,
			PayoutAmount:        payoutAmount,
			CompletedNodesCount: member.CompletedNodesCount,
		})
	}

	return PreviewResponse{
		ProjectID:               financePreview.ProjectID,
		ProjectCurrencyCode:     financePreview.ProjectCurrencyCode,
		TeamPoolAmount:          teamPoolAmount,
		TotalPointsInPool:       totalPoints,
		NumberOfEligibleMembers: eligibleMembers,
		RoundingPolicy:          roundingPolicy,
		Members:                 members,
	}
}

func formatUserName(member workflow.ProjectContributionUserSummary) string {
	if strings.TrimSpace(member.FullName) != "" {
		return strings.TrimSpace(member.FullName)
	}
	if strings.TrimSpace(member.Nickname) != "" {
		return strings.TrimSpace(member.Nickname)
	}
	if strings.TrimSpace(member.Email) != "" {
		return strings.TrimSpace(member.Email)
	}
	return "Unknown member"
}

func roundMoney(value float64) float64 {
	return math.Round(value*100) / 100
}

func roundShare(value float64) float64 {
	return math.Round(value*1000000) / 1000000
}
