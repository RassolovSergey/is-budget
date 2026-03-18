"use client";

import { FormEvent, useEffect, useMemo, useState } from "react";
import { useParams } from "next/navigation";

import { AppShell } from "../../../../components/AppShell";
import { useConfirm } from "../../../../components/ConfirmProvider";
import { useLocale } from "../../../../components/LocaleProvider";
import { ProjectNav } from "../../../../components/ProjectNav";
import { apiDownload, apiRequest } from "../../../../lib/api";
import { shortDate, todayISO } from "../../../../lib/format";
import type { CurrencyCode, ForecastRun, ForecastRunSummary, HistoricalCampaign, IncomeDaily, Project } from "../../../../lib/types";
import { useAuthGuard } from "../../../../lib/useAuthGuard";
import {
  ForecastChartsSection,
  ForecastCommandSection,
  ForecastDailyTableSection,
  ForecastExportSection,
  ForecastHistorySection,
  ForecastScenarioSection,
  ForecastSetupSection
} from "./ForecastSections";
import type { ForecastBandRow, ForecastDailyRow } from "./page-types";

function toIsoDate(value: string): string {
  return value.slice(0, 10);
}

function addDays(value: string, days: number): string {
  const date = new Date(`${value}T00:00:00Z`);
  if (Number.isNaN(date.getTime())) return value;
  date.setUTCDate(date.getUTCDate() + days);
  return date.toISOString().slice(0, 10);
}

function isoDateToUTC(isoLike: string): number {
  const iso = toIsoDate(isoLike);
  const [year, month, day] = iso.split("-").map((part) => Number(part));
  if (!Number.isFinite(year) || !Number.isFinite(month) || !Number.isFinite(day)) {
    const parsed = Date.parse(isoLike);
    return Number.isFinite(parsed) ? parsed : 0;
  }
  return Date.UTC(year, month - 1, day);
}

function formatUTCDate(ts: number, locale: string): string {
  if (!Number.isFinite(ts) || ts <= 0) return "—";
  return new Date(ts).toLocaleDateString(locale);
}

function currencySymbol(code: CurrencyCode): string {
  switch (code) {
    case "USD":
      return "$";
    case "RUB":
      return "₽";
    case "EUR":
    default:
      return "€";
  }
}

function formatMoney(amount: number, code: CurrencyCode, locale: string): string {
  return `${amount.toLocaleString(locale, { maximumFractionDigits: 2 })} ${currencySymbol(code)}`;
}

