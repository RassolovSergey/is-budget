"use client";

import { useEffect, useState } from "react";

import { displayHumanName } from "../../../../lib/displayName";
import type {
  ProjectMemberWithUser,
  UUID,
  WorkflowContributionUserSummary,
  WorkflowNode,
  WorkflowNodeStatus
} from "../../../../lib/types";

export type WorkflowNodeView = WorkflowNode & {
  assigneeName: string;
  parentIds: UUID[];
  parentTitles: string[];
  parentCount: number;
  isHighlighted: boolean;
  canTake: boolean;
  canComplete: boolean;
  canUnassign: boolean;
  isOwnedByMe: boolean;
  isCompletedByMe: boolean;
};

export type WorkflowBoardFilter = "all" | "my" | "ready" | "planned" | "done";

export type WorkflowCreateNodeInput = {
  title: string;
  description: string;
  contribution_points: number;
  parent_node_ids: UUID[];
};

export type WorkflowUpdateNodeInput = {
  title: string;
  description: string;
  contribution_points: number;
};

const statusOrder: WorkflowNodeStatus[] = ["ready", "in_progress", "planned", "done"];

function statusLabel(status: WorkflowNodeStatus, locale: "ru" | "en"): string {
  const map: Record<WorkflowNodeStatus, [string, string]> = {
    ready: ["Готово к старту", "Ready"],
    in_progress: ["В работе", "In progress"],
    planned: ["Заблокировано", "Planned"],
    done: ["Завершено", "Done"]
  };
  return locale === "en" ? map[status][1] : map[status][0];
}

function statusDescription(status: WorkflowNodeStatus, locale: "ru" | "en"): string {
  const map: Record<WorkflowNodeStatus, [string, string]> = {
    ready: ["Можно брать в работу прямо сейчас.", "This node can be picked up now."],
    in_progress: ["Нода уже закреплена за участником.", "This node is already assigned."],
    planned: ["Нода разблокируется после завершения родителей.", "This node unlocks after parent nodes are done."],
    done: ["Вклад уже зафиксирован в журнале.", "Contribution is already recorded in the ledger."]
  };
  return locale === "en" ? map[status][1] : map[status][0];
}

function emptyMessage(status: WorkflowNodeStatus, locale: "ru" | "en"): string {
  const map: Record<WorkflowNodeStatus, [string, string]> = {
    ready: ["Сейчас нет доступных нод.", "No nodes are ready right now."],
    in_progress: ["Никто ничего не взял в работу.", "Nothing is currently in progress."],
    planned: ["Нет заблокированных нод.", "No blocked nodes right now."],
    done: ["Завершённых нод пока нет.", "No completed nodes yet."]
  };
  return locale === "en" ? map[status][1] : map[status][0];
}

function formatDateTime(value: string | null | undefined, locale: "ru" | "en"): string {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "—";
  return date.toLocaleString(locale === "en" ? "en-US" : "ru-RU", {
    day: "2-digit",
    month: "short",
    hour: "2-digit",
    minute: "2-digit"
  });
}

function filterLabel(filter: WorkflowBoardFilter, locale: "ru" | "en"): string {
  const map: Record<WorkflowBoardFilter, [string, string]> = {
    all: ["Все", "All"],
    my: ["Мои задачи", "My tasks"],
    ready: ["Готово", "Ready"],
    planned: ["Заблокировано", "Blocked"],
    done: ["Завершено", "Done"]
  };
  return locale === "en" ? map[filter][1] : map[filter][0];
}

function filterDescription(filter: WorkflowBoardFilter, locale: "ru" | "en"): string {
  const map: Record<WorkflowBoardFilter, [string, string]> = {
    all: ["Все задачи проекта по статусам.", "All project tasks grouped by status."],
    my: ["Ноды, закреплённые за вами или завершённые вами.", "Nodes assigned to you or completed by you."],
    ready: ["Ноды, которые можно взять прямо сейчас.", "Nodes that can be taken right now."],
    planned: ["Заблокированные ноды, ожидающие родителей.", "Blocked nodes waiting on parent completion."],
    done: ["Уже завершённые ноды с зафиксированным вкладом.", "Completed nodes with recorded contribution."]
  };
  return locale === "en" ? map[filter][1] : map[filter][0];
}

