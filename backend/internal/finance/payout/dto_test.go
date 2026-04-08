package payout

import (
	"testing"

	"isbudget/backend/internal/common"
)

func TestUpsertRuleRequestAllowsGeneralUUIDRecipientUserID(t *testing.T) {
	validate := common.NewValidator()
	req := upsertRuleRequest{
		RecipientUserID: "11111111-1111-1111-1111-111111111111",
		Responsibility:  "Producer",
		Percent:         25,
		Level:           1,
	}

	if err := validate.Struct(req); err != nil {
		t.Fatalf("expected general UUID recipient_user_id to pass validation, got %v", err)
	}
}

func TestUpsertRuleRequestRejectsInvalidRecipientUserID(t *testing.T) {
	validate := common.NewValidator()
	req := upsertRuleRequest{
		RecipientUserID: "not-a-uuid",
		Responsibility:  "Producer",
		Percent:         25,
		Level:           1,
	}

	if err := validate.Struct(req); err == nil {
		t.Fatal("expected invalid recipient_user_id to fail validation")
	}
}

func TestUpsertRuleRequestRejectsMissingRecipientUserID(t *testing.T) {
	validate := common.NewValidator()
	req := upsertRuleRequest{
		Responsibility: "Producer",
		Percent:        25,
		Level:          1,
	}

	if err := validate.Struct(req); err == nil {
		t.Fatal("expected missing recipient_user_id to fail validation")
	}
}
