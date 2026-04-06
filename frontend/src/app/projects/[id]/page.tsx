"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { Area, Bar, BarChart, CartesianGrid, Cell, ComposedChart, Line, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";

import { AppShell } from "../../../components/AppShell";
import { useConfirm } from "../../../components/ConfirmProvider";
import { useLocale } from "../../../components/LocaleProvider";
import { ProjectNav } from "../../../components/ProjectNav";
import { useToast } from "../../../components/ToastProvider";
import { apiRequest } from "../../../lib/api";
import { toIntlLocale } from "../../../lib/i18n";
import { parseMemberLookup } from "../../../lib/memberLookup";
import { money, shortDate } from "../../../lib/format";
import { normalizeProjectInviteError } from "../../../lib/projectInviteErrors";
import { projectRoleOptions, projectRoleView } from "../../../lib/projectRoles";
import type { DashboardResponse, ForecastRun, Project, ProjectMemberWithUser, ProjectStatus, User } from "../../../lib/types";
import { useAuthGuard } from "../../../lib/useAuthGuard";

function completionDate(project: Project, locale: string): string | null {
  if (!project.activated_at) return null;
  const start = new Date(project.activated_at);
  if (Number.isNaN(start.getTime())) return null;
  start.setDate(start.getDate() + project.duration_days - 1);
  return start.toLocaleDateString(locale);
}

function toIsoDate(value: string): string {
  return value.slice(0, 10);
}

function addDays(value: string, days: number): string {
  const date = new Date(`${value}T00:00:00Z`);
  if (Number.isNaN(date.getTime())) return value;
  date.setUTCDate(date.getUTCDate() + days);
  return date.toISOString().slice(0, 10);
}

function todayIsoDate(): string {
  return new Date().toISOString().slice(0, 10);
}

function diffDaysInclusive(fromIso: string, toIso: string): number {
  const from = new Date(`${fromIso}T00:00:00Z`);
  const to = new Date(`${toIso}T00:00:00Z`);
  if (Number.isNaN(from.getTime()) || Number.isNaN(to.getTime())) return 1;
  const diffMs = to.getTime() - from.getTime();
  return Math.max(1, Math.floor(diffMs / 86_400_000) + 1);
}

function resolveDashboardDays(project: Project): number {
  if (!project.activated_at) return 30;
  const start = toIsoDate(project.activated_at);
  const today = todayIsoDate();
  const required = diffDaysInclusive(start, today);
  return Math.min(365, Math.max(30, required));
}

function buildCumulativeIncome(
  project: Project,
  series: DashboardResponse["daily_income_series"],
  forecastPoints: ForecastRun["points"] = []
) {
  if (!project.activated_at && !series.length) return [];

  const sumsByDate = new Map<string, number>();
  let latestEnteredDate: string | null = null;
  for (const row of series) {
    const key = toIsoDate(row.income_date);
    sumsByDate.set(key, (sumsByDate.get(key) || 0) + row.amount);
    if (!latestEnteredDate || key > latestEnteredDate) {
      latestEnteredDate = key;
    }
  }
  const forecastByDate = new Map<string, number>();
  for (const point of forecastPoints) {
    const key = toIsoDate(point.forecast_date);
    forecastByDate.set(key, (forecastByDate.get(key) || 0) + Math.max(point.yhat, 0));
  }

  let firstDate = series.length ? toIsoDate(series[0].income_date) : todayIsoDate();
  let lastDate = series.length ? toIsoDate(series[series.length - 1].income_date) : firstDate;

  if (project.activated_at) {
    firstDate = toIsoDate(project.activated_at);
    lastDate = addDays(firstDate, project.duration_days - 1);
  }

  if (lastDate < firstDate) {
    const temp = firstDate;
    firstDate = lastDate;
    lastDate = temp;
  }

  let visibleThroughDate = lastDate;
  if (project.activated_at) {
    const today = todayIsoDate();
    const defaultVisible = today < lastDate ? today : lastDate;
    visibleThroughDate = latestEnteredDate || defaultVisible;
  } else if (latestEnteredDate) {
    visibleThroughDate = latestEnteredDate;
  }
  if (visibleThroughDate > lastDate) {
    visibleThroughDate = lastDate;
  }
  if (visibleThroughDate < firstDate) {
    visibleThroughDate = firstDate;
  }

  const hasForecast = forecastByDate.size > 0;
  const result: Array<{
    date: string;
    date_iso: string;
    cumulative_amount: number;
    actual_amount: number | null;
    forecast_amount: number | null;
  }> = [];
  let cursor = firstDate;
  let runningActual = 0;
  let runningForecast: number | null = null;
  let guard = 0;

  for (const [date, amount] of sumsByDate.entries()) {
    if (date < firstDate) runningActual += amount;
  }

  while (cursor <= lastDate && guard < 2000) {
    runningActual += sumsByDate.get(cursor) || 0;
    const roundedActual = Number(runningActual.toFixed(2));
    let actualAmount: number | null = null;
    let forecastAmount: number | null = null;

    if (cursor <= visibleThroughDate) {
      actualAmount = roundedActual;
      if (hasForecast && cursor === visibleThroughDate) {
        forecastAmount = roundedActual;
        runningForecast = roundedActual;
      }
    } else if (hasForecast) {
      if (runningForecast === null) {
        runningForecast = roundedActual;
      }
      runningForecast += forecastByDate.get(cursor) || 0;
      forecastAmount = Number(runningForecast.toFixed(2));
    }

    const cumulative = actualAmount ?? forecastAmount ?? roundedActual;
    result.push({
      date: shortDate(cursor),
      date_iso: cursor,
      cumulative_amount: cumulative,
      actual_amount: actualAmount,
      forecast_amount: forecastAmount
    });
    const next = addDays(cursor, 1);
    if (next === cursor) break;
    cursor = next;
    guard += 1;
  }

  return result;
}

