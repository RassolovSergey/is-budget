"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { Cell, Pie, PieChart, ResponsiveContainer, Tooltip } from "recharts";

import { AppShell } from "../../../../components/AppShell";
import { useLocale } from "../../../../components/LocaleProvider";
import { ProjectNav } from "../../../../components/ProjectNav";
import { apiRequest } from "../../../../lib/api";
import { displayHumanName } from "../../../../lib/displayName";
import { forecastScenarioViews } from "../../../../lib/forecastScenarios";
import { money } from "../../../../lib/format";
import { projectRoleView } from "../../../../lib/projectRoles";
import type { MyIncomeResponse, Project, TeamIncomeAmount, TeamIncomeForecastResponse } from "../../../../lib/types";
import { useAuthGuard } from "../../../../lib/useAuthGuard";

export default function MyIncomePage() {
  const { id } = useParams<{ id: string }>();
  const { locale, t } = useLocale();
  const { token, ready } = useAuthGuard();
  const [project, setProject] = useState<Project | null>(null);
  const [income, setIncome] = useState<MyIncomeResponse | null>(null);
  const [teamForecast, setTeamForecast] = useState<TeamIncomeForecastResponse | null>(null);
  const [error, setError] = useState("");
  const [activeSlice, setActiveSlice] = useState(0);

  async function load() {
    if (!token) return;
    try {
      const [projectResp, myIncomeResp] = await Promise.all([
        apiRequest<Project>(`/projects/${id}`, { token }),
        apiRequest<MyIncomeResponse>(`/projects/${id}/my-income`, { token })
      ]);
      setProject(projectResp);
      setIncome(myIncomeResp);
      if (projectResp.current_user_role === "PM") {
        const teamForecastResp = await apiRequest<TeamIncomeForecastResponse>(`/projects/${id}/team-income-forecast`, { token });
        setTeamForecast(teamForecastResp);
      } else {
        setTeamForecast(null);
      }
    } catch (err) {
      setError((err as Error).message);
    }
  }

  useEffect(() => {
    if (ready) {
      void load();
    }
  }, [ready, id, token]);

  const tr = (ru: string, en: string) => (locale === "en" ? en : ru);
  const numberLocale = locale === "en" ? "en-US" : "ru-RU";
  const rateFormatter = new Intl.NumberFormat(numberLocale, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2
  });
  const rateLabel = (value: number) => rateFormatter.format(Number.isFinite(value) ? value : 0);
  const isAdministrator = project?.current_user_role === "PM";
  const displayResponsibility = (value: string) => {
    const normalized = value.trim().toLowerCase();
    if (!normalized) return "-";
    if (normalized === "administrator" || normalized === "administrator residual") {
      return tr("Администратор", "Administrator");
    }
    return value;
  };
  const sparklinePoints = useMemo(() => {
    const rate = Math.max(income?.total_conversion_rate || 1, 0.01);
    const delta = rate - 1;
    return [0, 0.18, 0.34, 0.5, 0.7, 0.86, 1].map((step) => Number((1 + delta * step).toFixed(4)));
  }, [income?.total_conversion_rate]);
  const sparklinePath = useMemo(() => {
    if (!sparklinePoints.length) return "";
    const max = Math.max(...sparklinePoints);
    const min = Math.min(...sparklinePoints);
    const span = Math.max(max - min, 0.0001);
    return sparklinePoints
      .map((point, index) => {
        const x = (index / Math.max(sparklinePoints.length - 1, 1)) * 100;
        const y = 100 - ((point - min) / span) * 100;
        return `${index === 0 ? "M" : "L"} ${x} ${y}`;
      })
      .join(" ");
  }, [sparklinePoints]);
  const pieData = useMemo(() => {
    if (!income) {
      return [];
    }
    const rows = [
      { name: tr("Расходы", "Expenses"), value: Math.max(income.expenses_total, 0), color: "#d66853" },
      { name: tr("Налоги и комиссии", "Taxes and fees"), value: Math.max(income.tax_total, 0), color: "#8b5cf6" },
      { name: tr("Выплаты команде", "Team payouts"), value: Math.max(income.distributed_to_others, 0), color: "#0f8a62" },
      { name: tr("Доход администратора", "Administrator income"), value: Math.max(income.total_project_amount, 0), color: "#183153" }
    ];
    return rows.filter((row) => row.value > 0);
  }, [income, locale]);
  const forecastProjection = income?.forecast_projection || null;
  const forecastScenarios = forecastScenarioViews(locale);
  const renderAmountCell = (amount?: TeamIncomeAmount | null) => {
    if (!amount) {
      return <span className="subtle">{tr("Нет прогноза", "No forecast")}</span>;
    }
    return (
      <div className="money-stack">
        <strong>{money(amount.project_amount, amount.project_currency_code)}</strong>
        <span>{money(amount.user_amount, amount.user_currency_code)}</span>
      </div>
    );
  };

  if (!ready || !project || !income) return null;

  return (
    <AppShell title={project.name} className="my-income-shell">
      <ProjectNav projectId={id} />
      {error && <p style={{ color: "var(--danger)" }}>{error}</p>}

      {isAdministrator && (
        <section className="card my-income-guidance-card">
          <div>
            <span className="shell-eyebrow">{tr("Роль страницы", "Page role")}</span>
            <h2 className="title">{tr("Здесь вы проверяете итоговые суммы", "This page is for reviewing final amounts")}</h2>
            <p className="subtle">
              {tr(
                "Настройка денежного потока и сохранение снимков находятся в финансовом конвейере. Здесь удобнее сверять личный итог, структуру суммы и прогноз выплат команде.",
                "Money-flow setup and snapshots live in Finance Pipeline. This page is best for reviewing your final amount, the amount structure, and team payout forecasts."
              )}
            </p>
          </div>
          <div className="my-income-guidance-actions">
            <Link href={`/projects/${id}/finance-pipeline`} className="button-like">
              {tr("Открыть финансовый конвейер", "Open Finance Pipeline")}
            </Link>
            <Link href={`/projects/${id}/campaign-lifecycle`} className="button-like secondary">
              {tr("Проверить импорт и жизненный цикл", "Review import and lifecycle")}
            </Link>
          </div>
        </section>
      )}

      {!isAdministrator && (
        <section className="card my-income-panel">
          <div className="my-income-hero">
            <div className="my-income-hero-main">
              <p className="shell-eyebrow">{tr("Личный доход", "Personal income")}</p>
              <h2>{money(income.total_user_amount, income.user_currency_code)}</h2>
              <p className="subtle" style={{ margin: 0 }}>
                {tr("Эквивалент в валюте проекта", "Equivalent in project currency")}:{" "}
                <strong style={{ color: "var(--ink)" }}>{money(income.total_project_amount, income.project_currency_code)}</strong>
              </p>
            </div>
            <div className="my-income-hero-side">
              <article className="my-income-stat-card my-income-stat-card-rate">
                <span>{tr("Курс конвертации", "Conversion rate")}</span>
                <strong>{rateLabel(income.total_conversion_rate || 1)}</strong>
                <div className="conversion-sparkline">
                  <svg viewBox="0 0 100 36" preserveAspectRatio="none" aria-hidden="true">
                    <path d={sparklinePath} />
                  </svg>
                  <p>{`1 ${income.project_currency_code} = ${rateLabel(income.total_conversion_rate || 1)} ${income.user_currency_code}`}</p>
                </div>
                <div className="conversion-trend-chip">
                  {income.total_conversion_rate >= 1 ? tr("Курс выше базовой единицы", "Rate above base unit") : tr("Курс ниже базовой единицы", "Rate below base unit")}
                </div>
              </article>
              <article className="my-income-stat-card">
                <span>{tr("Строк дохода", "Income lines")}</span>
                <strong>{income.items.length}</strong>
              </article>
            </div>
          </div>

          <div className="table-surface">
            <table>
              <thead>
                <tr>
                  <th>{tr("Получатель", "Recipient")}</th>
                  <th>{tr("Ответственность", "Responsibility")}</th>
                  <th>{tr("Уровень", "Level")}</th>
                  <th>{tr("Процент", "Percent")}</th>
                  <th>{tr("Основная сумма", "Main amount")}</th>
                  <th>{tr("В валюте проекта", "In project currency")}</th>
                </tr>
              </thead>
              <tbody>
                {income.items.map((item) => (
                  <tr key={item.rule_id}>
                    <td>{item.recipient_name}</td>
                    <td>{displayResponsibility(item.responsibility)}</td>
                    <td>{item.level}</td>
                    <td>{item.percent}%</td>
                    <td>
                      <div className="money-stack">
                        <strong>{money(item.user_amount, item.user_currency_code)}</strong>
                        <span>{tr("Профильная валюта", "Profile currency")}</span>
                      </div>
                    </td>
                    <td>{money(item.project_amount, item.project_currency_code)}</td>
                  </tr>
                ))}
                {!income.items.length && (
                  <tr>
                    <td colSpan={6} className="subtle">
                      {tr(
                        "Для вашего аккаунта пока нет рассчитанных выплат в текущей конфигурации проекта.",
                        "There are no calculated payouts for your account in the current project setup yet."
                      )}
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </section>
      )}

      {isAdministrator && (
        <section className="card my-income-panel">
          <div className="my-income-admin-hero">
            <div className="my-income-admin-main">
              <p className="shell-eyebrow">{tr("Финансовая сводка администратора", "Administrator income summary")}</p>
              <h2>{money(income.total_user_amount, income.user_currency_code)}</h2>
              <p className="subtle" style={{ margin: 0 }}>
                {tr("Ваш итог в валюте проекта", "Your total in project currency")}:{" "}
                <strong style={{ color: "var(--ink)" }}>{money(income.total_project_amount, income.project_currency_code)}</strong>
              </p>
              <p className="subtle" style={{ margin: "6px 0 0" }}>
                {tr("После расходов, налогов и выплат участникам проект оставляет администратору этот остаток.", "After expenses, taxes, and team payouts, the administrator receives this remainder.")}
              </p>
            </div>
            <div className="my-income-admin-side">
              <article className="my-income-stat-card my-income-stat-card-rate">
                <span>{tr("Курс конвертации", "Conversion rate")}</span>
                <strong>{rateLabel(income.total_conversion_rate || 1)}</strong>
                <div className="conversion-sparkline">
                  <svg viewBox="0 0 100 36" preserveAspectRatio="none" aria-hidden="true">
                    <path d={sparklinePath} />
                  </svg>
                  <p>{`1 ${income.project_currency_code} = ${rateLabel(income.total_conversion_rate || 1)} ${income.user_currency_code}`}</p>
                </div>
                <div className="conversion-trend-chip">
                  {income.total_conversion_rate >= 1 ? tr("Курс выше базовой единицы", "Rate above base unit") : tr("Курс ниже базовой единицы", "Rate below base unit")}
                </div>
              </article>
              <article className="my-income-stat-card">
                <span>{tr("Выплачено команде", "Paid to team")}</span>
                <strong>{money(income.distributed_to_others, income.project_currency_code)}</strong>
              </article>
            </div>
          </div>

          <div className="my-income-admin-grid">
            <article className="card my-income-breakdown-card">
              <h3>{tr("Ключевые суммы проекта", "Key project amounts")}</h3>
              <div className="my-income-breakdown-list">
                <div className="my-income-breakdown-row">
                  <span>{tr("Доход проекта", "Project income")}</span>
                  <strong>{money(income.income_total, income.project_currency_code)}</strong>
                </div>
                <div className="my-income-breakdown-row">
                  <span>{tr("Расходы проекта", "Project expenses")}</span>
                  <strong>{money(income.expenses_total, income.project_currency_code)}</strong>
                </div>
                <div className="my-income-breakdown-row">
                  <span>{tr("Налоги и комиссии", "Taxes and fees")}</span>
                  <strong>{money(income.tax_total, income.project_currency_code)}</strong>
                </div>
                <div className="my-income-breakdown-row">
                  <span>{tr("Доступно к распределению", "Available for sharing")}</span>
                  <strong>{money(income.available_for_sharing, income.project_currency_code)}</strong>
                </div>
                <div className="my-income-breakdown-row">
                  <span>{tr("Распределено по конвейеру", "Distributed by pipeline")}</span>
                  <strong>{money(income.distributed_amount, income.project_currency_code)}</strong>
                </div>
                <div className="my-income-breakdown-row highlight">
                  <span>{tr("Доход администратора", "Administrator income")}</span>
                  <strong>{money(income.total_project_amount, income.project_currency_code)}</strong>
                </div>
              </div>
            </article>

            <article className="card my-income-chart-card">
              <div className="my-income-chart-head">
                <h3>{tr("Структура итоговой суммы", "How the final amount is formed")}</h3>
                <p className="subtle" style={{ margin: 0 }}>
                  {tr("На что был разделён общий доход проекта.", "How the total project income was split.")}
                </p>
              </div>
              {!pieData.length && <p className="subtle">{tr("Недостаточно данных для диаграммы.", "Not enough data for chart.")}</p>}
              {!!pieData.length && (
                <ResponsiveContainer width="100%" height={320}>
                  <PieChart>
                    <Pie
                      data={pieData}
                      dataKey="value"
                      nameKey="name"
                      cx="50%"
                      cy="50%"
                      innerRadius={78}
                      outerRadius={112}
                      activeIndex={activeSlice}
                      paddingAngle={4}
                      onMouseEnter={(_, index) => setActiveSlice(index)}
                    >
                      {pieData.map((entry) => (
                        <Cell key={entry.name} fill={entry.color} />
                      ))}
                    </Pie>
                    <Tooltip
                      formatter={(value: number) => money(Number(value), income.project_currency_code)}
                      contentStyle={{ borderRadius: 14, borderColor: "var(--line)", boxShadow: "0 12px 28px var(--shadow)" }}
                    />
                  </PieChart>
                </ResponsiveContainer>
              )}
              <div className="my-income-chart-legend">
                {pieData.map((entry) => (
                  <div key={entry.name} className="my-income-chart-legend-row">
                    <span className="my-income-chart-dot" style={{ background: entry.color }} />
                    <span>{entry.name}</span>
                    <strong>{money(entry.value, income.project_currency_code)}</strong>
                  </div>
                ))}
              </div>
            </article>
          </div>
        </section>
      )}

      <section className="card my-income-panel">
        <div className="my-income-forecast-block">
          <div className="my-income-forecast-head">
            <p className="shell-eyebrow">{tr("Мой доход по прогнозу", "My forecast income")}</p>
            {forecastProjection ? (
              <>
                <h2>{money(forecastProjection.yhat.user_amount, income.user_currency_code)}</h2>
                <p className="subtle" style={{ margin: 0 }}>
                  {tr(
                    "Оценка по последнему прогнозу и текущей конфигурации проекта.",
                    "Estimate based on the latest forecast and current project setup."
                  )}
                </p>
                <p className="subtle" style={{ margin: "4px 0 0" }}>
                  {tr("Последний запуск", "Latest run")}: {new Date(forecastProjection.created_at).toLocaleDateString(numberLocale)}
                </p>
              </>
            ) : (
              <>
                <h2>{tr("Прогноз пока не запущен", "No forecast yet")}</h2>
                <p className="subtle" style={{ margin: 0 }}>
                  {tr(
                    "Запустите прогноз, чтобы увидеть ожидаемый доход.",
                    "Run a forecast to see the expected income."
                  )}
                </p>
              </>
            )}
          </div>

          {forecastProjection ? (
            <div className="my-income-forecast-range">
              {forecastScenarios.map((scenario) => {
                const value =
                  scenario.key === "poor"
                    ? forecastProjection.p10
                    : scenario.key === "average"
                      ? forecastProjection.yhat
                      : forecastProjection.p90;
                return (
                <article key={scenario.label} className="my-income-stat-card">
                  <span>{scenario.label}</span>
                  <strong>{money(value.user_amount, income.user_currency_code)}</strong>
                  <p className="subtle" style={{ margin: 0 }}>
                    {tr("В валюте проекта", "In project currency")}: {money(value.project_amount, income.project_currency_code)}
                  </p>
                </article>
                );
              })}
            </div>
          ) : (
            <div className="ledger-empty-state">
              <strong>{tr("Нет данных прогноза", "No forecast data")}</strong>
              <span>{tr("После запуска прогноза здесь появятся оценки личного дохода по сценариям.", "After running a forecast, your personal income scenarios will appear here.")}</span>
            </div>
          )}
        </div>
      </section>

      {isAdministrator && teamForecast && (
        <section className="card my-income-panel">
          <div className="my-income-forecast-block">
            <div className="my-income-forecast-head">
              <p className="shell-eyebrow">{tr("Прогноз выплат команде", "Team payout forecast")}</p>
              <h2>{tr("Сценарии выплат по всем участникам", "Member payout scenarios")}</h2>
              <p className="subtle" style={{ margin: 0 }}>
                {tr(
                  "Сравните текущие суммы и прогнозные выплаты участников в трёх сценариях.",
                  "Compare current amounts and projected member payouts across three scenarios."
                )}
              </p>
              <p className="subtle" style={{ margin: "4px 0 0" }}>
                {tr("Настройка денежного потока и сохранение снимков теперь находятся в ", "Money-flow setup and snapshots now live in ")}
                <Link href={`/projects/${id}/finance-pipeline`}>{tr("финансовом конвейере", "Finance Pipeline")}</Link>
                .
              </p>
              <p className="subtle" style={{ margin: "4px 0 0" }}>
                {tr(
                  "Эта таблица нужна для сверки ожидаемых выплат по участникам, а не для настройки самих правил движения денег.",
                  "Use this table to review expected member payouts, not to configure the money-flow rules themselves."
                )}
              </p>
              <p className="subtle" style={{ margin: "4px 0 0" }}>
                {teamForecast.forecast_available && teamForecast.latest_forecast_created_at
                  ? tr(
                      `Последний запуск прогноза: ${new Date(teamForecast.latest_forecast_created_at).toLocaleDateString(numberLocale)}.`,
                      `Latest forecast run: ${new Date(teamForecast.latest_forecast_created_at).toLocaleDateString(numberLocale)}.`
                    )
                  : tr(
                      "Текущие суммы показаны уже сейчас, но сценарии появятся после запуска прогноза.",
                      "Current earned amounts are available now, but scenario estimates appear only after a forecast run."
                    )}
              </p>
            </div>

            <div className="table-surface">
              <table>
                <thead>
                  <tr>
                    <th>{tr("Участник", "Member")}</th>
                    <th>{tr("Роль", "Role")}</th>
                    <th>{tr("Текущий доход", "Current earned")}</th>
                    {forecastScenarios.map((scenario) => (
                      <th key={scenario.key}>{scenario.label}</th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {teamForecast.members.map((member) => {
                    const roleView = projectRoleView(t, member.role);
                    return (
                      <tr key={member.user_id}>
                        <td>{displayHumanName(member, tr("Без имени", "Unnamed member"))}</td>
                        <td>{roleView.title || member.role}</td>
                        <td>{renderAmountCell(member.current_earned)}</td>
                        {forecastScenarios.map((scenario) => {
                          const amount =
                            scenario.key === "poor"
                              ? member.poor_scenario
                              : scenario.key === "average"
                                ? member.average_scenario
                                : member.good_scenario;
                          return <td key={scenario.key}>{renderAmountCell(amount)}</td>;
                        })}
                      </tr>
                    );
                  })}
                  {!teamForecast.members.length && (
                    <tr>
                      <td colSpan={6} className="subtle">
                        {tr("В проекте пока нет участников для расчёта.", "There are no project members to forecast yet.")}
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>
        </section>
      )}
    </AppShell>
  );
}
