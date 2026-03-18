"use client";

import { useRouter } from "next/navigation";
import type { Project, CurrencyCode, ProjectRole, ProjectStatus } from "../../lib/types";
import type { FolderRecord, FolderColor, FolderIcon, ProjectMenuState } from "./page-types";

type Translator = (key: string) => string;
type LocalTranslator = (ru: string, en: string) => string;

type ProjectsKanbanSectionProps = {
  tr: LocalTranslator;
  t: Translator;
  projects: Project[];
  columns: { key: ProjectStatus; labelKey: string }[];
  folderLabel: (folderName: string) => string;
};

type ProjectsFolderExplorerSectionProps = {
  tr: LocalTranslator;
  t: Translator;
  projects: Project[];
  activeFolder: string;
  allFoldersKey: string;
  folderRecords: FolderRecord[];
  folderDropTarget: string | null;
  draggedProjectId: string | null;
  folderActionBusy: boolean;
  folderComposerOpen: boolean;
  folderDraftName: string;
  folderDraftColor: FolderColor;
  folderDraftIcon: FolderIcon;
  folderDraftProjectIds: string[];
  folderProjectOptions: Project[];
  canDeleteActiveFolder: boolean;
  colorToken: (color: FolderColor) => string;
  folderGlyph: (icon: FolderIcon) => string;
  folderLabel: (folderName: string) => string;
  folderNames: string[];
  onSelectFolder: (folderName: string) => void;
  onToggleComposer: () => void;
  onDeleteFolder: () => void;
  onFolderDragOver: (folderName: string, e: React.DragEvent<HTMLButtonElement>) => void;
  onFolderDragLeave: (folderName: string) => void;
  onFolderDrop: (folderName: string, e: React.DragEvent<HTMLButtonElement>) => void;
  onFolderDraftNameChange: (value: string) => void;
  onFolderDraftColorChange: (color: FolderColor) => void;
  onFolderDraftIconChange: (icon: FolderIcon) => void;
  onToggleFolderProjectSelection: (projectId: string, checked: boolean) => void;
  onCreateFolder: () => void;
  projectStatusLabels: Record<ProjectStatus, string>;
};

type ProjectsBrowserSectionProps = {
  tr: LocalTranslator;
  t: Translator;
  activeFolderLabel: string;
  filteredProjects: Project[];
  draggedProjectId: string | null;
  colorToken: (color: FolderColor) => string;
  folderGlyph: (icon: FolderIcon) => string;
  folderRecordFor: (folderName: string) => FolderRecord;
  canonicalFolderName: (value?: string | null) => string;
  folderLabel: (folderName: string) => string;
  projectRoleLabels: Record<ProjectRole, string>;
  projectStatusLabels: Record<ProjectStatus, string>;
  onContextMenu: (projectId: string, x: number, y: number) => void;
  onProjectDragStart: (projectId: string, e: React.DragEvent<HTMLElement>) => void;
  onProjectDragEnd: () => void;
  onEditProject: (project: Project) => void;
  onDeleteProject: (project: Project) => void;
  onLeaveProject: (project: Project) => void;
};

type ProjectFormSectionProps = {
  tr: LocalTranslator;
  t: Translator;
  editingProjectId: string | null;
  name: string;
  description: string;
  durationDays: string;
  durationLocked: boolean;
  currencyCode: CurrencyCode;
  secondaryCurrencyCode: string;
  folderChoice: string;
  folderNames: string[];
  folderLabel: (folderName: string) => string;
  onSubmit: (e: React.FormEvent) => void;
  onNameChange: (value: string) => void;
  onDescriptionChange: (value: string) => void;
  onDurationChange: (value: string) => void;
  onCurrencyChange: (value: CurrencyCode) => void;
  onSecondaryCurrencyChange: (value: string) => void;
  onFolderChoiceChange: (value: string) => void;
  onReset: () => void;
};

type ProjectContextMenuProps = {
  tr: LocalTranslator;
  projectMenu: ProjectMenuState;
  project: Project | null;
  folderNames: string[];
  folderGlyph: (icon: FolderIcon) => string;
  folderRecordFor: (folderName: string) => FolderRecord;
  folderLabel: (folderName: string) => string;
  canonicalFolderName: (value?: string | null) => string;
  onMoveProject: (folderName: string) => void;
  onClose: () => void;
};

