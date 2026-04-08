"use client";

import type { CampaignConfig, CampaignLifecycleStage, CampaignStageSnapshot, ImportRunDetail, ImportRunSummary } from "../../../../lib/types";
import { money } from "../../../../lib/format";

export type CampaignConfigFormState = {
  current_stage: CampaignLifecycleStage;
  launch_at: string;
  late_pledges_enabled: boolean;
  pledge_over_time_enabled: boolean;
  pledge_manager_enabled: boolean;
  digital_only_fulfillment: boolean;
};

export type CampaignSnapshotFormState = {
  stage: CampaignLifecycleStage;
  followers_count: string;
  notes: string;
  gross_amount: string;
};

export type KickstarterImportFormState = {
  replace_existing: boolean;
};

export function stageLabel(stage: CampaignLifecycleStage, locale: "ru" | "en"): string {
  const map: Record<CampaignLifecycleStage, [string, string]> = {
    pre_launch: ["До запуска", "Pre-launch"],
    live_campaign: ["Активная кампания", "Live campaign"],
    post_campaign: ["После кампании", "Post-campaign"],
    closed: ["Закрыта", "Closed"]
  };
  return locale === "en" ? map[stage][1] : map[stage][0];
}

function importStatusLabel(status: ImportRunSummary["status"], locale: "ru" | "en"): string {
  const map: Record<ImportRunSummary["status"], [string, string]> = {
    pending: ["Обработка", "Pending"],
    success: ["Успешно", "Success"],
    failed: ["Ошибка", "Failed"]
  };
  return locale === "en" ? map[status][1] : map[status][0];
}

function importRecordStatusLabel(status: string, locale: "ru" | "en"): string {
  const map: Record<string, [string, string]> = {
    success: ["Успешно", "Success"],
    failed: ["Ошибка", "Failed"],
    skipped: ["Пропущено", "Skipped"]
  };
  const value = map[status];
  if (!value) return status;
  return locale === "en" ? value[1] : value[0];
}

function featureLabelParts(key: keyof CampaignConfigFormState, locale: "ru" | "en"): { primary: string; secondary?: string } {
  if (locale === "en") {
    const map: Record<string, string> = {
      late_pledges_enabled: "Late pledges enabled",
      pledge_over_time_enabled: "Pledge over time enabled",
      pledge_manager_enabled: "Pledge manager enabled",
      digital_only_fulfillment: "Digital-only fulfillment"
    };
    return { primary: map[key] };
  }

  const map: Record<string, { primary: string; secondary?: string }> = {
    late_pledges_enabled: { primary: "Поздние взносы", secondary: "Late pledges" },
    pledge_over_time_enabled: { primary: "Оплата частями", secondary: "Pledge over time" },
    pledge_manager_enabled: { primary: "Менеджер взносов", secondary: "Pledge manager" },
    digital_only_fulfillment: { primary: "Только цифровая доставка" }
  };
  return map[key];
}