function csvEscape(value: string): string {
  if (!/[",;\n]/.test(value)) return value;
  return `"${value.replaceAll("\"", "\"\"")}"`;
}

function triggerDownload(fileName: string, blob: Blob) {
  const objectURL = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = objectURL;
  link.download = fileName;
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(objectURL);
}

function renderForecastTooltip(
  content: unknown,
  currencyCode: CurrencyCode,
  locale: string,
  dateLocale: string,
  labels: {
    date: string;
    actual: string;
    totalBad: string;
    totalBase: string;
    totalGood: string;
  }
) {
  const props = content as {
    active?: boolean;
    label?: string | number;
    payload?: Array<{ dataKey?: string; value?: number }>;
  };

  if (!props.active || !props.payload || props.payload.length === 0) {
    return null;
  }

  const labelMap: Record<string, string> = {
    actualTotal: labels.actual,
    totalBad: labels.totalBad,
    totalBase: labels.totalBase,
    totalGood: labels.totalGood
  };
  const order: Record<string, number> = {
    actualTotal: 1,
    totalBad: 2,
    totalBase: 3,
    totalGood: 4
  };

  const rows = props.payload
    .filter((item) => item.dataKey && labelMap[item.dataKey])
    .sort((a, b) => (order[a.dataKey ?? ""] ?? 99) - (order[b.dataKey ?? ""] ?? 99));

  if (rows.length === 0) {
    return null;
  }

  return (
    <div
      style={{
        border: "1px solid var(--line)",
        borderRadius: 12,
        padding: "10px 12px",
        background: "var(--surface)",
        boxShadow: "0 8px 22px var(--shadow)"
      }}
    >
      <p style={{ margin: 0, color: "var(--muted)" }}>
        {labels.date}: {formatUTCDate(Number(props.label ?? 0), dateLocale)}
      </p>
      {rows.map((row) => (
        <p key={String(row.dataKey)} style={{ margin: "4px 0 0" }}>
          {labelMap[row.dataKey ?? ""]}: <b>{formatMoney(Number(row.value ?? 0), currencyCode, locale)}</b>
        </p>
      ))}
    </div>
  );
}

export default function ForecastPage() {
  const { id } = useParams<{ id: string }>();
  const { locale } = useLocale();
  const { token, ready } = useAuthGuard();
  const confirmAction = useConfirm();
  const appLocale = locale === "en" ? "en-US" : "ru-RU";
  const tr = (ru: string, en: string) => (locale === "en" ? en : ru);

  const [project, setProject] = useState<Project | null>(null);
  const [selectedRun, setSelectedRun] = useState<ForecastRun | null>(null);
  const [runHistory, setRunHistory] = useState<ForecastRunSummary[]>([]);
  const [selectedRunIDs, setSelectedRunIDs] = useState<string[]>([]);

  const [history, setHistory] = useState<IncomeDaily[]>([]);
  const [historyFrom, setHistoryFrom] = useState<string>("");
  const [historyTo, setHistoryTo] = useState<string>("");
  const [historicalCampaignsCount, setHistoricalCampaignsCount] = useState(0);

  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [loadingRunId, setLoadingRunId] = useState<string>("");
  const [deletingRunIds, setDeletingRunIds] = useState<string[]>([]);
  const [deletingSelected, setDeletingSelected] = useState(false);
  const [showRunHistory, setShowRunHistory] = useState(false);
  const [exportingReport, setExportingReport] = useState(false);

  async function loadRunByID(authToken: string, runID: string) {
    const run = await apiRequest<ForecastRun>(`/projects/${id}/forecast/${runID}`, { token: authToken });
    setSelectedRun(run);
  }

  async function loadRunHistory(authToken: string, preferredRunID = "") {
    const runs = await apiRequest<ForecastRunSummary[]>(`/projects/${id}/forecast/runs`, { token: authToken });
    setRunHistory(runs);

    if (runs.length === 0) {
      setSelectedRun(null);
      return;
    }

    const targetRunID =
      preferredRunID && runs.some((item) => item.run.id === preferredRunID)
        ? preferredRunID
        : runs[0].run.id;

    await loadRunByID(authToken, targetRunID);
  }

  useEffect(() => {
    const validIDs = new Set(runHistory.map((item) => item.run.id));
    setSelectedRunIDs((prev) => prev.filter((runID) => validIDs.has(runID)));
  }, [runHistory]);

  async function load() {
    if (!token) return;
    setError("");

    try {
      const projectResp = await apiRequest<Project>(`/projects/${id}`, { token });
      setProject(projectResp);

      const fallbackToday = todayISO();
      const fallbackStart = addDays(fallbackToday, -(projectResp.duration_days - 1));
      const campaignStart = projectResp.activated_at ? toIsoDate(projectResp.activated_at) : fallbackStart;
      const campaignEnd = addDays(campaignStart, projectResp.duration_days - 1);
      setHistoryFrom(campaignStart);
      setHistoryTo(campaignEnd);

      const historyResp = await apiRequest<IncomeDaily[]>(
        `/projects/${id}/income-daily?from=${campaignStart}&to=${campaignEnd}`,
        { token }
      );
      setHistory(historyResp);

      try {
        const encodedCategory = encodeURIComponent(projectResp.category || tr("Общее", "General"));
        const historicalResp = await apiRequest<HistoricalCampaign[]>(`/historical-campaigns?category=${encodedCategory}`, { token });
        setHistoricalCampaignsCount(historicalResp.length);
      } catch {
        setHistoricalCampaignsCount(0);
      }

      await loadRunHistory(token);
    } catch (err) {
      setError((err as Error).message);
    }
  }

  useEffect(() => {
    if (ready) {
      load();
    }
  }, [ready]);

  async function runForecast(e: FormEvent) {
    e.preventDefault();
    if (!project || !token) return;
    if (!(project.current_user_role === "PM" || project.current_user_role === "ANALYST")) {
      setError(tr("Недостаточно прав для запуска прогноза.", "Not enough permissions to run forecast."));
      return;
    }
    if (remainingDays <= 0) {
      setError(tr("Все дни кампании уже заполнены фактическими доходами. Прогноз больше не требуется.", "All campaign days are already filled with actual income. Forecast is no longer required."));
      return;
    }

    const horizon = Math.min(30, Math.max(1, remainingDays));
    setLoading(true);
    setError("");
    try {
      const run = await apiRequest<ForecastRun>(`/projects/${id}/forecast`, {
        method: "POST",
        token,
        body: { horizon_days: horizon }
      });
      setSelectedRun(run);
      setShowRunHistory(true);
      await loadRunHistory(token, run.run.id);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }

  async function selectRun(runID: string) {
    if (!token) return;
    setLoadingRunId(runID);
    setError("");
    try {
      await loadRunByID(token, runID);
    } catch {
      await loadRunHistory(token);
      setError(tr("Запуск не удалось открыть. История обновлена.", "Failed to open run. History has been refreshed."));
    } finally {
      setLoadingRunId("");
    }
  }

  async function removeRuns(runIDs: string[]) {
    if (!token || runIDs.length === 0) return;
    if (!(project?.current_user_role === "PM" || project?.current_user_role === "ANALYST")) {
      setError(tr("Недостаточно прав для удаления запусков.", "Not enough permissions to delete runs."));
      return;
    }

    const confirmed = await confirmAction({
      title: runIDs.length > 1 ? tr("Удалить выбранные запуски?", "Delete selected runs?") : tr("Удалить запуск прогноза?", "Delete forecast run?"),
      message:
        runIDs.length > 1
          ? tr(`Будет удалено запусков: ${runIDs.length}. Это действие нельзя отменить.`, `${runIDs.length} runs will be deleted. This action cannot be undone.`)
          : tr("Запись будет удалена из истории без возможности восстановления.", "This run will be permanently deleted from history."),
      confirmText: tr("Удалить", "Delete"),
      cancelText: tr("Отмена", "Cancel"),
      tone: "danger"
    });
    if (!confirmed) {
      return;
    }

    setDeletingRunIds(runIDs);
    setDeletingSelected(runIDs.length > 1);
    setError("");
    try {
      const results = await Promise.allSettled(
        runIDs.map((runID) =>
          apiRequest<void>(`/projects/${id}/forecast/${runID}`, {
            method: "DELETE",
            token
          })
        )
      );

      const failedCount = results.filter((result) => result.status === "rejected").length;
      if (failedCount > 0) {
        setError(tr(`Удалено не всё: ошибок ${failedCount}. Обновите и повторите.`, `Some runs were not deleted: ${failedCount} errors. Refresh and retry.`));
      }

      const preferredRunID =
        selectedRun && !runIDs.includes(selectedRun.run.id)
          ? selectedRun.run.id
          : "";
      await loadRunHistory(token, preferredRunID);
      setSelectedRunIDs((prev) => prev.filter((runID) => !runIDs.includes(runID)));
    } finally {
      setDeletingRunIds([]);
      setDeletingSelected(false);
    }
  }

  async function deleteRun(runID: string) {
    await removeRuns([runID]);
  }

  async function deleteSelectedRuns() {
    await removeRuns(selectedRunIDs);
  }

  function toggleRunSelection(runID: string, checked: boolean) {
    setSelectedRunIDs((prev) => {
      if (checked) {
        if (prev.includes(runID)) return prev;
        return [...prev, runID];
      }
      return prev.filter((item) => item !== runID);
    });
  }

  function toggleAllSelections(checked: boolean) {
    if (checked) {
      setSelectedRunIDs(runHistory.map((item) => item.run.id));
      return;
    }
    setSelectedRunIDs([]);
  }

  const historyMap = useMemo(() => {
    const map: Record<string, number> = {};
    for (const item of history) {
      map[toIsoDate(item.income_date)] = item.amount;
    }
    return map;
  }, [history]);

  const knownDatesSorted = useMemo(
    () => Array.from(new Set(history.map((item) => toIsoDate(item.income_date)))).sort(),
    [history]
  );

  const observedDaysCount = knownDatesSorted.length;
  const lastActualDateISO = knownDatesSorted[knownDatesSorted.length - 1] ?? "";
  const remainingDays = Math.max((project?.duration_days || 0) - observedDaysCount, 0);

  const currentIncomeTotal = useMemo(
    () => history.reduce((sum, item) => sum + item.amount, 0),
    [history]
  );

  const actualCumulativeData = useMemo(() => {
    const rows: Array<{ dateISO: string; date: string; total: number | null }> = [];
    if (!historyFrom || !historyTo || historyFrom > historyTo) {
      return rows;
    }

    const lastFilledDate = knownDatesSorted[knownDatesSorted.length - 1] || "";

    let cursor = historyFrom;
    let cumulative = 0;
    let guard = 0;
    while (cursor <= historyTo && guard < 1000) {
      cumulative += historyMap[cursor] ?? 0;
      rows.push({
        dateISO: cursor,
        date: shortDate(cursor),
        total: lastFilledDate && cursor <= lastFilledDate ? cumulative : null
      });
      cursor = addDays(cursor, 1);
      guard += 1;
    }
    return rows;
  }, [historyFrom, historyTo, knownDatesSorted, historyMap]);

  const forecastPointsAfterActual = useMemo(() => {
    if (!selectedRun) return [];
    if (!lastActualDateISO) return selectedRun.points;
    return selectedRun.points.filter((point) => toIsoDate(point.forecast_date) > lastActualDateISO);
  }, [selectedRun, lastActualDateISO]);

  const forecastTotals = useMemo(() => {
    if (!selectedRun || !project) {
      return {
        totalBad: currentIncomeTotal,
        totalBase: currentIncomeTotal,
        totalGood: currentIncomeTotal
      };
    }

    const addBase = forecastPointsAfterActual.reduce((sum, point) => sum + point.yhat, 0);
    const addBad = forecastPointsAfterActual.reduce((sum, point) => sum + point.p10, 0);
    const addGood = forecastPointsAfterActual.reduce((sum, point) => sum + point.p90, 0);

    return {
      totalBad: currentIncomeTotal + addBad,
      totalBase: currentIncomeTotal + addBase,
      totalGood: currentIncomeTotal + addGood
    };
  }, [selectedRun, forecastPointsAfterActual, currentIncomeTotal, project]);

  const forecastDailyData = useMemo<ForecastDailyRow[]>(() => {
    if (!selectedRun) return [];

    let cumBad = currentIncomeTotal;
    let cumBase = currentIncomeTotal;
    let cumGood = currentIncomeTotal;

    return forecastPointsAfterActual.map((point) => {
      const dateISO = toIsoDate(point.forecast_date);
      cumBad += point.p10;
      cumBase += point.yhat;
      cumGood += point.p90;
      return {
        dateISO,
        dateLabel: shortDate(dateISO),
        dailyBad: point.p10,
        dailyBase: point.yhat,
        dailyGood: point.p90,
        totalBad: cumBad,
        totalBase: cumBase,
        totalGood: cumGood
      };
    });
  }, [forecastPointsAfterActual, currentIncomeTotal]);

  const forecastBandData = useMemo<ForecastBandRow[]>(
    () => {
      const byDate = new Map<string, ForecastBandRow>();

      for (const row of actualCumulativeData) {
        const dateISO = toIsoDate(row.dateISO);
        byDate.set(dateISO, {
          ...(byDate.get(dateISO) || {}),
          ts: isoDateToUTC(dateISO),
          dateISO,
          dateLabel: shortDate(dateISO),
          actualTotal: row.total ?? undefined
        });
      }

      for (const row of forecastDailyData) {
        const dateISO = toIsoDate(row.dateISO);
        byDate.set(dateISO, {
          ...(byDate.get(dateISO) || {}),
          ts: isoDateToUTC(dateISO),
          dateISO,
          dateLabel: shortDate(dateISO),
          totalBad: row.totalBad,
          totalBase: row.totalBase,
          totalGood: row.totalGood,
          rangeBase: row.totalBad,
          rangeBand: Math.max(row.totalGood - row.totalBad, 0)
        });
      }

      return Array.from(byDate.values()).sort((left, right) => left.ts - right.ts);
    },
    [actualCumulativeData, forecastDailyData]
  );

  const forecastDomain = useMemo<[number, number]>(() => {
    if (historyFrom && historyTo) {
      return [isoDateToUTC(historyFrom), isoDateToUTC(historyTo)];
    }
    if (!forecastBandData.length) {
      const now = Date.now();
      return [now - 7 * 24 * 60 * 60 * 1000, now];
    }
    return [forecastBandData[0].ts, forecastBandData[forecastBandData.length - 1].ts];
  }, [historyFrom, historyTo, forecastBandData]);

  const forecastTickCount = useMemo(() => {
    if (!project) return 6;
    return Math.min(8, Math.max(4, project.duration_days));
  }, [project]);

  const actualChartTop = useMemo(() => {
    const values = actualCumulativeData.map((row) => (typeof row.total === "number" ? row.total : 0));
    const max = Math.max(0, ...values);
    if (max <= 0) return 100;
    return Math.max(100, Math.ceil((max * 1.1) / 10) * 10);
  }, [actualCumulativeData]);

  const forecastChartTop = useMemo(() => {
    const values = forecastBandData.flatMap((row) => [
      row.actualTotal || 0,
      row.totalBad || 0,
      row.totalBase || 0,
      row.totalGood || 0
    ]);
    const max = Math.max(0, ...values);
    if (max <= 0) return 100;
    return Math.max(100, Math.ceil((max * 1.1) / 10) * 10);
  }, [forecastBandData]);

  const allSelected = runHistory.length > 0 && selectedRunIDs.length === runHistory.length;
  const canForecastWrite = project?.current_user_role === "PM" || project?.current_user_role === "ANALYST";

  function downloadForecastCSV() {
    if (!project || !selectedRun) return;
    const header = [
      "run_id",
      "created_at",
      "date",
      "daily_bad",
      "daily_base",
      "daily_good",
      "total_bad",
      "total_base",
      "total_good"
    ];
    const rows = forecastDailyData.map((row) => [
      selectedRun.run.id,
      selectedRun.run.created_at,
      row.dateISO,
      row.dailyBad.toFixed(2),
      row.dailyBase.toFixed(2),
      row.dailyGood.toFixed(2),
      row.totalBad.toFixed(2),
      row.totalBase.toFixed(2),
      row.totalGood.toFixed(2)
    ]);
    const content = [header, ...rows].map((line) => line.map((cell) => csvEscape(String(cell))).join(";")).join("\n");
    const blob = new Blob([content], { type: "text/csv;charset=utf-8" });
    triggerDownload(`forecast-${project.name}-${selectedRun.run.id}.csv`, blob);
  }

  function downloadForecastJSON() {
    if (!project || !selectedRun) return;
    const payload = {
      project: {
        id: project.id,
        name: project.name,
        currency_code: project.currency_code
      },
      run: selectedRun.run,
      totals: forecastTotals,
      daily: forecastDailyData,
      actual_cumulative: actualCumulativeData
    };
    const blob = new Blob([JSON.stringify(payload, null, 2)], { type: "application/json;charset=utf-8" });
    triggerDownload(`forecast-${project.name}-${selectedRun.run.id}.json`, blob);
  }

  async function downloadForecastPDF() {
    if (!project || !selectedRun || !token) return;
    setExportingReport(true);
    setError("");
    try {
      const blob = await apiDownload(`/projects/${id}/forecast/${selectedRun.run.id}/report.pdf`, token);
      triggerDownload(`forecast-report-${project.name}-${selectedRun.run.id}.pdf`, blob);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setExportingReport(false);
    }
  }

  if (!ready || !project) return null;

  return (
    <AppShell title={project.name} className="forecast-shell">
      <ProjectNav projectId={id} />
      {error && <p style={{ color: "#bc4749" }}>{error}</p>}

      <ForecastCommandSection
        tr={tr}
        currentIncomeTotal={currentIncomeTotal}
        currencyCode={project.currency_code}
        locale={appLocale}
        observedDaysCount={observedDaysCount}
        remainingDays={remainingDays}
        historicalCampaignsCount={historicalCampaignsCount}
        formatMoney={formatMoney}
      />

      <ForecastSetupSection
        tr={tr}
        project={project}
        historicalCampaignsCount={historicalCampaignsCount}
        observedDaysCount={observedDaysCount}
        remainingDays={remainingDays}
        canForecastWrite={canForecastWrite}
        loading={loading}
        selectedRun={selectedRun}
        locale={appLocale}
        onRunForecast={runForecast}
      />

      <ForecastScenarioSection
        tr={tr}
        selectedRun={selectedRun}
        totals={forecastTotals}
        currencyCode={project.currency_code}
        locale={appLocale}
        formatMoney={formatMoney}
      />

      {selectedRun && (
        <ForecastExportSection
          tr={tr}
          exportingReport={exportingReport}
          onDownloadPDF={() => void downloadForecastPDF()}
          onDownloadCSV={downloadForecastCSV}
          onDownloadJSON={downloadForecastJSON}
        />
      )}

      <ForecastChartsSection
        tr={tr}
        project={project}
        appLocale={appLocale}
        actualCumulativeData={actualCumulativeData}
        forecastBandData={forecastBandData}
        actualChartTop={actualChartTop}
        forecastChartTop={forecastChartTop}
        forecastDomain={forecastDomain}
        forecastTickCount={forecastTickCount}
        lastActualDateISO={lastActualDateISO}
        selectedRun={selectedRun}
        shortDateLabel={shortDate}
        formatUTCDate={formatUTCDate}
        isoDateToUTC={isoDateToUTC}
        renderForecastTooltip={renderForecastTooltip}
        formatMoney={formatMoney}
      />

      {selectedRun && (
        <ForecastDailyTableSection
          tr={tr}
          rows={forecastDailyData}
          currencyCode={project.currency_code}
          locale={appLocale}
          formatMoney={formatMoney}
        />
      )}

      <ForecastHistorySection
        tr={tr}
        appLocale={appLocale}
        runHistory={runHistory}
        showRunHistory={showRunHistory}
        selectedRunIDs={selectedRunIDs}
        selectedRun={selectedRun}
        deletingRunIds={deletingRunIds}
        deletingSelected={deletingSelected}
        canForecastWrite={canForecastWrite}
        onToggleVisibility={() => setShowRunHistory((prev) => !prev)}
        onToggleAllSelections={toggleAllSelections}
        onDeleteSelectedRuns={() => void deleteSelectedRuns()}
        onToggleRunSelection={toggleRunSelection}
        onSelectRun={(runId) => void selectRun(runId)}
        onDeleteRun={(runId) => void deleteRun(runId)}
        formatMoney={formatMoney}
        currencyCode={project.currency_code}
      />
    </AppShell>
  );
}
