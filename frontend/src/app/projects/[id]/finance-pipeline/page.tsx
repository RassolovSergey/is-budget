"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";

import { AppShell } from "../../../../components/AppShell";
import { useConfirm } from "../../../../components/ConfirmProvider";
import { useLocale } from "../../../../components/LocaleProvider";
import { ProjectNav } from "../../../../components/ProjectNav";
import { useToast } from "../../../../components/ToastProvider";
import { ApiError, apiRequest } from "../../../../lib/api";
import type {
  AllocationBlock,
  AllocationPreview,
  PayoutSnapshotDetail,
  PayoutSnapshotSummary,
  Project,
  TeamDistributionMode,
  UUID
} from "../../../../lib/types";
import { useAuthGuard } from "../../../../lib/useAuthGuard";
import {
  FinancePipelineForm,
  FinancePipelineLevels,
  PayoutSnapshotsPanel,
  teamDistributionModeForBlock,
  type FinancePipelineFormState,
  initialFinancePipelineForm
} from "./FinancePipelineSections";

type ReorderPayloadItem = {
  id: UUID;
  level_index: number;
  order_index: number;
};

type SnapshotSettlementFormState = {
  settlement_status: "draft" | "approved" | "ready_for_settlement";
  settlement_note: string;
};

function sortBlocks(items: AllocationBlock[]): AllocationBlock[] {
  return [...items].sort((a, b) => (a.level_index === b.level_index ? a.order_index - b.order_index : a.level_index - b.level_index));
}

function buildPlacementPayload(items: AllocationBlock[]): ReorderPayloadItem[] {
  const sorted = sortBlocks(items);
  const counters = new Map<number, number>();
  return sorted.map((item) => {
    const nextOrder = (counters.get(item.level_index) || 0) + 1;
    counters.set(item.level_index, nextOrder);
    return {
      id: item.id,
      level_index: item.level_index,
      order_index: nextOrder
    };
  });
}

function formatFinancePipelineError(err: unknown, locale: "ru" | "en"): string {
  const tr = (ru: string, en: string) => (locale === "en" ? en : ru);
  if (err instanceof ApiError) {
    if (err.status === 403) {
      return tr("Только PM может менять конфигурацию финансового конвейера.", "Only the PM can change the finance pipeline configuration.");
    }
    const mapped: Record<string, [string, string]> = {
      "allocation block not found": ["Блок финансового конвейера не найден.", "The allocation block was not found."],
      "allocation block type is invalid": ["Некорректный тип блока.", "The block type is invalid."],
      "allocation block value type is invalid": ["Некорректный тип значения блока.", "The block value type is invalid."],
      "allocation block applies_to is invalid": ["Некорректная база расчёта.", "The calculation base is invalid."],
      "allocation block value must be greater than or equal to zero": ["Значение блока должно быть 0 или больше.", "The block value must be 0 or greater."],
      "allocation block level_index must be greater than or equal to 1": ["Уровень блока должен быть 1 или больше.", "The block level must be 1 or greater."],
      "team distribution mode is invalid": ["Некорректный режим распределения команде.", "The team distribution mode is invalid."],
      "invalid allocation block order payload": ["Не удалось переставить блоки между уровнями. Обновите список и попробуйте снова.", "Failed to reorder blocks between levels. Refresh the list and try again."],
      "only one full remainder team distribution block is allowed per project": ["В проекте может быть только один блок командного распределения, забирающий весь остаток.", "Only one full remainder team distribution block is allowed per project."]
    };
    if (mapped[err.message]) {
      return tr(mapped[err.message][0], mapped[err.message][1]);
    }
  }
  return (err as Error)?.message || tr("Не удалось выполнить действие в финансовом конвейере.", "Failed to perform the finance pipeline action.");
}

function formatSnapshotError(err: unknown, locale: "ru" | "en"): string {
  const tr = (ru: string, en: string) => (locale === "en" ? en : ru);
  if (err instanceof ApiError) {
    if (err.status === 403) {
      return tr("У вас нет доступа к этому действию со снимком.", "You do not have access to this snapshot action.");
    }
    if (err.status === 404) {
      return tr("Снимок не найден.", "The snapshot was not found.");
    }
    if (err.code === "invalid_settlement_transition") {
      return tr("Нельзя перевести снимок в этот статус напрямую. Сначала подтвердите его.", "This snapshot cannot move to that status directly. Approve it first.");
    }
    if (err.code === "invalid_settlement_status") {
      return tr("Некорректный статус подготовки выплаты для снимка.", "The settlement status is invalid.");
    }
  }
  return (err as Error)?.message || tr("Не удалось выполнить действие со снимком.", "Failed to perform the snapshot action.");
}