function nodeGuidance(node: WorkflowNodeView, locale: "ru" | "en"): string {
  if (node.status === "planned") {
    return locale === "en"
      ? "Blocked until all parent nodes are done."
      : "Заблокирована, пока не будут завершены все родительские ноды.";
  }
  if (node.status === "ready") {
    return locale === "en"
      ? "Ready to be taken into work."
      : "Готова к работе прямо сейчас.";
  }
  if (node.status === "in_progress") {
    if (node.isOwnedByMe) {
      return locale === "en"
        ? "Assigned to you. Complete it when the result is ready."
        : "Нода закреплена за вами. Завершите её, когда результат готов.";
    }
    return locale === "en"
      ? "Currently owned by another teammate."
      : "Сейчас нода находится в работе у другого участника.";
  }
  return locale === "en"
    ? "Completed and recorded in the contribution ledger."
    : "Завершена и записана в журнал вкладов.";
}

export function WorkflowViewControls({
  locale,
  activeFilter,
  counts,
  busy,
  onChange
}: {
  locale: "ru" | "en";
  activeFilter: WorkflowBoardFilter;
  counts: Record<WorkflowBoardFilter, number>;
  busy: boolean;
  onChange: (filter: WorkflowBoardFilter) => void;
}) {
  const filters: WorkflowBoardFilter[] = ["all", "my", "ready", "planned", "done"];

  return (
    <section className="card workflow-view-card">
      <div>
        <span className="shell-eyebrow">{locale === "en" ? "View" : "Вид"}</span>
        <h2 className="title">{filterLabel(activeFilter, locale)}</h2>
        <p className="subtle">{filterDescription(activeFilter, locale)}</p>
      </div>

      <div className="workflow-filter-row" role="tablist" aria-label={locale === "en" ? "Contribution views" : "Виды вклада команды"}>
        {filters.map((filter) => (
          <button
            key={filter}
            type="button"
            className={`workflow-filter-pill ${activeFilter === filter ? "is-active" : ""}`.trim()}
            disabled={busy}
            onClick={() => onChange(filter)}
          >
            <span>{filterLabel(filter, locale)}</span>
            <strong>{counts[filter]}</strong>
          </button>
        ))}
      </div>
    </section>
  );
}

export function WorkflowOverview({
  totalNodes,
  readyNodes,
  inProgressNodes,
  doneNodes,
  totalProjectPoints,
  completedPoints,
  completionPercentage,
  locale
}: {
  totalNodes: number;
  readyNodes: number;
  inProgressNodes: number;
  doneNodes: number;
  totalProjectPoints: number;
  completedPoints: number;
  completionPercentage: number;
  locale: "ru" | "en";
}) {
  const tr = (ru: string, en: string) => (locale === "en" ? en : ru);

  const items = [
    { label: tr("Всего нод", "Total nodes"), value: totalNodes },
    { label: tr("Готово", "Ready"), value: readyNodes },
    { label: tr("В работе", "In progress"), value: inProgressNodes },
    { label: tr("Завершено", "Done"), value: doneNodes },
    {
      label: tr("Очки проекта", "Project points"),
      value: tr(
        `Всего: ${totalProjectPoints} очков`,
        `Total: ${totalProjectPoints} pts`
      )
    },
    {
      label: tr("Прогресс по очкам", "Points completion"),
      value: tr(
        `Выполнено: ${completedPoints} / ${totalProjectPoints} (${completionPercentage.toFixed(0)}%)`,
        `Completed: ${completedPoints} / ${totalProjectPoints} (${completionPercentage.toFixed(0)}%)`
      )
    }
  ];

  return (
    <section className="workflow-kpis">
      {items.map((item) => (
        <article key={item.label} className="card workflow-kpi-card">
          <span>{item.label}</span>
          <strong>{item.value}</strong>
        </article>
      ))}
    </section>
  );
}

