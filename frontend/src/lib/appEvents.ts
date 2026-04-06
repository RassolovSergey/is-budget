export const INVITES_UPDATED_EVENT = "invites-updated";
export const PROJECTS_UPDATED_EVENT = "projects-updated";

export function dispatchInvitesUpdated() {
  if (typeof window === "undefined") return;
  window.dispatchEvent(new CustomEvent(INVITES_UPDATED_EVENT));
}

export function dispatchProjectsUpdated() {
  if (typeof window === "undefined") return;
  window.dispatchEvent(new CustomEvent(PROJECTS_UPDATED_EVENT));
}
