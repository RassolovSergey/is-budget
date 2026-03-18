import type { ProjectRole } from "./types";

export type ProjectRoleView = {
  code: ProjectRole;
  title: string;
  subtitle: string;
};

export const PROJECT_ROLE_ORDER: ProjectRole[] = ["TEAM", "CUSTOMER", "ANALYST", "PM"];
export const PROJECT_ROLE_ASSIGNABLE_ORDER: ProjectRole[] = ["TEAM", "CUSTOMER", "ANALYST"];

export function projectRoleView(t: (key: string) => string, role: ProjectRole): ProjectRoleView {
  switch (role) {
    case "PM":
      return { code: "PM", title: t("role.pm"), subtitle: t("role.desc.pm") };
    case "ANALYST":
      return { code: "ANALYST", title: t("role.analyst"), subtitle: t("role.desc.analyst") };
    case "TEAM":
      return { code: "TEAM", title: t("role.team"), subtitle: t("role.desc.team") };
    case "CUSTOMER":
      return { code: "CUSTOMER", title: t("role.customer"), subtitle: t("role.desc.customer") };
    default:
      return { code: role, title: t("role.default"), subtitle: "" };
  }
}

export function projectRoleOptions(
  t: (key: string) => string,
  opts?: { includeAdmin?: boolean }
): ProjectRoleView[] {
  const roles = opts?.includeAdmin ? PROJECT_ROLE_ORDER : PROJECT_ROLE_ASSIGNABLE_ORDER;
  return roles.map((role) => projectRoleView(t, role));
}