export function WorkflowManagePanel({
  locale,
  nodes,
  busy,
  onCreate
}: {
  locale: "ru" | "en";
  nodes: WorkflowNode[];
  busy: boolean;
  onCreate: (input: WorkflowCreateNodeInput) => Promise<boolean>;
}) {
  const tr = (ru: string, en: string) => (locale === "en" ? en : ru);
  const [isOpen, setIsOpen] = useState(false);
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [points, setPoints] = useState("1");
  const [parentNodeIDs, setParentNodeIDs] = useState<UUID[]>([]);
  const [localError, setLocalError] = useState("");

  function toggleParent(parentNodeID: UUID) {
    setParentNodeIDs((current) =>
      current.includes(parentNodeID)
        ? current.filter((value) => value !== parentNodeID)
        : [...current, parentNodeID]
    );
  }

  async function submit() {
    const normalizedTitle = title.trim();
    if (!normalizedTitle) {
      setLocalError(tr("Укажите название ноды.", "Enter a node title."));
      return;
    }

    const parsedPoints = Number(points);
    if (!Number.isFinite(parsedPoints) || parsedPoints < 0) {
      setLocalError(tr("Очки вклада должны быть числом 0 или больше.", "Contribution points must be a number 0 or greater."));
      return;
    }

    setLocalError("");
    const created = await onCreate({
      title: normalizedTitle,
      description: description.trim(),
      contribution_points: parsedPoints,
      parent_node_ids: parentNodeIDs
    });
    if (created) {
      setTitle("");
      setDescription("");
      setPoints("1");
      setParentNodeIDs([]);
    }
  }

  return (
    <section className="card workflow-manage-card">
      <details className="workflow-details" open={isOpen} onToggle={(event) => setIsOpen((event.target as HTMLDetailsElement).open)}>
        <summary className="workflow-details-summary">
          <div>
            <span className="shell-eyebrow">{tr("Управление", "Management")}</span>
          <h2 className="title">{tr("Создать ноду", "Create workflow node")}</h2>
          <p className="subtle">
            {tr(
                "Компактная панель для добавления новых задач и базовых зависимостей.",
                "Compact panel for adding new tasks and initial dependencies."
              )}
          </p>
          </div>
          <div className="workflow-details-meta">
            <span className="workflow-parent-chip">
              {tr("Доступно родительских нод", "Available parent nodes")}: {nodes.length}
            </span>
            <span className="workflow-parent-chip">
              {tr("Выбрано", "Selected")}: {parentNodeIDs.length}
            </span>
          </div>
        </summary>

        <div className="workflow-details-body">
          <div className="workflow-form-grid">
            <label className="workflow-field">
              <span>{tr("Название", "Title")}</span>
              <input
                value={title}
                onChange={(event) => setTitle(event.target.value)}
                placeholder={tr("Например, Prepare launch trailer", "For example, Prepare launch trailer")}
              />
            </label>

            <label className="workflow-field workflow-field-small">
              <span>{tr("Очки вклада", "Contribution points")}</span>
              <input
                type="number"
                min="0"
                step="1"
                value={points}
                onChange={(event) => setPoints(event.target.value)}
              />
            </label>

            <label className="workflow-field workflow-field-full">
              <span>{tr("Описание", "Description")}</span>
              <textarea
                rows={3}
                value={description}
                onChange={(event) => setDescription(event.target.value)}
                placeholder={tr("Коротко опишите результат этой ноды.", "Briefly describe the outcome of this node.")}
              />
            </label>
          </div>

          <div className="workflow-selector-card">
            <div className="workflow-selector-head">
              <strong>{tr("Родительские ноды", "Parent nodes")}</strong>
              <span>{parentNodeIDs.length}</span>
            </div>
            {nodes.length === 0 ? (
              <p className="workflow-inline-note">{tr("Сначала можно создать корневую ноду без зависимостей.", "You can start with a root node without dependencies.")}</p>
            ) : (
              <div className="workflow-checkbox-list">
                {nodes.map((node) => (
                  <label key={node.id} className="workflow-checkbox-row">
                    <input
                      type="checkbox"
                      checked={parentNodeIDs.includes(node.id)}
                      onChange={() => toggleParent(node.id)}
                    />
                    <span>
                      <strong>{node.title}</strong>
                      <small>{statusLabel(node.status, locale)}</small>
                    </span>
                  </label>
                ))}
              </div>
            )}
          </div>

          {localError ? <p className="workflow-inline-error">{localError}</p> : null}

          <div className="workflow-manage-actions">
            <button type="button" disabled={busy} onClick={() => void submit()}>
              {busy ? tr("Создаём...", "Creating...") : tr("Создать ноду", "Create node")}
            </button>
          </div>
        </div>
      </details>
    </section>
  );
}

