export type ForecastDailyRow = {
  dateISO: string;
  dateLabel: string;
  dailyBad: number;
  dailyBase: number;
  dailyGood: number;
  totalBad: number;
  totalBase: number;
  totalGood: number;
};

export type ForecastBandRow = {
  ts: number;
  dateISO: string;
  dateLabel: string;
  actualTotal?: number;
  totalBad?: number;
  totalBase?: number;
  totalGood?: number;
  rangeBase?: number;
  rangeBand?: number;
};
