package kickstarter

import "time"

type ImportRunSummary struct {
	ID               string     `json:"id"`
	ProjectID        string     `json:"project_id"`
	UploadedBy       string     `json:"uploaded_by"`
	UploadedByName   string     `json:"uploaded_by_name"`
	ImportType       string     `json:"import_type"`
	FileName         string     `json:"file_name"`
	Status           string     `json:"status"`
	ErrorMessage     *string    `json:"error_message,omitempty"`
	RowsProcessed    int32      `json:"rows_processed"`
	RowsSucceeded    int32      `json:"rows_succeeded"`
	RowsFailed       int32      `json:"rows_failed"`
	RowsSkipped      int32      `json:"rows_skipped"`
	TotalPledged     *float64   `json:"total_pledged,omitempty"`
	BackerCount      *int32     `json:"backer_count,omitempty"`
	RewardTiersCount *int32     `json:"reward_tiers_count,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
}

type ImportRecordResponse struct {
	ID               string     `json:"id"`
	RunID            string     `json:"run_id"`
	RowNumber        int32      `json:"row_number"`
	Status           string     `json:"status"`
	ErrorMessage     *string    `json:"error_message,omitempty"`
	PledgedAt        *time.Time `json:"pledged_at,omitempty"`
	PledgedAmount    *float64   `json:"pledged_amount,omitempty"`
	BackerIdentifier *string    `json:"backer_identifier,omitempty"`
	RewardTier       *string    `json:"reward_tier,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

type ImportRunDetail struct {
	Run     ImportRunSummary      `json:"run"`
	Records []ImportRecordResponse `json:"records"`
}
