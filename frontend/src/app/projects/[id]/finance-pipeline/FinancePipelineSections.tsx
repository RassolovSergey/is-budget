"use client";

import type {
  AllocationBlock,
  AllocationBlockAppliesTo,
  AllocationBlockType,
  AllocationBlockValueType,
  AllocationPreview,
  AllocationPreviewLevel,
  CurrencyCode,
  PayoutSnapshotBlock,
  PayoutSnapshotDetail,
  PayoutSnapshotSummary,
  TeamDistributionMode
} from "../../../../lib/types";
import { money } from "../../../../lib/format";

export type FinancePipelineFormState = {
  id?: string;
  name: string;
  type: AllocationBlockType;
  level_index: string;
  enabled: boolean;
  value_type: AllocationBlockValueType;
  value: string;
  applies_to: AllocationBlockAppliesTo;
  team_distribution_mode: TeamDistributionMode;
};

export const initialFinancePipelineForm: FinancePipelineFormState = {
  name: "",
  type: "tax",
  level_index: "1",
  enabled: true,
  value_type: "percent",
  value: "",
  applies_to: "gross",
  team_distribution_mode: "percent"
};

function isTeamDistributionType(type: string): boolean {
  return type === "team_distribution" || type === "team_pool_percent" || type === "team_pool_remainder";
}

export function teamDistributionModeForBlock(
  block: Pick<AllocationBlock, "type" | "value_type" | "value"> | { type: string; value_type: string; value: number }
): TeamDistributionMode {
  if (block.value_type === "fixed" && block.value === 0) {
    return "full_remainder";
  }
  return "percent";
}

function teamDistributionModeForSnapshotBlock(block: Pick<PayoutSnapshotBlock, "type" | "value_type" | "value">): TeamDistributionMode {
  if (block.type === "team_pool_remainder") {
    return "full_remainder";
  }
  if (block.value_type === "fixed" && block.value === 0) {
    return "full_remainder";
  }
  return "percent";
}

export function blockTypeLabel(type: string, locale: "ru" | "en"): string {
  const map: Record<string, [string, string]> = {
    tax: ["Налог", "Tax"],
    tax_percent: ["Налог", "Tax"],
    actual_expense: ["Фактический расход", "Actual expense"],
    fixed_cost: ["Фактический расход", "Actual expense"],
    custom_fee: ["Дополнительный сбор", "Custom fee"],
    studio_distribution: ["Распределение студии", "Studio distribution"],
    studio_percent: ["Распределение студии", "Studio distribution"],
    admin_distribution: ["Распределение админа", "Admin distribution"],
    admin_percent: ["Распределение админа", "Admin distribution"],
    team_distribution: ["Распределение команде", "Team distribution"],
    team_pool_percent: ["Распределение команде", "Team distribution"],
    team_pool_remainder: ["Распределение команде", "Team distribution"]
  };
  const value = map[type] || [type, type];
  return locale === "en" ? value[1] : value[0];
}

function appliesToLabel(value: AllocationBlockAppliesTo | string, locale: "ru" | "en"): string {
  return value === "gross"
    ? locale === "en"
      ? "from gross income"
      : "от валового дохода"
    : locale === "en"
      ? "from current remainder"
      : "от текущего остатка";
}

function valueTypeLabel(value: AllocationBlockValueType | string, locale: "ru" | "en"): string {
  return value === "percent"
    ? locale === "en"
      ? "percent"
      : "процент"
    : locale === "en"
      ? "fixed"
      : "фиксированная сумма";
}

function sourceLabel(source: string, locale: "ru" | "en"): string {
  if (source === "manual_override") {
    return locale === "en" ? "manual preview amount" : "ручная сумма предпросмотра";
  }
  return locale === "en" ? "actual project income" : "фактический доход проекта";
}

function settlementStatusLabel(status: string, locale: "ru" | "en"): string {
  const map: Record<string, [string, string]> = {
    draft: ["Черновик", "Draft"],
    approved: ["Подтверждён", "Approved"],
    ready_for_settlement: ["Готов к выплате", "Ready for settlement"]
  };
  const value = map[status] || [status, status];
  return locale === "en" ? value[1] : value[0];
}

