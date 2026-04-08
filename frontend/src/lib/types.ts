export type UUID = string;
export type CurrencyCode = "EUR" | "USD" | "RUB";
export type ProjectRole = "PM" | "ANALYST" | "TEAM" | "CUSTOMER";

export interface User {
  id: UUID;
  email: string;
  full_name: string;
  first_name: string;
  last_name: string;
  nickname: string;
  avatar_path?: string | null;
  default_currency_code: CurrencyCode;
}

export interface LoginResponse {
  token: string;
  user: User;
}

export type ProjectStatus = "preparation" | "active" | "completed";

export interface Project {
  id: UUID;
  name: string;
  description: string;
  duration_days: number;
  status: ProjectStatus;
  activated_at?: string | null;
  currency_code: CurrencyCode;
  secondary_currency_code?: CurrencyCode | null;
  category: string;
  current_user_role?: ProjectRole;
  created_at: string;
  updated_at: string;
}

export interface ProjectCategoryFolderSummary {
  category: string;
  active_projects_count: number;
  completed_projects_count: number;
  historical_campaigns_count: number;
}

export interface PlannedExpense {
  id: UUID;
  project_id: UUID;
  title: string;
  category: string;
  amount: number;
  due_date: string;
  is_mandatory: boolean;
  priority: number;
  note: string;
  created_at: string;
  updated_at: string;
}

export interface IncomeDaily {
  id: UUID;
  project_id: UUID;
  income_date: string;
  amount: number;
}

export interface ActualExpense {
  id: UUID;
  project_id: UUID;
  planned_expense_id?: UUID | null;
  title: string;
  category: string;
  amount: number;
  paid_at: string;
  note: string;
  created_at: string;
  updated_at: string;
}

export interface ProfitShareRule {
  id: UUID;
  project_id: UUID;
  recipient_user_id?: UUID | null;
  recipient_name: string;
  responsibility: string;
  currency_code: CurrencyCode;
  percent: number;
  level: number;
  applies_after_mandatory: boolean;
  created_at: string;
  updated_at: string;
}

export interface ProjectTaxRule {
  id: UUID;
  project_id: UUID;
  name: string;
  percent: number;
  level: number;
  created_at: string;
  updated_at: string;
}

export interface ProjectMemberWithUser {
  id: UUID;
  project_id: UUID;
  user_id: UUID;
  role: ProjectRole;
  email: string;
  first_name: string;
  last_name: string;
  full_name: string;
  nickname: string;
  avatar_path?: string | null;
  default_currency_code: CurrencyCode;
  created_at: string;
}

export type WorkflowNodeStatus = "planned" | "ready" | "in_progress" | "done";

export interface WorkflowNode {
  id: UUID;
  project_id: UUID;
  title: string;
  description: string;
  status: WorkflowNodeStatus;
  assignee_user_id?: UUID | null;
  contribution_points: number;
  completed_at?: string | null;
  completed_by?: UUID | null;
  created_at: string;
  updated_at: string;
}

export interface WorkflowDependency {
  id: UUID;
  parent_node_id: UUID;
  child_node_id: UUID;
  created_at: string;
}

export interface ContributionLedgerEntry {
  id: UUID;
  project_id: UUID;
  node_id: UUID;
  user_id: UUID;
  points: number;
  created_at: string;
}

export interface WorkflowProjectGraph {
  project_id: UUID;
  nodes: WorkflowNode[];
  dependencies: WorkflowDependency[];
  contribution_ledger: ContributionLedgerEntry[];
}

export interface WorkflowContributionUserSummary {
  user_id: UUID;
  email: string;
  full_name: string;
  nickname: string;
  earned_points: number;
  share_of_total_project_points: number;
  projected_points: number;
  projected_share_percentage: number;
  completed_nodes_count: number;
}

export interface WorkflowProjectSummary {
  project_id: UUID;
  total_project_points: number;
  completed_points: number;
  completion_percentage: number;
  users: WorkflowContributionUserSummary[];
}

export type AllocationBlockType =
  | "tax"
  | "actual_expense"
  | "custom_fee"
  | "studio_distribution"
  | "admin_distribution"
  | "team_distribution";

