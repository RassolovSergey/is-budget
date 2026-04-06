"use client";

import { FormEvent, useEffect, useMemo, useState } from "react";

import { AppShell } from "../../components/AppShell";
import { useConfirm } from "../../components/ConfirmProvider";
import { useLocale } from "../../components/LocaleProvider";
import { apiRequest } from "../../lib/api";
import { PROJECTS_UPDATED_EVENT } from "../../lib/appEvents";
import { useAuthGuard } from "../../lib/useAuthGuard";
import type { CurrencyCode, Project, ProjectCategoryFolderSummary, ProjectRole, ProjectStatus } from "../../lib/types";
import {
  ProjectContextMenu,
  ProjectFormSection,
  ProjectsBrowserSection,
  ProjectsFolderExplorerSection,
  ProjectsKanbanSection
} from "./ProjectsSections";
import {
  ALL_FOLDERS_KEY,
  DEFAULT_FOLDER,
  FOLDER_COLOR_OPTIONS,
  FOLDER_ICON_OPTIONS,
  KANBAN_COLUMNS,
  LOCAL_FOLDERS_STORAGE_KEY,
  type FolderColor,
  type FolderIcon,
  type FolderRecord,
  type ProjectMenuState
} from "./page-types";

function canonicalFolderName(value?: string | null): string {
  const trimmed = value?.trim() || "";
  const lowered = trimmed.toLowerCase();
  if (!trimmed || lowered === "general" || lowered === "общее") {
    return DEFAULT_FOLDER;
  }
  return trimmed;
}

function slugify(value: string): string {
  return value
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9а-яё]+/gi, "-")
    .replace(/^-+|-+$/g, "") || "folder";
}

function createFolderRecord(name: string, color: FolderColor = "slate", icon: FolderIcon = "folder"): FolderRecord {
  return {
    id: `folder-${slugify(name)}-${Math.random().toString(36).slice(2, 8)}`,
    name: canonicalFolderName(name),
    color,
    icon
  };
}

function normalizeFolderRecords(records: FolderRecord[], locale: string): FolderRecord[] {
  const map = new Map<string, FolderRecord>();
  for (const record of records) {
    const name = canonicalFolderName(record.name);
    if (!name || name === DEFAULT_FOLDER) continue;
    if (!map.has(name)) {
      map.set(name, {
        id: record.id || `folder-${slugify(name)}`,
        name,
        color: FOLDER_COLOR_OPTIONS.includes(record.color) ? record.color : "slate",
        icon: FOLDER_ICON_OPTIONS.includes(record.icon) ? record.icon : "folder"
      });
    }
  }
  return Array.from(map.values()).sort((a, b) => a.name.localeCompare(b.name, locale === "en" ? "en" : "ru"));
}

function loadStoredFolders(locale: string): FolderRecord[] {
  if (typeof window === "undefined") return [];
  try {
    const raw = window.localStorage.getItem(LOCAL_FOLDERS_STORAGE_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];

    if (parsed.every((value) => typeof value === "string")) {
      return normalizeFolderRecords(
        parsed.map((value, index) => createFolderRecord(value, FOLDER_COLOR_OPTIONS[index % FOLDER_COLOR_OPTIONS.length], FOLDER_ICON_OPTIONS[index % FOLDER_ICON_OPTIONS.length])),
        locale
      );
    }

    return normalizeFolderRecords(
      parsed.filter((value): value is FolderRecord => typeof value === "object" && value !== null && typeof value.name === "string"),
      locale
    );
  } catch {
    return [];
  }
}

function folderGlyph(icon: FolderIcon): string {
  switch (icon) {
    case "spark":
      return "✦";
    case "stack":
      return "▤";
    case "badge":
      return "⬢";
    default:
      return "◫";
  }
}

function colorToken(color: FolderColor): string {
  switch (color) {
    case "emerald":
      return "var(--folder-emerald)";
    case "violet":
      return "var(--folder-violet)";
    case "amber":
      return "var(--folder-amber)";
    case "rose":
      return "var(--folder-rose)";
    case "cyan":
      return "var(--folder-cyan)";
    default:
      return "var(--folder-slate)";
  }
}