export function ProjectsKanbanSection({
  tr,
  t,
  projects,
  columns,
  folderLabel
}: ProjectsKanbanSectionProps) {
  const router = useRouter();

  return (
    <section className="card projects-kanban-card projects-kanban-card-wide">
      <div className="section-head">
        <div>
          <h2 className="title" style={{ marginBottom: 6 }}>{tr("Доска по статусам", "Status board")}</h2>
          <p className="subtle" style={{ margin: 0 }}>
            {tr("Подготовка, активные и завершённые проекты на одной ширине, без сжатия по колонкам.", "Preparation, active, and completed projects across the full width.")}
          </p>
        </div>
      </div>
      <div className="kanban-board kanban-board-wide">
        {columns.map((column) => {
          const items = projects.filter((project) => project.status === column.key);
          return (
            <div key={column.key} className="kanban-column">
              <div className="kanban-column-head">
                <h3>{t(column.labelKey)}</h3>
                <span>{items.length}</span>
              </div>
              <div className="kanban-column-list">
                {items.map((project) => (
                  <button
                    key={project.id}
                    type="button"
                    className="kanban-item"
                    onClick={() => router.push(`/projects/${project.id}`)}
                  >
                    <strong>{project.name}</strong>
                    <span>
                      {folderLabel(project.category)} · {project.duration_days} {t("projects.days_short")} · {project.currency_code}
                      {project.secondary_currency_code ? ` / ${project.secondary_currency_code}` : ""}
                    </span>
                  </button>
                ))}
                {!items.length && <p className="subtle kanban-empty">{t("projects.kanban.empty")}</p>}
              </div>
            </div>
          );
        })}
      </div>
    </section>
  );
}