export type AllocationBlockValueType = "percent" | "fixed";
export type AllocationBlockAppliesTo = "gross" | "current_balance";
export type TeamDistributionMode = "percent" | "full_remainder";

export interface AllocationBlock {
  id: UUID;
  project_id: UUID;
  name: string;
  type: AllocationBlockType;
  level_index: number;
  order_index: number;
  enabled: boolean;
  value_type: AllocationBlockValueType;
  value: number;
  applies_to: AllocationBlockAppliesTo;
  created_at: string;
  updated_at: string;
}

export interface AllocationPreviewStep {
  block_id: UUID;
  name: string;
  type: AllocationBlockType;
  level_index: number;
  order_index: number;
  value_type: AllocationBlockValueType;
  value: number;
  applies_to: AllocationBlockAppliesTo;
  input_amount: number;
  allocated_amount: number;
  remaining_amount: number;
  is_team_pool: boolean;
  team_distribution_mode?: TeamDistributionMode | string;
}

export interface AllocationPreviewLevel {
  level_index: number;
  incoming_amount: number;
  total_allocated: number;
  remaining_amount: number;
  steps: AllocationPreviewStep[];
}

export interface AllocationPreview {
  project_id: UUID;
  project_currency_code: CurrencyCode;
  gross_amount: number;
  gross_amount_source: "project_income_total" | "manual_override" | string;
  total_deducted_before_team_pool: number;
  final_remaining_amount: number;
  final_team_pool_amount: number;
  levels: AllocationPreviewLevel[];
  steps: AllocationPreviewStep[];
}

export interface TeamPayoutPreviewMember {
  user_id: UUID;
  user_name: string;
  total_points: number;
  percentage_share: number;
  payout_amount: number;
  completed_nodes_count: number;
}

export interface TeamPayoutPreview {
  project_id: UUID;
  project_currency_code: CurrencyCode;
  team_pool_amount: number;
  total_points_in_pool: number;
  number_of_eligible_members: number;
  rounding_policy: string;
  members: TeamPayoutPreviewMember[];
}

export interface PayoutSnapshotSummary {
  id: UUID;
  project_id: UUID;
  created_by: UUID;
  created_by_name: string;
  created_at: string;
  project_currency_code: CurrencyCode;
  gross_amount: number;
  gross_amount_source: string;
  final_team_pool_amount: number;
  total_points: number;
  eligible_members_count: number;
  member_count: number;
  rounding_policy: string;
  settlement_status: "draft" | "approved" | "ready_for_settlement";
  settlement_note?: string | null;
  approved_by?: UUID | null;
  approved_by_name?: string | null;
  approved_at?: string | null;
}

export interface PayoutSnapshotMember {
  id: UUID;
  snapshot_id: UUID;
  user_id?: UUID | null;
  user_name: string;
  total_points: number;
  percentage_share: number;
  payout_amount: number;
  completed_nodes_count: number;
  order_index: number;
}

export interface PayoutSnapshotBlock {
  id: UUID;
  snapshot_id: UUID;
  source_block_id?: UUID | null;
  name: string;
  type: string;
  level_index: number;
  order_index: number;
  value_type: string;
  value: number;
  applies_to: string;
  input_amount: number;
  allocated_amount: number;
  remaining_amount: number;
  is_team_pool: boolean;
}

export interface PayoutSnapshotDetail extends PayoutSnapshotSummary {
  members: PayoutSnapshotMember[];
  blocks: PayoutSnapshotBlock[];
}

export type CampaignLifecycleStage = "pre_launch" | "live_campaign" | "post_campaign" | "closed";

export interface CampaignConfig {
  id: UUID;
  project_id: UUID;
  current_stage: CampaignLifecycleStage;
  launch_at?: string | null;
  late_pledges_enabled: boolean;
  pledge_over_time_enabled: boolean;
  pledge_manager_enabled: boolean;
  digital_only_fulfillment: boolean;
  created_at: string;
  updated_at: string;
}

export interface CampaignStageSnapshot {
  id: UUID;
  project_id: UUID;
  stage: CampaignLifecycleStage;
  created_by: UUID;
  created_by_name: string;
  created_at: string;
  followers_count?: number | null;
  notes?: string | null;
  gross_amount_snapshot?: number | null;
  team_pool_snapshot?: number | null;
}

