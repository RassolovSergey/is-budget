"use client";

import type { FormEvent } from "react";
import Link from "next/link";
import {
  Area,
  CartesianGrid,
  ComposedChart,
  Line,
  ReferenceLine,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis
} from "recharts";

import type { CurrencyCode, ForecastRun, ForecastRunSummary, Project } from "../../../../lib/types";
import { shortDate } from "../../../../lib/format";
import type { ForecastBandRow, ForecastDailyRow } from "./page-types";

type Translator = (ru: string, en: string) => string;

export function ForecastCommandSection({
  tr,
  currentIncomeTotal,
  currencyCode,
  locale,
  observedDaysCount,
  remainingDays,
  historicalCampaignsCount,
  formatMoney
}: {
  tr: Translator;
  currentIncomeTotal: number;
  currencyCode: CurrencyCode;
  locale: string;
  observedDaysCount: number;
  remainingDays: number;
  historicalCampaignsCount: number;
  formatMoney: (amount: number, code: CurrencyCode, locale: string) => string;
}) {
  return (
    <section className="forecast-command-grid" style={{ marginBottom: 14 }}>
      <div className="forecast-command-surface card">
        <div className="forecast-command-copy">
          <span className="shell-eyebrow">{tr("Прогноз", "Forecast")}</span>
          <h2 className="title">{tr("Прогноз дохода кампании", "Campaign revenue forecast")}</h2>
          <p className="subtle" style={{ margin: 0 }}>
            {tr(
              "Комбинируйте фактические доходы проекта и архив похожих кампаний, чтобы получить рабочий диапазон финального результата.",
              "Blend actual campaign income with historical category data to get a working range for the final result."
            )}
          </p>
        </div>
        <div className="forecast-command-stats">
          <article className="forecast-command-stat forecast-command-stat-accent">
            <span>{tr("Текущий факт", "Current actual")}</span>
            <strong>{formatMoney(currentIncomeTotal, currencyCode, locale)}</strong>
          </article>
          <article className="forecast-command-stat">
            <span>{tr("Прошло дней кампании", "Campaign days elapsed")}</span>
            <strong>{observedDaysCount}</strong>
          </article>
          <article className="forecast-command-stat">
            <span>{tr("Осталось дней", "Remaining days")}</span>
            <strong>{remainingDays}</strong>
          </article>
          <article className="forecast-command-stat">
            <span>{tr("Архив кампаний", "Historical archive")}</span>
            <strong>{historicalCampaignsCount}</strong>
          </article>
        </div>
      </div>
    </section>
  );
}

