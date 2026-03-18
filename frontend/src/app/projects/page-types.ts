import type { ProjectStatus } from "../../lib/types";

export const DEFAULT_FOLDER = "General";
export const ALL_FOLDERS_KEY = "__all__";
export const LOCAL_FOLDERS_STORAGE_KEY = "isbudget.project-folders.v3";

export const FOLDER_ICON_OPTIONS = ["folder", "spark", "stack", "badge"] as const;
export type FolderIcon = (typeof FOLDER_ICON_OPTIONS)[number];

export const FOLDER_COLOR_OPTIONS = ["slate", "emerald", "violet", "amber", "rose", "cyan"] as const;
export type FolderColor = (typeof FOLDER_COLOR_OPTIONS)[number];

export type FolderRecord = {
  id: string;
  name: string;
  color: FolderColor;
  icon: FolderIcon;
};

export type ProjectMenuState = {
  projectId: string;
  x: number;
  y: number;
} | null;

export const KANBAN_COLUMNS: { key: ProjectStatus; labelKey: string }[] = [
  { key: "preparation", labelKey: "projects.kanban.preparation" },
  { key: "active", labelKey: "projects.kanban.active" },
  { key: "completed", labelKey: "projects.kanban.completed" }
];
