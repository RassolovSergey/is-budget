"use client";

import Link from "next/link";
import { useEffect, useState } from "react";

import { AppShell } from "../../components/AppShell";
import { useLocale } from "../../components/LocaleProvider";
import { apiRequest } from "../../lib/api";
import { toIntlLocale } from "../../lib/i18n";
import { projectRoleView } from "../../lib/projectRoles";
import type { IncomingProjectInvite } from "../../lib/types";
import { useAuthGuard } from "../../lib/useAuthGuard";

export default function NotificationsPage() {
  const { t, locale } = useLocale();
  const { token, ready } = useAuthGuard();
  const [items, setItems] = useState<IncomingProjectInvite[]>([]);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [pendingToken, setPendingToken] = useState("");

  const tr = (ru: string, en: string) => (locale === "en" ? en : ru);

  async function load() {
    if (!token) return;
    try {
      setError("");
      const data = await apiRequest<IncomingProjectInvite[]>("/project-invites/incoming", { token });
      setItems(data);
    } catch (err) {
      setError((err as Error).message);
    }
  }

  useEffect(() => {
    if (ready) {
      load();
    }
  }, [ready]);

  async function acceptInvite(inviteToken: string) {
    if (!token) return;
    setPendingToken(inviteToken);
    setError("");
    setNotice("");
    try {
      await apiRequest(`/project-invites/${inviteToken}/accept`, {
        method: "POST",
        token
      });
      setNotice(tr("Приглашение принято.", "Invite accepted."));
      await load();
      window.dispatchEvent(new CustomEvent("invites-updated"));
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setPendingToken("");
    }
  }

  async function declineInvite(inviteToken: string) {
    if (!token) return;
    setPendingToken(inviteToken);
    setError("");
    setNotice("");
    try {
      await apiRequest(`/project-invites/${inviteToken}/decline`, {
        method: "POST",
        token
      });
      setNotice(tr("Приглашение отклонено.", "Invite declined."));
      await load();
      window.dispatchEvent(new CustomEvent("invites-updated"));
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setPendingToken("");
    }
  }

  if (!ready) return null;

  return (
    <AppShell title={t("app.notifications")}>
      {error && <p style={{ color: "#bc4749" }}>{error}</p>}
      {notice && <p style={{ color: "#2a9d8f" }}>{notice}</p>}

      <section className="card">
        <h2 className="title">{tr("Входящие приглашения", "Incoming invites")}</h2>
        <p className="subtle" style={{ marginTop: 0 }}>
          {tr("Принимайте или отклоняйте приглашения в проекты.", "Accept or decline project invitations.")}
        </p>

        {!items.length ? (
          <p className="subtle" style={{ marginBottom: 0 }}>
            {tr("Новых приглашений нет.", "No new invites.")}
          </p>
        ) : (
          <div className="grid" style={{ gap: 10 }}>
            {items.map((item) => {
              const roleView = projectRoleView(t, item.invited_role);
              const createdAt = new Date(item.created_at).toLocaleString(toIntlLocale(locale));
              const expiresAt = new Date(item.expires_at).toLocaleDateString(toIntlLocale(locale));
              return (
                <article key={item.id} className="card" style={{ padding: 12 }}>
                  <div style={{ display: "grid", gap: 6 }}>
                    <strong style={{ fontSize: "1.05rem" }}>{item.project_name}</strong>
                    <p className="subtle" style={{ margin: 0 }}>{item.project_description || tr("Без описания", "No description")}</p>
                    <p className="subtle" style={{ margin: 0 }}>
                      {tr("Роль", "Role")}: {roleView.title}
                    </p>
                    <p className="subtle" style={{ margin: 0 }}>
                      {tr("Пригласил", "Invited by")}: {item.invited_by_full_name || item.invited_by}
                    </p>
                    <p className="subtle" style={{ margin: 0 }}>
                      {tr("Валюта проекта", "Project currency")}: {item.project_currency_code}
                    </p>
                    <p className="subtle" style={{ margin: 0 }}>
                      {tr("Отправлено", "Sent")}: {createdAt} · {tr("Истекает", "Expires")}: {expiresAt}
                    </p>
                  </div>
                  <div style={{ display: "flex", flexWrap: "wrap", gap: 8, marginTop: 10 }}>
                    <button
                      type="button"
                      onClick={() => acceptInvite(item.token)}
                      disabled={pendingToken === item.token}
                    >
                      {tr("Принять", "Accept")}
                    </button>
                    <button
                      type="button"
                      className="warn"
                      onClick={() => declineInvite(item.token)}
                      disabled={pendingToken === item.token}
                    >
                      {tr("Отклонить", "Decline")}
                    </button>
                    <Link className="secondary topbar-link" href={`/projects/${item.project_id}`}>
                      {tr("Открыть проект", "Open project")}
                    </Link>
                  </div>
                </article>
              );
            })}
          </div>
        )}
      </section>
    </AppShell>
  );
}