export function ForecastSetupSection({
  tr,
  project,
  historicalCampaignsCount,
  observedDaysCount,
  remainingDays,
  canForecastWrite,
  loading,
  selectedRun,
  locale,
  onRunForecast
}: {
  tr: Translator;
  project: Project;
  historicalCampaignsCount: number;
  observedDaysCount: number;
  remainingDays: number;
  canForecastWrite: boolean;
  loading: boolean;
  selectedRun: ForecastRun | null;
  locale: string;
  onRunForecast: (e: FormEvent) => void;
}) {
  return (
    <div className="grid two forecast-top-grid" style={{ marginBottom: 14 }}>
      <section className="card forecast-setup-card">
        <div className="finance-card-head">
          <div>
            <span className="shell-eyebrow">{tr("Данные", "Data")}</span>
            <h2 className="title">{tr("Исторические кампании", "Historical campaigns")}</h2>
          </div>
        </div>
        <p className="subtle" style={{ marginTop: 0 }}>
          {tr("Для качества прогноза загружено прошлых кампаний", "Past campaigns loaded for forecast quality")}: <b>{historicalCampaignsCount}</b>
        </p>
        <Link
          className="card"
          href={`/historical-campaigns?category=${encodeURIComponent(project.category || tr("Общее", "General"))}`}
          style={{ display: "inline-block", padding: "10px 14px" }}
        >
          {tr("Открыть историю кампаний", "Open Campaign History")}
        </Link>
      </section>

      <section className="card forecast-setup-card forecast-setup-card-accent">
        <div className="finance-card-head">
          <div>
            <span className="shell-eyebrow">{tr("Прогноз", "Forecast")}</span>
            <h2 className="title">{tr("Запуск прогноза", "Run Forecast")}</h2>
          </div>
        </div>
        <form style={{ display: "flex", gap: 10, alignItems: "center", flexWrap: "wrap" }} onSubmit={onRunForecast}>
          <p className="subtle" style={{ margin: 0 }}>
            {tr("Прошло дней кампании", "Campaign days elapsed")}: <b>{observedDaysCount}</b> · {tr("Осталось дней", "Remaining days")}: <b>{remainingDays}</b> · {tr("Модель", "Model")}: <b>{tr("Гибридная краудфандинг-модель", "Hybrid crowdfunding model")}</b>
          </p>
          <button disabled={loading || !canForecastWrite || remainingDays <= 0}>
            {!canForecastWrite
              ? tr("Только просмотр", "Read-only")
              : remainingDays <= 0
                ? tr("Нет оставшихся дней", "No remaining days")
                : loading
                  ? tr("Выполняется...", "Running...")
                  : tr("Запустить новый расчет", "Run new forecast")}
          </button>
        </form>
        {!canForecastWrite && <p className="subtle">{tr("У вас только просмотр: запуск и удаление прогнозов недоступны.", "You have read-only access: running and deleting forecasts is unavailable.")}</p>}
        {selectedRun && (
          <p className="subtle" style={{ margin: "10px 0 0" }}>
            {tr("Выбран запуск от", "Selected run from")} {new Date(selectedRun.run.created_at).toLocaleString(locale)}
          </p>
        )}
      </section>
    </div>
  );
}

export function ForecastScenarioSection({
  tr,
  selectedRun,
  totals,
  currencyCode,
  locale,
  formatMoney
}: {
  tr: Translator;
  selectedRun: ForecastRun | null;
  totals: { totalBad: number; totalBase: number; totalGood: number };
  currencyCode: CurrencyCode;
  locale: string;
  formatMoney: (amount: number, code: CurrencyCode, locale: string) => string;
}) {
  return (
    <section className="card forecast-scenarios-card" style={{ marginBottom: 14 }}>
      <h2 className="title">{tr("Сценарии по выбранному запуску", "Scenarios for Selected Run")}</h2>
      {!selectedRun && <p className="subtle" style={{ marginTop: 0 }}>{tr("Запустите расчет прогноза или выберите запись в истории.", "Run a forecast or select a run from history.")}</p>}
      {selectedRun && (
        <div className="grid three forecast-scenarios-grid" style={{ marginTop: 10 }}>
          <div className="card forecast-scenario-card forecast-scenario-card-bad" style={{ padding: 12 }}>
            <p className="subtle" style={{ margin: 0 }}>{tr("Плохой вариант", "Bad scenario")}</p>
            <p className="kpi" style={{ margin: "8px 0 0" }}>{formatMoney(totals.totalBad, currencyCode, locale)}</p>
          </div>
          <div className="card forecast-scenario-card forecast-scenario-card-base" style={{ padding: 12 }}>
            <p className="subtle" style={{ margin: 0 }}>{tr("Средний вариант", "Base scenario")}</p>
            <p className="kpi" style={{ margin: "8px 0 0" }}>{formatMoney(totals.totalBase, currencyCode, locale)}</p>
          </div>
          <div className="card forecast-scenario-card forecast-scenario-card-good" style={{ padding: 12 }}>
            <p className="subtle" style={{ margin: 0 }}>{tr("Хороший вариант", "Good scenario")}</p>
            <p className="kpi" style={{ margin: "8px 0 0" }}>{formatMoney(totals.totalGood, currencyCode, locale)}</p>
          </div>
        </div>
      )}
    </section>
  );
}

