package teampreview

import (
	"context"
	"testing"

	"isbudget/backend/internal/finance/pipeline"
	"isbudget/backend/internal/workflow"

	"github.com/google/uuid"
)

type fakeFinanceProvider struct {
	response  pipeline.PreviewResponse
	lastInput pipeline.PreviewInput
}

func (f *fakeFinanceProvider) Preview(ctx context.Context, projectID uuid.UUID, input pipeline.PreviewInput) (pipeline.PreviewResponse, error) {
	f.lastInput = input
	return f.response, nil
}

type fakeWorkflowProvider struct {
	response workflow.ProjectContributionSummary
}

func (f *fakeWorkflowProvider) GetSummary(ctx context.Context, actorID, projectID uuid.UUID) (workflow.ProjectContributionSummary, error) {
	return f.response, nil
}

func TestComposePreviewNormalDistribution(t *testing.T) {
	projectID := uuid.New()
	preview := ComposePreview(
		pipeline.PreviewResponse{ProjectID: projectID, ProjectCurrencyCode: "EUR", FinalTeamPoolAmount: 1000},
		workflow.ProjectContributionSummary{
			ProjectID:       projectID,
			CompletedPoints: 10,
			Users: []workflow.ProjectContributionUserSummary{
				{UserID: uuid.New(), FullName: "Alice", EarnedPoints: 6},
				{UserID: uuid.New(), FullName: "Bob", EarnedPoints: 4},
			},
		},
	)

	if preview.TeamPoolAmount != 1000 {
		t.Fatalf("expected team pool 1000, got %.2f", preview.TeamPoolAmount)
	}
	if preview.Members[0].PayoutAmount != 600 || preview.Members[1].PayoutAmount != 400 {
		t.Fatalf("unexpected payouts: %+v", preview.Members)
	}
}

func TestComposePreviewZeroTotalPoints(t *testing.T) {
	projectID := uuid.New()
	preview := ComposePreview(
		pipeline.PreviewResponse{ProjectID: projectID, ProjectCurrencyCode: "EUR", FinalTeamPoolAmount: 800},
		workflow.ProjectContributionSummary{
			ProjectID:       projectID,
			CompletedPoints: 0,
			Users: []workflow.ProjectContributionUserSummary{
				{UserID: uuid.New(), FullName: "Alice", EarnedPoints: 0},
				{UserID: uuid.New(), FullName: "Bob", EarnedPoints: 0},
			},
		},
	)

	for _, member := range preview.Members {
		if member.PayoutAmount != 0 || member.PercentageShare != 0 {
			t.Fatalf("expected zero payouts when points are zero, got %+v", preview.Members)
		}
	}
}

func TestComposePreviewSingleContributorGetsFullPool(t *testing.T) {
	projectID := uuid.New()
	preview := ComposePreview(
		pipeline.PreviewResponse{ProjectID: projectID, ProjectCurrencyCode: "EUR", FinalTeamPoolAmount: 725.25},
		workflow.ProjectContributionSummary{
			ProjectID:       projectID,
			CompletedPoints: 5,
			Users: []workflow.ProjectContributionUserSummary{
				{UserID: uuid.New(), FullName: "Alice", EarnedPoints: 5},
				{UserID: uuid.New(), FullName: "Bob", EarnedPoints: 0},
			},
		},
	)

	if preview.Members[0].PayoutAmount != 725.25 {
		t.Fatalf("expected single contributor to get full pool, got %+v", preview.Members[0])
	}
	if preview.Members[1].PayoutAmount != 0 {
		t.Fatalf("expected zero-point member to get zero, got %+v", preview.Members[1])
	}
}

func TestComposePreviewZeroPointMembersGetZeroPayout(t *testing.T) {
	projectID := uuid.New()
	preview := ComposePreview(
		pipeline.PreviewResponse{ProjectID: projectID, ProjectCurrencyCode: "EUR", FinalTeamPoolAmount: 500},
		workflow.ProjectContributionSummary{
			ProjectID:       projectID,
			CompletedPoints: 10,
			Users: []workflow.ProjectContributionUserSummary{
				{UserID: uuid.New(), FullName: "Alice", EarnedPoints: 10},
				{UserID: uuid.New(), FullName: "Bob", EarnedPoints: 0},
			},
		},
	)

	if preview.Members[1].PayoutAmount != 0 || preview.Members[1].PercentageShare != 0 {
		t.Fatalf("expected zero-point member to stay at zero, got %+v", preview.Members[1])
	}
}

func TestComposePreviewConsistentRoundingBehavior(t *testing.T) {
	projectID := uuid.New()
	preview := ComposePreview(
		pipeline.PreviewResponse{ProjectID: projectID, ProjectCurrencyCode: "EUR", FinalTeamPoolAmount: 100},
		workflow.ProjectContributionSummary{
			ProjectID:       projectID,
			CompletedPoints: 3,
			Users: []workflow.ProjectContributionUserSummary{
				{UserID: uuid.New(), FullName: "Alice", EarnedPoints: 1},
				{UserID: uuid.New(), FullName: "Bob", EarnedPoints: 1},
				{UserID: uuid.New(), FullName: "Cara", EarnedPoints: 1},
			},
		},
	)

	if preview.Members[0].PayoutAmount != 33.33 || preview.Members[1].PayoutAmount != 33.33 || preview.Members[2].PayoutAmount != 33.34 {
		t.Fatalf("expected deterministic cent residual on final member, got %+v", preview.Members)
	}
}

func TestServiceUsesFinancePreviewResultAndGrossOverride(t *testing.T) {
	projectID := uuid.New()
	override := 1500.0
	financeProvider := &fakeFinanceProvider{
		response: pipeline.PreviewResponse{
			ProjectID:           projectID,
			ProjectCurrencyCode: "EUR",
			FinalTeamPoolAmount: 420,
		},
	}
	workflowProvider := &fakeWorkflowProvider{
		response: workflow.ProjectContributionSummary{
			ProjectID:       projectID,
			CompletedPoints: 7,
			Users: []workflow.ProjectContributionUserSummary{
				{UserID: uuid.New(), FullName: "Alice", EarnedPoints: 7},
			},
		},
	}
	svc := NewService(financeProvider, workflowProvider)

	preview, err := svc.Preview(context.Background(), uuid.New(), projectID, &override)
	if err != nil {
		t.Fatalf("Preview() error = %v", err)
	}
	if financeProvider.lastInput.GrossAmount == nil || *financeProvider.lastInput.GrossAmount != override {
		t.Fatalf("expected gross override to flow into finance preview, got %+v", financeProvider.lastInput)
	}
	if preview.TeamPoolAmount != 420 || preview.Members[0].PayoutAmount != 420 {
		t.Fatalf("expected payout preview to reflect finance preview result, got %+v", preview)
	}
}