function fallbackFolderRecord(name: string): FolderRecord {
  const normalized = canonicalFolderName(name);
  const hash = normalized.split("").reduce((acc, char) => acc + char.charCodeAt(0), 0);
  return {
    id: `fallback-${slugify(normalized)}`,
    name: normalized,
    color: FOLDER_COLOR_OPTIONS[hash % FOLDER_COLOR_OPTIONS.length],
    icon: FOLDER_ICON_OPTIONS[hash % FOLDER_ICON_OPTIONS.length]
  };
}

export default function ProjectsPage() {
  const confirmAction = useConfirm();
  const { t, locale } = useLocale();
  const { token, ready } = useAuthGuard();

  const [projects, setProjects] = useState<Project[]>([]);
  const [folderSummaries, setFolderSummaries] = useState<ProjectCategoryFolderSummary[]>([]);
  const [editingProjectId, setEditingProjectId] = useState<string | null>(null);
  const [editingOriginalStatus, setEditingOriginalStatus] = useState<ProjectStatus | null>(null);
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [durationDays, setDurationDays] = useState<string>("14");
  const [currencyCode, setCurrencyCode] = useState<CurrencyCode>("EUR");
  const [secondaryCurrencyCode, setSecondaryCurrencyCode] = useState<string>("");
  const [folderChoice, setFolderChoice] = useState(DEFAULT_FOLDER);
  const [activeFolder, setActiveFolder] = useState<string>(ALL_FOLDERS_KEY);
  const [customFolders, setCustomFolders] = useState<FolderRecord[]>([]);
  const [folderComposerOpen, setFolderComposerOpen] = useState(false);
  const [folderDraftName, setFolderDraftName] = useState("");
  const [folderDraftColor, setFolderDraftColor] = useState<FolderColor>("emerald");
  const [folderDraftIcon, setFolderDraftIcon] = useState<FolderIcon>("folder");
  const [folderDraftProjectIds, setFolderDraftProjectIds] = useState<string[]>([]);
  const [movingProjectId, setMovingProjectId] = useState<string | null>(null);
  const [draggedProjectId, setDraggedProjectId] = useState<string | null>(null);
  const [folderDropTarget, setFolderDropTarget] = useState<string | null>(null);
  const [folderActionBusy, setFolderActionBusy] = useState(false);
  const [projectMenu, setProjectMenu] = useState<ProjectMenuState>(null);
  const [error, setError] = useState("");

  const tr = (ru: string, en: string) => (locale === "en" ? en : ru);

  async function loadProjects() {
    if (!token) return;
    try {
      const [items, summaries] = await Promise.all([
        apiRequest<Project[]>("/projects", { token }),
        apiRequest<ProjectCategoryFolderSummary[]>("/projects/category-folders", { token })
      ]);
      setProjects(items);
      setFolderSummaries(summaries);
    } catch (err) {
      setError((err as Error).message);
    }
  }

  useEffect(() => {
    if (ready) {
      void loadProjects();
    }
  }, [ready, token]);

  useEffect(() => {
    if (typeof window === "undefined") return;
    const handleProjectsUpdated = () => {
      if (!ready || !token) return;
      void loadProjects();
    };
    window.addEventListener(PROJECTS_UPDATED_EVENT, handleProjectsUpdated);
    return () => window.removeEventListener(PROJECTS_UPDATED_EVENT, handleProjectsUpdated);
  }, [ready, token]);

  useEffect(() => {
    setCustomFolders(loadStoredFolders(locale));
  }, [locale]);

  useEffect(() => {
    if (typeof window === "undefined") return;
    window.localStorage.setItem(LOCAL_FOLDERS_STORAGE_KEY, JSON.stringify(customFolders));
  }, [customFolders]);

  useEffect(() => {
    const closeMenu = () => setProjectMenu(null);
    window.addEventListener("click", closeMenu);
    window.addEventListener("scroll", closeMenu, true);
    return () => {
      window.removeEventListener("click", closeMenu);
      window.removeEventListener("scroll", closeMenu, true);
    };
  }, []);

  const PROJECT_STATUS_LABELS: Record<ProjectStatus, string> = {
    preparation: t("status.preparation"),
    active: t("status.active"),
    completed: t("status.completed")
  };

  const PROJECT_ROLE_LABELS: Record<ProjectRole, string> = {
    PM: t("role.pm"),
    ANALYST: t("role.analyst"),
    TEAM: t("role.team"),
    CUSTOMER: t("role.customer")
  };

  const folderSummaryMap = useMemo(() => {
    const map = new Map<string, ProjectCategoryFolderSummary>();
    for (const item of folderSummaries) {
      const normalized = canonicalFolderName(item.category);
      map.set(normalized, { ...item, category: normalized });
    }
    return map;
  }, [folderSummaries]);

  const folderMap = useMemo(() => {
    const map = new Map<string, FolderRecord>();
    for (const folder of customFolders) {
      map.set(canonicalFolderName(folder.name), folder);
    }
    for (const project of projects) {
      const folderName = canonicalFolderName(project.category);
      if (folderName !== DEFAULT_FOLDER && !map.has(folderName)) {
        map.set(folderName, fallbackFolderRecord(folderName));
      }
    }
    if ((projects.some((project) => canonicalFolderName(project.category) === DEFAULT_FOLDER) || folderSummaryMap.has(DEFAULT_FOLDER)) && !map.has(DEFAULT_FOLDER)) {
      map.set(DEFAULT_FOLDER, { id: "general", name: DEFAULT_FOLDER, color: "slate", icon: "folder" });
    }
    return map;
  }, [customFolders, folderSummaryMap, projects]);

  const folderRecords = useMemo(
    () => Array.from(folderMap.values()).sort((a, b) => a.name.localeCompare(b.name, locale === "en" ? "en" : "ru")),
    [folderMap, locale]
  );

  const folderNames = useMemo(() => Array.from(new Set([DEFAULT_FOLDER, ...folderRecords.map((folder) => folder.name)])), [folderRecords]);

  const allFoldersSummary = useMemo<ProjectCategoryFolderSummary>(
    () => ({
      category: ALL_FOLDERS_KEY,
      active_projects_count: projects.filter((project) => project.status !== "completed").length,
      completed_projects_count: projects.filter((project) => project.status === "completed").length,
      historical_campaigns_count: folderSummaries.reduce((sum, item) => sum + item.historical_campaigns_count, 0)
    }),
    [projects, folderSummaries]
  );

  useEffect(() => {
    if (activeFolder !== ALL_FOLDERS_KEY && !folderNames.includes(activeFolder)) {
      setActiveFolder(ALL_FOLDERS_KEY);
    }
  }, [activeFolder, folderNames]);

  useEffect(() => {
    if (!editingProjectId && activeFolder !== ALL_FOLDERS_KEY) {
      setFolderChoice(activeFolder);
    }
  }, [activeFolder, editingProjectId]);

  const filteredProjects = activeFolder === ALL_FOLDERS_KEY
    ? projects
    : projects.filter((project) => canonicalFolderName(project.category) === activeFolder);

  const folderProjectOptions = useMemo(
    () => projects.filter((project) => project.current_user_role === "PM"),
    [projects]
  );

  const activeFolderProjects = activeFolder === ALL_FOLDERS_KEY
    ? []
    : projects.filter((project) => canonicalFolderName(project.category) === activeFolder);

  const canDeleteActiveFolder = activeFolder !== ALL_FOLDERS_KEY
    && activeFolder !== DEFAULT_FOLDER
    && activeFolderProjects.every((project) => project.current_user_role === "PM");

  function folderLabel(folderName: string): string {
    return canonicalFolderName(folderName) === DEFAULT_FOLDER ? t("projects.general") : folderName;
  }

  function folderRecordFor(folderName: string): FolderRecord {
    const normalized = canonicalFolderName(folderName);
    if (normalized === DEFAULT_FOLDER) {
      return { id: "general", name: DEFAULT_FOLDER, color: "slate", icon: "folder" };
    }
    return folderMap.get(normalized) || fallbackFolderRecord(normalized);
  }

  function folderSummaryFor(folderName: string): ProjectCategoryFolderSummary {
    const normalized = canonicalFolderName(folderName);
    return folderSummaryMap.get(normalized) || {
      category: normalized,
      active_projects_count: 0,
      completed_projects_count: 0,
      historical_campaigns_count: 0
    };
  }

  async function upsertProject(e: FormEvent) {
    e.preventDefault();
    setError("");

    const parsedDuration = Number(durationDays);
    if (durationDays.trim() === "" || Number.isNaN(parsedDuration) || parsedDuration < 3 || parsedDuration > 30) {
      setError(t("projects.err.duration"));
      return;
    }

    const resolvedFolder = canonicalFolderName(folderChoice);
    if (!resolvedFolder.trim()) {
      setError(tr("Укажите папку проекта.", "Specify a project folder."));
      return;
    }

    try {
      const body = {
        name,
        description,
        duration_days: parsedDuration,
        currency_code: currencyCode,
        secondary_currency_code: secondaryCurrencyCode || undefined,
        category: resolvedFolder
      };
      if (editingProjectId) {
        await apiRequest<Project>(`/projects/${editingProjectId}`, {
          method: "PUT",
          token,
          body
        });
      } else {
        await apiRequest<Project>("/projects", {
          method: "POST",
          token,
          body
        });
      }
      resetForm();
      await loadProjects();
    } catch (err) {
      setError((err as Error).message);
    }
  }

  function startEdit(project: Project) {
    const canonicalCategory = canonicalFolderName(project.category);
    setEditingProjectId(project.id);
    setEditingOriginalStatus(project.status);
    setName(project.name);
    setDescription(project.description);
    setDurationDays(String(project.duration_days));
    setCurrencyCode(project.currency_code);
    setSecondaryCurrencyCode(project.secondary_currency_code || "");
    setFolderChoice(canonicalCategory);
  }

  function resetForm() {
    setEditingProjectId(null);
    setEditingOriginalStatus(null);
    setName("");
    setDescription("");
    setDurationDays("14");
    setCurrencyCode("EUR");
    setSecondaryCurrencyCode("");
    setFolderChoice(activeFolder !== ALL_FOLDERS_KEY ? activeFolder : DEFAULT_FOLDER);
  }

  async function moveProjectToFolder(project: Project, nextFolder: string) {
    if (!token || project.current_user_role !== "PM") return;
    const resolvedFolder = canonicalFolderName(nextFolder);
    if (resolvedFolder === canonicalFolderName(project.category)) return;

    setMovingProjectId(project.id);
    setError("");
    try {
      await apiRequest<Project>(`/projects/${project.id}`, {
        method: "PUT",
        token,
        body: {
          name: project.name,
          description: project.description,
          duration_days: project.duration_days,
          currency_code: project.currency_code,
          secondary_currency_code: project.secondary_currency_code || undefined,
          category: resolvedFolder
        }
      });
      await loadProjects();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setMovingProjectId(null);
    }
  }

  function toggleFolderProjectSelection(projectId: string, checked: boolean) {
    setFolderDraftProjectIds((prev) => {
      if (checked) {
        return prev.includes(projectId) ? prev : [...prev, projectId];
      }
      return prev.filter((id) => id !== projectId);
    });
  }

  async function createFolderFromComposer() {
    const folderName = canonicalFolderName(folderDraftName);
    if (!folderName.trim() || folderName.length < 2) {
      setError(tr("Укажите название папки длиной не меньше 2 символов.", "Specify a folder name with at least 2 characters."));
      return;
    }
    if (folderNames.includes(folderName)) {
      setError(tr("Папка с таким названием уже существует.", "A folder with this name already exists."));
      return;
    }

    setError("");
    setFolderActionBusy(true);
    try {
      const record = createFolderRecord(folderName, folderDraftColor, folderDraftIcon);
      setCustomFolders((prev) => normalizeFolderRecords([...prev, record], locale));
      const toMove = projects.filter((project) => folderDraftProjectIds.includes(project.id) && project.current_user_role === "PM");
      for (const project of toMove) {
        await moveProjectToFolder(project, folderName);
      }
      setFolderDraftName("");
      setFolderDraftProjectIds([]);
      setFolderDraftColor("emerald");
      setFolderDraftIcon("folder");
      setFolderComposerOpen(false);
      setActiveFolder(folderName);
      setFolderChoice(folderName);
    } finally {
      setFolderActionBusy(false);
    }
  }

  async function deleteFolder(folderName: string) {
    const normalized = canonicalFolderName(folderName);
    if (normalized === DEFAULT_FOLDER) return;
    const folderProjects = projects.filter((project) => canonicalFolderName(project.category) === normalized);
    if (folderProjects.some((project) => project.current_user_role !== "PM")) {
      setError(tr("Папку нельзя удалить, пока в ней есть проекты без прав администратора у вас.", "This folder cannot be deleted while it contains projects you do not administer."));
      return;
    }

    const confirmed = await confirmAction({
      title: tr("Удалить папку?", "Delete folder?"),
      message: folderProjects.length > 0
        ? tr(
            `Папка «${folderLabel(normalized)}» будет удалена, а ${folderProjects.length} проект(ов) перемещены в папку «${t("projects.general")}".`,
            `Folder "${folderLabel(normalized)}" will be deleted, and ${folderProjects.length} project(s) will be moved to "${t("projects.general")}".`
          )
        : tr(
            `Папка «${folderLabel(normalized)}» будет удалена.`,
            `Folder "${folderLabel(normalized)}" will be deleted.`
          ),
      confirmText: tr("Удалить папку", "Delete folder"),
      cancelText: t("projects.form.cancel"),
      tone: "danger"
    });
    if (!confirmed) return;

    setFolderActionBusy(true);
    setError("");
    try {
      for (const project of folderProjects) {
        await moveProjectToFolder(project, DEFAULT_FOLDER);
      }
      setCustomFolders((prev) => prev.filter((folder) => canonicalFolderName(folder.name) !== normalized));
      if (activeFolder === normalized) {
        setActiveFolder(ALL_FOLDERS_KEY);
      }
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setFolderActionBusy(false);
    }
  }

  async function leaveProject(project: Project) {
    const confirmed = await confirmAction({
      title: t("projects.confirm.leave_title"),
      message: `${t("projects.confirm.leave_message")} «${project.name}».`,
      confirmText: t("projects.confirm.leave_button"),
      cancelText: t("projects.form.cancel"),
      tone: "danger"
    });
    if (!confirmed) return;

    setError("");
    try {
      await apiRequest(`/projects/${project.id}/leave`, { method: "POST", token });
      if (editingProjectId === project.id) {
        resetForm();
      }
      await loadProjects();
    } catch (err) {
      setError((err as Error).message);
    }
  }

  async function deleteProject(project: Project) {
    const confirmed = await confirmAction({
      title: t("projects.confirm.delete_title"),
      message: `${t("projects.confirm.delete_message")} «${project.name}».`,
      confirmText: t("projects.confirm.delete_button"),
      cancelText: t("projects.form.cancel"),
      tone: "danger"
    });
    if (!confirmed) return;

    setError("");
    try {
      await apiRequest(`/projects/${project.id}`, { method: "DELETE", token });
      if (editingProjectId === project.id) {
        resetForm();
      }
      await loadProjects();
    } catch (err) {
      setError((err as Error).message);
    }
  }

  if (!ready) return null;

  const durationLocked = Boolean(editingProjectId && editingOriginalStatus && editingOriginalStatus !== "preparation");
  const activeFolderLabel = activeFolder === ALL_FOLDERS_KEY ? tr("Все папки", "All folders") : folderLabel(activeFolder);

  return (
    <AppShell title={t("projects.page_title")} className="projects-shell">
      {error && <p style={{ color: "var(--danger)", margin: 0 }}>{error}</p>}

      <ProjectsKanbanSection
        tr={tr}
        t={t}
        projects={projects}
        columns={KANBAN_COLUMNS}
        folderLabel={(folderName) => folderLabel(canonicalFolderName(folderName))}
      />

      <ProjectsFolderExplorerSection
        tr={tr}
        t={t}
        activeFolder={activeFolder}
        allFoldersKey={ALL_FOLDERS_KEY}
        folderRecords={folderRecords}
        folderDropTarget={folderDropTarget}
        draggedProjectId={draggedProjectId}
        folderActionBusy={folderActionBusy}
        folderComposerOpen={folderComposerOpen}
        folderDraftName={folderDraftName}
        folderDraftColor={folderDraftColor}
        folderDraftIcon={folderDraftIcon}
        folderDraftProjectIds={folderDraftProjectIds}
        folderProjectOptions={folderProjectOptions}
        canDeleteActiveFolder={canDeleteActiveFolder}
        colorToken={colorToken}
        folderGlyph={folderGlyph}
        folderLabel={(folderName) => folderLabel(canonicalFolderName(folderName))}
        allFoldersSummary={allFoldersSummary}
        folderSummaryFor={folderSummaryFor}
        folderNames={folderNames}
        onSelectFolder={setActiveFolder}
        onToggleComposer={() => setFolderComposerOpen((prev) => !prev)}
        onDeleteFolder={() => void deleteFolder(activeFolder)}
        onFolderDragOver={(folderName, e) => {
          if (!draggedProjectId) return;
          e.preventDefault();
          e.dataTransfer.dropEffect = "move";
          setFolderDropTarget(folderName);
        }}
        onFolderDragLeave={(folderName) => {
          if (folderDropTarget === folderName) setFolderDropTarget(null);
        }}
        onFolderDrop={async (folderName, e) => {
          if (!draggedProjectId) return;
          e.preventDefault();
          const dragged = projects.find((project) => project.id === draggedProjectId);
          setFolderDropTarget(null);
          setDraggedProjectId(null);
          if (!dragged) return;
          await moveProjectToFolder(dragged, folderName);
        }}
        onFolderDraftNameChange={setFolderDraftName}
        onFolderDraftColorChange={setFolderDraftColor}
        onFolderDraftIconChange={setFolderDraftIcon}
        onToggleFolderProjectSelection={toggleFolderProjectSelection}
        onCreateFolder={() => void createFolderFromComposer()}
        projectStatusLabels={PROJECT_STATUS_LABELS}
      />

      <div className="grid two projects-main-grid">
        <ProjectsBrowserSection
          tr={tr}
          t={t}
          activeFolderLabel={activeFolderLabel}
          filteredProjects={filteredProjects}
          draggedProjectId={draggedProjectId}
          colorToken={colorToken}
          folderGlyph={folderGlyph}
          folderRecordFor={folderRecordFor}
          canonicalFolderName={canonicalFolderName}
          folderLabel={folderLabel}
          projectRoleLabels={PROJECT_ROLE_LABELS}
          projectStatusLabels={PROJECT_STATUS_LABELS}
          onContextMenu={(projectId, x, y) => setProjectMenu({ projectId, x, y })}
          onProjectDragStart={(projectId, e) => {
            const project = projects.find((item) => item.id === projectId);
            if (!project || project.current_user_role !== "PM") return;
            setDraggedProjectId(project.id);
            e.dataTransfer.effectAllowed = "move";
            e.dataTransfer.setData("text/plain", project.id);
          }}
          onProjectDragEnd={() => {
            setDraggedProjectId(null);
            setFolderDropTarget(null);
          }}
          onEditProject={startEdit}
          onDeleteProject={(project) => void deleteProject(project)}
          onLeaveProject={(project) => void leaveProject(project)}
        />

        <ProjectFormSection
          tr={tr}
          t={t}
          editingProjectId={editingProjectId}
          name={name}
          description={description}
          durationDays={durationDays}
          durationLocked={durationLocked}
          currencyCode={currencyCode}
          secondaryCurrencyCode={secondaryCurrencyCode}
          folderChoice={folderChoice}
          folderNames={folderNames}
          folderLabel={folderLabel}
          onSubmit={upsertProject}
          onNameChange={setName}
          onDescriptionChange={setDescription}
          onDurationChange={setDurationDays}
          onCurrencyChange={setCurrencyCode}
          onSecondaryCurrencyChange={setSecondaryCurrencyCode}
          onFolderChoiceChange={setFolderChoice}
          onReset={resetForm}
        />
      </div>

      <ProjectContextMenu
        tr={tr}
        projectMenu={projectMenu}
        project={projects.find((item) => item.id === projectMenu?.projectId) ?? null}
        folderNames={folderNames}
        folderGlyph={folderGlyph}
        folderRecordFor={folderRecordFor}
        folderLabel={folderLabel}
        canonicalFolderName={canonicalFolderName}
        onMoveProject={(folderName) => {
          const project = projects.find((item) => item.id === projectMenu?.projectId);
          if (project) {
            void moveProjectToFolder(project, folderName);
          }
        }}
        onClose={() => setProjectMenu(null)}
      />
    </AppShell>
  );
}