export type ImportRunStatus = "pending" | "success" | "failed";
export type ImportRecordStatus = "success" | "failed" | "skipped";

export interface ImportRunSummary {
  id: UUID;
  project_id: UUID;
  uploaded_by: UUID;
  uploaded_by_name: string;
  import_type: string;
  file_name: string;
  status: ImportRunStatus;
  error_message?: string | null;
  rows_processed: number;
  rows_succeeded: number;
  rows_failed: number;
  rows_skipped: number;
  total_pledged?: number | null;
  backer_count?: number | null;
  reward_tiers_count?: number | null;
  created_at: string;
  completed_at?: string | null;
}

export interface ImportRecord {
  id: UUID;
  run_id: UUID;
  row_number: number;
  status: ImportRecordStatus;
  error_message?: string | null;
  pledged_at?: string | null;
  pledged_amount?: number | null;
  backer_identifier?: string | null;
  reward_tier?: string | null;
  created_at: string;
}

export interface ImportRunDetail {
  run: ImportRunSummary;
  records: ImportRecord[];
}

export interface ProjectInvite {
  id: UUID;
  project_id: UUID;
  token: string;
  invited_role: "PM" | "ANALYST" | "TEAM" | "CUSTOMER";
  invited_by: UUID;
  invited_user_id?: UUID | null;
  accepted_by?: UUID | null;
  accepted_at?: string | null;
  declined_by?: UUID | null;
  declined_at?: string | null;
  expires_at: string;
  created_at: string;
}

export interface ProjectInviteView extends ProjectInvite {
  invited_user_email?: string | null;
  invited_user_full_name?: string | null;
  invited_user_nickname?: string | null;
  invited_by_full_name?: string | null;
  status: "pending" | "accepted" | "declined" | "expired";
}

export interface IncomingProjectInvite {
  id: UUID;
  token: string;
  project_id: UUID;
  project_name: string;
  project_description: string;
  project_status: "preparation" | "active" | "completed";
  project_currency_code: CurrencyCode;
  invited_role: ProjectRole;
  invited_by: UUID;
  invited_by_full_name?: string | null;
  expires_at: string;
  created_at: string;
}

export interface PayoutPreviewItem {
  rule_id: UUID;
  recipient_user_id?: UUID | null;
  recipient_name: string;
  responsibility: string;
  level?: number;
  percent: number;
  payout_amount: number;
  currency_code: CurrencyCode;
  conversion_rate: number;
  payout_amount_converted: number;
  viewer_currency_code?: CurrencyCode;
  viewer_conversion_rate?: number;
  viewer_amount_converted?: number;
}

export interface PayoutPreview {
  project_currency_code: CurrencyCode;
  secondary_currency_code?: CurrencyCode;
  viewer_currency_code?: CurrencyCode;
  distributed_amount: number;
  distributed_percent: number;
  remaining_amount: number;
  remaining_percent: number;
  remaining_currency_code: CurrencyCode;
  remaining_conversion_rate: number;
  remaining_amount_converted: number;
  tax_levels: Array<{
    level: number;
    base_amount: number;
    percent_total: number;
    tax_amount: number;
    remaining_after: number;
  }>;
  summary: {
    income_total: number;
    expenses_total: number;
    planned_total: number;
    net_profit_to_date: number;
    mandatory_total: number;
    mandatory_paid: number;
    mandatory_remaining: number;
    available_before_tax: number;
    tax_percent_total: number;
    tax_total: number;
    available_for_sharing: number;
  };
  items: PayoutPreviewItem[];
}

export interface PayoutSnapshotFile {
  file_name: string;
  saved_at: string;
  size_bytes: number;
  download_url: string;
}

export interface CalculatePayoutsResponse {
  preview: PayoutPreview;
  payouts: Array<{
    id: UUID;
    project_id: UUID;
    rule_id?: UUID | null;
    calculated_at: string;
    recipient_name: string;
    amount: number;
    currency_code: CurrencyCode;
    amount_converted: number;
    conversion_rate: number;
    status: string;
  }>;
  snapshot: PayoutSnapshotFile;
}

