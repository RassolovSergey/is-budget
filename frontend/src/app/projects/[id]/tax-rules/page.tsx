"use client";

import { FormEvent, useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";

import { AppShell } from "../../../../components/AppShell";
import { useConfirm } from "../../../../components/ConfirmProvider";
import { useLocale } from "../../../../components/LocaleProvider";
import { ProjectNav } from "../../../../components/ProjectNav";
import { apiRequest } from "../../../../lib/api";
import { money } from "../../../../lib/format";
import type { PayoutPreview, Project, ProjectTaxRule } from "../../../../lib/types";
import { useAuthGuard } from "../../../../lib/useAuthGuard";

type FormState = {
  id?: string;
  name: string;
  percent: string;
  level: string;
};

const initialForm: FormState = {
  name: "",
  percent: "",
  level: "1"
};

export default function TaxRulesPage() {
  const { id } = useParams<{ id: string }>();
  const confirmAction = useConfirm();
  const { locale } = useLocale();
  const { token, ready } = useAuthGuard();
  const [project, setProject] = useState<Project | null>(null);
  const [rules, setRules] = useState<ProjectTaxRule[]>([]);
  const [preview, setPreview] = useState<PayoutPreview | null>(null);
  const [form, setForm] = useState<FormState>(initialForm);
  const [draggedRuleId, setDraggedRuleId] = useState<string | null>(null);
  const [dragOverLevel, setDragOverLevel] = useState<number | null>(null);
  const [movingRuleId, setMovingRuleId] = useState<string | null>(null);
  const [prependBeforeFirst, setPrependBeforeFirst] = useState(false);
  const [error, setError] = useState("");

  async function load() {
    if (!token) return;
    try {
      const [projectResp, rulesResp, previewResp] = await Promise.all([
        apiRequest<Project>(`/projects/${id}`, { token }),
        apiRequest<ProjectTaxRule[]>(`/projects/${id}/tax-rules`, { token }),
        apiRequest<PayoutPreview>(`/projects/${id}/payout-preview`, { token })
      ]);
      setProject(projectResp);
      setRules(rulesResp);
      setPreview(previewResp);
    } catch (err) {
      setError((err as Error).message);
    }
  }

  useEffect(() => {
    if (ready) load();
  }, [ready, id, token]);

  const tr = (ru: string, en: string) => (locale === "en" ? en : ru);
  const legacyReadOnly = true;

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    if (legacyReadOnly) {
      setError(tr("Этот legacy-раздел доступен только для просмотра. Основные налоги и сборы теперь настраиваются в «Финансовом конвейере».", "This legacy section is read-only. Configure taxes and fees in Finance Pipeline."));
      return;
    }
    setError("");
    const percent = Number(form.percent);
    if (form.percent.trim() === "" || Number.isNaN(percent) || percent < 0 || percent > 100) {
      setError(tr("Процент должен быть числом от 0 до 100.", "Percent must be a number from 0 to 100."));
      return;
    }
    const level = !form.id && prependBeforeFirst ? 1 : Number(form.level);
    if ((!prependBeforeFirst || !!form.id) && (form.level.trim() === "" || !Number.isInteger(level) || level < 1)) {
      setError(tr("Уровень должен быть целым числом от 1.", "Level must be an integer starting from 1."));
      return;
    }
    const levelValidation = !form.id && prependBeforeFirst ? { ok: true, message: "" } : validateLevelInput(level, form.id);
    if (!levelValidation.ok) {
      setError(levelValidation.message);
      return;
    }

    const body = {
      name: form.name,
      percent,
      level,
      prepend_before_first: !form.id && prependBeforeFirst
    };

    try {
      if (form.id) {
        await apiRequest(`/projects/${id}/tax-rules/${form.id}`, { method: "PUT", token, body });
      } else {
        await apiRequest(`/projects/${id}/tax-rules`, { method: "POST", token, body });
      }
      setForm(initialForm);
      setPrependBeforeFirst(false);
      await load();
    } catch (err) {
      setError((err as Error).message);
    }
  }

  function startEdit(rule: ProjectTaxRule) {
    setForm({
      id: rule.id,
      name: rule.name,
      percent: String(rule.percent),
      level: String(rule.level)
    });
    setPrependBeforeFirst(false);
  }

  async function onDelete(ruleId: string) {
    if (legacyReadOnly) {
      setError(tr("Этот legacy-раздел доступен только для просмотра. Основные налоги и сборы теперь настраиваются в «Финансовом конвейере».", "This legacy section is read-only. Configure taxes and fees in Finance Pipeline."));
      return;
    }
    const confirmed = await confirmAction({
      title: tr("Удалить налог/комиссию?", "Delete tax/fee rule?"),
      message: tr("Запись будет удалена без возможности восстановления.", "This record will be permanently deleted."),
      confirmText: tr("Удалить", "Delete"),
      cancelText: tr("Отмена", "Cancel"),
      tone: "danger"
    });
    if (!confirmed) return;

    try {
      await apiRequest(`/projects/${id}/tax-rules/${ruleId}`, { method: "DELETE", token });
      await load();
    } catch (err) {
      setError((err as Error).message);
    }
  }

  function getMaxLevelExcluding(ruleId?: string): number {
    let maxLevel = 0;
    for (const rule of rules) {
      if (ruleId && rule.id === ruleId) continue;
      if (rule.level > maxLevel) maxLevel = rule.level;
    }
    return maxLevel;
  }

  function validateLevelInput(level: number, currentRuleId?: string): { ok: boolean; message: string } {
    const maxLevel = getMaxLevelExcluding(currentRuleId);
    if (level > maxLevel + 1) {
      return {
        ok: false,
        message: tr(
          `Нельзя перескакивать уровни. Сначала создайте уровень ${maxLevel + 1}.`,
          `Cannot skip levels. First create level ${maxLevel + 1}.`
        )
      };
    }
    return { ok: true, message: "" };
  }

  async function moveRuleToLevel(rule: ProjectTaxRule, targetLevel: number) {
    if (!token) return;
    if (legacyReadOnly) {
      setError(tr("Этот legacy-раздел доступен только для просмотра. Основные налоги и сборы теперь настраиваются в «Финансовом конвейере».", "This legacy section is read-only. Configure taxes and fees in Finance Pipeline."));
      return;
    }
    const levelValidation = validateLevelInput(targetLevel, rule.id);
    if (!levelValidation.ok) {
      setError(levelValidation.message);
      return;
    }

    if (rule.level === targetLevel) return;
    setMovingRuleId(rule.id);
    setError("");
    try {
      await apiRequest(`/projects/${id}/tax-rules/${rule.id}`, {
        method: "PUT",
        token,
        body: {
          name: rule.name,
          percent: rule.percent,
          level: targetLevel
        }
      });
      await load();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setMovingRuleId(null);
    }
  }

  if (!ready || !project) return null;

  const currencyCode = project.currency_code;
  const canFinanceWrite = false;
  const levelsMap = new Map<number, ProjectTaxRule[]>();
  for (const rule of rules) {
    if (!levelsMap.has(rule.level)) levelsMap.set(rule.level, []);
    levelsMap.get(rule.level)?.push(rule);
  }
  const levels = Array.from(levelsMap.keys()).sort((a, b) => a - b);
  const maxLevel = levels.length ? levels[levels.length - 1] : 0;
  const levelHint =
    !form.id && prependBeforeFirst
      ? ""
      : Number.isInteger(Number(form.level))
        ? validateLevelInput(Number(form.level), form.id).message
        : "";
  const previewLevelMap = new Map((preview?.tax_levels || []).map((item) => [item.level, item]));

  return (
    <AppShell title={project.name} className="tax-rules-shell">
      <ProjectNav projectId={id} />
      {error && <p style={{ color: "#bc4749" }}>{error}</p>}

      <section className="card legacy-route-note">
        <strong>{tr("Legacy-раздел налогов", "Legacy tax section")}</strong>
        <p>
          {tr(
            "Эта страница оставлена для совместимости со старыми проектами. Новые уровни налогов и сборов теперь настраиваются в «Финансовом конвейере».",
            "This page remains for compatibility with older projects. New tax and fee levels should now be configured in Finance Pipeline."
          )}
        </p>
        <Link href={`/projects/${id}/finance-pipeline`} className="inline-link">
          {tr("Открыть финансовый конвейер", "Open Finance Pipeline")}
        </Link>
      </section>

      <div className="grid two tax-rules-top-grid">
        <section className="card finance-form-card">
          <div className="finance-card-head">
            <div>
              <span className="shell-eyebrow">{tr("Налоги", "Taxes")}</span>
              <h2 className="title">{tr("Налоги и комиссии", "Taxes and Fees")}</h2>
            </div>
            <p className="subtle" style={{ margin: 0 }}>
              {tr(
                "Фиксируйте налоговые удержания и комиссии по уровням. Каждый следующий уровень считается от остатка после предыдущего.",
                "Configure taxes and fee deductions by levels. Each next level is calculated from the remainder after the previous one."
              )}
            </p>
          </div>
          <form className="grid" onSubmit={onSubmit}>
            <label>
              {tr("Название", "Name")}
              <input
                placeholder={tr("Например: Kickstarter, НДФЛ, комиссия банка", "For example: Kickstarter, income tax, bank fee")}
                value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
                disabled={!canFinanceWrite}
                required
              />
            </label>
            <div className="finance-form-inline">
              <label>
                {tr("Процент", "Percent")}
                <input
                  type="number"
                  min="0"
                  max="100"
                  step="0.01"
                  placeholder={tr("Процент (0-100)", "Percent (0-100)")}
                  value={form.percent}
                  onChange={(e) => setForm({ ...form, percent: e.target.value })}
                  disabled={!canFinanceWrite}
                  required
                />
              </label>
              <label>
                {tr("Уровень", "Level")}
                <input
                  type="number"
                  min="1"
                  step="1"
                  placeholder={tr("Уровень (1, 2, 3...)", "Level (1, 2, 3...)")}
                  value={!form.id && prependBeforeFirst ? "1" : form.level}
                  onChange={(e) => setForm({ ...form, level: e.target.value })}
                  disabled={!canFinanceWrite || (!form.id && prependBeforeFirst)}
                  required
                />
              </label>
            </div>
            {!form.id && (
              <label style={{ display: "flex", alignItems: "center", gap: 8 }}>
                <input
                  type="checkbox"
                  checked={prependBeforeFirst}
                  onChange={(e) => setPrependBeforeFirst(e.target.checked)}
                  disabled={!canFinanceWrite}
                />
                <span>{tr("Вставить перед первым уровнем", "Insert before the first level")}</span>
              </label>
            )}
            {!form.id && prependBeforeFirst && (
              <p className="subtle" style={{ margin: 0 }}>
                {tr(
                  "Новый налог станет уровнем 1, а все существующие уровни автоматически сдвинутся на +1.",
                  "The new tax rule becomes level 1 and all existing levels shift by +1 automatically."
                )}
              </p>
            )}
            {levelHint && <p style={{ color: "#bc4749", margin: 0 }}>{levelHint}</p>}
            <p className="subtle" style={{ margin: 0 }}>
              {tr(
                "Налоги одного уровня считаются от одной базы и суммируются. Следующий уровень считается от остатка после предыдущего уровня.",
                "Taxes in the same level are calculated from the same base and summed. The next level is calculated from the remainder after the previous level."
              )}
            </p>
            <div style={{ display: "flex", gap: 8 }}>
              <button disabled={!canFinanceWrite}>{form.id ? tr("Обновить", "Update") : tr("Создать", "Create")}</button>
              {form.id && (
                <button
                  className="secondary"
                  type="button"
                  onClick={() => {
                    setForm(initialForm);
                    setPrependBeforeFirst(false);
                  }}
                  disabled={!canFinanceWrite}
                >
                  {tr("Отмена", "Cancel")}
                </button>
              )}
            </div>
            {!canFinanceWrite && (
              <p className="subtle" style={{ margin: 0 }}>
                {tr(
                  "У вас только просмотр. Изменение налогов доступно ролям PM/ANALYST.",
                  "You have read-only access. Editing taxes is available for PM/ANALYST roles."
                )}
              </p>
            )}
          </form>
        </section>

        <section className="card tax-board-card">
          <h2 className="title">{tr("Список налогов и комиссий", "Tax and Fee Rules")}</h2>
          <p className="subtle" style={{ marginTop: 0 }}>
            {tr(
              "Налоги сгруппированы по уровням. Перетаскивайте карточки между уровнями.",
              "Taxes are grouped by levels. Drag cards between levels."
            )}
          </p>
          {!rules.length && <p className="subtle">{tr("Налоги и комиссии пока не добавлены.", "No tax and fee rules yet.")}</p>}

          {!!rules.length && (
            <div className="tax-levels-stack">
              {levels.map((level) => (
                <section
                  key={level}
                  className={`tax-level-group${dragOverLevel === level ? " drop-target" : ""}`}
                  onDragOver={(e) => {
                    if (!canFinanceWrite) return;
                    e.preventDefault();
                    e.dataTransfer.dropEffect = "move";
                    setDragOverLevel(level);
                  }}
                  onDragLeave={() => {
                    if (dragOverLevel === level) setDragOverLevel(null);
                  }}
                  onDrop={async (e) => {
                    if (!canFinanceWrite) return;
                    e.preventDefault();
                    if (!draggedRuleId) return;
                    const moved = rules.find((r) => r.id === draggedRuleId);
                    setDragOverLevel(null);
                    setDraggedRuleId(null);
                    if (!moved) return;
                    await moveRuleToLevel(moved, level);
                  }}
                >
                  <div className="tax-level-head">
                    <span>{tr("Уровень", "Level")} {level}</span>
                    <span>{((levelsMap.get(level) || []).reduce((sum, rule) => sum + rule.percent, 0)).toFixed(2)}%</span>
                  </div>
                  {previewLevelMap.get(level) && (
                    <div className="tax-level-metrics-grid">
                      <article className="tax-level-metric-card">
                        <span>{tr("База", "Base")}</span>
                        <strong>{money(previewLevelMap.get(level)!.base_amount, currencyCode)}</strong>
                      </article>
                      <article className="tax-level-metric-card">
                        <span>{tr("Налог уровня", "Level tax")}</span>
                        <strong>{money(previewLevelMap.get(level)!.tax_amount, currencyCode)}</strong>
                      </article>
                      <article className="tax-level-metric-card">
                        <span>{tr("Остаток", "Remainder")}</span>
                        <strong>{money(previewLevelMap.get(level)!.remaining_after, currencyCode)}</strong>
                      </article>
                    </div>
                  )}
                  <div className="tax-level-list">
                    {(levelsMap.get(level) || []).map((rule) => (
                      <article
                        key={rule.id}
                        className={`tax-rule-card${draggedRuleId === rule.id ? " dragging" : ""}`}
                        draggable={canFinanceWrite && movingRuleId !== rule.id}
                        onDragStart={(e) => {
                          if (!canFinanceWrite) return;
                          setDraggedRuleId(rule.id);
                          e.dataTransfer.effectAllowed = "move";
                        }}
                        onDragEnd={() => {
                          setDraggedRuleId(null);
                          setDragOverLevel(null);
                        }}
                      >
                        <div className="tax-rule-card-main">
                          <strong>{rule.name}</strong>
                          <span>{rule.percent}%</span>
                        </div>
                        <p className="subtle tax-rule-card-note" style={{ margin: 0 }}>
                          {tr("Назначение платежа", "Payment purpose")}: {rule.name}
                        </p>
                        <div className="tax-rules-actions">
                          <button type="button" className="secondary" onClick={() => startEdit(rule)} disabled={!canFinanceWrite}>{tr("Редактировать", "Edit")}</button>
                          <button type="button" className="warn" onClick={() => onDelete(rule.id)} disabled={!canFinanceWrite}>{tr("Удалить", "Delete")}</button>
                        </div>
                      </article>
                    ))}
                  </div>
                </section>
              ))}

              <section
                className={`tax-level-group tax-level-new${dragOverLevel === maxLevel + 1 ? " drop-target" : ""}`}
                onDragOver={(e) => {
                  if (!canFinanceWrite) return;
                  e.preventDefault();
                  e.dataTransfer.dropEffect = "move";
                  setDragOverLevel(maxLevel + 1);
                }}
                onDragLeave={() => {
                  if (dragOverLevel === maxLevel + 1) setDragOverLevel(null);
                }}
                onDrop={async (e) => {
                  if (!canFinanceWrite) return;
                  e.preventDefault();
                  if (!draggedRuleId) return;
                  const moved = rules.find((r) => r.id === draggedRuleId);
                  setDragOverLevel(null);
                  setDraggedRuleId(null);
                  if (!moved) return;
                  await moveRuleToLevel(moved, maxLevel + 1);
                }}
              >
                <div className="tax-level-head">{tr("Новый уровень", "New level")} {maxLevel + 1}</div>
                <p className="subtle" style={{ margin: 0 }}>{tr("Перетащите сюда налог, чтобы создать следующий уровень.", "Drag a tax here to create the next level.")}</p>
              </section>
            </div>
          )}
        </section>
      </div>

      {preview && (
        <section className="card" style={{ marginTop: 14 }}>
          <h2 className="title">{tr("Влияние на прибыль", "Impact on Profit")}</h2>
          <div className="grid three tax-impact-grid">
            <article className="tax-impact-item tax-impact-item-accent">
              <p className="subtle">{tr("Сумма после фиксированных трат", "Amount after fixed costs")}</p>
              <strong>{money(preview.summary.available_before_tax, currencyCode)}</strong>
              <span>{tr("Считается до налогов и комиссий", "Calculated before taxes and fees")}</span>
            </article>
            <article className="tax-impact-item">
              <p className="subtle">{tr("Налоги и комиссии", "Taxes and fees")} ({preview.summary.tax_percent_total}%)</p>
              <strong>{money(preview.summary.tax_total, currencyCode)}</strong>
              <span>{tr("Сумма всех удержаний по правилам", "All deductions by tax rules")}</span>
            </article>
            <article className="tax-impact-item tax-impact-item-success">
              <p className="subtle">{tr("Доступно к распределению", "Available for sharing")}</p>
              <strong>{money(preview.summary.available_for_sharing, currencyCode)}</strong>
              <span>{tr("Эта сумма пойдёт в правила распределения прибыли", "This amount moves into profit-sharing rules")}</span>
            </article>
          </div>
        </section>
      )}
    </AppShell>
  );
}