export function CampaignLifecycleConfigPanel({
  locale,
  canManage,
  form,
  busy,
  error,
  onChange,
  onSubmit
}: {
  locale: "ru" | "en";
  canManage: boolean;
  form: CampaignConfigFormState;
  busy: boolean;
  error: string;
  onChange: (next: CampaignConfigFormState) => void;
  onSubmit: () => void;
}) {
  const tr = (ru: string, en: string) => (locale === "en" ? en : ru);

  return (
    <section className="card lifecycle-config-card">
      <div className="finance-pipeline-panel-head">
        <div>
          <span className="shell-eyebrow">{tr("Раздел 1", "Section 1")}</span>
          <h2 className="title">{tr("Настройки кампании", "Campaign settings")}</h2>
          <p className="subtle">
            {tr(
              "Управляйте текущей стадией кампании, моментом запуска и основными параметрами кампании в одном месте.",
              "Manage the current campaign stage, launch milestone, and core campaign settings in one place."
            )}
          </p>
        </div>
      </div>

      <div className="grid finance-form-grid">
        <label>
          <span>{tr("Текущая стадия", "Current stage")}</span>
          <select
            value={form.current_stage}
            disabled={!canManage || busy}
            onChange={(event) => onChange({ ...form, current_stage: event.target.value as CampaignLifecycleStage })}
          >
            {(["pre_launch", "live_campaign", "post_campaign", "closed"] as CampaignLifecycleStage[]).map((stage) => (
              <option key={stage} value={stage}>
                {stageLabel(stage, locale)}
              </option>
            ))}
          </select>
        </label>

        <label>
          <span>{tr("Момент запуска", "Launch milestone")}</span>
          <input
            type="datetime-local"
            value={form.launch_at}
            disabled={!canManage || busy}
            onChange={(event) => onChange({ ...form, launch_at: event.target.value })}
          />
        </label>
      </div>

      <div className="lifecycle-toggle-grid">
        {(
          [
            "late_pledges_enabled",
            "pledge_over_time_enabled",
            "pledge_manager_enabled",
            "digital_only_fulfillment"
          ] as Array<keyof CampaignConfigFormState>
        ).map((key) => (
          <label key={key} className="finance-pipeline-toggle-field">
            <span className="lifecycle-toggle-copy">
              <strong>{featureLabelParts(key, locale).primary}</strong>
              {featureLabelParts(key, locale).secondary ? <small>{featureLabelParts(key, locale).secondary}</small> : null}
            </span>
            <input
              type="checkbox"
              checked={Boolean(form[key])}
              disabled={!canManage || busy}
              onChange={(event) => onChange({ ...form, [key]: event.target.checked })}
            />
          </label>
        ))}
      </div>

      <p className="subtle lifecycle-helper-copy">
        {tr(
          "Эти параметры влияют на механику кампании и последующие выплаты.",
          "These settings affect campaign mechanics and downstream payout behavior."
        )}
      </p>

      {error ? <p className="workflow-inline-error">{error}</p> : null}

      <div className="finance-pipeline-form-actions">
        <button type="button" disabled={!canManage || busy} onClick={onSubmit}>
          {busy ? tr("Сохраняем...", "Saving...") : tr("Сохранить настройки", "Save settings")}
        </button>
      </div>
    </section>
  );
}