export function WorkflowBoard({
  nodesByStatus,
  allNodes,
  members,
  locale,
  busyKey,
  canManage,
  onTake,
  onUnassign,
  onAssign,
  onComplete,
  onUpdate,
  onAddDependencies,
  onRemoveDependency,
  hideEmptySections = false,
  emptyStateMessage
}: {
  nodesByStatus: Record<WorkflowNodeStatus, WorkflowNodeView[]>;
  allNodes: WorkflowNode[];
  members: ProjectMemberWithUser[];
  locale: "ru" | "en";
  busyKey: string;
  canManage: boolean;
  onTake: (nodeId: UUID) => void;
  onUnassign: (nodeId: UUID) => void;
  onAssign: (nodeId: UUID, userId: UUID) => Promise<boolean>;
  onComplete: (nodeId: UUID) => void;
  onUpdate: (nodeId: UUID, input: WorkflowUpdateNodeInput) => Promise<boolean>;
  onAddDependencies: (nodeId: UUID, parentNodeIDs: UUID[]) => Promise<boolean>;
  onRemoveDependency: (nodeId: UUID, parentNodeID: UUID) => Promise<boolean>;
  hideEmptySections?: boolean;
  emptyStateMessage?: string;
}) {
  const visibleStatuses = hideEmptySections
    ? statusOrder.filter((status) => nodesByStatus[status].length > 0)
    : statusOrder;

  if (visibleStatuses.length === 0) {
    return (
      <section className="card workflow-status-card">
        <div className="workflow-empty-state">
          <p>{emptyStateMessage || (locale === "en" ? "No nodes match this view yet." : "Для этого вида пока нет нод.")}</p>
        </div>
      </section>
    );
  }

  return (
    <div className="workflow-sections">
      {visibleStatuses.map((status) => (
        <WorkflowStatusSection
          key={status}
          status={status}
          locale={locale}
          nodes={nodesByStatus[status]}
          allNodes={allNodes}
          members={members}
          busyKey={busyKey}
          canManage={canManage}
          onTake={onTake}
          onUnassign={onUnassign}
          onAssign={onAssign}
          onComplete={onComplete}
          onUpdate={onUpdate}
          onAddDependencies={onAddDependencies}
          onRemoveDependency={onRemoveDependency}
        />
      ))}
    </div>
  );
}

function WorkflowStatusSection({
  status,
  nodes,
  allNodes,
  members,
  locale,
  busyKey,
  canManage,
  onTake,
  onUnassign,
  onAssign,
  onComplete,
  onUpdate,
  onAddDependencies,
  onRemoveDependency
}: {
  status: WorkflowNodeStatus;
  nodes: WorkflowNodeView[];
  allNodes: WorkflowNode[];
  members: ProjectMemberWithUser[];
  locale: "ru" | "en";
  busyKey: string;
  canManage: boolean;
  onTake: (nodeId: UUID) => void;
  onUnassign: (nodeId: UUID) => void;
  onAssign: (nodeId: UUID, userId: UUID) => Promise<boolean>;
  onComplete: (nodeId: UUID) => void;
  onUpdate: (nodeId: UUID, input: WorkflowUpdateNodeInput) => Promise<boolean>;
  onAddDependencies: (nodeId: UUID, parentNodeIDs: UUID[]) => Promise<boolean>;
  onRemoveDependency: (nodeId: UUID, parentNodeID: UUID) => Promise<boolean>;
}) {
  const tr = (ru: string, en: string) => (locale === "en" ? en : ru);

  return (
    <section className="card workflow-status-card">
      <div className="workflow-status-head">
        <div>
          <span className={`workflow-status-pill workflow-status-${status}`}>{statusLabel(status, locale)}</span>
          <h2 className="title">{statusLabel(status, locale)}</h2>
          <p className="subtle">{statusDescription(status, locale)}</p>
        </div>
        <div className="workflow-status-count">
          <strong>{nodes.length}</strong>
          <span>{tr("нод", "nodes")}</span>
        </div>
      </div>

      {nodes.length === 0 ? (
        <div className="workflow-empty-state">
          <p>{emptyMessage(status, locale)}</p>
        </div>
      ) : (
        <div className="workflow-node-grid">
          {nodes.map((node) => (
            <WorkflowNodeCard
              key={node.id}
              node={node}
              allNodes={allNodes}
              members={members}
              locale={locale}
              busyKey={busyKey}
              canManage={canManage}
              onTake={onTake}
              onUnassign={onUnassign}
              onAssign={onAssign}
              onComplete={onComplete}
              onUpdate={onUpdate}
              onAddDependencies={onAddDependencies}
              onRemoveDependency={onRemoveDependency}
            />
          ))}
        </div>
      )}
    </section>
  );
}

