import type {
  CurrencyCode,
  HistoricalCampaignDay
} from "../../lib/types";

export type CampaignForm = {
  name: string;
  category: string;
  duration_days: number;
  currency_code: CurrencyCode;
  started_at: string;
  ended_at: string;
  days: HistoricalCampaignDay[];
};

export type CumulativeRow = {
  day_number: number;
  cumulative_income: number;
  day_income: number;
};

export type HistorySort =
  | "created_desc"
  | "created_asc"
  | "period_desc"
  | "period_asc"
  | "income_desc"
  | "income_asc";
