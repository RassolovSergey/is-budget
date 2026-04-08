"use client";

import Link from "next/link";
import { useEffect, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";

import { useLocale } from "./LocaleProvider";
import { apiRequest } from "../lib/api";
import { dispatchInvitesUpdated, dispatchProjectsUpdated, INVITES_UPDATED_EVENT } from "../lib/appEvents";
import { toIntlLocale } from "../lib/i18n";
import { projectRoleView } from "../lib/projectRoles";
import { getToken } from "../lib/session";
import type { IncomingProjectInvite } from "../lib/types";
import { useLiveRefresh } from "../lib/useLiveRefresh";

const MAX_VISIBLE_ITEMS = 6;

export function NotificationsBell() {
  const { locale, t } = useLocale();
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const [items, setItems] = useState<IncomingProjectInvite[]>([]);
  const [error, setError] = useState("");
  const [pendingToken, setPendingToken] = useState("");
  const rootRef = useRef<HTMLDivElement | null>(null);
  const buttonRef = useRef<HTMLButtonElement | null>(null);
  const popoverRef = useRef<HTMLDivElement | null>(null);
  const [popoverStyle, setPopoverStyle] = useState<{ top: number; right: number; width: number }>({
    top: 0,
    right: 0,
    width: 420
  });

  const tr = (ru: string, en: string) => (locale === "en" ? en : ru);

  const unreadCount = items.length;

  const topItems = useMemo(() => items.slice(0, MAX_VISIBLE_ITEMS), [items]);

  async function loadInvites() {
    const token = getToken();
    if (!token) {
      setItems([]);
      return;
    }
    setLoading(true);
    setError("");
    try {
      const data = await apiRequest<IncomingProjectInvite[]>("/project-invites/incoming", { token });
      setItems(data);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }

  async function acceptInvite(inviteToken: string) {
    const token = getToken();
    if (!token) return;
    setPendingToken(inviteToken);
    setError("");
    try {
      await apiRequest(`/project-invites/${inviteToken}/accept`, { method: "POST", token });
      await loadInvites();
      dispatchInvitesUpdated();
      dispatchProjectsUpdated();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setPendingToken("");
    }
  }

  async function declineInvite(inviteToken: string) {
    const token = getToken();
    if (!token) return;
    setPendingToken(inviteToken);
    setError("");
    try {
      await apiRequest(`/project-invites/${inviteToken}/decline`, { method: "POST", token });
      await loadInvites();
      dispatchInvitesUpdated();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setPendingToken("");
    }
  }

  useEffect(() => {
    loadInvites();
  }, []);

  useLiveRefresh(loadInvites, {
    enabled: true,
    intervalMs: 7000,
    eventNames: [INVITES_UPDATED_EVENT]
  });

  useEffect(() => {
    if (!open) return;
    const syncPopoverPosition = () => {
      const rect = buttonRef.current?.getBoundingClientRect();
      if (!rect) return;
      setPopoverStyle({
        top: rect.bottom + 10,
        right: Math.max(window.innerWidth - rect.right, 14),
        width: Math.min(420, window.innerWidth - 28)
      });
    };
    syncPopoverPosition();
    const onMouseDown = (event: MouseEvent) => {
      const target = event.target as Node;
      const clickedInsideRoot = !!rootRef.current?.contains(target);
      const clickedInsidePopover = !!popoverRef.current?.contains(target);
      if (!clickedInsideRoot && !clickedInsidePopover) {
        setOpen(false);
      }
    };
    const onEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") setOpen(false);
    };
    window.addEventListener("resize", syncPopoverPosition);
    window.addEventListener("scroll", syncPopoverPosition, true);
    document.addEventListener("mousedown", onMouseDown);
    document.addEventListener("keydown", onEscape);
    return () => {
      window.removeEventListener("resize", syncPopoverPosition);
      window.removeEventListener("scroll", syncPopoverPosition, true);
      document.removeEventListener("mousedown", onMouseDown);
      document.removeEventListener("keydown", onEscape);
    };
  }, [open]);

  useEffect(() => {
    if (open) {
      loadInvites();
    }
  }, [open]);

  const bellLabel = tr("Открыть уведомления", "Open notifications");

  return (
    <div ref={rootRef} className="notifications-bell-wrap">
      <button
        ref={buttonRef}
        type="button"
        className={`secondary notifications-bell-btn ${open ? "active" : ""}`}
        aria-label={bellLabel}
        title={bellLabel}
        onClick={() => setOpen((v) => !v)}
      >
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path d="M6 9a6 6 0 1 1 12 0v4.1l1.5 2.2c.4.6-.1 1.4-.9 1.4H5.4c-.8 0-1.3-.8-.9-1.4L6 13.1V9Z" />
          <path d="M10 18.8a2 2 0 0 0 4 0" />
        </svg>
        {unreadCount > 0 ? (
          <span className="topbar-badge notifications-bell-badge">{unreadCount > 99 ? "99+" : unreadCount}</span>
        ) : null}
      </button>

      {open && typeof document !== "undefined"
        ? createPortal(
            <div
              ref={popoverRef}
              className="notifications-popover card"
              role="dialog"
              aria-label={t("app.notifications")}
              style={{
                top: popoverStyle.top,
                right: popoverStyle.right,
                width: popoverStyle.width
              }}
            >
          <div className="notifications-popover-head">
            <strong>{t("app.notifications")}</strong>
            <Link className="notifications-open-all" href="/notifications" onClick={() => setOpen(false)}>
              {tr("Открыть все", "Open all")}
            </Link>
          </div>

          {loading ? <p className="subtle">{tr("Загрузка...", "Loading...")}</p> : null}
          {!loading && error ? <p className="notifications-error">{error}</p> : null}
          {!loading && !error && !topItems.length ? (
            <p className="subtle">{tr("Новых уведомлений нет.", "No new notifications.")}</p>
          ) : null}

          {!loading && !error && topItems.length ? (
            <div className="notifications-list">
              {topItems.map((item) => {
                const role = projectRoleView(t, item.invited_role);
                const expiresAt = new Date(item.expires_at).toLocaleDateString(toIntlLocale(locale));
                return (
                  <article key={item.id} className="notifications-item">
                    <div className="notifications-item-main">
                      <strong>{item.project_name}</strong>
                      <p className="subtle">{tr("Роль", "Role")}: {role.title}</p>
                      <p className="subtle">{tr("Истекает", "Expires")}: {expiresAt}</p>
                    </div>
                    <div className="notifications-item-actions">
                      <button
                        type="button"
                        className="notifications-action-accept"
                        disabled={pendingToken === item.token}
                        onClick={() => acceptInvite(item.token)}
                      >
                        {tr("Принять", "Accept")}
                      </button>
                      <button
                        type="button"
                        className="warn notifications-action-decline"
                        disabled={pendingToken === item.token}
                        onClick={() => declineInvite(item.token)}
                      >
                        {tr("Отклонить", "Decline")}
                      </button>
                    </div>
                  </article>
                );
              })}
            </div>
          ) : null}
            </div>,
            document.body
          )
        : null}
    </div>
  );
}