export interface MyIncomeItem {
  rule_id: UUID;
  recipient_name: string;
  responsibility: string;
  level: number;
  percent: number;
  project_amount: number;
  project_currency_code: CurrencyCode;
  user_amount: number;
  user_currency_code: CurrencyCode;
  conversion_rate_to_user: number;
}

export interface MyIncomeResponse {
  project_id: UUID;
  project_currency_code: CurrencyCode;
  user_currency_code: CurrencyCode;
  income_total: number;
  expenses_total: number;
  net_profit_to_date: number;
  tax_total: number;
  available_for_sharing: number;
  distributed_amount: number;
  distributed_to_others: number;
  total_project_amount: number;
  total_user_amount: number;
  total_conversion_rate: number;
  matching_strategy: string;
  forecast_projection?: {
    run_id: UUID;
    created_at: string;
    p10: {
      projected_income_total: number;
      project_amount: number;
      user_amount: number;
    };
    yhat: {
      projected_income_total: number;
      project_amount: number;
      user_amount: number;
    };
    p90: {
      projected_income_total: number;
      project_amount: number;
      user_amount: number;
    };
  } | null;
  items: MyIncomeItem[];
}

export interface TeamIncomeAmount {
  project_amount: number;
  project_currency_code: CurrencyCode;
  user_amount: number;
  user_currency_code: CurrencyCode;
  conversion_rate_to_user: number;
}

export interface TeamIncomeForecastResponse {
  project_id: UUID;
  project_currency_code: CurrencyCode;
  forecast_available: boolean;
  latest_forecast_run_id?: UUID | null;
  latest_forecast_created_at?: string | null;
  scenario_model: {
    poor: {
      key: string;
      label: string;
      forecast_key: "p10";
      description: string;
    };
    average: {
      key: string;
      label: string;
      forecast_key: "yhat";
      description: string;
    };
    good: {
      key: string;
      label: string;
      forecast_key: "p90";
      description: string;
    };
  };
  members: Array<{
    user_id: UUID;
    full_name: string;
    role: ProjectRole;
    current_earned: TeamIncomeAmount;
    poor_scenario?: TeamIncomeAmount | null;
    average_scenario?: TeamIncomeAmount | null;
    good_scenario?: TeamIncomeAmount | null;
  }>;
}

export interface DashboardResponse {
  daily_income_series: IncomeDaily[];
  planned_expenses: PlannedExpense[];
  actual_expenses: ActualExpense[];
  income_total: number;
  expenses_total: number;
  net_profit_to_date: number;
  mandatory_total: number;
  mandatory_paid: number;
  mandatory_remaining: number;
  available_before_tax: number;
  tax_percent_total: number;
  tax_total: number;
  available_for_sharing: number;
  payout_preview: PayoutPreviewItem[];
  last_forecast_run?: {
    id: UUID;
    created_at: string;
    horizon_days: number;
    model_version: string;
    input_hash: string;
    seed: number;
  } | null;
}

export interface ForecastRun {
  run: ForecastRunMeta;
  points: {
    id: UUID;
    run_id: UUID;
    forecast_date: string;
    yhat: number;
    p10: number;
    p90: number;
  }[];
}

export interface ForecastRunMeta {
  id: UUID;
  project_id: UUID;
  created_at: string;
  horizon_days: number;
  model_version: string;
  input_hash: string;
  seed: number;
}

export interface ForecastRunSummary {
  run: ForecastRunMeta;
  forecast_total_yhat: number;
  forecast_total_p10: number;
  forecast_total_p90: number;
}

export interface HistoricalCampaignExpense {
  category: string;
  amount: number;
}

export interface HistoricalCampaignDay {
  day_number: number;
  income_amount: number;
  expenses: HistoricalCampaignExpense[];
}

export interface HistoricalCampaign {
  id: UUID;
  project_id?: UUID | null;
  owner_user_id: UUID;
  category: string;
  source?: "manual" | "completed_project";
  name: string;
  duration_days: number;
  currency_code: CurrencyCode;
  started_at: string;
  ended_at: string;
  created_at: string;
  updated_at: string;
  days: HistoricalCampaignDay[];
}

export interface UpsertHistoricalCampaignRequest {
  name: string;
  category: string;
  duration_days: number;
  currency_code: CurrencyCode;
  started_at: string;
  ended_at: string;
  days: HistoricalCampaignDay[];
}