type MemberPayoutStats = {
  hasRules: boolean;
  payoutAmountProject: number;
  payoutAmountDisplay: number;
  payoutCurrencyCode: DashboardResponse["payout_preview"][number]["currency_code"];
  effectivePercent: number;
};

function round2(value: number): number {
  return Math.round((value + Number.EPSILON) * 100) / 100;
}

function buildMemberPayoutMap(
  items: DashboardResponse["payout_preview"],
  availableForSharing: number
): Map<string, MemberPayoutStats> {
  const result = new Map<string, MemberPayoutStats>();

  for (const item of items) {
    if (!item.recipient_user_id) continue;
    const current = result.get(item.recipient_user_id) || {
      hasRules: true,
      payoutAmountProject: 0,
      payoutAmountDisplay: 0,
      payoutCurrencyCode: item.currency_code,
      effectivePercent: 0
    };
    current.payoutAmountProject = round2(current.payoutAmountProject + item.payout_amount);
    current.payoutAmountDisplay = round2(current.payoutAmountDisplay + item.payout_amount_converted);
    current.payoutCurrencyCode = item.currency_code;
    result.set(item.recipient_user_id, current);
  }

  if (availableForSharing > 0) {
    for (const [memberID, stats] of result.entries()) {
      stats.effectivePercent = round2((stats.payoutAmountProject / availableForSharing) * 100);
      result.set(memberID, stats);
    }
    return result;
  }

  const leveled = items
    .filter((item) => !!item.recipient_user_id)
    .map((item, index) => ({ item, index }))
    .sort((left, right) => {
      const leftLevel = left.item.level || 1;
      const rightLevel = right.item.level || 1;
      if (leftLevel === rightLevel) return left.index - right.index;
      return leftLevel - rightLevel;
    });

  let remaining = 100;
  for (let i = 0; i < leveled.length; ) {
    const currentLevel = leveled[i].item.level || 1;
    const levelBase = remaining;
    let levelSpent = 0;

    while (i < leveled.length && (leveled[i].item.level || 1) === currentLevel) {
      const row = leveled[i].item;
      const part = levelBase * (row.percent / 100);
      levelSpent += part;

      const memberID = row.recipient_user_id as string;
      const current = result.get(memberID) || {
        hasRules: true,
        payoutAmountProject: 0,
        payoutAmountDisplay: 0,
        payoutCurrencyCode: row.currency_code,
        effectivePercent: 0
      };
      current.effectivePercent += part;
      result.set(memberID, current);
      i += 1;
    }

    if (levelSpent > remaining) levelSpent = remaining;
    remaining -= levelSpent;
    if (remaining < 0) remaining = 0;
  }

  for (const [memberID, stats] of result.entries()) {
    stats.effectivePercent = round2(stats.effectivePercent);
    stats.payoutAmountProject = round2(stats.payoutAmountProject);
    stats.payoutAmountDisplay = round2(stats.payoutAmountDisplay);
    result.set(memberID, stats);
  }

  return result;
}