export function ForecastExportSection({
  tr,
  exportingReport,
  onDownloadPDF,
  onDownloadCSV,
  onDownloadJSON
}: {
  tr: Translator;
  exportingReport: boolean;
  onDownloadPDF: () => void;
  onDownloadCSV: () => void;
  onDownloadJSON: () => void;
}) {
  return (
    <section className="card forecast-export-card forecast-export-card-rich" style={{ marginBottom: 14 }}>
      <div className="section-head">
        <h2 className="title" style={{ margin: 0 }}>{tr("Экспорт отчёта", "Export report")}</h2>
      </div>
      <p className="subtle" style={{ marginTop: 0 }}>
        {tr(
          "Скачайте расчёт в удобном формате: PDF для отправки команде, CSV/JSON для внешней аналитики.",
          "Download this run in the right format: PDF for presentation, CSV/JSON for external analysis."
        )}
      </p>
      <div className="forecast-report-actions">
        <button type="button" onClick={onDownloadPDF} disabled={exportingReport}>
          {exportingReport ? tr("Генерация...", "Generating...") : tr("Скачать PDF-отчёт", "Download PDF report")}
        </button>
        <button type="button" className="secondary" onClick={onDownloadCSV}>
          {tr("Скачать CSV", "Download CSV")}
        </button>
        <button type="button" className="secondary" onClick={onDownloadJSON}>
          {tr("Скачать JSON", "Download JSON")}
        </button>
      </div>
    </section>
  );
}

