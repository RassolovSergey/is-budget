"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { useParams } from "next/navigation";

import { AppShell } from "../../../../components/AppShell";
import { ProjectNav } from "../../../../components/ProjectNav";
import { useLocale } from "../../../../components/LocaleProvider";
import { useToast } from "../../../../components/ToastProvider";
import { ApiError, apiRequest } from "../../../../lib/api";
import { displayHumanName } from "../../../../lib/displayName";
import type {
  Project,
  ProjectMemberWithUser,
  User,
  UUID,
  WorkflowNodeStatus,
  WorkflowProjectGraph,
  WorkflowProjectSummary
} from "../../../../lib/types";
import { useAuthGuard } from "../../../../lib/useAuthGuard";
import {
  WorkflowBoard,
  WorkflowViewControls,
  WorkflowManagePanel,
  WorkflowOverview,
  WorkflowSummaryPanel,
  type WorkflowBoardFilter,
  type WorkflowCreateNodeInput,
  type WorkflowNodeView,
  type WorkflowUpdateNodeInput
} from "./WorkflowSections";

const emptyGroupedNodes: Record<WorkflowNodeStatus, WorkflowNodeView[]> = {
  ready: [],
  in_progress: [],
  planned: [],
  done: []
};

export default function WorkflowPage() {
  const { id } = useParams<{ id: string }>();
  const { locale } = useLocale();
  const { showToast } = useToast();
  const { token, ready } = useAuthGuard();

  const [project, setProject] = useState<Project | null>(null);
  const [me, setMe] = useState<User | null>(null);
  const [members, setMembers] = useState<ProjectMemberWithUser[]>([]);
  const [graph, setGraph] = useState<WorkflowProjectGraph | null>(null);
  const [summary, setSummary] = useState<WorkflowProjectSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [successMessage, setSuccessMessage] = useState("");
  const [busyKey, setBusyKey] = useState("");
  const [highlightedNodeIds, setHighlightedNodeIds] = useState<UUID[]>([]);
  const [activeFilter, setActiveFilter] = useState<WorkflowBoardFilter>("all");
  const highlightTimerRef = useRef<number | null>(null);

  const tr = (ru: string, en: string) => (locale === "en" ? en : ru);
  const appLocale = locale === "en" ? "en" : "ru";

  function formatWorkflowError(err: unknown): string {
    if (!(err instanceof ApiError)) {
      return (err as Error)?.message || tr("Не удалось выполнить действие с вкладом команды.", "Failed to perform the team contribution action.");
    }

    const message = err.message;
    const mappedMessages: Record<string, [string, string]> = {
      "one or more parent nodes were not found": [
        "Одна или несколько родительских нод не найдены.",
        "One or more parent nodes were not found."
      ],
      "all parent nodes must belong to the same project": [
        "Все родительские ноды должны принадлежать этому проекту.",
        "All parent nodes must belong to this project."
      ],
      "duplicate parent nodes are not allowed": [
        "Нельзя добавлять одну и ту же родительскую ноду дважды.",
        "You cannot add the same parent node twice."
      ],
      "completed nodes cannot change dependencies": [
        "У завершённых нод нельзя менять зависимости.",
        "Completed nodes cannot change dependencies."
      ],
      "parent node not found for this workflow project": [
        "Родительская нода не найдена в этом проекте.",
        "The parent node was not found in this project."
      ],
      "parent and child nodes must belong to the same project": [
        "Связь можно создавать только между нодами одного проекта.",
        "Dependencies can only link nodes from the same project."
      ],
      "this dependency already exists": [
        "Такая зависимость уже существует.",
        "That dependency already exists."
      ],
      "a node cannot depend on itself": [
        "Нода не может зависеть от самой себя.",
        "A node cannot depend on itself."
      ],
      "this dependency would create a circular workflow": [
        "Эта зависимость создаст цикл в графе работ.",
        "This dependency would create a cycle."
      ],
      "dependency link not found": [
        "Эта зависимость уже удалена или не существует.",
        "That dependency was already removed or does not exist."
      ],
      "completed nodes cannot be returned": [
        "Завершённую задачу уже нельзя вернуть обратно.",
        "A completed task cannot be returned."
      ],
      "this node is not assigned right now": [
        "Эта задача сейчас никому не назначена.",
        "This task is not assigned right now."
      ],
      "only in-progress nodes can be returned": [
        "Вернуть можно только задачу в статусе «В работе».",
        "Only an in-progress task can be returned."
      ],
      "only the assignee or a project manager can return this node": [
        "Вернуть задачу может только её исполнитель или менеджер проекта.",
        "Only the assignee or a project manager can return this task."
      ]
    };

    const mapped = mappedMessages[message];
    if (mapped) {
      return tr(mapped[0], mapped[1]);
    }

    if (err.status === 403) {
      return tr(
        "У вас нет прав для этого действия в графе работ.",
        "You do not have permission for this team contribution action."
      );
    }
    if (err.status === 409) {
      return tr(
        "Конфликт состояния графа работ. Обновите страницу и попробуйте снова.",
        "Task state conflict. Refresh and try again."
      );
    }

    return message || tr("Не удалось выполнить действие с вкладом команды.", "Failed to perform the team contribution action.");
  }

  async function load(previousGraph?: WorkflowProjectGraph | null) {
    if (!token) return;

    try {
      setLoading(true);
      setError("");

      const [projectResp, meResp, membersResp, graphResp, summaryResp] = await Promise.all([
        apiRequest<Project>(`/projects/${id}`, { token }),
        apiRequest<User>("/auth/me", { token }),
        apiRequest<ProjectMemberWithUser[]>(`/projects/${id}/members`, { token }),
        apiRequest<WorkflowProjectGraph>(`/workflow/project/${id}`, { token }),
        apiRequest<WorkflowProjectSummary>(`/workflow/project/${id}/summary`, { token })
      ]);

      setProject(projectResp);
      setMe(meResp);
      setMembers(membersResp);
      setGraph(graphResp);
      setSummary(summaryResp);

      if (previousGraph) {
        const previouslyPlanned = new Set(
          previousGraph.nodes.filter((node) => node.status === "planned").map((node) => node.id)
        );
        const unlocked = graphResp.nodes
          .filter((node) => node.status === "ready" && previouslyPlanned.has(node.id))
          .map((node) => node.id);

        setHighlightedNodeIds(unlocked);
        if (highlightTimerRef.current) {
          window.clearTimeout(highlightTimerRef.current);
        }
        if (unlocked.length > 0) {
          highlightTimerRef.current = window.setTimeout(() => {
            setHighlightedNodeIds([]);
          }, 2800);
        }
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
    return () => {
      if (highlightTimerRef.current) {
        window.clearTimeout(highlightTimerRef.current);
      }
    };
  }, [ready, id, token]);

  const membersByUserId = useMemo(() => {
    const map = new Map<UUID, ProjectMemberWithUser>();
    for (const member of members) {
      map.set(member.user_id, member);
    }
    return map;
  }, [members]);

  const canManageWorkflow = useMemo(() => {
    if (!me) return false;
    return project?.current_user_role === "PM" || membersByUserId.get(me.id)?.role === "PM";
  }, [project, membersByUserId, me]);

  const groupedNodes = useMemo(() => {
    if (!graph || !me) return emptyGroupedNodes;

    const nodesById = new Map(graph.nodes.map((node) => [node.id, node]));
    const parentIdsByChild = new Map<UUID, UUID[]>();

    for (const dependency of graph.dependencies) {
      const current = parentIdsByChild.get(dependency.child_node_id) || [];
      current.push(dependency.parent_node_id);
      parentIdsByChild.set(dependency.child_node_id, current);
    }

    const grouped: Record<WorkflowNodeStatus, WorkflowNodeView[]> = {
      ready: [],
      in_progress: [],
      planned: [],
      done: []
    };

    for (const node of graph.nodes) {
      const parentIds = parentIdsByChild.get(node.id) || [];
      const parentTitles = parentIds
        .map((parentId) => nodesById.get(parentId)?.title)
        .filter(Boolean) as string[];
      const assignee = node.assignee_user_id ? membersByUserId.get(node.assignee_user_id) : undefined;
      const isOwnedByMe = node.assignee_user_id === me.id;

      grouped[node.status].push({
        ...node,
        assigneeName: assignee ? displayHumanName(assignee, tr("Без имени", "Unnamed")) : tr("Не назначен", "Unassigned"),
        parentIds,
        parentTitles,
        parentCount: parentTitles.length,
        isHighlighted: highlightedNodeIds.includes(node.id),
        isOwnedByMe,
        isCompletedByMe: node.completed_by === me.id,
        canTake: node.status === "ready" && !node.assignee_user_id,
        canComplete: node.status === "in_progress" && isOwnedByMe,
        canUnassign: node.status === "in_progress" && (isOwnedByMe || canManageWorkflow)
      });
    }

    return grouped;
  }, [canManageWorkflow, graph, highlightedNodeIds, me, membersByUserId, locale]);

  const totalNodes = graph?.nodes.length || 0;
  const readyNodes = groupedNodes.ready.length;
  const inProgressNodes = groupedNodes.in_progress.length;
  const doneNodes = groupedNodes.done.length;
  const totalProjectPoints = summary?.total_project_points || 0;
  const completedPoints = summary?.completed_points || 0;
  const completionPercentage = summary?.completion_percentage || 0;
  const filteredGroupedNodes = useMemo(() => {
    if (activeFilter === "all") return groupedNodes;

    const emptyState = {
      ready: [] as WorkflowNodeView[],
      in_progress: [] as WorkflowNodeView[],
      planned: [] as WorkflowNodeView[],
      done: [] as WorkflowNodeView[]
    };

    if (activeFilter === "my") {
      return {
        ...emptyState,
        in_progress: groupedNodes.in_progress.filter((node) => node.isOwnedByMe),
        done: groupedNodes.done.filter((node) => node.isCompletedByMe)
      };
    }

    if (activeFilter === "ready") {
      return { ...emptyState, ready: groupedNodes.ready };
    }

    if (activeFilter === "planned") {
      return { ...emptyState, planned: groupedNodes.planned };
    }

    return { ...emptyState, done: groupedNodes.done };
  }, [activeFilter, groupedNodes]);
  const filterCounts = useMemo(
    () => ({
      all: totalNodes,
      my:
        groupedNodes.in_progress.filter((node) => node.isOwnedByMe).length +
        groupedNodes.done.filter((node) => node.isCompletedByMe).length,
      ready: groupedNodes.ready.length,
      planned: groupedNodes.planned.length,
      done: groupedNodes.done.length
    }),
    [groupedNodes, totalNodes]
  );
  const summaryUsers = useMemo(
    () =>
      [...(summary?.users || [])].sort((left, right) => {
        if (right.earned_points !== left.earned_points) return right.earned_points - left.earned_points;
        return displayHumanName(left, tr("Без имени", "Unnamed")).localeCompare(
          displayHumanName(right, tr("Без имени", "Unnamed"))
        );
      }),
    [locale, summary]
  );

  async function runMutation(key: string, action: () => Promise<void>, successMessage: string): Promise<boolean> {
    if (!token || busyKey) return false;
    const previousGraph = graph;

    try {
      setBusyKey(key);
      setError("");
      setSuccessMessage("");
      await action();
      setSuccessMessage(successMessage);
      showToast({ message: successMessage, tone: "success" });
      await load(previousGraph);
      return true;
    } catch (err) {
      setSuccessMessage("");
      setError(formatWorkflowError(err));
      return false;
    } finally {
      setBusyKey("");
    }
  }

  async function takeNode(nodeId: UUID) {
    if (!me) return;
    await runMutation(
      `assign:${nodeId}`,
      async () => {
        await apiRequest(`/workflow/node/${nodeId}/assign`, {
          method: "POST",
          token,
          body: { user_id: me.id }
        });
      },
      tr("Нода назначена на вас.", "Node assigned to you.")
    );
  }

  async function assignNode(nodeId: UUID, userId: UUID): Promise<boolean> {
    return runMutation(
      `assign:${nodeId}`,
      async () => {
        await apiRequest(`/workflow/node/${nodeId}/assign`, {
          method: "POST",
          token,
          body: { user_id: userId }
        });
      },
      tr("Назначение обновлено.", "Assignment updated.")
    );
  }

  async function unassignNode(nodeId: UUID) {
    await runMutation(
      `unassign:${nodeId}`,
      async () => {
        await apiRequest(`/workflow/node/${nodeId}/unassign`, {
          method: "POST",
          token
        });
      },
      tr("Задача возвращена в доступные.", "Task returned to ready.")
    );
  }

  async function completeNode(nodeId: UUID) {
    await runMutation(
      `complete:${nodeId}`,
      async () => {
        await apiRequest(`/workflow/node/${nodeId}/complete`, {
          method: "POST",
          token
        });
      },
      tr("Нода завершена, вклад записан.", "Node completed and contribution recorded.")
    );
  }

  async function createNode(input: WorkflowCreateNodeInput): Promise<boolean> {
    return runMutation(
      "create",
      async () => {
        await apiRequest("/workflow/node", {
          method: "POST",
          token,
          body: {
            project_id: id,
            ...input
          }
        });
      },
      tr("Нода создана.", "Node created.")
    );
  }

  async function updateNode(nodeId: UUID, input: WorkflowUpdateNodeInput): Promise<boolean> {
    return runMutation(
      `update:${nodeId}`,
      async () => {
        await apiRequest(`/workflow/node/${nodeId}`, {
          method: "PUT",
          token,
          body: input
        });
      },
      tr("Нода обновлена.", "Node updated.")
    );
  }

  async function addDependencies(nodeId: UUID, parentNodeIds: UUID[]): Promise<boolean> {
    return runMutation(
      `dependency:${nodeId}`,
      async () => {
        await apiRequest(`/workflow/node/${nodeId}/dependency`, {
          method: "POST",
          token,
          body: { parent_node_ids: parentNodeIds }
        });
      },
      tr("Зависимости обновлены.", "Dependencies updated.")
    );
  }

  async function removeDependency(nodeId: UUID, parentNodeId: UUID): Promise<boolean> {
    return runMutation(
      `dependency-remove:${nodeId}:${parentNodeId}`,
      async () => {
        await apiRequest(`/workflow/node/${nodeId}/dependency/${parentNodeId}`, {
          method: "DELETE",
          token
        });
      },
      tr("Зависимость удалена.", "Dependency removed.")
    );
  }

  return (
    <AppShell title={project?.name || tr("Вклад команды", "Team Contribution")} className="workflow-shell">
      <ProjectNav projectId={id} />

      <section className="card workflow-hero-card">
        <div>
          <span className="shell-eyebrow">{tr("Вклад команды", "Team Contribution")}</span>
          <h2 className="title">{tr("Задачи, очки и доля команды в проекте", "Tasks, points, and team share in the project")}</h2>
          <p className="subtle">
            {tr(
              "Здесь видно, какие задачи уже выполнены, сколько очков заработано и как складывается вклад команды по проекту.",
              "This is where you see completed work, earned points, and how team contribution is building across the project."
            )}
          </p>
        </div>
        <div className="workflow-hero-note">
          <span>{tr("Ваш режим", "Your mode")}</span>
          <strong>
            {canManageWorkflow
              ? tr("Управление вкладом команды", "Team contribution management")
              : tr("Работа с задачами", "Task execution")}
          </strong>
          <span>{tr("Текущий пользователь", "Current user")}</span>
          <strong>{me ? displayHumanName(me, tr("Без имени", "Unnamed")) : tr("Загрузка...", "Loading...")}</strong>
        </div>
      </section>

      {error ? (
        <section className="card workflow-feedback workflow-feedback-error">
          <strong>{tr("Не удалось обновить вклад команды.", "Failed to update team contribution.")}</strong>
          <p>{error}</p>
          <button type="button" className="secondary" onClick={() => void load(graph)}>
            {tr("Попробовать снова", "Try again")}
          </button>
        </section>
      ) : null}

      {!error && successMessage ? (
        <section className="card workflow-feedback workflow-feedback-success">
          <strong>{tr("Данные по вкладу команды обновлены.", "Team contribution updated.")}</strong>
          <p>{successMessage}</p>
        </section>
      ) : null}

      {busyKey ? (
        <section className="card workflow-feedback workflow-feedback-busy">
          <strong>{tr("Применяем изменения...", "Applying changes...")}</strong>
          <p className="subtle">
            {tr(
              "Обновляем ноды, зависимости и сводку вкладов, чтобы экран оставался согласованным.",
              "Refreshing nodes, dependencies, and contribution summary to keep the screen consistent."
            )}
          </p>
        </section>
      ) : null}

      {loading && !graph ? (
        <section className="card workflow-feedback">
          <strong>{tr("Загружаем вклад команды...", "Loading team contribution...")}</strong>
          <p className="subtle">
            {tr(
              "Получаем задачи, статусы, участников проекта и сводку по очкам.",
              "Fetching tasks, statuses, project members, and points summary."
            )}
          </p>
        </section>
      ) : null}

      {graph && summary ? (
        <>
          <WorkflowOverview
            totalNodes={totalNodes}
            readyNodes={readyNodes}
            inProgressNodes={inProgressNodes}
            doneNodes={doneNodes}
            totalProjectPoints={totalProjectPoints}
            completedPoints={completedPoints}
            completionPercentage={completionPercentage}
            locale={appLocale}
          />

          <div className="workflow-layout">
            <div className="workflow-main-column">
              {canManageWorkflow ? (
                <WorkflowManagePanel
                  locale={appLocale}
                  nodes={graph.nodes}
                  busy={busyKey === "create"}
                  onCreate={createNode}
                />
              ) : null}

              <WorkflowViewControls
                locale={appLocale}
                activeFilter={activeFilter}
                counts={filterCounts}
                busy={Boolean(busyKey)}
                onChange={setActiveFilter}
              />

              <WorkflowBoard
                nodesByStatus={filteredGroupedNodes}
                allNodes={graph.nodes}
                members={members}
                locale={appLocale}
                busyKey={busyKey}
                canManage={canManageWorkflow}
                onTake={takeNode}
                onUnassign={unassignNode}
                onAssign={assignNode}
                onComplete={completeNode}
                onUpdate={updateNode}
                onAddDependencies={addDependencies}
                onRemoveDependency={removeDependency}
                hideEmptySections={activeFilter !== "all"}
                emptyStateMessage={
                  activeFilter === "my"
                    ? tr("У вас пока нет нод в этом представлении.", "You do not have any nodes in this view yet.")
                    : tr("В этом представлении сейчас нет нод.", "There are no nodes in this view right now.")
                }
              />
            </div>

            <div className="workflow-side-column">
              <WorkflowSummaryPanel
                users={summaryUsers}
                totalProjectPoints={totalProjectPoints}
                completedPoints={completedPoints}
                completionPercentage={completionPercentage}
                locale={appLocale}
              />
            </div>
          </div>

          {totalNodes === 0 ? (
            <section className="card workflow-feedback">
              <strong>{tr("Пока нет задач с очками вклада.", "No contribution tasks yet.")}</strong>
              <p className="subtle">
                {canManageWorkflow
                  ? tr(
                      "Создайте первую ноду в панели управления выше и при необходимости сразу свяжите её с родителями.",
                      "Create the first node in the management panel above and attach parents right away if needed."
                    )
                  : tr(
                      "Менеджер проекта ещё не настроил задачи для вклада команды.",
                      "The project manager has not configured team contribution tasks yet."
                    )}
              </p>
            </section>
          ) : null}
        </>
      ) : null}
    </AppShell>
  );
}