export default function DashboardPage() {
  const { id } = useParams<{ id: string }>();
  const confirmAction = useConfirm();
  const { t, locale } = useLocale();
  const { showToast } = useToast();
  const { token, ready } = useAuthGuard();
  const [project, setProject] = useState<Project | null>(null);
  const [dashboard, setDashboard] = useState<DashboardResponse | null>(null);
  const [latestForecast, setLatestForecast] = useState<ForecastRun | null>(null);
  const [members, setMembers] = useState<ProjectMemberWithUser[]>([]);
  const [canManageMembers, setCanManageMembers] = useState(false);
  const [memberLookup, setMemberLookup] = useState("");
  const [memberRole, setMemberRole] = useState<"PM" | "ANALYST" | "TEAM" | "CUSTOMER">("TEAM");
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [activating, setActivating] = useState(false);
  const tr = (ru: string, en: string) => (locale === "en" ? en : ru);

  async function loadData() {
    if (!token) return;
    try {
      const projectResp = await apiRequest<Project>(`/projects/${id}`, { token });
      const days = resolveDashboardDays(projectResp);
      const [dashboardResp, me] = await Promise.all([
        apiRequest<DashboardResponse>(`/projects/${id}/dashboard?days=${days}`, { token }),
        apiRequest<User>("/auth/me", { token })
      ]);
      setProject(projectResp);
      setDashboard(dashboardResp);
      if (dashboardResp.last_forecast_run?.id) {
        try {
          const latest = await apiRequest<ForecastRun>(`/projects/${id}/forecast/${dashboardResp.last_forecast_run.id}`, { token });
          setLatestForecast(latest);
        } catch {
          setLatestForecast(null);
        }
      } else {
        setLatestForecast(null);
      }

      try {
        const membersResp = await apiRequest<ProjectMemberWithUser[]>(`/projects/${id}/members`, { token });
        setMembers(membersResp);
        const myMember = membersResp.find((member) => member.user_id === me.id);
        setCanManageMembers(myMember?.role === "PM");
      } catch {
        setMembers([]);
        setCanManageMembers(false);
      }
    } catch (err) {
      setError((err as Error).message);
    }
  }

  useEffect(() => {
    if (ready) loadData();
  }, [ready, id, token]);

  async function activateProject() {
    if (!token) return;
    const confirmed = await confirmAction({
      title: t("dashboard.activate.title"),
      message: t("dashboard.activate.message"),
      confirmText: t("dashboard.activate.confirm"),
      cancelText: t("projects.form.cancel"),
      tone: "danger"
    });
    if (!confirmed) return;

    setActivating(true);
    setError("");
    try {
      await apiRequest<Project>(`/projects/${id}/activate`, { method: "POST", token });
      await loadData();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setActivating(false);
    }
  }

  async function addMember() {
    const lookup = parseMemberLookup(memberLookup);
    if (!lookup) {
      setError(tr("Укажите корректный email, user ID (UUID) или никнейм.", "Provide a valid email, user ID (UUID), or nickname."));
      return;
    }
    setError("");
    setNotice("");
    try {
      await apiRequest(`/projects/${id}/members`, {
        method: "POST",
        token,
        body: {
          ...lookup,
          role: memberRole
        }
      });
      setMemberLookup("");
      setNotice(tr("Приглашение отправлено. Участник появится после подтверждения.", "Invite sent. Member will appear after confirmation."));
      await loadData();
    } catch (err) {
      const inviteError = normalizeProjectInviteError(err, locale);
      if (inviteError.toastMessage) {
        showToast({ message: inviteError.toastMessage, tone: "warning" });
      }
      setError(inviteError.suppressInline ? "" : inviteError.inlineMessage);
    }
  }

  if (!ready) return null;

  if (!project || !dashboard) {
    return (
      <AppShell title={t("dashboard.page_fallback_title")}>
        <ProjectNav projectId={id} />
        {error ? (
          <p style={{ color: "#bc4749" }}>{error}</p>
        ) : (
          <p className="subtle">{t("dashboard.loading")}</p>
        )}
      </AppShell>
    );
  }

  const PROJECT_STATUS_LABELS: Record<ProjectStatus, string> = {
    preparation: t("status.preparation"),
    active: t("status.active"),
    completed: t("status.completed")
  };
  const revenueData = buildCumulativeIncome(project, dashboard.daily_income_series, latestForecast?.points ?? []);
  const currencyCode = project.currency_code;
  const appLocale = toIntlLocale(locale);
  const axisMoney = new Intl.NumberFormat(appLocale, {
    style: "currency",
    currency: currencyCode,
    maximumFractionDigits: 0
  });
  const chartTopBase = Math.max(0, ...revenueData.map((point) => point.cumulative_amount));
  const chartTop = chartTopBase > 0 ? Math.ceil(chartTopBase * 1.08) : 100;
  const percentFormatter = new Intl.NumberFormat(appLocale, {
    minimumFractionDigits: 0,
    maximumFractionDigits: 2
  });
  const memberPayoutMap = buildMemberPayoutMap(dashboard.payout_preview, dashboard.available_for_sharing);

  let collectedNow = 0;
  for (const point of revenueData) {
    if (point.actual_amount !== null) {
      collectedNow = point.actual_amount;
    }
  }

  const summaryData = [
    { name: t("dashboard.summary.income"), value: dashboard.income_total, color: "#1da971" },
    { name: t("dashboard.summary.expense"), value: dashboard.expenses_total, color: "#d66853" },
    { name: t("dashboard.summary.net_profit"), value: dashboard.net_profit_to_date, color: "#22395e" },
    { name: t("dashboard.summary.available"), value: dashboard.available_for_sharing, color: "#5b7cfa" }
  ];
  const expectedCompletion = completionDate(project, appLocale);
  const canProjectWrite = project.current_user_role === "PM";
  const roleOptions = projectRoleOptions(t);
  const selectedRoleView = projectRoleView(t, memberRole);

  return (
    <AppShell title={project.name} className="dashboard-shell">
      {project.description && <p className="subtle page-intro">{project.description}</p>}
      <ProjectNav projectId={id} />
      {error && <p style={{ color: "#bc4749" }}>{error}</p>}
      {notice && <p style={{ color: "#2a9d8f" }}>{notice}</p>}

      <section className="card dashboard-hero">
        <div className="project-status-row">
          <div>
            <p className="subtle dashboard-hero-label" style={{ margin: 0 }}>{t("dashboard.status.current")}</p>
            <div className="kpi" style={{ fontSize: "1.15rem" }}>{PROJECT_STATUS_LABELS[project.status]}</div>
            <p className="subtle" style={{ margin: "6px 0 0" }}>
              {t("dashboard.status.duration")}: {project.duration_days} {t("projects.days_short")}
            </p>
            <p className="subtle" style={{ margin: "6px 0 0" }}>
              {t("dashboard.status.currency")}: {currencyCode}
            </p>
            {project.secondary_currency_code && (
              <p className="subtle" style={{ margin: "6px 0 0" }}>
                {t("dashboard.status.secondary_currency")}: {project.secondary_currency_code}
              </p>
            )}
            {project.status === "active" && project.activated_at && (
              <p className="subtle" style={{ margin: "6px 0 0" }}>
                {t("dashboard.status.activated")}: {shortDate(project.activated_at)} · {t("dashboard.status.auto_complete")}: {expectedCompletion ?? "—"}
              </p>
            )}
            {project.status === "completed" && (
              <p className="subtle" style={{ margin: "6px 0 0" }}>
                {t("dashboard.status.completed")}: {shortDate(project.updated_at)}
              </p>
            )}
          </div>
          {project.status === "preparation" && (
            <button type="button" onClick={activateProject} disabled={activating || !canProjectWrite}>
              {canProjectWrite ? (activating ? t("dashboard.activate.progress") : t("dashboard.activate.action")) : t("dashboard.activate.denied")}
            </button>
          )}
        </div>
      </section>

      <section className="card dashboard-team-card" style={{ marginBottom: 14 }}>
        <h2 className="title">{t("dashboard.team.title")}</h2>
        {canManageMembers ? (
          <>
            <div className="dashboard-team-toolbar">
              <input
                className="dashboard-team-lookup"
                placeholder={tr("Email, user ID (UUID) или никнейм (@nickname)", "Email, user ID (UUID), or nickname (@nickname)")}
                value={memberLookup}
                onChange={(e) => setMemberLookup(e.target.value)}
              />
              <select
                className="dashboard-team-role-select"
                value={memberRole}
                onChange={(e) => setMemberRole(e.target.value as typeof memberRole)}
              >
                {roleOptions.map((opt) => (
                  <option key={opt.code} value={opt.code}>{opt.title}</option>
                ))}
              </select>
              <button type="button" className="fixed-width-button dashboard-team-add-button" onClick={addMember}>{t("dashboard.team.add")}</button>
              <div className="subtle dashboard-role-helper">
                {selectedRoleView.subtitle}
              </div>
            </div>
            <div className="table-surface">
            <table>
              <thead>
                <tr>
                  <th>{t("dashboard.team.col.name")}</th>
                  <th>{t("dashboard.team.col.email")}</th>
                  <th>{t("dashboard.team.col.role")}</th>
                  <th>{t("dashboard.team.col.percent")}</th>
                  <th>{t("dashboard.team.col.payout")}</th>
                </tr>
              </thead>
              <tbody>
                {members.map((member) => (
                  <tr key={member.id}>
                    {(() => {
                      const payout = memberPayoutMap.get(member.user_id);
                      return (
                        <>
                    <td>{member.full_name}</td>
                    <td>{member.email}</td>
                    <td>
                      {(() => {
                        const view = projectRoleView(t, member.role);
                        return (
                      <div style={{ display: "grid", gap: 2 }}>
                        <strong style={{ fontSize: "1rem", lineHeight: 1.2 }}>
                          {view.title}
                        </strong>
                        <span className="subtle" style={{ fontSize: "0.85rem", lineHeight: 1.2 }}>
                          {view.subtitle}
                        </span>
                      </div>
                        );
                      })()}
                    </td>
                    <td>{payout?.hasRules ? `${percentFormatter.format(payout.effectivePercent)}%` : t("dashboard.team.no_value")}</td>
                    <td>
                      {payout?.hasRules ? (
                        <div className="money-stack">
                          <strong>{money(payout.payoutAmountDisplay, payout.payoutCurrencyCode)}</strong>
                          <span>{money(payout.payoutAmountProject, currencyCode)}</span>
                        </div>
                      ) : t("dashboard.team.no_value")}
                    </td>
                        </>
                      );
                    })()}
                  </tr>
                ))}
              </tbody>
            </table>
            </div>
          </>
        ) : (
          <p className="subtle" style={{ margin: 0 }}>
            {t("dashboard.team.readonly")}
          </p>
        )}
      </section>

      <div className="dashboard-kpi-grid">
        <article className="card dashboard-kpi-card">
          <p className="dashboard-kpi-label">{t("dashboard.kpi.net_profit")}</p>
          <div className="dashboard-kpi-body">
            <div className="kpi">{money(dashboard.net_profit_to_date, currencyCode)}</div>
          </div>
          <p className="dashboard-kpi-meta">{tr("После расходов и текущих поступлений", "After current income and expenses")}</p>
        </article>
        <article className="card dashboard-kpi-card">
          <p className="dashboard-kpi-label">{tr("Фиксированные траты к оплате", "Fixed obligations left")}</p>
          <div className="dashboard-kpi-body">
            <div className="kpi">{money(dashboard.mandatory_remaining, currencyCode)}</div>
          </div>
          <p className="dashboard-kpi-meta">{tr("Сумма обязательных платежей, которую ещё нужно закрыть до расчёта налогов", "Mandatory payments still to be covered before tax calculation")}</p>
        </article>
        <article className="card dashboard-kpi-card">
          <p className="dashboard-kpi-label">{t("dashboard.kpi.expenses")}</p>
          <div className="dashboard-kpi-body">
            <div className="kpi">{money(dashboard.expenses_total, currencyCode)}</div>
          </div>
          <p className="dashboard-kpi-meta">{tr("Все подтверждённые расходы проекта на сегодня", "All confirmed project expenses to date")}</p>
        </article>
        <article className="card dashboard-kpi-card">
          <p className="dashboard-kpi-label">{t("dashboard.kpi.taxes")}</p>
          <div className="dashboard-kpi-body">
            <div className="kpi">{money(dashboard.tax_total, currencyCode)}</div>
          </div>
          <p className="dashboard-kpi-meta">
            {t("dashboard.kpi.total")}: {percentFormatter.format(dashboard.tax_percent_total)}%
          </p>
        </article>
        <article className="card dashboard-kpi-card dashboard-kpi-card-accent">
          <p className="dashboard-kpi-label">{t("dashboard.kpi.available")}</p>
          <div className="dashboard-kpi-body">
            <div className="kpi">{money(dashboard.available_for_sharing, currencyCode)}</div>
          </div>
          <p className="dashboard-kpi-meta">{tr("Остаток после фиксированных трат и налогов", "Remainder after fixed costs and taxes")}</p>
        </article>
      </div>

      <div className="grid two dashboard-charts-grid" style={{ marginTop: 14 }}>
        <section className="card dashboard-chart-card">
          <div className="dashboard-chart-head">
            <h2 className="title">{t("dashboard.chart.cumulative_income")}</h2>
            <div className="dashboard-chart-meta">
              <span className="subtle">
                {t("dashboard.chart.collected")}: <strong style={{ color: "var(--ink)" }}>{money(collectedNow, currencyCode)}</strong>
              </span>
              <span className="subtle dashboard-legend-item">
                <span className="dashboard-legend-dot actual" />
                {tr("Факт", "Actual")}
              </span>
              <span className="subtle dashboard-legend-item">
                <span className="dashboard-legend-dot forecast" />
                {tr("Прогноз", "Forecast")}
              </span>
            </div>
          </div>
          <div className="dashboard-chart-frame">
          <ResponsiveContainer width="100%" height={300}>
            <ComposedChart data={revenueData} margin={{ top: 8, right: 8, left: 2, bottom: 0 }}>
              <defs>
                <linearGradient id="fundingAreaFillActual" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor="#22c55e" stopOpacity={0.42} />
                  <stop offset="100%" stopColor="#22c55e" stopOpacity={0.08} />
                </linearGradient>
                <linearGradient id="fundingAreaFillForecast" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor="#f59e0b" stopOpacity={0.35} />
                  <stop offset="100%" stopColor="#f59e0b" stopOpacity={0.08} />
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 3" vertical={false} />
              <XAxis dataKey="date_iso" tickFormatter={shortDate} interval="preserveStartEnd" minTickGap={22} />
              <YAxis domain={[0, chartTop]} tickFormatter={(value) => axisMoney.format(Number(value))} width={80} />
              <Tooltip
                content={({ active, payload, label }) => {
                  if (!active || !payload?.length) return null;
                  const seen = new Set<string>();
                  const items: Array<{ label: string; value: number }> = [];
                  for (const row of payload) {
                    const key = String(row.dataKey || row.name || "");
                    if (!key || seen.has(key)) continue;
                    seen.add(key);
                    const value = Number(row.value);
                    if (!Number.isFinite(value)) continue;
                    let itemLabel = t("dashboard.chart.tooltip.value");
                    if (key === "actual_amount") itemLabel = tr("Фактический накопленный доход", "Actual cumulative income");
                    if (key === "forecast_amount") itemLabel = tr("Прогноз накопленного дохода", "Forecast cumulative income");
                    items.push({ label: itemLabel, value });
                  }
                  if (!items.length) return null;
                  return (
                    <div
                      style={{
                        background: "var(--surface)",
                        border: "1px solid var(--line)",
                        borderRadius: 10,
                        padding: "10px 12px",
                        boxShadow: "0 8px 22px rgba(20, 33, 61, 0.16)"
                      }}
                    >
                      <div style={{ fontWeight: 700, marginBottom: 6 }}>{t("dashboard.chart.tooltip.date")}: {String(label || "")}</div>
                      {items.map((item) => (
                        <div key={item.label} style={{ display: "flex", justifyContent: "space-between", gap: 10 }}>
                          <span>{item.label}</span>
                          <strong>{money(item.value, currencyCode)}</strong>
                        </div>
                      ))}
                    </div>
                  );
                }}
              />
              <Area
                type="monotone"
                dataKey="actual_amount"
                name={tr("Фактический накопленный доход", "Actual cumulative income")}
                stroke="none"
                fill="url(#fundingAreaFillActual)"
                isAnimationActive={false}
                connectNulls={false}
              />
              <Area
                type="monotone"
                dataKey="forecast_amount"
                name={tr("Прогноз накопленного дохода", "Forecast cumulative income")}
                stroke="none"
                fill="url(#fundingAreaFillForecast)"
                isAnimationActive={false}
                connectNulls={false}
              />
              <Line
                type="monotone"
                dataKey="actual_amount"
                name={tr("Фактический накопленный доход", "Actual cumulative income")}
                stroke="#22c55e"
                strokeWidth={2.8}
                dot={{ r: 3, stroke: "#22c55e", strokeWidth: 1.2, fill: "#a7f3c4" }}
                activeDot={{ r: 5 }}
                connectNulls={false}
                isAnimationActive={false}
              />
              <Line
                type="monotone"
                dataKey="forecast_amount"
                name={tr("Прогноз накопленного дохода", "Forecast cumulative income")}
                stroke="#f59e0b"
                strokeWidth={2.6}
                dot={{ r: 2.6, stroke: "#f59e0b", strokeWidth: 1.1, fill: "#fde68a" }}
                activeDot={{ r: 4.5 }}
                connectNulls={false}
                isAnimationActive={false}
              />
            </ComposedChart>
          </ResponsiveContainer>
          </div>
        </section>

        <section className="card dashboard-chart-card">
          <div className="dashboard-chart-head">
            <h2 className="title">{t("dashboard.chart.summary")}</h2>
            <div className="dashboard-chart-meta">
              <span className="subtle">{tr("Ключевые финансовые показатели проекта", "Key project financial indicators")}</span>
            </div>
          </div>
          <div className="dashboard-chart-frame">
          <ResponsiveContainer width="100%" height={300}>
            <BarChart data={summaryData} layout="vertical" margin={{ top: 8, right: 12, left: 10, bottom: 0 }} barCategoryGap={16}>
              <defs>
                <linearGradient id="dashboardIncomeBar" x1="0" y1="0" x2="1" y2="0">
                  <stop offset="0%" stopColor="#0f8a62" />
                  <stop offset="100%" stopColor="#38c98b" />
                </linearGradient>
                <linearGradient id="dashboardExpenseBar" x1="0" y1="0" x2="1" y2="0">
                  <stop offset="0%" stopColor="#d66853" />
                  <stop offset="100%" stopColor="#ef8f7d" />
                </linearGradient>
                <linearGradient id="dashboardProfitBar" x1="0" y1="0" x2="1" y2="0">
                  <stop offset="0%" stopColor="#183153" />
                  <stop offset="100%" stopColor="#33598d" />
                </linearGradient>
                <linearGradient id="dashboardAvailableBar" x1="0" y1="0" x2="1" y2="0">
                  <stop offset="0%" stopColor="#4c6fff" />
                  <stop offset="100%" stopColor="#7d97ff" />
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 3" horizontal={false} />
              <XAxis
                type="number"
                tickFormatter={(value) => axisMoney.format(Number(value))}
                axisLine={false}
                tickLine={false}
                tick={{ fill: "var(--muted)", fontSize: 12 }}
              />
              <YAxis
                type="category"
                dataKey="name"
                axisLine={false}
                tickLine={false}
                width={118}
                tick={{ fill: "var(--ink)", fontSize: 13, fontWeight: 600 }}
              />
              <Tooltip
                content={({ active, payload, label }) => {
                  if (!active || !payload?.length) return null;
                  const row = payload[0];
                  const value = Number(row.value);
                  if (!Number.isFinite(value)) return null;
                  return (
                    <div
                      style={{
                        background: "color-mix(in srgb, var(--surface) 96%, transparent)",
                        border: "1px solid var(--line)",
                        borderRadius: 12,
                        padding: "10px 12px",
                        boxShadow: "0 12px 28px var(--shadow)",
                        minWidth: 168
                      }}
                    >
                      <div style={{ fontSize: "0.78rem", color: "var(--muted)", marginBottom: 4 }}>
                        {tr("Показатель", "Metric")}
                      </div>
                      <div style={{ fontWeight: 700, marginBottom: 6 }}>{String(label)}</div>
                      <div style={{ display: "flex", justifyContent: "space-between", gap: 12 }}>
                        <span style={{ color: "var(--muted)" }}>{tr("Сумма", "Amount")}</span>
                        <strong>{money(value, currencyCode)}</strong>
                      </div>
                    </div>
                  );
                }}
              />
              <Bar dataKey="value" radius={[0, 12, 12, 0]} barSize={18}>
                {summaryData.map((entry, index) => (
                  <Cell
                    key={entry.name}
                    fill={index === 0 ? "url(#dashboardIncomeBar)" : index === 1 ? "url(#dashboardExpenseBar)" : index === 2 ? "url(#dashboardProfitBar)" : "url(#dashboardAvailableBar)"}
                  />
                ))}
              </Bar>
            </BarChart>
          </ResponsiveContainer>
          </div>
        </section>
      </div>

    </AppShell>
  );
}