export function ForecastChartsSection({
  tr,
  project,
  appLocale,
  actualCumulativeData,
  forecastBandData,
  actualChartTop,
  forecastChartTop,
  forecastDomain,
  forecastTickCount,
  lastActualDateISO,
  selectedRun,
  shortDateLabel,
  formatUTCDate,
  isoDateToUTC,
  renderForecastTooltip,
  formatMoney
}: {
  tr: Translator;
  project: Project;
  appLocale: string;
  actualCumulativeData: Array<{ dateISO: string; total: number | null }>;
  forecastBandData: ForecastBandRow[];
  actualChartTop: number;
  forecastChartTop: number;
  forecastDomain: [number, number];
  forecastTickCount: number;
  lastActualDateISO: string;
  selectedRun: ForecastRun | null;
  shortDateLabel: (value: string) => string;
  formatUTCDate: (ts: number, locale: string) => string;
  isoDateToUTC: (value: string) => number;
  renderForecastTooltip: (
    content: unknown,
    currencyCode: CurrencyCode,
    locale: string,
    dateLocale: string,
    labels: { date: string; actual: string; totalBad: string; totalBase: string; totalGood: string }
  ) => React.ReactNode;
  formatMoney: (amount: number, code: CurrencyCode, locale: string) => string;
}) {
  return (
    <div className="grid two" style={{ marginBottom: 14 }}>
      <section className="card dashboard-chart-card forecast-chart-card">
        <div className="dashboard-chart-head">
          <h2 className="title">{tr("Фактический накопленный доход", "Actual cumulative income")}</h2>
          <div className="dashboard-chart-meta">
            <span className="dashboard-legend-item">
              <span className="dashboard-legend-dot actual" />
              {tr("Факт", "Actual")}
            </span>
          </div>
        </div>
        <div className="dashboard-chart-frame">
          <ResponsiveContainer width="100%" height={320}>
            <ComposedChart data={actualCumulativeData} margin={{ left: 2, right: 10, top: 10, bottom: 6 }}>
              <defs>
                <linearGradient id="actualIncomeFill" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor="#2a9d8f" stopOpacity={0.35} />
                  <stop offset="100%" stopColor="#2a9d8f" stopOpacity={0.03} />
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="4 4" stroke="var(--line)" />
              <XAxis dataKey="dateISO" tickFormatter={shortDateLabel} hide={actualCumulativeData.length > 20} tick={{ fill: "var(--muted)" }} />
              <YAxis domain={[0, actualChartTop]} tick={{ fill: "var(--muted)" }} />
              <Tooltip
                labelFormatter={(label) => `${tr("Дата", "Date")}: ${shortDate(String(label))}`}
                formatter={(value, key) => [
                  formatMoney(Number(value), project.currency_code, appLocale),
                  key === "total" ? tr("Накопленный доход", "Cumulative income") : String(key)
                ]}
                contentStyle={{ borderRadius: 14, borderColor: "var(--line)", boxShadow: "0 10px 26px var(--shadow)" }}
              />
              <Area type="monotone" dataKey="total" fill="url(#actualIncomeFill)" stroke="none" />
              <Line type="monotone" dataKey="total" stroke="#2a9d8f" strokeWidth={3} dot={{ r: 3, fill: "#2a9d8f" }} />
            </ComposedChart>
          </ResponsiveContainer>
        </div>
      </section>

      <section className="card dashboard-chart-card forecast-chart-card">
        <div className="dashboard-chart-head">
          <h2 className="title">{tr("Прогноз накопленного дохода", "Cumulative income forecast")}</h2>
          <div className="dashboard-chart-meta">
            <span className="dashboard-legend-item">
              <span className="dashboard-legend-dot actual" />
              {tr("Факт", "Actual")}
            </span>
            <span className="dashboard-legend-item">
              <span className="dashboard-legend-dot forecast" />
              {tr("Средний прогноз", "Base forecast")}
            </span>
          </div>
        </div>
        <div className="dashboard-chart-frame">
          <ResponsiveContainer width="100%" height={320}>
            <ComposedChart data={forecastBandData} margin={{ left: 2, right: 10, top: 10, bottom: 6 }}>
              <defs>
                <linearGradient id="forecastRangeFill" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor="#f9c784" stopOpacity={0.4} />
                  <stop offset="100%" stopColor="#f9c784" stopOpacity={0.08} />
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="4 4" stroke="var(--line)" />
              <XAxis
                type="number"
                dataKey="ts"
                domain={forecastDomain}
                tickCount={forecastTickCount}
                tickFormatter={(value) => formatUTCDate(Number(value), appLocale)}
                tick={{ fill: "var(--muted)" }}
              />
              <YAxis domain={[0, forecastChartTop]} tick={{ fill: "var(--muted)" }} />
              <Tooltip
                content={(props) =>
                  renderForecastTooltip(props, project.currency_code, appLocale, appLocale, {
                    date: tr("Дата", "Date"),
                    actual: tr("Факт", "Actual"),
                    totalBad: tr("Плохой сценарий", "Bad scenario"),
                    totalBase: tr("Средний сценарий", "Base scenario"),
                    totalGood: tr("Хороший сценарий", "Good scenario")
                  })
                }
              />
              {lastActualDateISO && (
                <ReferenceLine
                  x={isoDateToUTC(lastActualDateISO)}
                  stroke="var(--ok)"
                  strokeDasharray="4 4"
                  label={{
                    value: tr("Граница факта", "Actual boundary"),
                    fill: "var(--muted)",
                    position: "insideTopLeft",
                    fontSize: 11
                  }}
                />
              )}
              <Line type="monotone" dataKey="actualTotal" name={tr("Фактический накопленный доход", "Actual cumulative income")} stroke="#2a9d8f" strokeWidth={2.4} dot={{ r: 2.5, fill: "#2a9d8f" }} connectNulls />
              <Area dataKey="rangeBase" stackId="band" fill="transparent" stroke="none" />
              <Area dataKey="rangeBand" stackId="band" fill="url(#forecastRangeFill)" stroke="none" />
              <Line type="monotone" dataKey="totalBad" name={tr("Плохой сценарий", "Bad scenario")} stroke="#577590" strokeDasharray="5 5" dot={false} strokeWidth={1.8} connectNulls />
              <Line type="monotone" dataKey="totalGood" name={tr("Хороший сценарий", "Good scenario")} stroke="#577590" strokeDasharray="5 5" dot={false} strokeWidth={1.8} connectNulls />
              <Line type="monotone" dataKey="totalBase" name={tr("Средний сценарий", "Base scenario")} stroke="#f3722c" strokeWidth={3} dot={{ r: 3, fill: "#f3722c" }} connectNulls />
            </ComposedChart>
          </ResponsiveContainer>
        </div>
        {!selectedRun && <p className="subtle" style={{ marginTop: 8 }}>{tr("Нет выбранного запуска прогноза.", "No forecast run selected.")}</p>}
      </section>
    </div>
  );
}

