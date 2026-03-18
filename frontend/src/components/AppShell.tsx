"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";

import { LanguageToggle } from "./LanguageToggle";
import { NotificationsBell } from "./NotificationsBell";
import { useLocale } from "./LocaleProvider";
import { ThemeToggle } from "./ThemeToggle";
import { clearToken } from "../lib/session";

export function AppShell({
  title,
  children,
  className = ""
}: {
  title: string;
  children: React.ReactNode;
  className?: string;
}) {
  const pathname = usePathname();
  const router = useRouter();
  const { t } = useLocale();
  const isHistoryActive = pathname.startsWith("/historical-campaigns");
  const isProjectsActive = pathname.startsWith("/projects");
  const isProfileActive = pathname.startsWith("/profile");
  const sectionLabel = pathname.startsWith("/profile")
    ? t("app.profile")
    : isHistoryActive
      ? t("app.history")
      : isProjectsActive
        ? t("app.projects")
        : "IS Budget";

  return (
    <div className={`page page-shell ${className}`.trim()}>
      <div className="shell-orb shell-orb-a" aria-hidden="true" />
      <div className="shell-orb shell-orb-b" aria-hidden="true" />
      <header className="card shell-header">
        <div className="shell-heading">
          <span className="shell-eyebrow">{sectionLabel}</span>
          <h1>{title}</h1>
        </div>
        <div className="topbar-controls">
          <div className="topbar-tools">
            <ThemeToggle />
            <LanguageToggle />
            <NotificationsBell />
          </div>
          <div className="topbar-switch topbar-main-nav" aria-label={t("app.navigation")}>
            <Link className={`topbar-switch-item ${isProfileActive ? "active" : ""}`} href="/profile">
              {t("app.profile")}
            </Link>
            <Link className={`topbar-switch-item ${isHistoryActive ? "active" : ""}`} href="/historical-campaigns">
              {t("app.history")}
            </Link>
            <Link className={`topbar-switch-item ${isProjectsActive ? "active" : ""}`} href="/projects">
              {t("app.projects")}
            </Link>
          </div>
          <button
            className="secondary topbar-logout"
            onClick={() => {
              clearToken();
              router.push("/login");
            }}
          >
            {t("app.logout")}
          </button>
        </div>
      </header>
      <div className="shell-content">{children}</div>
    </div>
  );
}