export function ProjectsFolderExplorerSection({
  tr,
  t,
  projects,
  activeFolder,
  allFoldersKey,
  folderRecords,
  folderDropTarget,
  draggedProjectId,
  folderActionBusy,
  folderComposerOpen,
  folderDraftName,
  folderDraftColor,
  folderDraftIcon,
  folderDraftProjectIds,
  folderProjectOptions,
  canDeleteActiveFolder,
  colorToken,
  folderGlyph,
  folderLabel,
  folderNames,
  onSelectFolder,
  onToggleComposer,
  onDeleteFolder,
  onFolderDragOver,
  onFolderDragLeave,
  onFolderDrop,
  onFolderDraftNameChange,
  onFolderDraftColorChange,
  onFolderDraftIconChange,
  onToggleFolderProjectSelection,
  onCreateFolder,
  projectStatusLabels
}: ProjectsFolderExplorerSectionProps) {
  return (
    <section className="card projects-folders-card projects-folder-explorer-card">
      <div className="section-head">
        <div>
          <h2 className="title" style={{ marginBottom: 6 }}>{t("projects.folders_title")}</h2>
          <p className="subtle" style={{ margin: 0 }}>
            {tr(
              "Управляйте папками как в файловом менеджере: создавайте их отдельно от проектов и переносите проекты внутрь drag-and-drop или через контекстное меню.",
              "Manage folders like in a file manager: create them separately from projects and move projects inside via drag-and-drop or the context menu."
            )}
          </p>
        </div>
        <div className="folder-section-actions">
          {activeFolder !== allFoldersKey && canDeleteActiveFolder && (
            <button type="button" className="secondary" onClick={onDeleteFolder} disabled={folderActionBusy}>
              {tr("Удалить папку", "Delete folder")}
            </button>
          )}
          <button type="button" className="secondary" onClick={onToggleComposer}>
            {folderComposerOpen ? tr("Скрыть форму", "Hide form") : tr("Новая папка", "New folder")}
          </button>
        </div>
      </div>

      <div className="folder-rail folder-rail-drive">
        <button
          type="button"
          className={`folder-drive-card ${activeFolder === allFoldersKey ? "active" : ""}`}
          onClick={() => onSelectFolder(allFoldersKey)}
        >
          <span className="folder-drive-cap" />
          <div className="folder-drive-icon shell-folder-icon shell-folder-icon-neutral">◧</div>
          <div className="folder-drive-body">
            <strong>{tr("Все папки", "All folders")}</strong>
            <span>{projects.length} {t("projects.count").toLowerCase()}</span>
          </div>
        </button>

        {folderRecords.map((folder) => {
          const count = projects.filter((project) => folderLabel(project.category) === folderLabel(folder.name)).length;
          return (
            <button
              key={folder.id}
              type="button"
              className={`folder-drive-card ${activeFolder === folder.name ? "active" : ""} ${folderDropTarget === folder.name ? "drop-target" : ""}`}
              style={{ ["--folder-accent" as string]: colorToken(folder.color) }}
              onClick={() => onSelectFolder(folder.name)}
              onDragOver={(e) => onFolderDragOver(folder.name, e)}
              onDragLeave={() => onFolderDragLeave(folder.name)}
              onDrop={(e) => onFolderDrop(folder.name, e)}
            >
              <span className="folder-drive-cap" />
              <div className="folder-drive-icon">{folderGlyph(folder.icon)}</div>
              <div className="folder-drive-body">
                <strong>{folderLabel(folder.name)}</strong>
                <span>{count} {t("projects.count").toLowerCase()}</span>
              </div>
            </button>
          );
        })}

        <button
          type="button"
          className={`folder-drive-card folder-drive-card-create ${folderComposerOpen ? "active" : ""}`}
          onClick={onToggleComposer}
        >
          <span className="folder-drive-cap" />
          <div className="folder-drive-icon shell-folder-icon shell-folder-icon-create">+</div>
          <div className="folder-drive-body">
            <strong>{tr("Создать папку", "Create folder")}</strong>
            <span>{tr("Пустая или с проектами", "Empty or prefilled")}</span>
          </div>
        </button>
      </div>

      {folderComposerOpen && (
        <div className="folder-composer folder-composer-drive">
          <div className="folder-composer-grid folder-composer-grid-drive">
            <label>
              {tr("Название папки", "Folder name")}
              <input value={folderDraftName} onChange={(e) => onFolderDraftNameChange(e.target.value)} placeholder={tr("Например: Terrain · Launch", "For example: Terrain · Launch")} />
            </label>
            <label>
              {tr("Цвет", "Color")}
              <div className="folder-color-picker">
                {(["slate", "emerald", "violet", "amber", "rose", "cyan"] as FolderColor[]).map((color) => (
                  <button
                    key={color}
                    type="button"
                    className={`folder-color-swatch ${folderDraftColor === color ? "active" : ""}`}
                    style={{ ["--folder-accent" as string]: colorToken(color) }}
                    onClick={() => onFolderDraftColorChange(color)}
                    aria-label={color}
                  />
                ))}
              </div>
            </label>
            <label>
              {tr("Значок", "Icon")}
              <div className="folder-icon-picker">
                {(["folder", "spark", "stack", "badge"] as FolderIcon[]).map((icon) => (
                  <button
                    key={icon}
                    type="button"
                    className={`folder-icon-chip ${folderDraftIcon === icon ? "active" : ""}`}
                    onClick={() => onFolderDraftIconChange(icon)}
                  >
                    <span>{folderGlyph(icon)}</span>
                  </button>
                ))}
              </div>
            </label>
            <div className="folder-composer-actions folder-composer-actions-drive">
              <button type="button" onClick={onCreateFolder} disabled={folderActionBusy}>
                {folderActionBusy ? tr("Сохраняю...", "Saving...") : tr("Создать папку", "Create folder")}
              </button>
            </div>
          </div>
          <div className="folder-composer-projects">
            <div className="folder-composer-head">
              <strong>{tr("Сразу переместить проекты", "Move projects immediately")}</strong>
              <span className="subtle">{tr("Необязательно. Можно выбрать несколько проектов и сразу отправить их в новую папку.", "Optional. Pick several projects and place them into the new folder immediately.")}</span>
            </div>
            <div className="folder-composer-list">
              {folderProjectOptions.map((project) => (
                <label key={project.id} className="folder-composer-item">
                  <input
                    type="checkbox"
                    checked={folderDraftProjectIds.includes(project.id)}
                    onChange={(e) => onToggleFolderProjectSelection(project.id, e.target.checked)}
                  />
                  <div>
                    <strong>{project.name}</strong>
                    <span>{folderLabel(project.category)} · {projectStatusLabels[project.status]}</span>
                  </div>
                </label>
              ))}
              {!folderProjectOptions.length && (
                <p className="subtle" style={{ margin: 0 }}>
                  {tr("Нет проектов с правами администратора для быстрого переноса.", "There are no administrator-owned projects available for quick move.")}
                </p>
              )}
            </div>
          </div>
        </div>
      )}
    </section>
  );
}