export function ForecastDailyTableSection({
  tr,
  rows,
  currencyCode,
  locale,
  formatMoney
}: {
  tr: Translator;
  rows: ForecastDailyRow[];
  currencyCode: CurrencyCode;
  locale: string;
  formatMoney: (amount: number, code: CurrencyCode, locale: string) => string;
}) {
  return (
    <section className="card forecast-daily-card" style={{ marginBottom: 14 }}>
      <h2 className="title">{tr("Прогноз по дням", "Daily forecast")}</h2>
      <div className="forecast-legend">
        <span className="forecast-pill daily">{tr("За день", "Daily")}</span>
        <span className="forecast-pill total">{tr("Накопительно", "Cumulative")}</span>
        <span className="forecast-pill bad">{tr("Плохой", "Bad")}</span>
        <span className="forecast-pill base">{tr("Средний", "Base")}</span>
        <span className="forecast-pill good">{tr("Хороший", "Good")}</span>
      </div>
      <div className="forecast-table-scroll">
        <table className="forecast-table">
          <thead>
            <tr>
              <th rowSpan={2}>{tr("Дата", "Date")}</th>
              <th colSpan={3} className="forecast-group-head daily-group">{tr("За день", "Daily")}</th>
              <th colSpan={3} className="forecast-group-head total-group">{tr("Накопительно", "Cumulative")}</th>
            </tr>
            <tr>
              <th className="bad-col">{tr("Плохой", "Bad")}</th>
              <th className="base-col">{tr("Средний", "Base")}</th>
              <th className="good-col">{tr("Хороший", "Good")}</th>
              <th className="bad-col">{tr("Плохой", "Bad")}</th>
              <th className="base-col">{tr("Средний", "Base")}</th>
              <th className="good-col">{tr("Хороший", "Good")}</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((row) => (
              <tr key={row.dateISO}>
                <td>{row.dateLabel}</td>
                <td className="bad-col">{formatMoney(row.dailyBad, currencyCode, locale)}</td>
                <td className="base-col">{formatMoney(row.dailyBase, currencyCode, locale)}</td>
                <td className="good-col">{formatMoney(row.dailyGood, currencyCode, locale)}</td>
                <td className="bad-col">{formatMoney(row.totalBad, currencyCode, locale)}</td>
                <td className="base-col">{formatMoney(row.totalBase, currencyCode, locale)}</td>
                <td className="good-col">{formatMoney(row.totalGood, currencyCode, locale)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}

export function ForecastHistorySection({
  tr,
  appLocale,
  runHistory,
  showRunHistory,
  selectedRunIDs,
  selectedRun,
  deletingRunIds,
  deletingSelected,
  canForecastWrite,
  onToggleVisibility,
  onToggleAllSelections,
  onDeleteSelectedRuns,
  onToggleRunSelection,
  onSelectRun,
  onDeleteRun,
  formatMoney,
  currencyCode
}: {
  tr: Translator;
  appLocale: string;
  runHistory: ForecastRunSummary[];
  showRunHistory: boolean;
  selectedRunIDs: string[];
  selectedRun: ForecastRun | null;
  deletingRunIds: string[];
  deletingSelected: boolean;
  canForecastWrite: boolean;
  onToggleVisibility: () => void;
  onToggleAllSelections: (checked: boolean) => void;
  onDeleteSelectedRuns: () => void;
  onToggleRunSelection: (runId: string, checked: boolean) => void;
  onSelectRun: (runId: string) => void;
  onDeleteRun: (runId: string) => void;
  formatMoney: (amount: number, code: CurrencyCode, locale: string) => string;
  currencyCode: CurrencyCode;
}) {
  const allSelected = runHistory.length > 0 && selectedRunIDs.length === runHistory.length;

  return (
    <section className="card forecast-history-card">
      <div className="section-head">
        <h2 className="title" style={{ margin: 0 }}>{tr("История запусков", "Run history")}</h2>
        <button type="button" className="secondary" onClick={onToggleVisibility}>
          {showRunHistory ? tr("Скрыть", "Hide") : tr("Показать", "Show")}
        </button>
      </div>
      <p className="subtle" style={{ marginTop: 0 }}>
        {tr("Всего запусков", "Total runs")}: <b>{runHistory.length}</b>
      </p>
      {showRunHistory && (
        <>
          {runHistory.length > 0 && (
            <div style={{ display: "flex", gap: 8, marginBottom: 10, flexWrap: "wrap", alignItems: "center" }}>
              <label style={{ display: "inline-flex", alignItems: "center", gap: 6, color: "var(--muted)" }}>
                <input type="checkbox" checked={allSelected} disabled={!canForecastWrite} onChange={(event) => onToggleAllSelections(event.target.checked)} />
                {tr("Выбрать все", "Select all")}
              </label>
              <button type="button" className="warn" disabled={!canForecastWrite || selectedRunIDs.length === 0 || deletingSelected} onClick={onDeleteSelectedRuns}>
                {deletingSelected
                  ? tr("Удаление...", "Deleting...")
                  : tr(`Удалить выбранные (${selectedRunIDs.length})`, `Delete selected (${selectedRunIDs.length})`)}
              </button>
            </div>
          )}
          {runHistory.length === 0 && <p className="subtle" style={{ marginTop: 0 }}>{tr("Запусков пока нет.", "No runs yet.")}</p>}
          {runHistory.length > 0 && (
            <div className="forecast-history-list">
              {runHistory.map((item) => {
                const active = selectedRun?.run.id === item.run.id;
                const isDeleting = deletingRunIds.includes(item.run.id);
                const isChecked = selectedRunIDs.includes(item.run.id);
                return (
                  <div key={item.run.id} className={`card forecast-history-item${active ? " active" : ""}`}>
                    <div style={{ display: "grid", gridTemplateColumns: "auto minmax(0,1fr)", gap: 10, alignItems: "start" }}>
                      <label style={{ display: "inline-flex", alignItems: "center", marginTop: 2 }}>
                        <input
                          type="checkbox"
                          checked={isChecked}
                          disabled={!canForecastWrite || isDeleting || deletingSelected}
                          onChange={(event) => onToggleRunSelection(item.run.id, event.target.checked)}
                        />
                      </label>
                      <div>
                        <div style={{ display: "flex", justifyContent: "space-between", gap: 10, flexWrap: "wrap" }}>
                          <b>{new Date(item.run.created_at).toLocaleString(appLocale)}</b>
                        </div>
                        <div className="subtle" style={{ marginTop: 4 }}>
                          {tr("Прогнозный прирост", "Forecast growth")}: {formatMoney(item.forecast_total_yhat, currencyCode, appLocale)} · {" "}
                          {tr("Плохой", "Bad")}: {formatMoney(item.forecast_total_p10, currencyCode, appLocale)} · {" "}
                          {tr("Хороший", "Good")}: {formatMoney(item.forecast_total_p90, currencyCode, appLocale)}
                        </div>
                        <div style={{ display: "flex", gap: 8, marginTop: 10, flexWrap: "wrap" }}>
                          <button type="button" className="secondary" disabled={isDeleting || deletingSelected} onClick={() => onSelectRun(item.run.id)}>
                            {tr("Открыть", "Open")}
                          </button>
                          <button type="button" className="warn" disabled={!canForecastWrite || isDeleting || deletingSelected} onClick={() => onDeleteRun(item.run.id)}>
                            {isDeleting ? tr("Удаление...", "Deleting...") : tr("Удалить", "Delete")}
                          </button>
                        </div>
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </>
      )}
    </section>
  );
}
