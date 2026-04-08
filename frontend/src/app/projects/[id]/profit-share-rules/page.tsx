"use client";

import { FormEvent, useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";

import { AppShell } from "../../../../components/AppShell";
import { useConfirm } from "../../../../components/ConfirmProvider";
import { useLocale } from "../../../../components/LocaleProvider";
import { ProjectNav } from "../../../../components/ProjectNav";
import { ApiError, apiDownload, apiRequest } from "../../../../lib/api";
import { money } from "../../../../lib/format";
import type { Locale } from "../../../../lib/i18n";
import { parseMemberLookup } from "../../../../lib/memberLookup";
import { projectRoleOptions, projectRoleView } from "../../../../lib/projectRoles";
import type { CalculatePayoutsResponse, PayoutPreview, ProfitShareRule, Project, ProjectMemberWithUser, User } from "../../../../lib/types";
import { useAuthGuard } from "../../../../lib/useAuthGuard";

type FormState = {
  id?: string;
  recipient_user_id: string;
  responsibility: string;
  level: string;
  percent: string;
};

const initialForm: FormState = {
  recipient_user_id: "",
  responsibility: "",
  level: "1",
  percent: ""
};

const MAX_PROFIT_LEVEL = 20;

function levelLabel(level: number, locale: Locale): string {
  return locale === "en" ? `Level ${level}` : `Уровень ${level}`;
}

function displayMemberName(member: ProjectMemberWithUser): string {
  const firstLast = `${member.first_name || ""} ${member.last_name || ""}`.trim();
  if (firstLast) return firstLast;
  if (member.full_name?.trim()) return member.full_name.trim();
  if (member.nickname?.trim()) return member.nickname.trim();
  if (member.email?.trim()) return member.email.trim();
  return "Unknown member";
}

function normalizeProfitShareRuleError(err: unknown, locale: Locale): string {
  const tr = (ru: string, en: string) => (locale === "en" ? en : ru);
  const message = err instanceof Error ? err.message : "";

  if (err instanceof ApiError) {
    if (err.status === 400 && err.code === "invalid_request" && err.message === "recipient_user_id is required") {
      return tr(
        "Выберите участника проекта для правила распределения.",
        "Select a project member for this payout rule."
      );
    }

    if (err.status === 400 && err.code === "invalid_request" && err.message === "recipient_user_id must belong to a project member") {
      return tr(
        "Выберите корректного участника проекта для привязки правила.",
        "Select a valid project member to bind this rule."
      );
    }

    if (err.status === 400 && err.code === "validation_error" && err.details?.recipient_user_id) {
      return tr(
        "Некорректный идентификатор участника. Выберите получателя из списка команды.",
        "Invalid member identifier. Select the recipient from the project team list."
      );
    }

    if (err.status === 400) {
      return tr(
        "Не удалось сохранить правило. Проверьте поля формы и повторите.",
        "Failed to save the rule. Check the form fields and try again."
      );
    }
  }

  return message || tr("Не удалось сохранить правило распределения прибыли.", "Failed to save the profit-sharing rule.")
}

function compressRuleLevels(rules: ProfitShareRule[]): {
  normalized: ProfitShareRule[];
  repairs: Array<{ rule: ProfitShareRule; level: number }>;
} {
  if (!rules.length) return { normalized: [], repairs: [] };

  const uniqueLevels = Array.from(new Set(rules.map((rule) => rule.level))).sort((a, b) => a - b);
  const levelMap = new Map<number, number>();
  uniqueLevels.forEach((oldLevel, index) => levelMap.set(oldLevel, index + 1));

  const repairs: Array<{ rule: ProfitShareRule; level: number }> = [];
  const normalized = rules.map((rule) => {
    const nextLevel = levelMap.get(rule.level) || 1;
    if (nextLevel !== rule.level) {
      repairs.push({ rule, level: nextLevel });
    }
    return {
      ...rule,
      level: nextLevel,
      applies_after_mandatory: nextLevel > 1
    };
  });

  normalized.sort((a, b) => {
    if (a.level === b.level) {
      return new Date(a.created_at).getTime() - new Date(b.created_at).getTime();
    }
    return a.level - b.level;
  });
  return { normalized, repairs };
}

export default function ProfitShareRulesPage() {
  const { id } = useParams<{ id: string }>();
  const confirmAction = useConfirm();
  const { t, locale } = useLocale();
  const { token, ready } = useAuthGuard();
  const [project, setProject] = useState<Project | null>(null);
  const [rules, setRules] = useState<ProfitShareRule[]>([]);
  const [members, setMembers] = useState<ProjectMemberWithUser[]>([]);
  const [preview, setPreview] = useState<PayoutPreview | null>(null);
  const [form, setForm] = useState<FormState>(initialForm);
  const [canManageMembers, setCanManageMembers] = useState(false);
  const [memberLookup, setMemberLookup] = useState("");
  const [memberRole, setMemberRole] = useState<"PM" | "ANALYST" | "TEAM" | "CUSTOMER">("TEAM");
  const [draggedRuleId, setDraggedRuleId] = useState<string | null>(null);
  const [dragOverLevel, setDragOverLevel] = useState<number | null>(null);
  const [movingRuleId, setMovingRuleId] = useState<string | null>(null);
  const [prependBeforeFirst, setPrependBeforeFirst] = useState(false);
  const [isSavingSnapshot, setIsSavingSnapshot] = useState(false);
  const [isRepairingLevels, setIsRepairingLevels] = useState(false);
  const [notice, setNotice] = useState("");
  const [error, setError] = useState("");
  const tr = (ru: string, en: string) => (locale === "en" ? en : ru);
  const legacyReadOnly = true;
  const displayResponsibility = (value: string) => {
    const normalized = value.trim().toLowerCase();
    if (!normalized) return "-";
    if (normalized === "administrator" || normalized === "administrator residual") {
      return tr("Администратор", "Administrator");
    }
    return value;
  };

  async function load() {
    if (!token) return;
    try {
      const [projectResp, rulesResp, previewResp, membersResp, meResp] = await Promise.all([
        apiRequest<Project>(`/projects/${id}`, { token }),
        apiRequest<ProfitShareRule[]>(`/projects/${id}/profit-share-rules`, { token }),
        apiRequest<PayoutPreview>(`/projects/${id}/payout-preview`, { token }),
        apiRequest<ProjectMemberWithUser[]>(`/projects/${id}/members`, { token }),
        apiRequest<User>("/auth/me", { token })
      ]);
      setProject(projectResp);
      const compressed = compressRuleLevels(rulesResp);
      setRules(compressed.normalized);
      setPreview(previewResp);
      setMembers(membersResp);
      const selfMember = membersResp.find((member) => member.user_id === meResp.id);
      const canManage = !legacyReadOnly && selfMember?.role === "PM";
      setCanManageMembers(canManage);

      if (canManage && compressed.repairs.length > 0 && !isRepairingLevels) {
        setIsRepairingLevels(true);
        try {
          for (const repair of compressed.repairs) {
            await apiRequest(`/projects/${id}/profit-share-rules/${repair.rule.id}`, {
              method: "PUT",
              token,
              body: {
                recipient_user_id: repair.rule.recipient_user_id || undefined,
                responsibility: repair.rule.responsibility,
                percent: repair.rule.percent,
                level: repair.level,
                applies_after_mandatory: repair.level > 1
              }
            });
          }
          const repairedRulesResp = await apiRequest<ProfitShareRule[]>(`/projects/${id}/profit-share-rules`, { token });
          const repaired = compressRuleLevels(repairedRulesResp);
          setRules(repaired.normalized);
          setNotice(tr("Уровни правил автоматически приведены к непрерывному порядку.", "Rule levels were automatically normalized to a continuous order."));
        } catch (repairErr) {
          setError((repairErr as Error).message);
        } finally {
          setIsRepairingLevels(false);
        }
      }
    } catch (err) {
      setError((err as Error).message);
    }
  }

  useEffect(() => {
    if (ready) load();
  }, [ready, id, token]);

  function getLevelSums(excludeRuleId?: string): Map<number, number> {
    const sums = new Map<number, number>();
    for (const rule of rules) {
      if (excludeRuleId && rule.id === excludeRuleId) continue;
      sums.set(rule.level, (sums.get(rule.level) || 0) + rule.percent);
    }
    return sums;
  }

  function getMaxLevelExcluding(ruleId?: string): number {
    let maxLevel = 0;
    for (const rule of rules) {
      if (ruleId && rule.id === ruleId) continue;
      if (rule.level > maxLevel) maxLevel = rule.level;
    }
    return maxLevel;
  }

  function findBlockingPreviousLevel(targetLevel: number, excludeRuleId?: string): number | null {
    const sums = getLevelSums(excludeRuleId);
    for (let level = 1; level < targetLevel; level += 1) {
      if ((sums.get(level) || 0) >= 99.999999) {
        return level;
      }
    }
    return null;
  }

  function validateLevelPercentBudget(level: number, percent: number, currentRuleId?: string): { ok: boolean; message: string } {
    const sums = getLevelSums(currentRuleId);
    const currentSum = sums.get(level) || 0;
    if (currentSum + percent > 100.000001) {
      return {
        ok: false,
        message: tr(`Сумма процентов на уровне ${level} не может превышать 100%.`, `The total percent at level ${level} cannot exceed 100%.`)
      };
    }
    return { ok: true, message: "" };
  }

  function validateLevelInput(level: number, currentRuleId?: string): { ok: boolean; message: string } {
    if (level > MAX_PROFIT_LEVEL) {
      return {
        ok: false,
        message: tr(`Максимальное количество уровней: ${MAX_PROFIT_LEVEL}.`, `Maximum number of levels: ${MAX_PROFIT_LEVEL}.`)
      };
    }
    const maxLevel = getMaxLevelExcluding(currentRuleId);
    if (level > maxLevel + 1) {
      return {
        ok: false,
        message: tr(
          `Нельзя перескакивать уровни. Сначала создайте уровень ${maxLevel + 1}.`,
          `Cannot skip levels. Create level ${maxLevel + 1} first.`
        )
      };
    }
    const blockedBy = findBlockingPreviousLevel(level, currentRuleId);
    if (blockedBy !== null) {
      return {
        ok: false,
        message: tr(
          `Невозможно создать ${level}-й уровень: уровень ${blockedBy} уже распределяет 100%.`,
          `Cannot create level ${level}: level ${blockedBy} already allocates 100%.`
        )
      };
    }
    return { ok: true, message: "" };
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    if (legacyReadOnly) {
      setError(tr("Этот legacy-раздел доступен только для просмотра. Основное распределение денег теперь настраивается через «Финансовый конвейер».", "This legacy section is read-only. Configure the main money distribution in Finance Pipeline."));
      return;
    }
    setError("");
    const percent = Number(form.percent);
    if (form.percent.trim() === "" || Number.isNaN(percent) || percent < 0 || percent > 100) {
      setError(tr("Процент должен быть числом от 0 до 100.", "Percent must be a number from 0 to 100."));
      return;
    }
    if (!form.recipient_user_id) {
      setError(tr("Выберите участника проекта для этого правила.", "Select a project member for this rule."));
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
    const budgetValidation = !form.id && prependBeforeFirst ? { ok: true, message: "" } : validateLevelPercentBudget(level, percent, form.id);
    if (!budgetValidation.ok) {
      setError(budgetValidation.message);
      return;
    }

    const body = {
      recipient_user_id: form.recipient_user_id || undefined,
      responsibility: form.responsibility,
      percent,
      level,
      applies_after_mandatory: level > 1,
      prepend_before_first: !form.id && prependBeforeFirst
    };
    try {
      if (form.id) {
        await apiRequest(`/projects/${id}/profit-share-rules/${form.id}`, { method: "PUT", token, body });
      } else {
        await apiRequest(`/projects/${id}/profit-share-rules`, { method: "POST", token, body });
      }
      setForm(initialForm);
      setPrependBeforeFirst(false);
      await load();
    } catch (err) {
      setError(normalizeProfitShareRuleError(err, locale));
    }
  }

  function startEdit(rule: ProfitShareRule) {
    setForm({
      id: rule.id,
      recipient_user_id: rule.recipient_user_id || "",
      responsibility: rule.responsibility,
      level: String(rule.level || 1),
      percent: String(rule.percent)
    });
    setPrependBeforeFirst(false);
  }

  async function onDelete(ruleId: string) {
    if (legacyReadOnly) {
      setError(tr("Этот legacy-раздел доступен только для просмотра. Основное распределение денег теперь настраивается через «Финансовый конвейер».", "This legacy section is read-only. Configure the main money distribution in Finance Pipeline."));
      return;
    }
    const confirmed = await confirmAction({
      title: tr("Удалить правило распределения?", "Delete profit-sharing rule?"),
      message: tr("Правило будет удалено без возможности восстановления.", "The rule will be permanently deleted."),
      confirmText: tr("Удалить", "Delete"),
      cancelText: tr("Отмена", "Cancel"),
      tone: "danger"
    });
    if (!confirmed) return;

    try {
      await apiRequest(`/projects/${id}/profit-share-rules/${ruleId}`, { method: "DELETE", token });
      await load();
    } catch (err) {
      setError((err as Error).message);
    }
  }

  async function addMember() {
    if (legacyReadOnly) {
      setError(tr("Этот legacy-раздел доступен только для просмотра. Управление составом команды продолжайте через основной проект и «Финансовый конвейер».", "This legacy section is read-only. Continue team and finance setup through the main project and Finance Pipeline."));
      return;
    }
    if (!token) return;
    const lookup = parseMemberLookup(memberLookup);
    if (!lookup) {
      setError(tr("Укажите корректный email, user ID (UUID) или никнейм.", "Provide a valid email, user ID (UUID), or nickname."));
      return;
    }
    setError("");
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
      await load();
    } catch (err) {
      setError((err as Error).message);
    }
  }

  async function moveRuleToLevel(rule: ProfitShareRule, targetLevel: number) {
    if (!token) return;
    if (legacyReadOnly) {
      setError(tr("Этот legacy-раздел доступен только для просмотра. Основное распределение денег теперь настраивается через «Финансовый конвейер».", "This legacy section is read-only. Configure the main money distribution in Finance Pipeline."));
      return;
    }
    const levelValidation = validateLevelInput(targetLevel, rule.id);
    if (!levelValidation.ok) {
      setError(levelValidation.message);
      return;
    }
    const budgetValidation = validateLevelPercentBudget(targetLevel, rule.percent, rule.id);
    if (!budgetValidation.ok) {
      setError(budgetValidation.message);
      return;
    }

    if (rule.level === targetLevel) return;
    setMovingRuleId(rule.id);
    setError("");
    try {
      await apiRequest(`/projects/${id}/profit-share-rules/${rule.id}`, {
        method: "PUT",
        token,
        body: {
          recipient_user_id: rule.recipient_user_id || undefined,
          responsibility: rule.responsibility,
          percent: rule.percent,
          level: targetLevel,
          applies_after_mandatory: targetLevel > 1
        }
      });
      await load();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setMovingRuleId(null);
    }
  }

  async function calculatePayouts() {
    if (legacyReadOnly) {
      setError(tr("Этот legacy-раздел доступен только для просмотра. Для новых снимков и settlement используйте «Финансовый конвейер».", "This legacy section is read-only. Use Finance Pipeline for new snapshots and settlement."));
      return;
    }
    setError("");
    setNotice("");
    setIsSavingSnapshot(true);
    try {
      const result = await apiRequest<CalculatePayoutsResponse>(`/projects/${id}/payouts/calculate`, { method: "POST", token });
      if (result.snapshot?.download_url) {
        const pdf = await apiDownload(result.snapshot.download_url, token);
        const objectURL = URL.createObjectURL(pdf);
        const anchor = document.createElement("a");
        anchor.href = objectURL;
        anchor.download = result.snapshot.file_name || `payout_snapshot_${Date.now()}.pdf`;
        document.body.appendChild(anchor);
        anchor.click();
        anchor.remove();
        URL.revokeObjectURL(objectURL);
        setNotice(tr("Снимок выплат сохранён и скачан в PDF.", "Payout snapshot was saved and downloaded as PDF."));
      }
      await load();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setIsSavingSnapshot(false);
    }
  }

  if (!ready || !project) return null;
  const currencyCode = project.currency_code;
  const canFinanceWrite = false;

  const levelsMap = new Map<number, ProfitShareRule[]>();
  for (const rule of rules) {
    const level = rule.level || 1;
    if (!levelsMap.has(level)) levelsMap.set(level, []);
    levelsMap.get(level)?.push(rule);
  }
  const levels = Array.from(levelsMap.keys()).sort((a, b) => a - b);
  const maxLevel = levels.length ? levels[levels.length - 1] : 0;
  const levelSums = getLevelSums();
  const fullLevel = levels.find((level) => (levelSums.get(level) || 0) >= 99.999999) || null;
  const nextDnDLevel = maxLevel + 1;
  const canCreateNextLevel = fullLevel === null && nextDnDLevel <= MAX_PROFIT_LEVEL;
  const levelHint =
    !form.id && prependBeforeFirst
      ? ""
      : Number.isInteger(Number(form.level))
        ? validateLevelInput(Number(form.level), form.id).message
        : "";
  const distributedAmount = preview ? preview.distributed_amount : 0;
  const distributedPercent = preview ? preview.distributed_percent : 0;
  const membersByID = new Map(members.map((member) => [member.user_id, member]));
  const linkedMember = form.recipient_user_id ? membersByID.get(form.recipient_user_id) : undefined;
  const linkedMemberName = linkedMember ? displayMemberName(linkedMember) : "";
  const roleOptions = projectRoleOptions(t);
  const selectedMemberRoleView = projectRoleView(t, memberRole);

  return (
    <AppShell title={project.name} className="profit-share-shell">
      <ProjectNav projectId={id} />
      {error && <p style={{ color: "#bc4749" }}>{error}</p>}
      {notice && <p style={{ color: "#2a9d8f" }}>{notice}</p>}

      <section className="card legacy-route-note">
        <strong>{tr("Legacy-раздел правил распределения", "Legacy distribution rules section")}</strong>
        <p>
          {tr(
            "Эта логика больше не является основной моделью денежного потока. Для новых проектов используйте «Финансовый конвейер», а этот раздел считайте совместимым legacy-экраном.",
            "This logic is no longer the main money-flow model. Use Finance Pipeline for new projects, and treat this section as a legacy compatibility screen."
          )}
        </p>
        <Link href={`/projects/${id}/finance-pipeline`} className="inline-link">
          {tr("Открыть финансовый конвейер", "Open Finance Pipeline")}
        </Link>
      </section>

      <section className="card profit-team-card" style={{ marginBottom: 14 }}>
        <div className="finance-card-head">
          <div>
            <span className="shell-eyebrow">{tr("Команда", "Team")}</span>
            <h2 className="title">{tr("Команда и доступ", "Team and access")}</h2>
          </div>
          <p className="subtle" style={{ margin: 0 }}>
            {tr(
              "Связывайте правила распределения с участниками проекта. Валюта выплаты определяется по профилю получателя.",
              "Link distribution rules to project members. Payout currency is resolved from the recipient profile."
            )}
          </p>
        </div>
        <p className="subtle" style={{ marginTop: 0, marginBottom: 10 }}>
          <Link href={`/projects/${id}/members`}>{tr("Открыть полное управление участниками", "Open full member management")}</Link>
        </p>
        {canManageMembers ? (
          <div>
            <div>
              <p className="subtle" style={{ marginTop: 0 }}>{tr("Добавить участника", "Add member")}</p>
              <div className="grid" style={{ gap: 8 }}>
                <input
                  placeholder={tr("Email, user ID (UUID) или никнейм (@nickname)", "Email, user ID (UUID), or nickname (@nickname)")}
                  value={memberLookup}
                  onChange={(e) => setMemberLookup(e.target.value)}
                />
                <div style={{ display: "flex", gap: 8 }}>
                  <select value={memberRole} onChange={(e) => setMemberRole(e.target.value as typeof memberRole)}>
                    {roleOptions.map((opt) => (
                      <option key={opt.code} value={opt.code}>{opt.title}</option>
                    ))}
                  </select>
                  <button type="button" className="fixed-width-button" onClick={addMember}>{tr("Добавить", "Add")}</button>
                </div>
                <p className="subtle" style={{ margin: "2px 0 0", fontSize: "0.85rem" }}>
                  {selectedMemberRoleView.subtitle}
                </p>
              </div>
            </div>
          </div>
        ) : (
          <p className="subtle" style={{ marginTop: 0 }}>
            {tr("Добавление участников доступно роли PM.", "Adding members is available for PM role only.")}
          </p>
        )}

        {!!members.length && (
          <div style={{ marginTop: 12 }} className="team-members-block">
            <div className="team-members-block-head">
              <h3>{tr("Участники проекта", "Project members")}</h3>
              <span>{members.length}</span>
            </div>
            <table>
              <thead>
                <tr>
                  <th>{tr("Имя", "Name")}</th>
                  <th>{tr("Роль", "Role")}</th>
                  <th>{tr("Выплата", "Payout")}</th>
                </tr>
              </thead>
              <tbody>
                {members.map((member) => (
                  <tr key={member.id}>
                    <td>
                      <div style={{ display: "grid", gap: 2 }}>
                        <strong style={{ fontSize: "1rem", lineHeight: 1.2 }}>
                          {displayMemberName(member)}
                        </strong>
                        <span className="subtle" style={{ fontSize: "0.85rem", lineHeight: 1.2 }}>
                          {member.email}
                        </span>
                      </div>
                    </td>
                    <td>
                      <div style={{ display: "grid", gap: 2 }}>
                        <strong style={{ fontSize: "1rem", lineHeight: 1.2 }}>
                          {projectRoleView(t, member.role).title}
                        </strong>
                        <span className="subtle" style={{ fontSize: "0.85rem", lineHeight: 1.2 }}>
                          {projectRoleView(t, member.role).subtitle}
                        </span>
                      </div>
                    </td>
                    <td>{member.default_currency_code}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

      </section>

      <div className="grid two profit-share-top-grid">
        <section className="card finance-form-card">
          <div className="finance-card-head">
            <div>
              <span className="shell-eyebrow">{tr("Правило", "Rule")}</span>
              <h2 className="title">{form.id ? tr("Редактировать", "Edit") : tr("Добавить", "Add")} {tr("правило распределения прибыли", "profit-sharing rule")}</h2>
            </div>
            <p className="subtle" style={{ margin: 0 }}>
              {tr(
                "Ручная схема распределения прибыли по уровням. Следующий уровень считает процент от остатка предыдущего.",
                "Manual profit sharing by levels. Each next level is based on the previous level remainder."
              )}
            </p>
          </div>
          <form className="grid" onSubmit={onSubmit}>
            <label style={{ display: "grid", gap: 6 }}>
              {tr("Получатель из команды (строгое связывание)", "Recipient from project team (strict binding)")}
              <select
                value={form.recipient_user_id}
                onChange={(e) => setForm({ ...form, recipient_user_id: e.target.value })}
                disabled={!canFinanceWrite}
                required
              >
                <option value="">{tr("Выберите участника проекта", "Select a project member")}</option>
                {members.map((member) => (
                  <option key={member.user_id} value={member.user_id}>
                    {displayMemberName(member)}
                  </option>
                ))}
              </select>
            </label>
            <p className="subtle" style={{ margin: 0 }}>
              {linkedMember
                ? tr(
                    `${linkedMemberName} — ${projectRoleView(t, linkedMember.role).title}. Имя получателя подтянется автоматически из аккаунта.`,
                    `${linkedMemberName} — ${projectRoleView(t, linkedMember.role).title}. Recipient name is derived automatically from the linked account.`
                  )
                : tr(
                    "Правило можно создать только для действующего участника проекта.",
                    "Rules can only be created for current project members."
                  )}
            </p>
            <p className="subtle" style={{ margin: 0 }}>
              {linkedMember
                ? tr(
                    `Валюта выплаты автоматически берётся из профиля участника: ${linkedMember.default_currency_code}.`,
                    `Payout currency is taken automatically from the member profile: ${linkedMember.default_currency_code}.`
                  )
                : tr(`Базовая валюта проекта: ${currencyCode}.`, `Project base currency: ${currencyCode}.`)}
            </p>
            <input
              placeholder={tr("За что отвечает (должность / зона ответственности)", "Responsibility (position / area)")}
              value={form.responsibility}
              onChange={(e) => setForm({ ...form, responsibility: e.target.value })}
              disabled={!canFinanceWrite}
              required
            />
            <div className="level-picker">
              <label className="level-picker-label">{tr("Уровень распределения", "Distribution level")}</label>
              <div className="level-picker-row">
                <select
                  value={!form.id && prependBeforeFirst ? "1" : form.level}
                  onChange={(e) => setForm({ ...form, level: e.target.value })}
                  disabled={!canFinanceWrite || (!form.id && prependBeforeFirst)}
                >
                  {Array.from({ length: MAX_PROFIT_LEVEL }, (_, index) => index + 1).map((level) => (
                    <option key={level} value={String(level)}>
                      {levelLabel(level, locale)}
                    </option>
                  ))}
                </select>
              </div>
              <p className="subtle" style={{ margin: "4px 0 0" }}>
                {tr(
                  `Доступно до ${MAX_PROFIT_LEVEL} уровней. Уровни создаются последовательно.`,
                  `Up to ${MAX_PROFIT_LEVEL} levels are available. Levels are created sequentially.`
                )}
              </p>
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
                  "Новое правило станет уровнем 1, а все существующие уровни автоматически сдвинутся на +1.",
                  "The new rule becomes level 1 and all existing levels shift by +1 automatically."
                )}
              </p>
            )}
            <input
              type="number"
              min="0"
              max="100"
              step="1"
              placeholder={tr("Процент от прибыли (0-100)", "Profit percent (0-100)")}
              value={form.percent}
              onChange={(e) => setForm({ ...form, percent: e.target.value })}
              disabled={!canFinanceWrite}
              required
            />
            {levelHint && <p style={{ color: "#bc4749", margin: 0 }}>{levelHint}</p>}
            <p className="subtle" style={{ margin: 0 }}>
              {tr(
                "На одном уровне сумма процентов не может превышать 100%. Следующий уровень считается от остатка после предыдущего.",
                "At one level, total percent cannot exceed 100%. The next level is calculated from the remainder after the previous level."
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
                  "У вас только просмотр. Изменение правил доступно ролям PM/ANALYST.",
                  "You have read-only access. Editing rules is available for PM/ANALYST roles."
                )}
              </p>
            )}
          </form>
        </section>

        <section className="card tax-board-card">
          <h2 className="title">{tr("Правила по уровням", "Rules by levels")}</h2>
          <p className="subtle" style={{ marginTop: 0 }}>
            {tr("Перетаскивайте карточки между уровнями.", "Drag cards between levels.")}
          </p>
          {!rules.length && <p className="subtle">{tr("Правила пока не добавлены.", "No rules added yet.")}</p>}

          {!!rules.length && (
            <div className="tax-levels-stack">
              {levels.map((level) => {
                const levelPercent = levelSums.get(level) || 0;
                const levelIsFull = levelPercent >= 99.999999;
                return (
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
                      const ruleId = e.dataTransfer.getData("text/plain") || draggedRuleId;
                      if (!ruleId) return;
                      const moved = rules.find((r) => r.id === ruleId);
                      setDragOverLevel(null);
                      setDraggedRuleId(null);
                      if (!moved) return;
                      await moveRuleToLevel(moved, level);
                    }}
                  >
                    <div className="tax-level-head">
                      {levelLabel(level, locale)} • {levelPercent.toFixed(2)}%
                    </div>
                    {levelIsFull && (
                      <p className="subtle" style={{ marginTop: 0 }}>
                        {tr(
                          "На этом уровне распределено 100%. Следующие уровни не нужны: остаток будет 0.",
                          "This level allocates 100%. Next levels are unnecessary: remainder is 0."
                        )}
                      </p>
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
                            e.dataTransfer.setData("text/plain", rule.id);
                          }}
                          onDragEnd={() => {
                            setDraggedRuleId(null);
                            setDragOverLevel(null);
                          }}
                        >
                          {(() => {
                            const linkedRuleMember = rule.recipient_user_id ? membersByID.get(rule.recipient_user_id) : undefined;
                            const roleView = linkedRuleMember ? projectRoleView(t, linkedRuleMember.role) : null;
                            const recipientLabel = linkedRuleMember ? displayMemberName(linkedRuleMember) : rule.recipient_name;
                            return (
                              <>
                                <div className="tax-rule-card-main">
                                  <strong>{recipientLabel}</strong>
                                  <span>{rule.percent}%</span>
                                </div>
                                <p className="subtle" style={{ margin: 0 }}>
                                  {displayResponsibility(rule.responsibility)} · {(linkedRuleMember?.default_currency_code || rule.currency_code || currencyCode)}
                                </p>
                                {roleView && (
                                  <p className="subtle" style={{ margin: 0 }}>
                                    {roleView.title}
                                  </p>
                                )}
                              </>
                            );
                          })()}
                          <div className="profit-rules-actions">
                            <button type="button" className="secondary" onClick={() => startEdit(rule)} disabled={!canFinanceWrite}>{tr("Редактировать", "Edit")}</button>
                            <button type="button" className="warn" onClick={() => onDelete(rule.id)} disabled={!canFinanceWrite}>{tr("Удалить", "Delete")}</button>
                          </div>
                        </article>
                      ))}
                    </div>
                  </section>
                );
              })}

              <section
                className={`tax-level-group tax-level-new${dragOverLevel === nextDnDLevel ? " drop-target" : ""}`}
                onDragOver={(e) => {
                  if (!canFinanceWrite) return;
                  if (!canCreateNextLevel) return;
                  e.preventDefault();
                  e.dataTransfer.dropEffect = "move";
                  setDragOverLevel(nextDnDLevel);
                }}
                onDragLeave={() => {
                  if (dragOverLevel === nextDnDLevel) setDragOverLevel(null);
                }}
                onDrop={async (e) => {
                  if (!canFinanceWrite) return;
                  e.preventDefault();
                  const ruleId = e.dataTransfer.getData("text/plain") || draggedRuleId;
                  if (!canCreateNextLevel || !ruleId) return;
                  const moved = rules.find((r) => r.id === ruleId);
                  setDragOverLevel(null);
                  setDraggedRuleId(null);
                  if (!moved) return;
                  await moveRuleToLevel(moved, nextDnDLevel);
                }}
              >
                <div className="tax-level-head">{tr("Новый уровень", "New level")} {nextDnDLevel}</div>
                {canCreateNextLevel ? (
                  <p className="subtle" style={{ margin: 0 }}>
                    {tr("Перетащите сюда правило, чтобы создать следующий уровень.", "Drag a rule here to create the next level.")}
                  </p>
                ) : nextDnDLevel > MAX_PROFIT_LEVEL ? (
                  <p className="subtle" style={{ margin: 0 }}>
                    {tr(`Достигнут лимит уровней: ${MAX_PROFIT_LEVEL}.`, `Level limit reached: ${MAX_PROFIT_LEVEL}.`)}
                  </p>
                ) : (
                  <p className="subtle" style={{ margin: 0 }}>
                    {tr(
                      `Следующий уровень недоступен: на уровне ${fullLevel} уже распределено 100%.`,
                      `Next level unavailable: level ${fullLevel} already allocates 100%.`
                    )}
                  </p>
                )}
              </section>
            </div>
          )}
        </section>
      </div>

      {preview && (
        <section className="card profit-preview-card" style={{ marginTop: 14 }}>
          <h2 className="title">{tr("Предпросмотр выплат", "Payout preview")}</h2>
          <p className="subtle" style={{ marginTop: 0 }}>
            {preview.viewer_currency_code
              ? tr(
                  `Сначала показана сумма в валюте проекта (${currencyCode}), ниже — эквивалент в вашей валюте (${preview.viewer_currency_code}).`,
                  `Each payout shows the project-currency amount first (${currencyCode}), then the equivalent in your currency (${preview.viewer_currency_code}).`
                )
              : tr(
                  `Сначала показана сумма в валюте проекта (${currencyCode}). Если конвертация недоступна, предпросмотр остаётся в базовой валюте проекта.`,
                  `Each payout shows the project-currency amount first (${currencyCode}). If conversion is unavailable, the preview stays in the project base currency.`
                )}
          </p>
          <div className="grid three tax-impact-grid" style={{ marginBottom: 12 }}>
            <article className="tax-impact-item">
              <p className="subtle">{tr("Чистая прибыль", "Net profit")}</p>
              <strong>{money(preview.summary.net_profit_to_date, currencyCode)}</strong>
              <span>{tr("После доходов и расходов проекта", "After project income and expenses")}</span>
            </article>
            <article className="tax-impact-item">
              <p className="subtle">{tr("Налоги и комиссии", "Taxes and fees")} ({preview.summary.tax_percent_total}%)</p>
              <strong>{money(preview.summary.tax_total, currencyCode)}</strong>
              <span>{tr("Удержания до выплат участникам", "Deductions before member payouts")}</span>
            </article>
            <article className="tax-impact-item tax-impact-item-success">
              <p className="subtle">{tr("Распределено по правилам", "Allocated by rules")} ({distributedPercent}%)</p>
              <strong>{money(distributedAmount, currencyCode)}</strong>
              <span>{tr("Остаток автоматически получает администратор", "The remainder is automatically assigned to the administrator")}</span>
            </article>
          </div>

          <table>
            <thead>
              <tr>
                <th>{tr("Получатель", "Recipient")}</th>
                <th>{tr("Ответственность", "Responsibility")}</th>
                <th>{tr("Уровень", "Level")}</th>
                <th>{tr("Процент", "Percent")}</th>
                <th>{tr("Сумма выплаты / ваша валюта", "Payout / your currency")}</th>
              </tr>
            </thead>
            <tbody>
              {preview.items.map((item, index) => (
                <tr key={`${item.rule_id}-${index}`}>
                  {(() => {
                    const linkedPreviewMember = item.recipient_user_id ? membersByID.get(item.recipient_user_id) : undefined;
                    const roleView = linkedPreviewMember ? projectRoleView(t, linkedPreviewMember.role) : null;
                    const recipientLabel = linkedPreviewMember ? displayMemberName(linkedPreviewMember) : item.recipient_name;
                    return (
                      <>
                        <td>
                          <div className="money-stack">
                            <strong>{recipientLabel}</strong>
                            <span>{roleView?.title || "—"}</span>
                          </div>
                        </td>
                        <td>{displayResponsibility(item.responsibility)}</td>
                        <td>{levelLabel(item.level || 1, locale)}</td>
                        <td>{item.percent}%</td>
                        <td>
                          <div className="money-stack">
                            <strong>{money(item.payout_amount, currencyCode)}</strong>
                            {item.viewer_amount_converted !== undefined && preview.viewer_currency_code ? (
                              <span>{money(item.viewer_amount_converted, preview.viewer_currency_code)}</span>
                            ) : (
                              <span>{tr("Конвертация недоступна", "Conversion unavailable")}</span>
                            )}
                          </div>
                        </td>
                      </>
                    );
                  })()}
                </tr>
              ))}
              {!preview.items.length && (
                <tr>
                  <td colSpan={5} className="subtle">{tr("Правила распределения прибыли не настроены.", "Profit-sharing rules are not configured.")}</td>
                </tr>
              )}
            </tbody>
          </table>

          <div style={{ marginTop: 12 }}>
            <button onClick={calculatePayouts} disabled={isSavingSnapshot || !canFinanceWrite}>
              {isSavingSnapshot ? tr("Сохраняем PDF...", "Saving PDF...") : tr("Сохранить снимок выплат", "Save payout snapshot")}
            </button>
          </div>
        </section>
      )}
    </AppShell>
  );
}
