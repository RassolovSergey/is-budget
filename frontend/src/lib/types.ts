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
  full_name: string;
  nickname: string;
  avatar_path?: string | null;
  default_currency_code: CurrencyCode;
  created_at: string;
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
}

export interface PayoutPreview {
  project_currency_code: CurrencyCode;
  secondary_currency_code?: CurrencyCode;
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