export function CampaignStageSnapshotPanel({
  locale,
  canManage,
  currencyCode,
  form,
  busy,
  error,
  snapshots,
  selectedSnapshot,
  onChange,
  onSubmit,
  onOpen
}: {
  locale: "ru" | "en";
  canManage: boolean;
  currencyCode: "EUR" | "USD" | "RUB";
  form: CampaignSnapshotFormState;
  busy: boolean;
  error: string;
  snapshots: CampaignStageSnapshot[];
  selectedSnapshot: CampaignStageSnapshot | null;
  onChange: (next: CampaignSnapshotFormState) => void;
  onSubmit: () => void;
  onOpen: (snapshotID: string) => void;
}) {
  const tr = (ru: string, en: string) => (locale === "en" ? en : ru);

  return (
    <section className="card lifecycle-snapshot-card" id="stage-snapshot">
      <div className="finance-pipeline-panel-head">
        <div>
          <span className="shell-eyebrow">{tr("Раздел 2", "Section 2")}</span>
          <h2 className="title">{tr("Снимок стадии", "Stage snapshot")}</h2>
          <p className="subtle">
            {tr(
              "Сохраните контрольную точку кампании, чтобы зафиксировать состояние проекта на выбранном этапе.",
              "Save a campaign checkpoint to freeze the project state at the selected stage."
            )}
          </p>
        </div>
      </div>

      <div className="grid finance-form-grid">
        <label>
          <span>{tr("Стадия снимка", "Snapshot stage")}</span>
          <select
            value={form.stage}
            disabled={!canManage || busy}
            onChange={(event) => onChange({ ...form, stage: event.target.value as CampaignLifecycleStage })}
          >
            {(["pre_launch", "live_campaign", "post_campaign", "closed"] as CampaignLifecycleStage[]).map((stage) => (
              <option key={stage} value={stage}>
                {stageLabel(stage, locale)}
              </option>
            ))}
          </select>
        </label>

        <label>
          <span>{tr("Число подписчиков", "Followers count")}</span>
          <input
            type="number"
            min="0"
            step="1"
            value={form.followers_count}
            disabled={!canManage || busy}
            onChange={(event) => onChange({ ...form, followers_count: event.target.value })}
          />
        </label>

        <label>
          <span>{tr("Валовой доход для фиксации финансовых значений", "Gross override for frozen finance values")}</span>
          <input
            type="number"
            min="0"
            step="0.01"
            value={form.gross_amount}
            disabled={!canManage || busy}
            onChange={(event) => onChange({ ...form, gross_amount: event.target.value })}
          />
        </label>

        <label className="finance-form-span-full">
          <span>{tr("Заметки", "Notes")}</span>
          <textarea
            rows={3}
            value={form.notes}
            disabled={!canManage || busy}
            onChange={(event) => onChange({ ...form, notes: event.target.value })}
            placeholder={tr("Например: число подписчиков зафиксировано перед объявлением запуска.", "For example: followers measured before launch announcement.")}
          />
        </label>
      </div>

      {error ? <p className="workflow-inline-error">{error}</p> : null}

      <div className="finance-pipeline-form-actions">
        <button type="button" disabled={!canManage || busy} onClick={onSubmit}>
          {busy ? tr("Сохраняем...", "Saving...") : tr("Сохранить снимок стадии", "Save stage snapshot")}
        </button>
      </div>

      {!snapshots.length && (
        <div className="finance-pipeline-empty card">
          <strong>{tr("Снимков стадии пока нет", "No stage snapshots yet")}</strong>
          <p>{tr("Сохраните первую контрольную точку, чтобы зафиксировать важный этап кампании.", "Save the first checkpoint to freeze an important campaign milestone.")}</p>
        </div>
      )}

      {!!snapshots.length && (
        <div className="finance-snapshot-list">
          {snapshots.map((snapshot) => (
            <button
              key={snapshot.id}
              type="button"
              className={`finance-snapshot-row ${selectedSnapshot?.id === snapshot.id ? "is-active" : ""}`.trim()}
              onClick={() => onOpen(snapshot.id)}
            >
              <div className="finance-snapshot-row-main">
                <strong>{stageLabel(snapshot.stage, locale)}</strong>
                <p className="subtle">
                  {new Date(snapshot.created_at).toLocaleString(locale === "en" ? "en-US" : "ru-RU")} · {snapshot.created_by_name}
                </p>
              </div>
              <div className="finance-snapshot-row-metrics">
                <div>
                  <span>{tr("Подписчики", "Followers")}</span>
                  <strong>{snapshot.followers_count ?? "—"}</strong>
                </div>
                <div>
                  <span>{tr("Валовой доход", "Gross")}</span>
                  <strong>{snapshot.gross_amount_snapshot != null ? money(snapshot.gross_amount_snapshot, currencyCode) : "—"}</strong>
                </div>
                <div>
                  <span>{tr("Командный пул", "Team pool")}</span>
                  <strong>{snapshot.team_pool_snapshot != null ? money(snapshot.team_pool_snapshot, currencyCode) : "—"}</strong>
                </div>
              </div>
            </button>
          ))}
        </div>
      )}

      {selectedSnapshot && (
        <div className="finance-snapshot-detail">
          <div className="finance-pipeline-kpis">
            <article className="card finance-pipeline-kpi">
              <span>{tr("Зафиксированная стадия", "Frozen stage")}</span>
              <strong>{stageLabel(selectedSnapshot.stage, locale)}</strong>
              <p>{selectedSnapshot.created_by_name}</p>
            </article>
            <article className="card finance-pipeline-kpi">
              <span>{tr("Снимок подписчиков", "Followers snapshot")}</span>
              <strong>{selectedSnapshot.followers_count ?? "—"}</strong>
              <p>{tr("Особенно полезно до запуска", "Especially useful for pre-launch")}</p>
            </article>
            <article className="card finance-pipeline-kpi">
              <span>{tr("Снимок валового дохода", "Gross snapshot")}</span>
              <strong>{selectedSnapshot.gross_amount_snapshot != null ? money(selectedSnapshot.gross_amount_snapshot, currencyCode) : "—"}</strong>
              <p>{tr("Зафиксировано в момент сохранения", "Frozen at save time")}</p>
            </article>
            <article className="card finance-pipeline-kpi finance-pipeline-kpi-pool">
              <span>{tr("Снимок командного пула", "Team pool snapshot")}</span>
              <strong>{selectedSnapshot.team_pool_snapshot != null ? money(selectedSnapshot.team_pool_snapshot, currencyCode) : "—"}</strong>
              <p>{tr("Текущее значение конвейера на момент сохранения", "Current pipeline value at snapshot creation")}</p>
            </article>
          </div>

          {selectedSnapshot.notes && (
            <div className="card">
              <strong>{tr("Заметки к снимку", "Snapshot notes")}</strong>
              <p className="subtle" style={{ marginBottom: 0 }}>{selectedSnapshot.notes}</p>
            </div>
          )}
        </div>
      )}
    </section>
  );
}

