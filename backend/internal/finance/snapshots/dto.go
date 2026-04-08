package snapshots

type createSnapshotRequest struct {
	GrossAmount *float64 `json:"gross_amount" validate:"omitempty,gte=0"`
}

type updateSettlementRequest struct {
	SettlementStatus string  `json:"settlement_status" validate:"required,oneof=draft approved ready_for_settlement"`
	SettlementNote   *string `json:"settlement_note" validate:"omitempty,max=1000"`
}