function blockValueLabel(block: Pick<AllocationBlock, "type" | "value_type" | "value" | "applies_to">, locale: "ru" | "en", currencyCode: CurrencyCode): string {
  if (isTeamDistributionType(block.type)) {
    return teamDistributionModeForBlock(block as AllocationBlock) === "full_remainder"
      ? locale === "en"
        ? "full remainder of the level"
        : "весь остаток уровня"
      : `${block.value}% ${locale === "en" ? "of the current remainder" : "от текущего остатка"}`;
  }
  if (block.value_type === "percent") {
    return `${block.value}% ${appliesToLabel(block.applies_to, locale)}`;
  }
  return money(block.value, currencyCode);
}

function blockValueLabelForSnapshot(block: PayoutSnapshotBlock, locale: "ru" | "en", currencyCode: CurrencyCode): string {
  if (isTeamDistributionType(block.type)) {
    return teamDistributionModeForSnapshotBlock(block) === "full_remainder"
      ? locale === "en"
        ? "full remainder of the level"
        : "весь остаток уровня"
      : `${block.value}% ${locale === "en" ? "of the current remainder" : "от текущего остатка"}`;
  }
  if (block.value_type === "percent") {
    return `${block.value}% ${appliesToLabel(block.applies_to, locale)}`;
  }
  return money(block.value, currencyCode);
}

function levelLabel(levelIndex: number, locale: "ru" | "en"): string {
  return locale === "en" ? `Level ${levelIndex}` : `Уровень ${levelIndex}`;
}

function groupBlocksByLevel<T extends { level_index: number; order_index: number }>(items: T[]): Array<{ level_index: number; items: T[] }> {
  const sorted = [...items].sort((a, b) => (a.level_index === b.level_index ? a.order_index - b.order_index : a.level_index - b.level_index));
  const groups = new Map<number, T[]>();
  for (const item of sorted) {
    if (!groups.has(item.level_index)) {
      groups.set(item.level_index, []);
    }
    groups.get(item.level_index)!.push(item);
  }
  return Array.from(groups.entries()).map(([level_index, levelItems]) => ({ level_index, items: levelItems }));
}

function previewLevelByIndex(preview: AllocationPreview | null, levelIndex: number): AllocationPreviewLevel | null {
  return preview?.levels.find((level) => level.level_index === levelIndex) || null;
}