export function KickstarterImportPanel({
  locale,
  canManage,
  currencyCode,
  form,
  busy,
  error,
  imports,
  selectedImport,
  onChange,
  onPickFile,
  onOpen
}: {
  locale: "ru" | "en";
  canManage: boolean;
  currencyCode: "EUR" | "USD" | "RUB";
  form: KickstarterImportFormState;
  busy: boolean;
  error: string;
  imports: ImportRunSummary[];
  selectedImport: ImportRunDetail | null;
  onChange: (next: KickstarterImportFormState) => void;
  onPickFile: (file: File) => void;
  onOpen: (runID: string) => void;
}) {
  const tr = (ru: string, en: string) => (locale === "en" ? en : ru);

  return (
    <section className="card lifecycle-snapshot-card" id="kickstarter-import">
      <div className="finance-pipeline-panel-head">
        <div>
          <span className="shell-eyebrow">{tr("Раздел 3", "Section 3")}</span>
          <h2 className="title">{tr("Импорт данных Kickstarter", "Kickstarter data import")}</h2>
          <p className="subtle">
            {tr(
              "Загрузите CSV-файл из Kickstarter, чтобы обновить доходы проекта.",
              "Upload a CSV file from Kickstarter to update project income."
            )}
          </p>
        </div>
      </div>

      <div className="finance-pipeline-toggle-field lifecycle-import-toggle">
        <span className="lifecycle-toggle-copy">
          <strong>{tr("Перезаписать существующие данные по датам", "Replace existing date-based data")}</strong>
          <small>
            {tr(
              "Если в системе уже есть данные за те же даты, они будут заменены",
              "If the system already has data for the same dates, it will be replaced"
            )}
          </small>
        </span>
        <input
          type="checkbox"
          checked={form.replace_existing}
          disabled={!canManage || busy}
          onChange={(event) => onChange({ replace_existing: event.target.checked })}
        />
      </div>

      <div className="finance-pipeline-form-actions">
        <label className="button-like">
          <input
            type="file"
            accept=".csv,text/csv"
            disabled={!canManage || busy}
            style={{ display: "none" }}
            onChange={(event) => {
              const file = event.target.files?.[0];
              if (file) onPickFile(file);
              event.currentTarget.value = "";
            }}
          />
          {busy ? tr("Импортируем...", "Importing...") : tr("Импортировать CSV", "Import CSV")}
        </label>
      </div>

      {error ? <p className="workflow-inline-error">{error}</p> : null}

      {!imports.length && (
        <div className="finance-pipeline-empty card">
          <strong>{tr("Импортов пока нет", "No imports yet")}</strong>
          <p>
            {tr(
              "Загрузите CSV-файл из Kickstarter, чтобы автоматически заполнить доходы проекта",
              "Upload a Kickstarter CSV file to automatically fill project income"
            )}
          </p>
        </div>
      )}

      {!!imports.length && (
        <div className="finance-snapshot-list">
          {imports.map((item) => (
            <button
              key={item.id}
              type="button"
              className={`finance-snapshot-row ${selectedImport?.run.id === item.id ? "is-active" : ""}`.trim()}
              onClick={() => onOpen(item.id)}
            >
              <div className="finance-snapshot-row-main">
                <strong>{item.file_name}</strong>
                <p className="subtle">
                  {new Date(item.created_at).toLocaleString(locale === "en" ? "en-US" : "ru-RU")} · {item.uploaded_by_name}
                </p>
              </div>
              <div className="finance-snapshot-row-metrics">
                <div>
                  <span>{tr("Статус", "Status")}</span>
                  <strong>{importStatusLabel(item.status, locale)}</strong>
                </div>
                <div>
                  <span>{tr("Строки", "Rows")}</span>
                  <strong>{item.rows_succeeded}/{item.rows_processed}</strong>
                </div>
                <div>
                  <span>{tr("Всего собрано", "Total pledged")}</span>
                  <strong>{item.total_pledged != null ? money(item.total_pledged, currencyCode) : "—"}</strong>
                </div>
              </div>
            </button>
          ))}
        </div>
      )}

      {selectedImport && (
        <div className="finance-snapshot-detail">
          <div className="finance-pipeline-kpis">
            <article className="card finance-pipeline-kpi">
              <span>{tr("Обработано строк", "Rows processed")}</span>
              <strong>{selectedImport.run.rows_processed}</strong>
              <p>{tr("Успешно / с ошибкой / пропущено", "Success / failed / skipped")}: {selectedImport.run.rows_succeeded} / {selectedImport.run.rows_failed} / {selectedImport.run.rows_skipped}</p>
            </article>
            <article className="card finance-pipeline-kpi">
              <span>{tr("Бэкеры", "Backers")}</span>
              <strong>{selectedImport.run.backer_count ?? "—"}</strong>
              <p>{tr("Уникальные бэкеры, если CSV содержит идентификаторы", "Distinct backers if the CSV provides identifiers")}</p>
            </article>
            <article className="card finance-pipeline-kpi">
              <span>{tr("Уровни наград", "Reward tiers")}</span>
              <strong>{selectedImport.run.reward_tiers_count ?? "—"}</strong>
              <p>{tr("Уникальные уровни наград, найденные в CSV", "Distinct tiers detected in the CSV")}</p>
            </article>
            <article className="card finance-pipeline-kpi finance-pipeline-kpi-pool">
              <span>{tr("Импортированная сумма сборов", "Imported pledged total")}</span>
              <strong>{selectedImport.run.total_pledged != null ? money(selectedImport.run.total_pledged, currencyCode) : "—"}</strong>
              <p>{tr("Чистая сумма сборов, записанная в даты доходов проекта", "Net pledge total written into project income dates")}</p>
            </article>
          </div>

          {selectedImport.run.error_message ? (
            <p className="workflow-inline-error">{selectedImport.run.error_message}</p>
          ) : null}

          {!!selectedImport.records.length && (
            <div className="card">
              <strong>{tr("Записи импорта", "Import records")}</strong>
              <div className="finance-snapshot-list">
                {selectedImport.records.slice(0, 12).map((record) => (
                  <div key={record.id} className="finance-snapshot-row">
                    <div className="finance-snapshot-row-main">
                      <strong>{tr("Строка", "Row")} #{record.row_number}</strong>
                      <p className="subtle">
                        {record.pledged_at ? new Date(record.pledged_at).toLocaleDateString(locale === "en" ? "en-US" : "ru-RU") : tr("Дата не сопоставлена", "No mapped date")}
                      </p>
                    </div>
                    <div className="finance-snapshot-row-metrics">
                      <div>
                        <span>{tr("Статус", "Status")}</span>
                        <strong>{importRecordStatusLabel(record.status, locale)}</strong>
                      </div>
                      <div>
                        <span>{tr("Сумма", "Amount")}</span>
                        <strong>{record.pledged_amount != null ? money(record.pledged_amount, currencyCode) : "—"}</strong>
                      </div>
                      <div>
                        <span>{tr("Награда", "Reward")}</span>
                        <strong>{record.reward_tier || "—"}</strong>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}
    </section>
  );
}