export function ProjectsBrowserSection({
  tr,
  t,
  activeFolderLabel,
  filteredProjects,
  draggedProjectId,
  colorToken,
  folderGlyph,
  folderRecordFor,
  canonicalFolderName,
  folderLabel,
  projectRoleLabels,
  projectStatusLabels,
  onContextMenu,
  onProjectDragStart,
  onProjectDragEnd,
  onEditProject,
  onDeleteProject,
  onLeaveProject
}: ProjectsBrowserSectionProps) {
  const router = useRouter();

  return (
    <section className="card projects-list-card projects-browser-card">
      <div className="section-head">
        <div>
          <h2 className="title" style={{ marginBottom: 6 }}>{t("projects.your_projects")}</h2>
          <p className="subtle" style={{ margin: 0 }}>
            {tr("Открыта папка", "Opened folder")}: <strong style={{ color: "var(--ink)" }}>{activeFolderLabel}</strong>
          </p>
        </div>
      </div>

      <div className="project-browser-grid">
        {filteredProjects.map((project) => {
          const folder = folderRecordFor(project.category);
          return (
            <article
              key={project.id}
              className={`project-drive-card ${draggedProjectId === project.id ? "dragging" : ""}`}
              draggable={project.current_user_role === "PM"}
              style={{ ["--folder-accent" as string]: colorToken(folder.color) }}
              onContextMenu={(e) => {
                if (project.current_user_role !== "PM") return;
                e.preventDefault();
                onContextMenu(project.id, e.clientX, e.clientY);
              }}
              onDragStart={(e) => onProjectDragStart(project.id, e)}
              onDragEnd={onProjectDragEnd}
            >
              <div className="project-drive-head">
                <div className="project-drive-icon">{folderGlyph(folder.icon)}</div>
                <div className="project-drive-head-copy">
                  <strong>{project.name}</strong>
                  <span>{folderLabel(canonicalFolderName(project.category))}</span>
                </div>
              </div>

              <p className="project-description">{project.description || t("projects.no_description")}</p>

              <div className="project-badges">
                <span className="project-badge">{projectStatusLabels[project.status]}</span>
                <span className="project-badge project-badge-role">
                  {project.current_user_role ? projectRoleLabels[project.current_user_role] : t("role.default")}
                </span>
                <span className="project-badge">{project.duration_days} {t("projects.days_short")}</span>
                <span className="project-badge">{project.currency_code}{project.secondary_currency_code ? ` / ${project.secondary_currency_code}` : ""}</span>
              </div>

              <div className="project-drive-actions">
                <button type="button" className="secondary" onClick={() => router.push(`/projects/${project.id}`)}>
                  {t("projects.open")}
                </button>
                {project.current_user_role === "PM" && (
                  <button type="button" className="secondary" onClick={() => onEditProject(project)}>
                    {t("projects.edit")}
                  </button>
                )}
                {project.current_user_role === "PM" ? (
                  <button type="button" className="warn" onClick={() => onDeleteProject(project)}>
                    {t("projects.delete")}
                  </button>
                ) : (
                  <button type="button" className="warn" onClick={() => onLeaveProject(project)}>
                    {t("projects.leave")}
                  </button>
                )}
              </div>
            </article>
          );
        })}
        {!filteredProjects.length && <p className="subtle" style={{ margin: 0 }}>{t("projects.empty")}</p>}
      </div>
    </section>
  );
}