export function FinancePipelineForm({
  locale,
  canManage,
  form,
  busy,
  error,
  onChange,
  onSubmit,
  onCancel
}: {
  locale: "ru" | "en";
  canManage: boolean;
  form: FinancePipelineFormState;
  busy: boolean;
  error: string;
  onChange: (next: FinancePipelineFormState) => void;
  onSubmit: () => void;
  onCancel: () => void;
}) {
  const tr = (ru: string, en: string) => (locale === "en" ? en : ru);
  const isTeamDistribution = form.type === "team_distribution";

  return (
    <section className="card finance-pipeline-form-card">
      <div className="finance-pipeline-panel-head">
        <div>
          <span className="shell-eyebrow">{tr("Конструктор потока", "Money flow constructor")}</span>
          <h2 className="title">{form.id ? tr("Редактировать блок потока", "Edit flow block") : tr("Добавить блок потока", "Add flow block")}</h2>
          <p className="subtle">
            {tr(
              "Блок добавляется на конкретный уровень. Для первого уровня база всегда валовой доход, для следующих уровней — остаток после предыдущего уровня.",
              "Every block belongs to a specific level. Level 1 always uses gross income, and all following levels use the remainder from the previous level."
            )}
          </p>
        </div>
        {!canManage && <span className="finance-pipeline-readonly-chip">{tr("Только просмотр", "Read only")}</span>}
      </div>

      <div className="grid finance-form-grid">
        <label>
          <span>{tr("Название блока", "Block name")}</span>
          <input
            value={form.name}
            disabled={!canManage || busy}
            onChange={(event) => onChange({ ...form, name: event.target.value })}
            placeholder={tr("Например, НДС, реклама, доля студии", "For example, VAT, ads, studio share")}
          />
        </label>

        <label>
          <span>{tr("Уровень", "Level")}</span>
          <input
            type="number"
            min="1"
            step="1"
            value={form.level_index}
            disabled={!canManage || busy}
            onChange={(event) => onChange({ ...form, level_index: event.target.value })}
          />
        </label>

        <label>
          <span>{tr("Тип блока", "Block type")}</span>
          <select
            value={form.type}
            disabled={!canManage || busy}
            onChange={(event) => {
              const type = event.target.value as AllocationBlockType;
              onChange({
                ...form,
                type,
                value_type: type === "team_distribution" ? "percent" : form.value_type,
                applies_to: type === "team_distribution" ? "current_balance" : form.applies_to,
                team_distribution_mode: type === "team_distribution" ? form.team_distribution_mode : "percent",
                value: type === "team_distribution" && form.team_distribution_mode === "full_remainder" ? "0" : form.value
              });
            }}
          >
            {(["tax", "actual_expense", "custom_fee", "admin_distribution", "studio_distribution", "team_distribution"] as AllocationBlockType[]).map((type) => (
              <option key={type} value={type}>
                {blockTypeLabel(type, locale)}
              </option>
            ))}
          </select>
        </label>

        {isTeamDistribution ? (
          <label>
            <span>{tr("Режим распределения команде", "Team distribution mode")}</span>
            <select
              value={form.team_distribution_mode}
              disabled={!canManage || busy}
              onChange={(event) =>
                onChange({
                  ...form,
                  team_distribution_mode: event.target.value as TeamDistributionMode,
                  value_type: event.target.value === "full_remainder" ? "fixed" : "percent",
                  applies_to: "current_balance",
                  value: event.target.value === "full_remainder" ? "0" : form.value
                })
              }
            >
              <option value="percent">{tr("Процент от текущего остатка", "Percent of current remainder")}</option>
              <option value="full_remainder">{tr("Весь остаток уровня", "Full remainder of the level")}</option>
            </select>
          </label>
        ) : (
          <label>
            <span>{tr("Тип значения", "Value type")}</span>
            <select
              value={form.value_type}
              disabled={!canManage || busy}
              onChange={(event) => onChange({ ...form, value_type: event.target.value as AllocationBlockValueType })}
            >
              <option value="percent">{tr("Процент", "Percent")}</option>
              <option value="fixed">{tr("Фиксированная сумма", "Fixed amount")}</option>
            </select>
          </label>
        )}

        {(!isTeamDistribution || form.team_distribution_mode === "percent") && (
          <label>
            <span>{tr("Значение", "Value")}</span>
            <input
              type="number"
              min="0"
              step="0.01"
              value={form.value}
              disabled={!canManage || busy}
              onChange={(event) => onChange({ ...form, value: event.target.value })}
            />
          </label>
        )}

        <label className="finance-pipeline-toggle-field">
          <span>{tr("Активен", "Enabled")}</span>
          <input
            type="checkbox"
            checked={form.enabled}
            disabled={!canManage || busy}
            onChange={(event) => onChange({ ...form, enabled: event.target.checked })}
          />
        </label>
      </div>

      {isTeamDistribution && (
        <p className="finance-pipeline-inline-note">
          {form.team_distribution_mode === "full_remainder"
            ? tr(
                "Этот блок передаст в командный пул весь остаток уровня и обнулит сумму, идущую дальше.",
                "This block will send the full remainder of the level into the team pool and zero out the amount that moves forward."
              )
            : tr(
                "Этот блок передаст в командный пул процент от текущего остатка уровня. Остаток перейдёт дальше.",
                "This block will send a percent of the current level remainder into the team pool. The rest will continue to the next level."
              )}
        </p>
      )}

      {error ? <p className="workflow-inline-error">{error}</p> : null}

      <div className="finance-pipeline-form-actions">
        <button type="button" disabled={!canManage || busy} onClick={onSubmit}>
          {busy ? tr("Сохраняем...", "Saving...") : form.id ? tr("Сохранить блок", "Save block") : tr("Добавить блок", "Add block")}
        </button>
        {form.id && (
          <button type="button" className="ghost-button" disabled={!canManage || busy} onClick={onCancel}>
            {tr("Отмена", "Cancel")}
          </button>
        )}
      </div>
    </section>
  );
}