export default function FinancePipelinePage() {
  const { id } = useParams<{ id: string }>();
  const confirmAction = useConfirm();
  const { locale } = useLocale();
  const { token, ready } = useAuthGuard();
  const { showToast } = useToast();

  const [project, setProject] = useState<Project | null>(null);
  const [blocks, setBlocks] = useState<AllocationBlock[]>([]);
  const [preview, setPreview] = useState<AllocationPreview | null>(null);
  const [snapshots, setSnapshots] = useState<PayoutSnapshotSummary[]>([]);
  const [selectedSnapshot, setSelectedSnapshot] = useState<PayoutSnapshotDetail | null>(null);
  const [form, setForm] = useState<FinancePipelineFormState>(initialFinancePipelineForm);
  const [isFormOpen, setIsFormOpen] = useState(false);
  const [loading, setLoading] = useState(true);
  const [busyKey, setBusyKey] = useState("");
  const [error, setError] = useState("");
  const [formError, setFormError] = useState("");
  const [notice, setNotice] = useState("");
  const [previewGrossInput, setPreviewGrossInput] = useState("");
  const [settlementForm, setSettlementForm] = useState<SnapshotSettlementFormState>({
    settlement_status: "draft",
    settlement_note: ""
  });

  const tr = (ru: string, en: string) => (locale === "en" ? en : ru);
  const appLocale = locale === "en" ? "en" : "ru";

  const canManage = project?.current_user_role === "PM";

  const parsedGrossAmount = useMemo(() => {
    const normalized = previewGrossInput.trim();
    if (!normalized) return null;
    const value = Number(normalized);
    if (!Number.isFinite(value) || value < 0) return Number.NaN;
    return value;
  }, [previewGrossInput]);

  async function load(options?: { grossOverride?: number | null; preserveGrossInput?: boolean }) {
    if (!token) return;
    try {
      setLoading(true);
      setError("");

      const previewBody = options?.grossOverride != null ? { gross_amount: options.grossOverride } : {};
      const [projectResp, blocksResp, previewResp, snapshotsResp] = await Promise.all([
        apiRequest<Project>(`/projects/${id}`, { token }),
        apiRequest<AllocationBlock[]>(`/projects/${id}/allocation-blocks`, { token }),
        apiRequest<AllocationPreview>(`/projects/${id}/allocation-preview`, {
          method: "POST",
          token,
          body: previewBody
        }),
        apiRequest<PayoutSnapshotSummary[]>(`/projects/${id}/payout-snapshots`, { token })
      ]);

      setProject(projectResp);
      setBlocks(sortBlocks(blocksResp));
      setPreview(previewResp);
      setSnapshots(snapshotsResp);
      if (selectedSnapshot && !snapshotsResp.find((item) => item.id === selectedSnapshot.id)) {
        setSelectedSnapshot(null);
      }
      if (!options?.preserveGrossInput && !previewGrossInput.trim()) {
        setPreviewGrossInput(String(previewResp.gross_amount));
      }
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    if (ready) {
      void load();
    }
  }, [ready, id, token]);

  useEffect(() => {
    if (!selectedSnapshot) return;
    setSettlementForm({
      settlement_status: selectedSnapshot.settlement_status,
      settlement_note: selectedSnapshot.settlement_note || ""
    });
  }, [selectedSnapshot]);

  function resetForm() {
    setForm(initialFinancePipelineForm);
    setFormError("");
    setIsFormOpen(false);
  }

  function startEdit(block: AllocationBlock) {
    setFormError("");
    setIsFormOpen(true);
    setForm({
      id: block.id,
      name: block.name,
      type: block.type,
      level_index: String(block.level_index),
      enabled: block.enabled,
      value_type: block.value_type,
      value: String(block.value),
      applies_to: block.applies_to,
      team_distribution_mode: teamDistributionModeForBlock(block)
    });
  }

  async function submitForm() {
    if (!token || !canManage) return;

    const normalizedName = form.name.trim();
    if (!normalizedName) {
      setFormError(tr("Укажите название блока.", "Enter a block name."));
      return;
    }

    const parsedLevel = Number(form.level_index.trim());
    if (!Number.isInteger(parsedLevel) || parsedLevel < 1) {
      setFormError(tr("Укажите уровень 1 или больше.", "Enter a level 1 or greater."));
      return;
    }

    const isTeamDistribution = form.type === "team_distribution";
    const parsedValue = isTeamDistribution && form.team_distribution_mode === "full_remainder" ? 0 : Number(form.value);
    if (!isTeamDistribution || form.team_distribution_mode === "percent") {
      if (!Number.isFinite(parsedValue) || parsedValue < 0) {
        setFormError(tr("Значение блока должно быть числом 0 или больше.", "The block value must be a number 0 or greater."));
        return;
      }
    }

    const body = {
      name: normalizedName,
      type: form.type,
      level_index: parsedLevel,
      enabled: form.enabled,
      value_type: isTeamDistribution ? (form.team_distribution_mode === "full_remainder" ? "fixed" : "percent") : form.value_type,
      value: parsedValue,
      applies_to: isTeamDistribution ? "current_balance" : form.applies_to,
      team_distribution_mode: isTeamDistribution ? form.team_distribution_mode : undefined
    };

    try {
      setBusyKey(form.id ? `save:${form.id}` : "create");
      setFormError("");
      setNotice("");

      if (form.id) {
        await apiRequest(`/projects/${id}/allocation-blocks/${form.id}`, {
          method: "PUT",
          token,
          body
        });
      } else {
        await apiRequest(`/projects/${id}/allocation-blocks`, {
          method: "POST",
          token,
          body
        });
      }

      setNotice(form.id ? tr("Блок обновлён.", "Block updated.") : tr("Блок добавлен.", "Block added."));
      resetForm();
      const grossOverride = parsedGrossAmount != null && !Number.isNaN(parsedGrossAmount) ? parsedGrossAmount : null;
      await load({ grossOverride, preserveGrossInput: true });
    } catch (err) {
      setFormError(formatFinancePipelineError(err, appLocale));
    } finally {
      setBusyKey("");
    }
  }

  async function deleteBlock(block: AllocationBlock) {
    if (!token || !canManage) return;
    const confirmed = await confirmAction({
      title: tr("Удалить блок потока?", "Delete flow block?"),
      message: tr(`Блок «${block.name}» будет удалён из финансового конвейера.`, `Block "${block.name}" will be removed from the finance pipeline.`),
      confirmText: tr("Удалить", "Delete"),
      tone: "danger"
    });
    if (!confirmed) return;

    try {
      setBusyKey(`delete:${block.id}`);
      setNotice("");
      await apiRequest(`/projects/${id}/allocation-blocks/${block.id}`, {
        method: "DELETE",
        token
      });
      setNotice(tr("Блок удалён.", "Block deleted."));
      if (form.id === block.id) {
        resetForm();
      }
      const grossOverride = parsedGrossAmount != null && !Number.isNaN(parsedGrossAmount) ? parsedGrossAmount : null;
      await load({ grossOverride, preserveGrossInput: true });
    } catch (err) {
      setError(formatFinancePipelineError(err, appLocale));
    } finally {
      setBusyKey("");
    }
  }

  async function persistReorder(nextBlocks: AllocationBlock[], successNotice: string) {
    if (!token || !canManage) return;
    try {
      setNotice("");
      const items = buildPlacementPayload(nextBlocks);
      await apiRequest(`/projects/${id}/allocation-blocks/reorder`, {
        method: "PUT",
        token,
        body: { items }
      });
      setNotice(successNotice);
      const grossOverride = parsedGrossAmount != null && !Number.isNaN(parsedGrossAmount) ? parsedGrossAmount : null;
      await load({ grossOverride, preserveGrossInput: true });
    } catch (err) {
      setError(formatFinancePipelineError(err, appLocale));
    } finally {
      setBusyKey("");
    }
  }

  async function moveBlockAcrossLevels(block: AllocationBlock, direction: "prev" | "next") {
    if (!token || !canManage) return;
    const targetLevel = direction === "prev" ? Math.max(1, block.level_index - 1) : block.level_index + 1;
    if (targetLevel === block.level_index) return;

    const maxOrderInTarget = blocks
      .filter((item) => item.level_index === targetLevel)
      .reduce((max, item) => Math.max(max, item.order_index), 0);

    const nextBlocks = blocks.map((item) =>
      item.id === block.id
        ? {
            ...item,
            level_index: targetLevel,
            order_index: maxOrderInTarget + 1
          }
        : item
    );

    setBusyKey(`level:${block.id}`);
    await persistReorder(nextBlocks, "");
    showToast({
      message: tr("Блок перемещён между уровнями.", "Block moved between levels."),
      tone: "success"
    });
  }

  async function recalculatePreview() {
    if (parsedGrossAmount != null && Number.isNaN(parsedGrossAmount)) {
      setError(tr("Валовой доход для предпросмотра должен быть числом 0 или больше.", "Preview gross amount must be a number 0 or greater."));
      return;
    }

    try {
      setBusyKey("preview");
      setError("");
      setNotice("");
      await load({
        grossOverride: parsedGrossAmount != null ? parsedGrossAmount : null,
        preserveGrossInput: true
      });
    } finally {
      setBusyKey("");
    }
  }

  async function createSnapshot() {
    if (!token || !canManage) return;
    if (parsedGrossAmount != null && Number.isNaN(parsedGrossAmount)) {
      setError(tr("Валовой доход для снимка должен быть числом 0 или больше.", "Snapshot gross amount must be a number 0 or greater."));
      return;
    }

    try {
      setBusyKey("snapshot:create");
      setError("");
      setNotice("");
      const created = await apiRequest<PayoutSnapshotDetail>(`/projects/${id}/payout-snapshots`, {
        method: "POST",
        token,
        body: parsedGrossAmount != null ? { gross_amount: parsedGrossAmount } : {}
      });
      setSelectedSnapshot(created);
      setNotice(tr("Снимок сохранён.", "Snapshot saved."));
      await load({
        grossOverride: parsedGrossAmount != null ? parsedGrossAmount : null,
        preserveGrossInput: true
      });
    } catch (err) {
      setError(formatSnapshotError(err, appLocale));
    } finally {
      setBusyKey("");
    }
  }

  async function openSnapshot(snapshotID: string) {
    if (!token) return;
    try {
      setBusyKey(`snapshot:${snapshotID}`);
      setError("");
      const detail = await apiRequest<PayoutSnapshotDetail>(`/payout-snapshots/${snapshotID}`, { token });
      setSelectedSnapshot(detail);
    } catch (err) {
      setError(formatSnapshotError(err, appLocale));
    } finally {
      setBusyKey("");
    }
  }

  async function saveSnapshotSettlement() {
    if (!token || !canManage || !selectedSnapshot) return;
    try {
      setBusyKey("snapshot:settlement");
      setError("");
      setNotice("");
      const updated = await apiRequest<PayoutSnapshotDetail>(`/payout-snapshots/${selectedSnapshot.id}/settlement`, {
        method: "PUT",
        token,
        body: {
          settlement_status: settlementForm.settlement_status,
          settlement_note: settlementForm.settlement_note.trim() || undefined
        }
      });
      setSelectedSnapshot(updated);
      setNotice(
        settlementForm.settlement_status === "ready_for_settlement"
          ? tr("Снимок отмечен как готовый к выплате.", "Snapshot marked as ready for settlement.")
          : tr("Статус снимка обновлён.", "Snapshot status updated.")
      );
      const grossOverride = parsedGrossAmount != null && !Number.isNaN(parsedGrossAmount) ? parsedGrossAmount : null;
      await load({ grossOverride, preserveGrossInput: true });
    } catch (err) {
      setError(formatSnapshotError(err, appLocale));
    } finally {
      setBusyKey("");
    }
  }

  if (!ready || loading || !project) return null;

  return (
    <AppShell title={project.name}>
      <ProjectNav projectId={id} />

      <section className="card workflow-hero-card finance-pipeline-hero-card">
        <div>
          <span className="shell-eyebrow">{tr("Финансовый конвейер", "Finance Pipeline")}</span>
          <h1 className="title">{tr("Финансовый конвейер", "Finance Pipeline")}</h1>
          <p>
            {tr(
              "Настройте, как деньги проходят через налоги, расходы и распределение",
              "Set up how money flows through taxes, expenses, and distributions"
            )}
          </p>
          {!canManage && (
            <p className="workflow-inline-note">
              {tr(
                "У вас доступ только на чтение: менять конфигурацию конвейера может только PM.",
                "You have read-only access: only the PM can change the pipeline."
              )}
            </p>
          )}
        </div>
      </section>

      {error ? <p className="workflow-inline-error">{error}</p> : null}
      {notice ? <p className="workflow-inline-success">{notice}</p> : null}

      <section className="card finance-ops-flow-card">
        <div>
          <span className="shell-eyebrow">{tr("Операционный поток", "Operations flow")}</span>
          <h2 className="title">{tr("Что делать дальше", "What to do next")}</h2>
        </div>
        <div className="finance-ops-flow-grid">
          <Link href={`/projects/${id}/campaign-lifecycle`} className="finance-ops-step">
            <strong>{tr("1. Обновить доходы", "1. Refresh income")}</strong>
            <p>{tr("Импортируйте CSV Kickstarter и проверьте жизненный цикл кампании.", "Import the Kickstarter CSV and review the campaign lifecycle.")}</p>
          </Link>
          <Link href={`/projects/${id}/finance-pipeline`} className="finance-ops-step">
            <strong>{tr("2. Проверить денежный поток", "2. Review money flow")}</strong>
            <p>{tr("Настройте уровни конвейера и проверьте командный пул.", "Adjust pipeline levels and verify the team pool.")}</p>
          </Link>
          <Link href={`/projects/${id}/my-income`} className="finance-ops-step">
            <strong>{tr("3. Сверить прогнозы", "3. Review forecasts")}</strong>
            <p>{tr("Посмотрите личные и командные прогнозы выплат после изменений.", "Review personal and team payout forecasts after updates.")}</p>
          </Link>
        </div>
      </section>

      <section className="card finance-pipeline-controls-card">
        <div className="finance-pipeline-preview-toolbar">
          <label>
            <span>{tr("Валовой доход для предпросмотра", "Preview gross amount")}</span>
            <input type="number" min="0" step="0.01" value={previewGrossInput} onChange={(event) => setPreviewGrossInput(event.target.value)} />
          </label>
          <button type="button" disabled={busyKey === "preview"} onClick={() => void recalculatePreview()}>
            {busyKey === "preview" ? tr("Считаем...", "Calculating...") : tr("Обновить предпросмотр", "Refresh preview")}
          </button>
        </div>

        {canManage && (
          <button
            type="button"
            className="finance-pipeline-add-button"
            onClick={() => {
              setForm(initialFinancePipelineForm);
              setFormError("");
              setIsFormOpen(true);
            }}
          >
            {tr("Добавить блок", "Add block")}
          </button>
        )}
      </section>

      <FinancePipelineLevels
        locale={appLocale}
        currencyCode={project.currency_code}
        blocks={blocks}
        preview={preview}
        canManage={Boolean(canManage)}
        busyKey={busyKey}
        onEdit={startEdit}
        onDelete={(block) => void deleteBlock(block)}
        onMoveAcrossLevels={(block, direction) => void moveBlockAcrossLevels(block, direction)}
      />

      <PayoutSnapshotsPanel
        locale={appLocale}
        currencyCode={project.currency_code}
        canManage={Boolean(canManage)}
        creating={busyKey === "snapshot:create"}
        snapshots={snapshots}
        selectedSnapshot={selectedSnapshot}
        settlementForm={settlementForm}
        settlementBusy={busyKey === "snapshot:settlement"}
        onChangeSettlement={setSettlementForm}
        onSaveSettlement={() => void saveSnapshotSettlement()}
        onCreate={() => void createSnapshot()}
        onOpen={(snapshotID) => void openSnapshot(snapshotID)}
      />

      {isFormOpen ? (
        <div className="confirm-overlay" role="presentation" onClick={resetForm}>
          <section className="card finance-pipeline-modal" role="dialog" aria-modal="true" onClick={(event) => event.stopPropagation()}>
            <div className="finance-pipeline-modal-head">
              <h2 className="title" style={{ marginBottom: 0 }}>
                {form.id ? tr("Редактировать блок", "Edit block") : tr("Добавить блок", "Add block")}
              </h2>
              <button type="button" className="secondary" onClick={resetForm}>
                {tr("Закрыть", "Close")}
              </button>
            </div>
            <FinancePipelineForm
              locale={appLocale}
              canManage={Boolean(canManage)}
              form={form}
              busy={busyKey !== ""}
              error={formError}
              onChange={setForm}
              onSubmit={() => void submitForm()}
              onCancel={resetForm}
            />
          </section>
        </div>
      ) : null}
    </AppShell>
  );
}