function WorkflowNodeCard({
  node,
  allNodes,
  members,
  locale,
  busyKey,
  canManage,
  onTake,
  onUnassign,
  onAssign,
  onComplete,
  onUpdate,
  onAddDependencies,
  onRemoveDependency
}: {
  node: WorkflowNodeView;
  allNodes: WorkflowNode[];
  members: ProjectMemberWithUser[];
  locale: "ru" | "en";
  busyKey: string;
  canManage: boolean;
  onTake: (nodeId: UUID) => void;
  onUnassign: (nodeId: UUID) => void;
  onAssign: (nodeId: UUID, userId: UUID) => Promise<boolean>;
  onComplete: (nodeId: UUID) => void;
  onUpdate: (nodeId: UUID, input: WorkflowUpdateNodeInput) => Promise<boolean>;
  onAddDependencies: (nodeId: UUID, parentNodeIDs: UUID[]) => Promise<boolean>;
  onRemoveDependency: (nodeId: UUID, parentNodeID: UUID) => Promise<boolean>;
}) {
  const tr = (ru: string, en: string) => (locale === "en" ? en : ru);
  const [isAdminOpen, setIsAdminOpen] = useState(false);
  const [isEditing, setIsEditing] = useState(false);
  const [isDependencyOpen, setIsDependencyOpen] = useState(false);
  const [title, setTitle] = useState(node.title);
  const [description, setDescription] = useState(node.description);
  const [points, setPoints] = useState(String(node.contribution_points));
  const [selectedAssigneeID, setSelectedAssigneeID] = useState<UUID | "">(node.assignee_user_id || "");
  const [selectedParentIDs, setSelectedParentIDs] = useState<UUID[]>([]);
  const [localError, setLocalError] = useState("");

  useEffect(() => {
    setTitle(node.title);
    setDescription(node.description);
    setPoints(String(node.contribution_points));
    setSelectedAssigneeID(node.assignee_user_id || "");
    setSelectedParentIDs([]);
  }, [node.id, node.title, node.description, node.contribution_points, node.assignee_user_id]);

  useEffect(() => {
    if (!busyKey) {
      setLocalError("");
    }
  }, [busyKey]);

  const availableParents = allNodes.filter(
    (candidate) => candidate.id !== node.id && !node.parentIds.includes(candidate.id)
  );
  const canAdminAssign = canManage && (node.status === "ready" || node.status === "in_progress");
  const assignBusy = busyKey === `assign:${node.id}`;
  const unassignBusy = busyKey === `unassign:${node.id}`;
  const completeBusy = busyKey === `complete:${node.id}`;
  const updateBusy = busyKey === `update:${node.id}`;
  const dependencyBusy = busyKey === `dependency:${node.id}`;
  const existingParents = node.parentIds.map((parentId, index) => ({
    id: parentId,
    title: node.parentTitles[index] || parentId
  }));
  const takeDisabled = !node.canTake || assignBusy || unassignBusy;
  const completeDisabled = !node.canComplete || completeBusy;
  const statusToneLabel =
    node.status === "planned"
      ? tr("Заблокировано", "Blocked")
      : node.status === "ready"
        ? tr("Можно брать", "Ready to take")
        : node.status === "in_progress"
          ? tr("Нода в работе", "In progress")
          : tr("Вклад записан", "Contribution recorded");
  const adminSummary =
    existingParents.length > 0
      ? tr(`Зависимостей: ${existingParents.length}`, `Dependencies: ${existingParents.length}`)
      : tr("Без зависимостей", "No dependencies");

  function toggleParent(parentNodeID: UUID) {
    setSelectedParentIDs((current) =>
      current.includes(parentNodeID)
        ? current.filter((value) => value !== parentNodeID)
        : [...current, parentNodeID]
    );
  }

  async function submitUpdate() {
    const normalizedTitle = title.trim();
    if (!normalizedTitle) {
      setLocalError(tr("Название ноды не может быть пустым.", "Node title cannot be empty."));
      return;
    }

    const parsedPoints = Number(points);
    if (!Number.isFinite(parsedPoints) || parsedPoints < 0) {
      setLocalError(tr("Очки вклада должны быть числом 0 или больше.", "Contribution points must be a number 0 or greater."));
      return;
    }

    setLocalError("");
    const saved = await onUpdate(node.id, {
      title: normalizedTitle,
      description: description.trim(),
      contribution_points: parsedPoints
    });
    if (saved) {
      setIsEditing(false);
    }
  }

  async function submitAssignment() {
    if (!selectedAssigneeID) {
      setLocalError(tr("Выберите участника для назначения.", "Select a member to assign."));
      return;
    }
    setLocalError("");
    await onAssign(node.id, selectedAssigneeID);
  }

  async function submitDependencies() {
    if (selectedParentIDs.length === 0) {
      setLocalError(tr("Выберите хотя бы одну родительскую ноду.", "Select at least one parent node."));
      return;
    }
    setLocalError("");
    const added = await onAddDependencies(node.id, selectedParentIDs);
    if (added) {
      setSelectedParentIDs([]);
      setIsDependencyOpen(false);
    }
  }

  async function removeDependency(parentNodeID: UUID) {
    setLocalError("");
    const removed = await onRemoveDependency(node.id, parentNodeID);
    if (removed) {
      setSelectedParentIDs((current) => current.filter((value) => value !== parentNodeID));
    }
  }

  return (
    <article
      className={`workflow-node-card workflow-node-${node.status} ${node.isHighlighted ? "workflow-node-highlighted" : ""}`.trim()}
    >
      <div className="workflow-node-topline">
        <div>
          <h3>{node.title}</h3>
          <div className="workflow-node-badges">
            <span className={`workflow-status-pill workflow-status-${node.status}`}>{statusLabel(node.status, locale)}</span>
            <span className="workflow-point-badge">
              {node.contribution_points} {tr("очк.", "pts")}
            </span>
            <span className={`workflow-state-chip workflow-state-chip-${node.status}`}>{statusToneLabel}</span>
          </div>
        </div>
        {node.parentCount > 0 ? (
          <span className="workflow-parent-chip">
            {tr("Зависимостей", "Dependencies")}: {node.parentCount}
          </span>
        ) : (
          <span className="workflow-parent-chip">{tr("Без блокеров", "No blockers")}</span>
        )}
      </div>

      {node.description ? <p className="workflow-node-description">{node.description}</p> : null}

      <div className={`workflow-node-guidance workflow-node-guidance-${node.status}`}>
        <strong>{tr("Следующий шаг", "Next step")}</strong>
        <span>{nodeGuidance(node, locale)}</span>
      </div>

      <div className="workflow-node-meta">
        <div>
          <span>{tr("Исполнитель", "Assignee")}</span>
          <strong>{node.assigneeName}</strong>
        </div>
        <div>
          <span>{tr("Родители", "Parents")}</span>
          <strong>{node.parentTitles.length > 0 ? node.parentTitles.join(", ") : tr("Нет", "None")}</strong>
        </div>
        <div>
          <span>{tr("Обновлено", "Updated")}</span>
          <strong>{formatDateTime(node.updated_at, locale)}</strong>
        </div>
      </div>

      <div className="workflow-node-actions">
        <button
          type="button"
          className={!node.canTake ? "secondary" : undefined}
          disabled={takeDisabled}
          title={!node.canTake ? nodeGuidance(node, locale) : undefined}
          onClick={() => onTake(node.id)}
        >
          {assignBusy
            ? tr("Берём...", "Taking...")
            : node.status === "planned"
              ? tr("Заблокировано", "Blocked")
              : node.status === "done"
                ? tr("Уже завершено", "Already done")
                : tr("Взять в работу", "Take")}
        </button>

        {node.canUnassign ? (
          <button
            type="button"
            className="secondary"
            disabled={unassignBusy}
            onClick={() => onUnassign(node.id)}
          >
            {unassignBusy ? tr("Возвращаем...", "Unassigning...") : tr("Вернуть", "Unassign")}
          </button>
        ) : null}

        <button
          type="button"
          className={!node.canComplete ? "secondary" : undefined}
          disabled={completeDisabled}
          title={!node.canComplete ? nodeGuidance(node, locale) : undefined}
          onClick={() => onComplete(node.id)}
        >
          {completeBusy
            ? tr("Завершаем...", "Completing...")
            : node.status === "done"
              ? tr("Завершено", "Completed")
              : tr("Завершить", "Complete")}
        </button>

        <span className="workflow-node-note">{nodeGuidance(node, locale)}</span>
      </div>

      {canManage ? (
        <details className="workflow-admin-panel workflow-details" open={isAdminOpen} onToggle={(event) => setIsAdminOpen((event.target as HTMLDetailsElement).open)}>
          <summary className="workflow-details-summary workflow-details-summary-compact">
            <div>
              <strong>{tr("Инструменты менеджера", "Manager tools")}</strong>
              <p className="subtle">
                {canAdminAssign
                  ? tr("Назначение, редактирование и зависимости этой ноды.", "Assignment, editing, and dependency controls for this node.")
                  : tr("Управление доступно, но часть действий ограничена состоянием ноды.", "Management is available, but some actions are limited by node state.")}
              </p>
            </div>
            <div className="workflow-details-meta">
              <span className="workflow-parent-chip">{adminSummary}</span>
              <span className="workflow-parent-chip">{tr("Исполнитель", "Assignee")}: {node.assigneeName}</span>
            </div>
          </summary>

          <div className="workflow-details-body">
            <div className="workflow-admin-toolbar">
              <strong>{tr("Управление нодой", "Node management")}</strong>
              <div className="workflow-admin-toolbar-actions">
                <button type="button" className="secondary" onClick={() => setIsEditing((current) => !current)}>
                  {isEditing ? tr("Скрыть редактирование", "Hide edit form") : tr("Редактировать", "Edit")}
                </button>
                {node.status !== "done" ? (
                  <button type="button" className="secondary" onClick={() => setIsDependencyOpen((current) => !current)}>
                    {isDependencyOpen ? tr("Скрыть зависимости", "Hide dependencies") : tr("Зависимости", "Dependencies")}
                  </button>
                ) : null}
              </div>
            </div>

            {canAdminAssign ? (
              <div className="workflow-admin-assign">
                <label className="workflow-field">
                  <span>{tr("Назначить участнику", "Assign to member")}</span>
                  <select
                    value={selectedAssigneeID}
                    onChange={(event) => setSelectedAssigneeID(event.target.value as UUID | "")}
                  >
                    <option value="">{tr("Выберите участника", "Select member")}</option>
                    {members.map((member) => (
                      <option key={member.id} value={member.user_id}>
                        {displayHumanName(member, tr("Без имени", "Unnamed member"))} · {member.role}
                      </option>
                    ))}
                  </select>
                </label>
                <button type="button" disabled={assignBusy} onClick={() => void submitAssignment()}>
                  {assignBusy ? tr("Назначаем...", "Assigning...") : tr("Назначить", "Assign")}
                </button>
              </div>
            ) : (
              <p className="workflow-inline-note">
                {node.status === "planned"
                  ? tr("Назначение откроется, когда нода станет ready.", "Assignment becomes available when the node is ready.")
                  : tr("Завершённые ноды больше не переназначаются.", "Completed nodes cannot be reassigned anymore.")}
              </p>
            )}

            {isEditing ? (
              <div className="workflow-inline-card">
                <div className="workflow-form-grid">
                  <label className="workflow-field">
                    <span>{tr("Название", "Title")}</span>
                    <input value={title} onChange={(event) => setTitle(event.target.value)} />
                  </label>

                  <label className="workflow-field workflow-field-small">
                    <span>{tr("Очки вклада", "Contribution points")}</span>
                    <input
                      type="number"
                      min="0"
                      step="1"
                      value={points}
                      onChange={(event) => setPoints(event.target.value)}
                    />
                  </label>

                  <label className="workflow-field workflow-field-full">
                    <span>{tr("Описание", "Description")}</span>
                    <textarea rows={3} value={description} onChange={(event) => setDescription(event.target.value)} />
                  </label>
                </div>

                <div className="workflow-inline-actions">
                  <button type="button" disabled={updateBusy} onClick={() => void submitUpdate()}>
                    {updateBusy ? tr("Сохраняем...", "Saving...") : tr("Сохранить ноду", "Save node")}
                  </button>
                </div>
              </div>
            ) : null}

            {isDependencyOpen ? (
              <div className="workflow-inline-card">
                <div className="workflow-selector-card">
                  <div className="workflow-selector-head">
                    <strong>{tr("Текущие зависимости", "Current dependencies")}</strong>
                    <span>{existingParents.length}</span>
                  </div>
                  {existingParents.length === 0 ? (
                    <p className="workflow-inline-note">
                      {tr("У этой ноды пока нет родительских связей.", "This node does not have parent dependencies yet.")}
                    </p>
                  ) : (
                    <div className="workflow-dependency-list">
                      {existingParents.map((parent) => {
                        const removeBusy = busyKey === `dependency-remove:${node.id}:${parent.id}`;
                        return (
                          <div key={parent.id} className="workflow-dependency-row">
                            <span>
                              <strong>{parent.title}</strong>
                            </span>
                            <button
                              type="button"
                              className="secondary"
                              disabled={removeBusy}
                              onClick={() => void removeDependency(parent.id)}
                            >
                              {removeBusy ? tr("Удаляем...", "Removing...") : tr("Убрать", "Remove")}
                            </button>
                          </div>
                        );
                      })}
                    </div>
                  )}
                </div>

                <div className="workflow-selector-card">
                  <div className="workflow-selector-head">
                    <strong>{tr("Новые родительские ноды", "New parent nodes")}</strong>
                    <span>{selectedParentIDs.length}</span>
                  </div>
                  {availableParents.length === 0 ? (
                    <p className="workflow-inline-note">
                      {tr("Нет доступных нод для новой зависимости.", "No available nodes for a new dependency.")}
                    </p>
                  ) : (
                    <div className="workflow-checkbox-list">
                      {availableParents.map((parent) => (
                        <label key={parent.id} className="workflow-checkbox-row">
                          <input
                            type="checkbox"
                            checked={selectedParentIDs.includes(parent.id)}
                            onChange={() => toggleParent(parent.id)}
                          />
                          <span>
                            <strong>{parent.title}</strong>
                            <small>{statusLabel(parent.status, locale)}</small>
                          </span>
                        </label>
                      ))}
                    </div>
                  )}
                </div>

                <div className="workflow-inline-actions">
                  <button
                    type="button"
                    disabled={dependencyBusy || availableParents.length === 0}
                    onClick={() => void submitDependencies()}
                  >
                    {dependencyBusy ? tr("Добавляем...", "Adding...") : tr("Добавить зависимости", "Add dependencies")}
                  </button>
                </div>
              </div>
            ) : null}

            {localError ? <p className="workflow-inline-error">{localError}</p> : null}
          </div>
        </details>
      ) : null}
    </article>
  );
}