export function ProjectFormSection({
  tr,
  t,
  editingProjectId,
  name,
  description,
  durationDays,
  durationLocked,
  currencyCode,
  secondaryCurrencyCode,
  folderChoice,
  folderNames,
  folderLabel,
  onSubmit,
  onNameChange,
  onDescriptionChange,
  onDurationChange,
  onCurrencyChange,
  onSecondaryCurrencyChange,
  onFolderChoiceChange,
  onReset
}: ProjectFormSectionProps) {
  return (
    <section className="card project-form-card">
      <div className="section-head">
        <div>
          <h2 className="title" style={{ marginBottom: 6 }}>{editingProjectId ? t("projects.form.edit_title") : t("projects.form.create_title")}</h2>
          <p className="subtle" style={{ margin: 0 }}>
            {tr(
              "Проект появится в выбранной папке. Новые папки создаются отдельно в верхнем блоке, как в файловом менеджере.",
              "The project will appear in the selected folder. New folders are created separately in the top block, like in a file manager."
            )}
          </p>
        </div>
      </div>

      <form className="grid" onSubmit={onSubmit}>
        <label>
          {t("projects.form.name")}
          <input value={name} onChange={(e) => onNameChange(e.target.value)} required />
        </label>
        <label>
          {t("projects.form.description")}
          <textarea value={description} onChange={(e) => onDescriptionChange(e.target.value)} rows={4} />
        </label>
        <label>
          {tr("Папка проекта", "Project folder")}
          <select value={folderChoice} onChange={(e) => onFolderChoiceChange(e.target.value)}>
            {folderNames.map((folderName) => (
              <option key={folderName} value={folderName}>
                {folderLabel(folderName)}
              </option>
            ))}
          </select>
        </label>
        <label>
          {t("projects.form.duration")}
          <input type="number" min={3} max={30} value={durationDays} onChange={(e) => onDurationChange(e.target.value)} disabled={durationLocked} required />
        </label>
        {durationLocked && <p className="subtle" style={{ margin: 0 }}>{t("projects.form.duration_locked")}</p>}
        <label>
          {t("projects.form.currency")}
          <select value={currencyCode} onChange={(e) => onCurrencyChange(e.target.value as CurrencyCode)}>
            <option value="EUR">EUR (€)</option>
            <option value="USD">USD ($)</option>
            <option value="RUB">RUB (₽)</option>
          </select>
        </label>
        <label>
          {t("projects.form.secondary_currency")}
          <select value={secondaryCurrencyCode} onChange={(e) => onSecondaryCurrencyChange(e.target.value)}>
            <option value="">{t("projects.form.none_selected")}</option>
            {currencyCode !== "EUR" && <option value="EUR">EUR (€)</option>}
            {currencyCode !== "USD" && <option value="USD">USD ($)</option>}
            {currencyCode !== "RUB" && <option value="RUB">RUB (₽)</option>}
          </select>
        </label>
        <div style={{ display: "flex", gap: 8 }}>
          <button>{editingProjectId ? t("projects.form.save") : t("projects.form.create")}</button>
          {editingProjectId && (
            <button type="button" className="secondary" onClick={onReset}>
              {t("projects.form.cancel")}
            </button>
          )}
        </div>
      </form>
    </section>
  );
}

export function ProjectContextMenu({
  tr,
  projectMenu,
  project,
  folderNames,
  folderGlyph,
  folderRecordFor,
  folderLabel,
  canonicalFolderName,
  onMoveProject,
  onClose
}: ProjectContextMenuProps) {
  if (!projectMenu || !project || project.current_user_role !== "PM") {
    return null;
  }

  return (
    <div
      className="project-context-menu"
      style={{ top: projectMenu.y + 8, left: projectMenu.x + 8 }}
      onClick={(e) => e.stopPropagation()}
    >
      <div className="project-context-menu-head">
        <strong>{tr("Переместить в папку", "Move to folder")}</strong>
        <span>{project.name}</span>
      </div>
      <div className="project-context-menu-list">
        {folderNames
          .filter((folderName) => canonicalFolderName(folderName) !== canonicalFolderName(project.category))
          .map((folderName) => (
            <button
              key={folderName}
              type="button"
              className="project-context-menu-item"
              onClick={() => {
                onMoveProject(folderName);
                onClose();
              }}
            >
              <span>{folderGlyph(folderRecordFor(folderName).icon)}</span>
              <span>{folderLabel(folderName)}</span>
            </button>
          ))}
      </div>
    </div>
  );
}
