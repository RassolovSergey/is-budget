"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { useParams } from "next/navigation";

import { AppShell } from "../../../../components/AppShell";
import { useLocale } from "../../../../components/LocaleProvider";
import { ProjectNav } from "../../../../components/ProjectNav";
import { apiRequest } from "../../../../lib/api";
import { currencySymbol, money, todayISO } from "../../../../lib/format";
import type { IncomeDaily, Project } from "../../../../lib/types";
import { useAuthGuard } from "../../../../lib/useAuthGuard";

function toIsoDate(value: string): string {
  return value.slice(0, 10);
}

function addDays(value: string, days: number): string {
  const date = new Date(`${value}T00:00:00Z`);
  if (Number.isNaN(date.getTime())) return value;
  date.setUTCDate(date.getUTCDate() + days);
  return date.toISOString().slice(0, 10);
}

function campaignDatesFrom(startDate: string, durationDays: number): string[] {
  const out: string[] = [];
  let cursor = startDate;
  for (let i = 0; i < durationDays; i += 1) {
    out.push(cursor);
    cursor = addDays(cursor, 1);
  }
  return out;
}

function round2(value: number): number {
  return Math.round((value + Number.EPSILON) * 100) / 100;
}

function normalizeNumberInput(raw: string): string {
  return raw.replace(/,/g, ".").replace(/[^0-9.\-]/g, "");
}

function parseNonNegative(value: string): number | null {
  const normalized = normalizeNumberInput(value).trim();
  if (!normalized) return null;
  const num = Number(normalized);
  if (!Number.isFinite(num) || num < 0) return null;
  return round2(num);
}

type SaveState = "idle" | "saving" | "saved" | "error";