export function WorkflowSummaryPanel({
  users,
  totalProjectPoints,
  completedPoints,
  completionPercentage,
  locale
}: {
  users: WorkflowContributionUserSummary[];
  totalProjectPoints: number;
  completedPoints: number;
  completionPercentage: number;
  locale: "ru" | "en";
}) {
  const tr = (ru: string, en: string) => (locale === "en" ? en : ru);

  return (
    <aside className="card workflow-summary-card">
      <div className="workflow-summary-head">
        <div>
          <span className="shell-eyebrow">{tr("Вклад команды", "Team contribution")}</span>
          <h2 className="title">{tr("Сводка по очкам", "Points summary")}</h2>
          <p className="subtle">
            {tr(
              "Сводка показывает весь объём очков проекта, уже выполненные очки и текущую долю каждого участника.",
              "This summary shows the full project points, completed points, and each member's current share."
            )}
          </p>
        </div>
        <div className="workflow-summary-total">
          <span>{tr("Всего очков проекта", "Total project points")}</span>
          <strong>{totalProjectPoints}</strong>
        </div>
      </div>

      <div className="workflow-summary-project-stats">
        <span>{tr(`Всего: ${totalProjectPoints} очков`, `Total: ${totalProjectPoints} pts`)}</span>
        <strong>
          {tr(
            `Выполнено: ${completedPoints} / ${totalProjectPoints} (${completionPercentage.toFixed(0)}%)`,
            `Completed: ${completedPoints} / ${totalProjectPoints} (${completionPercentage.toFixed(0)}%)`
          )}
        </strong>
      </div>

      {users.length === 0 ? (
        <div className="workflow-empty-state">
          <p>{tr("У проекта пока нет участников.", "This project has no members yet.")}</p>
        </div>
      ) : (
        <div className="workflow-summary-list">
          {users.map((user) => (
            <article key={user.user_id} className="workflow-summary-row">
              <div className="workflow-summary-topline">
                <div>
                  <h3>{displayHumanName(user, tr("Без имени", "Unnamed member"))}</h3>
                  <p>{tr("Вклад по выполненным задачам и текущим назначениям", "Contribution from completed work and current assignments")}</p>
                </div>
                <strong>{user.earned_points}</strong>
              </div>

              <div className="workflow-summary-progress">
                <div
                  className="workflow-summary-progress-fill"
                  style={{ width: `${Math.max(0, Math.min(user.share_of_total_project_points, 100))}%` }}
                />
              </div>

              <div className="workflow-summary-metrics">
                <span>
                  {tr("Заработано", "Earned")}: <strong>{user.earned_points}</strong>
                </span>
                <span>
                  {tr("Доля проекта", "Project share")}: <strong>{user.share_of_total_project_points.toFixed(1)}%</strong>
                </span>
                <span>
                  {tr("Ожидаемая доля", "Projected share")}: <strong>{user.projected_share_percentage.toFixed(1)}%</strong>
                </span>
                <span>
                  {tr("Завершено нод", "Completed nodes")}: <strong>{user.completed_nodes_count}</strong>
                </span>
              </div>
            </article>
          ))}
        </div>
      )}
    </aside>
  );
}