export function FinancePipelineLevels({
  locale,
  currencyCode,
  blocks,
  preview,
  canManage,
  busyKey,
  onEdit,
  onDelete,
  onMoveAcrossLevels
}: {
  locale: "ru" | "en";
  currencyCode: CurrencyCode;
  blocks: AllocationBlock[];
  preview: AllocationPreview | null;
  canManage: boolean;
  busyKey: string;
  onEdit: (block: AllocationBlock) => void;
  onDelete: (block: AllocationBlock) => void;
  onMoveAcrossLevels: (block: AllocationBlock, direction: "prev" | "next") => void;
}) {
  const tr = (ru: string, en: string) => (locale === "en" ? en : ru);
  const groups = groupBlocksByLevel(blocks);

  return (
    <section className="card finance-pipeline-list-card">
      <div className="finance-pipeline-panel-head">
        <div>
          <span className="shell-eyebrow">{tr("Уровни конвейера", "Pipeline levels")}</span>
          <h2 className="title">{tr("Слои денежного потока", "Money flow layers")}</h2>
        </div>
      </div>

      {!groups.length && (
        <div className="finance-pipeline-empty card">
          <strong>{tr("Блоков пока нет", "No blocks yet")}</strong>
          <p>{tr("Добавьте первый блок, чтобы собрать слои денежного потока проекта.", "Add the first block to start building the project money flow.")}</p>
        </div>
      )}

      <div className="finance-pipeline-levels">
        {groups.map((group) => {
          const levelPreview = previewLevelByIndex(preview, group.level_index);
          return (
            <section key={group.level_index} className="finance-pipeline-level-card">
              <div className="finance-pipeline-level-head">
                <div>
                  <span className="workflow-status-badge status-in_progress">{levelLabel(group.level_index, locale)}</span>
                </div>
                {levelPreview && (
                  <div className="finance-pipeline-level-metrics">
                    <span>{tr("Вход", "Incoming")}: <strong>{money(levelPreview.incoming_amount, currencyCode)}</strong></span>
                    <span>{tr("Списано / распределено", "Allocated")}: <strong>{money(levelPreview.total_allocated, currencyCode)}</strong></span>
                    <span>{tr("Остаток после уровня", "Remaining after level")}: <strong>{money(levelPreview.remaining_amount, currencyCode)}</strong></span>
                  </div>
                )}
              </div>

              <div className="finance-pipeline-blocks">
                {group.items.map((block) => (
                  <article key={block.id} className={`finance-pipeline-block ${!block.enabled ? "is-disabled" : ""}`.trim()}>
                    {(() => {
                      const previewStep = levelPreview?.steps.find((step) => step.block_id === block.id) || null;
                      return (
                        <>
                    <div className="finance-pipeline-block-top">
                      <div className="finance-pipeline-block-main">
                        <div className="finance-pipeline-block-order">{block.order_index}</div>
                        <div>
                          <h3>{block.name}</h3>
                          <p className="subtle">
                            {blockTypeLabel(block.type, locale)} · {blockValueLabel(block, locale, currencyCode)}
                          </p>
                        </div>
                      </div>
                      <div className="finance-pipeline-block-badges">
                        <span className={`workflow-status-badge status-${block.enabled ? "ready" : "planned"}`}>{block.enabled ? tr("Активен", "Enabled") : tr("Выключен", "Disabled")}</span>
                        <span className="workflow-parent-chip">{levelLabel(block.level_index, locale)}</span>
                        {!isTeamDistributionType(block.type) && <span className="workflow-parent-chip">{valueTypeLabel(block.value_type, locale)}</span>}
                        {isTeamDistributionType(block.type) && (
                          <span className="workflow-parent-chip">
                            {teamDistributionModeForBlock(block) === "full_remainder" ? tr("Весь остаток", "Full remainder") : tr("Процент остатка", "Percent of remainder")}
                          </span>
                        )}
                      </div>
                    </div>

                    <div className="finance-pipeline-block-meta">
                      <span>{tr("Уровень", "Level")}: {block.level_index}</span>
                      {previewStep ? (
                        <span>
                          {tr("Сейчас забирает", "Currently takes")}: {money(previewStep.allocated_amount, currencyCode)}
                        </span>
                      ) : null}
                      {previewStep && previewStep.is_team_pool && previewStep.team_distribution_mode === "full_remainder" ? (
                        <span>{tr("Остаток после блока: 0", "Remaining after block: 0")}</span>
                      ) : null}
                    </div>

                    {canManage && (
                      <div className="finance-pipeline-block-actions">
                        <div className="finance-pipeline-block-structure-actions">
                          <button
                            type="button"
                            className="icon-button"
                            aria-label={tr("На уровень выше", "Move to previous level")}
                            title={tr("На уровень выше", "Move to previous level")}
                            disabled={busyKey !== "" || block.level_index === 1}
                            onClick={() => onMoveAcrossLevels(block, "prev")}
                          >
                            ↑
                          </button>
                          <button
                            type="button"
                            className="icon-button"
                            aria-label={tr("На уровень ниже", "Move to next level")}
                            title={tr("На уровень ниже", "Move to next level")}
                            disabled={busyKey !== ""}
                            onClick={() => onMoveAcrossLevels(block, "next")}
                          >
                            ↓
                          </button>
                        </div>
                        <div className="finance-pipeline-block-primary-actions">
                          <button type="button" className="ghost-button" disabled={busyKey !== ""} onClick={() => onEdit(block)}>
                            {tr("Редактировать", "Edit")}
                          </button>
                          <button type="button" className="ghost-button danger-text" disabled={busyKey !== ""} onClick={() => onDelete(block)}>
                            {tr("Удалить", "Delete")}
                          </button>
                        </div>
                      </div>
                    )}
                        </>
                      );
                    })()}
                  </article>
                ))}
              </div>
            </section>
          );
        })}
      </div>
    </section>
  );
}