export default function IncomeDailyPage() {
  const { id } = useParams<{ id: string }>();
  const { locale } = useLocale();
  const { token, ready } = useAuthGuard();

  const [project, setProject] = useState<Project | null>(null);
  const [campaignDates, setCampaignDates] = useState<string[]>([]);
  const [values, setValues] = useState<Record<string, string>>({});
  const [error, setError] = useState("");
  const [saveState, setSaveState] = useState<SaveState>("idle");

  const loadingRef = useRef(false);
  const saveTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const saveInFlightRef = useRef(false);
  const saveAgainRef = useRef(false);
  const baselineRef = useRef<string>("");

  const tr = (ru: string, en: string) => (locale === "en" ? en : ru);

  function serializeCurrent(nextValues = values): string {
    if (!campaignDates.length) return "";
    return JSON.stringify(campaignDates.map((date) => [date, (nextValues[date] ?? "").trim()]));
  }

  const hasUnsavedChanges = useMemo(() => serializeCurrent() !== baselineRef.current, [values, campaignDates]);

  async function load() {
    if (!token) return;
    loadingRef.current = true;
    setError("");
    setSaveState("idle");
    try {
      const projectResp = await apiRequest<Project>(`/projects/${id}`, { token });
      setProject(projectResp);

      const fallbackEnd = todayISO();
      const fallbackStart = addDays(fallbackEnd, -(projectResp.duration_days - 1));
      const campaignStart = projectResp.activated_at ? toIsoDate(projectResp.activated_at) : fallbackStart;
      const allCampaignDates = campaignDatesFrom(campaignStart, projectResp.duration_days);
      setCampaignDates(allCampaignDates);

      const from = allCampaignDates[0];
      const to = allCampaignDates[allCampaignDates.length - 1] || fallbackEnd;
      const incomeResp = await apiRequest<IncomeDaily[]>(`/projects/${id}/income-daily?from=${from}&to=${to}`, { token });

      const dailyByDate: Record<string, number> = {};
      for (const row of incomeResp) {
        dailyByDate[toIsoDate(row.income_date)] = round2(Number(row.amount));
      }

      const sortedKnownDates = Object.keys(dailyByDate).sort();
      const lastKnownDate = sortedKnownDates[sortedKnownDates.length - 1] || "";

      let running = 0;
      const cumulative: Record<string, string> = {};
      for (const date of allCampaignDates) {
        running = round2(running + (dailyByDate[date] || 0));
        if (lastKnownDate && date <= lastKnownDate) {
          cumulative[date] = String(running);
        } else {
          cumulative[date] = "";
        }
      }

      setValues(cumulative);
      baselineRef.current = JSON.stringify(allCampaignDates.map((date) => [date, (cumulative[date] ?? "").trim()]));
    } catch (err) {
      setError((err as Error).message);
      setSaveState("error");
    } finally {
      loadingRef.current = false;
    }
  }

  useEffect(() => {
    if (ready) {
      void load();
    }
  }, [ready, id, token]);

  function clearSaveTimer() {
    if (saveTimerRef.current) {
      clearTimeout(saveTimerRef.current);
      saveTimerRef.current = null;
    }
  }

  async function performSave() {
    if (!token || !project || !campaignDates.length) return;
    if (!hasUnsavedChanges) return;

    if (saveInFlightRef.current) {
      saveAgainRef.current = true;
      return;
    }

    const points: Array<{ date: string; cumulative_amount: number | null }> = [];
    for (const date of campaignDates) {
      const raw = values[date] ?? "";
      const parsed = parseNonNegative(raw);
      if (raw.trim() !== "" && parsed === null) {
        setError(tr(`Некорректная сумма для ${date}. Используйте число >= 0.`, `Invalid amount for ${date}. Use a number >= 0.`));
        setSaveState("error");
        return;
      }
      points.push({ date, cumulative_amount: parsed });
    }

    saveInFlightRef.current = true;
    setSaveState("saving");
    setError("");

    try {
      await apiRequest(`/projects/${id}/income-daily/cumulative`, {
        method: "PUT",
        token,
        body: { points }
      });
      baselineRef.current = serializeCurrent();
      setSaveState("saved");
      setTimeout(() => {
        if (!saveInFlightRef.current) setSaveState("idle");
      }, 900);
    } catch (err) {
      setError((err as Error).message);
      setSaveState("error");
    } finally {
      saveInFlightRef.current = false;
      if (saveAgainRef.current) {
        saveAgainRef.current = false;
        void performSave();
      }
    }
  }

  function queueSave(immediate = false) {
    if (loadingRef.current) return;
    clearSaveTimer();
    if (immediate) {
      void performSave();
      return;
    }
    saveTimerRef.current = setTimeout(() => {
      void performSave();
    }, 650);
  }

  useEffect(() => {
    if (!project) return;
    if (!hasUnsavedChanges) return;
    queueSave(false);
    return () => clearSaveTimer();
  }, [values, campaignDates, project]);

  if (!ready || !project) return null;

  const canFinanceWrite = project.current_user_role === "PM" || project.current_user_role === "ANALYST";
  const projectCurrency = project.currency_code;
  const campaignStart = campaignDates[0] || "—";
  const campaignEnd = campaignDates[campaignDates.length - 1] || "—";
  const filledDays = campaignDates.filter((date) => (values[date] ?? "").trim() !== "").length;
  const latestFilledDate = [...campaignDates].reverse().find((date) => (values[date] ?? "").trim() !== "") || "";
  const latestValue = latestFilledDate ? Number(values[latestFilledDate] || 0) : 0;

  return (
    <AppShell title={project.name} className="income-daily-shell">
      <ProjectNav projectId={id} />

      <section className="card income-daily-card">
        <div className="income-daily-head">
          <div>
            <span className="shell-eyebrow">{tr("Доходы", "Income")}</span>
            <h2 className="title" style={{ margin: 0 }}>{tr("Накопленный доход по дням", "Cumulative income by day")}</h2>
            <p className="subtle income-daily-description">
              {tr(
                "Заполняйте накопительный доход по дням кампании. Автосохранение работает в фоне, а возвраты учитываются автоматически через пересчёт дельт.",
                "Enter cumulative campaign income day by day. Autosave runs in the background and refunds are handled automatically through delta recalculation."
              )}
            </p>
          </div>
          <div className={`income-save-badge state-${saveState}`}>
            {saveState === "saving" && tr("Сохранение...", "Saving...")}
            {saveState === "saved" && tr("Сохранено", "Saved")}
            {saveState === "error" && tr("Ошибка", "Error")}
            {(saveState === "idle" || !hasUnsavedChanges) && tr("Автосохранение", "Autosave")}
          </div>
        </div>

        <div className="income-summary-grid">
          <article className="income-summary-card income-summary-card-accent">
            <span>{tr("Последняя зафиксированная сумма", "Latest recorded total")}</span>
            <strong>{money(latestValue, projectCurrency)}</strong>
            <p>{latestFilledDate || tr("Данных пока нет", "No data yet")}</p>
          </article>
          <article className="income-summary-card">
            <span>{tr("Заполнено дней", "Filled days")}</span>
            <strong>{filledDays}</strong>
            <p>{tr("из", "of")} {project.duration_days}</p>
          </article>
          <article className="income-summary-card">
            <span>{tr("Период кампании", "Campaign period")}</span>
            <strong>{campaignStart}</strong>
            <p>{campaignEnd}</p>
          </article>
        </div>

        {error && <p style={{ color: "#bc4749", marginTop: 6 }}>{error}</p>}
        {!canFinanceWrite && (
          <p className="subtle" style={{ marginTop: 6 }}>
            {tr(
              "У вас только просмотр. Изменение доходов доступно ролям PM/ANALYST.",
              "You have read-only access. Editing income is available for PM/ANALYST roles."
            )}
          </p>
        )}

        <div className="income-days-grid income-days-grid-premium" style={{ marginTop: 12 }}>
          {campaignDates.map((date, index) => {
            const isFilled = (values[date] ?? "").trim() !== "";
            return (
              <article key={date} className={`income-day-item ${isFilled ? "filled" : "empty"}`}>
                <div className="income-day-head">
                  <div>
                    <strong>{tr("День", "Day")} {index + 1}</strong>
                    <span className="subtle">{date}</span>
                  </div>
                  <span className={`income-day-status ${isFilled ? "filled" : "empty"}`}>
                    {isFilled ? tr("Заполнен", "Filled") : tr("Ожидает", "Pending")}
                  </span>
                </div>
                <label className="income-day-field">
                  {tr("Накопленная сумма", "Cumulative amount")}
                  <div className="input-with-suffix">
                    <input
                      type="text"
                      inputMode="decimal"
                      placeholder={tr("Введите накопленный доход", "Enter cumulative income")}
                      value={values[date] ?? ""}
                      onChange={(e) => {
                        const next = normalizeNumberInput(e.target.value);
                        setValues((prev) => ({ ...prev, [date]: next }));
                      }}
                      onBlur={() => queueSave(true)}
                      disabled={!canFinanceWrite}
                    />
                    <span className="input-suffix">{currencySymbol(projectCurrency)}</span>
                  </div>
                </label>
              </article>
            );
          })}
        </div>
      </section>
    </AppShell>
  );
}
