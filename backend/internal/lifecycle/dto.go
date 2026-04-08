package lifecycle

type updateCampaignConfigRequest struct {
	CurrentStage           string  `json:"current_stage" validate:"required,oneof=pre_launch live_campaign post_campaign closed"`
	LaunchAt               *string `json:"launch_at" validate:"omitempty,datetime=2006-01-02T15:04"`
	LatePledgesEnabled     bool    `json:"late_pledges_enabled"`
	PledgeOverTimeEnabled  bool    `json:"pledge_over_time_enabled"`
	PledgeManagerEnabled   bool    `json:"pledge_manager_enabled"`
	DigitalOnlyFulfillment bool    `json:"digital_only_fulfillment"`
}

type createCampaignStageSnapshotRequest struct {
	Stage          string   `json:"stage" validate:"required,oneof=pre_launch live_campaign post_campaign closed"`
	FollowersCount *int32   `json:"followers_count" validate:"omitempty,gte=0"`
	Notes          *string  `json:"notes"`
	GrossAmount    *float64 `json:"gross_amount" validate:"omitempty,gte=0"`
}