export function PayoutSnapshotsPanel({
  locale,
  currencyCode,
  canManage,
  creating,
  snapshots,
  selectedSnapshot,
  settlementForm,
  settlementBusy,
  onChangeSettlement,
  onSaveSettlement,
  onCreate,
  onOpen
}: {
  locale: "ru" | "en";
  currencyCode: CurrencyCode;
  canManage: boolean;
  creating: boolean;
  snapshots: PayoutSnapshotSummary[];
  selectedSnapshot: PayoutSnapshotDetail | null;
  settlementForm: { settlement_status: "draft" | "approved" | "ready_for_settlement"; settlement_note: string };
  settlementBusy: boolean;
  onChangeSettlement: (next: { settlement_status: "draft" | "approved" | "ready_for_settlement"; settlement_note: string }) => void;
  onSaveSettlement: () => void;
  onCreate: () => void;
  onOpen: (snapshotID: string) => void;
}) {
  const tr = (ru: string, en: string) => (locale === "en" ? en : ru);
  const snapshotLevelGroups = selectedSnapshot ? groupBlocksByLevel(selectedSnapshot.blocks) : [];

  return (
    <section className="card finance-snapshot-card">
      <div className="finance-pipeline-panel-head">
        <div>
          <span className="shell-eyebrow">{tr("Исторические снимки", "Historical snapshots")}</span>
          <h2 className="title">{tr("Зафиксированные снимки распределения", "Frozen payout records")}</h2>
          <p className="subtle">
            {tr(
              "Снимок фиксирует текущий валовой доход, уровни конвейера, командный пул и предпросмотр выплат. Дальнейшие изменения проекта его не меняют.",
              "A snapshot freezes the current gross, pipeline levels, team pool, and payout preview. Later project changes do not alter it."
            )}
          </p>
        </div>
        {canManage && (
          <button type="button" disabled={creating} onClick={onCreate}>
            {creating ? tr("Сохраняем...", "Saving...") : tr("Сохранить снимок", "Save snapshot")}
          </button>
        )}
      </div>

      {!snapshots.length && (
        <div className="finance-pipeline-empty card">
          <strong>{tr("Снимков пока нет", "No snapshots yet")}</strong>
          <p>{tr("Сохраните текущий денежный поток, чтобы зафиксировать его как историческую запись.", "Save the current money flow to freeze it as a historical record.")}</p>
        </div>
      )}

      {!!snapshots.length && (
        <div className="finance-snapshot-list">
          {snapshots.map((snapshot) => {
            const isActive = selectedSnapshot?.id === snapshot.id;
            return (
              <button key={snapshot.id} type="button" className={`finance-snapshot-row ${isActive ? "is-active" : ""}`.trim()} onClick={() => onOpen(snapshot.id)}>
                <div className="finance-snapshot-row-main">
                  <strong>{new Date(snapshot.created_at).toLocaleString(locale === "en" ? "en-US" : "ru-RU")}</strong>
                  <p className="subtle">{tr("Создал", "Created by")}: {snapshot.created_by_name}</p>
                </div>
                <div className="finance-snapshot-row-metrics">
                  <div>
                    <span>{tr("Статус", "Status")}</span>
                    <strong>{settlementStatusLabel(snapshot.settlement_status, locale)}</strong>
                  </div>
                  <div>
                    <span>{tr("Валовой доход", "Gross")}</span>
                    <strong>{money(snapshot.gross_amount, currencyCode)}</strong>
                  </div>
                  <div>
                    <span>{tr("Командный пул", "Team pool")}</span>
                    <strong>{money(snapshot.final_team_pool_amount, currencyCode)}</strong>
                  </div>
                  <div>
                    <span>{tr("Участников", "Members")}</span>
                    <strong>{snapshot.member_count}</strong>
                  </div>
                </div>
              </button>
            );
          })}
        </div>
      )}

      {selectedSnapshot && (
        <div className="finance-snapshot-detail">
          <div className="finance-pipeline-kpis">
            <article className="card finance-pipeline-kpi">
              <span>{tr("Снимок валового дохода", "Gross snapshot")}</span>
              <strong>{money(selectedSnapshot.gross_amount, currencyCode)}</strong>
              <p>{sourceLabel(selectedSnapshot.gross_amount_source, locale)}</p>
            </article>
            <article className="card finance-pipeline-kpi finance-pipeline-kpi-pool">
              <span>{tr("Снимок командного пула", "Team pool snapshot")}</span>
              <strong>{money(selectedSnapshot.final_team_pool_amount, currencyCode)}</strong>
              <p>{tr("Зафиксированное значение командного распределения", "Frozen team distribution amount")}</p>
            </article>
            <article className="card finance-pipeline-kpi">
              <span>{tr("Всего очков", "Total points")}</span>
              <strong>{selectedSnapshot.total_points}</strong>
              <p>{tr("На момент сохранения", "At the moment of saving")}</p>
            </article>
            <article className="card finance-pipeline-kpi">
              <span>{tr("Статус подготовки выплаты", "Settlement status")}</span>
              <strong>{settlementStatusLabel(selectedSnapshot.settlement_status, locale)}</strong>
              <p>
                {selectedSnapshot.approved_at
                  ? tr(
                      `Подтвердил: ${selectedSnapshot.approved_by_name || "—"} · ${new Date(selectedSnapshot.approved_at).toLocaleString(locale === "en" ? "en-US" : "ru-RU")}`,
                      `Approved by ${selectedSnapshot.approved_by_name || "—"} · ${new Date(selectedSnapshot.approved_at).toLocaleString(locale === "en" ? "en-US" : "ru-RU")}`
                    )
                  : tr("Пока без подтверждения", "Not approved yet")}
              </p>
            </article>
          </div>

          <section className="card finance-snapshot-settlement-card">
            <div className="finance-pipeline-panel-head">
              <div>
                <span className="shell-eyebrow">{tr("Подготовка выплаты", "Settlement foundation")}</span>
                <h3 className="title">{tr("Статус выплаты", "Settlement status")}</h3>
                <p className="subtle">
                  {tr(
                    "Зафиксируйте, находится ли снимок в черновике, подтверждён ли он и готов ли к следующему этапу выплаты.",
                    "Track whether the snapshot is still draft, already approved, or ready for the next settlement step."
                  )}
                </p>
              </div>
            </div>

            <div className="grid finance-form-grid">
              <label>
                <span>{tr("Статус", "Status")}</span>
                <select
                  value={settlementForm.settlement_status}
                  disabled={!canManage || settlementBusy}
                  onChange={(event) =>
                    onChangeSettlement({
                      ...settlementForm,
                      settlement_status: event.target.value as "draft" | "approved" | "ready_for_settlement"
                    })
                  }
                >
                  <option value="draft">{settlementStatusLabel("draft", locale)}</option>
                  <option value="approved">{settlementStatusLabel("approved", locale)}</option>
                  <option value="ready_for_settlement">{settlementStatusLabel("ready_for_settlement", locale)}</option>
                </select>
              </label>

              <label className="finance-form-span-full">
                <span>{tr("Заметка", "Note")}</span>
                <textarea
                  rows={3}
                  value={settlementForm.settlement_note}
                  disabled={!canManage || settlementBusy}
                  onChange={(event) => onChangeSettlement({ ...settlementForm, settlement_note: event.target.value })}
                  placeholder={tr("Например: подтверждено PM перед подготовкой фактической выплаты.", "For example: approved by the PM before preparing the actual payout.")}
                />
              </label>
            </div>

            {selectedSnapshot.settlement_note ? (
              <p className="subtle" style={{ margin: 0 }}>
                {tr("Текущая заметка", "Current note")}: {selectedSnapshot.settlement_note}
              </p>
            ) : null}

            {canManage ? (
              <div className="finance-pipeline-form-actions">
                <button type="button" disabled={settlementBusy} onClick={onSaveSettlement}>
                  {settlementBusy ? tr("Сохраняем...", "Saving...") : tr("Сохранить статус", "Save status")}
                </button>
              </div>
            ) : null}
          </section>

          <div className="finance-team-payout-list">
            {selectedSnapshot.members.map((member) => (
              <article key={member.id} className="finance-team-payout-row">
                <div className="finance-team-payout-main">
                  <strong>{member.user_name}</strong>
                  <p className="subtle">{tr("Завершённых нод", "Completed nodes")}: {member.completed_nodes_count}</p>
                </div>
                <div className="finance-team-payout-metrics">
                  <div>
                    <span>{tr("Очки", "Points")}</span>
                    <strong>{member.total_points}</strong>
                  </div>
                  <div>
                    <span>{tr("Доля", "Share")}</span>
                    <strong>{member.percentage_share.toFixed(2)}%</strong>
                  </div>
                  <div>
                    <span>{tr("Зафиксированная выплата", "Frozen payout")}</span>
                    <strong>{money(member.payout_amount, currencyCode)}</strong>
                  </div>
                </div>
              </article>
            ))}
          </div>

          {!!snapshotLevelGroups.length && (
            <div className="finance-snapshot-blocks">
              <strong>{tr("Зафиксированные уровни конвейера", "Frozen pipeline levels")}</strong>
              <div className="finance-pipeline-levels">
                {snapshotLevelGroups.map((group) => (
                  <section key={group.level_index} className="finance-pipeline-level-card finance-pipeline-level-card-preview">
                    <div className="finance-pipeline-level-head">
                      <div>
                        <span className="workflow-status-badge status-in_progress">{levelLabel(group.level_index, locale)}</span>
                      </div>
                    </div>
                    <div className="finance-pipeline-steps">
                      {group.items.map((block) => (
                        <article key={block.id} className={`finance-pipeline-step ${block.is_team_pool ? "is-team-pool" : ""}`.trim()}>
                          <div className="finance-pipeline-step-head">
                            <div>
                              <strong>#{block.order_index} {block.name}</strong>
                              <p className="subtle">
                                {blockTypeLabel(block.type, locale)} · {blockValueLabelForSnapshot(block, locale, currencyCode)}
                              </p>
                            </div>
                            <span className={`workflow-status-badge status-${block.is_team_pool ? "done" : "in_progress"}`}>
                              {block.is_team_pool ? tr("Командное распределение", "Team distribution") : tr("Списание", "Deduction")}
                            </span>
                          </div>
                          <div className="finance-pipeline-step-grid">
                            <div>
                              <span>{tr("База блока", "Block input")}</span>
                              <strong>{money(block.input_amount, currencyCode)}</strong>
                            </div>
                            <div>
                              <span>{tr("Блок забирает", "Allocated")}</span>
                              <strong>{money(block.allocated_amount, currencyCode)}</strong>
                            </div>
                            <div>
                              <span>{tr("Остаток", "Remaining")}</span>
                              <strong>{money(block.remaining_amount, currencyCode)}</strong>
                            </div>
                          </div>
                        </article>
                      ))}
                    </div>
                  </section>
                ))}
              </div>
            </div>
          )}
        </div>
      )}
    </section>
  );
}
