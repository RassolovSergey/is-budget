package pipeline

type upsertAllocationBlockRequest struct {
	Name                 string  `json:"name" validate:"required,min=2,max=160"`
	Type                 string  `json:"type" validate:"required,oneof=tax actual_expense custom_fee admin_distribution studio_distribution team_distribution tax_percent fixed_cost studio_percent admin_percent team_pool_percent team_pool_remainder"`
	LevelIndex           int32   `json:"level_index" validate:"gte=1"`
	Enabled              *bool   `json:"enabled"`
	ValueType            string  `json:"value_type" validate:"omitempty,oneof=percent fixed"`
	Value                float64 `json:"value" validate:"gte=0"`
	AppliesTo            string  `json:"applies_to" validate:"omitempty,oneof=gross current_balance"`
	TeamDistributionMode string  `json:"team_distribution_mode" validate:"omitempty,oneof=percent full_remainder"`
}

type reorderAllocationBlockItemRequest struct {
	ID         string `json:"id" validate:"required,uuid"`
	LevelIndex int32  `json:"level_index" validate:"required,gte=1"`
	OrderIndex int32  `json:"order_index" validate:"required,gte=1"`
}

type reorderAllocationBlocksRequest struct {
	Items      []reorderAllocationBlockItemRequest `json:"items" validate:"omitempty,min=1,dive"`
	OrderedIDs []string                            `json:"ordered_ids" validate:"omitempty,min=1,dive,required,uuid"`
}

type previewAllocationRequest struct {
	GrossAmount *float64 `json:"gross_amount" validate:"omitempty,gte=0"`
}
