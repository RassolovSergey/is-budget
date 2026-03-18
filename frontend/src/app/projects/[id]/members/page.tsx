"use client";

import { useEffect, useMemo, useState } from "react";
import { useParams } from "next/navigation";

import { AppShell } from "../../../../components/AppShell";
import { useConfirm } from "../../../../components/ConfirmProvider";
import { useLocale } from "../../../../components/LocaleProvider";
import { ProjectNav } from "../../../../components/ProjectNav";
import { apiRequest } from "../../../../lib/api";
import { parseMemberLookup } from "../../../../lib/memberLookup";
import { projectRoleOptions, projectRoleView } from "../../../../lib/projectRoles";
import type { Project, ProjectInviteView, ProjectMemberWithUser, ProjectRole, User } from "../../../../lib/types";
import { useAuthGuard } from "../../../../lib/useAuthGuard";

export default function ProjectMembersPage() {
  const { id } = useParams<{ id: string }>();
  const confirmAction = useConfirm();
  const { t, locale } = useLocale();
  const { token, ready } = useAuthGuard();

  const [project, setProject] = useState<Project | null>(null);
  const [me, setMe] = useState<User | null>(null);
  const [members, setMembers] = useState<ProjectMemberWithUser[]>([]);
  const [invites, setInvites] = useState<ProjectInviteView[]>([]);
  const [canManageMembers, setCanManageMembers] = useState(false);

  const [memberLookup, setMemberLookup] = useState("");
  const [memberRole, setMemberRole] = useState<ProjectRole>("TEAM");

  const [notice, setNotice] = useState("");
  const [error, setError] = useState("");

  async function load() {
    if (!token) return;
    try {
      const [projectResp, membersResp, meResp] = await Promise.all([
        apiRequest<Project>(`/projects/${id}`, { token }),
        apiRequest<ProjectMemberWithUser[]>(`/projects/${id}/members`, { token }),
        apiRequest<User>("/auth/me", { token })
      ]);

      setProject(projectResp);
      setMembers(membersResp);
      setMe(meResp);

      const selfMember = membersResp.find((member) => member.user_id === meResp.id);
      const canManage = selfMember?.role === "PM";
      setCanManageMembers(canManage);
      if (canManage) {
        const invitesResp = await apiRequest<ProjectInviteView[]>(`/projects/${id}/invites`, { token });
        setInvites(invitesResp);
      } else {
        setInvites([]);
      }
    } catch (err) {
      setError((err as Error).message);
    }
  }

  useEffect(() => {
    if (ready) {
      void load();
    }
  }, [ready, id, token]);

  const pmCount = useMemo(() => members.filter((member) => member.role === "PM").length, [members]);
  const pendingInvites = useMemo(() => invites.filter((invite) => invite.status === "pending"), [invites]);
  const roleOptions = projectRoleOptions(t);
  const selectedMemberRoleView = projectRoleView(t, memberRole);
  const tr = (ru: string, en: string) => (locale === "en" ? en : ru);

  async function addMember() {
    if (!token || !canManageMembers) return;
    const lookup = parseMemberLookup(memberLookup);
    if (!lookup) {
      setError(tr("Укажите корректный email, user ID (UUID) или никнейм.", "Provide a valid email, user ID (UUID), or nickname."));
      return;
    }
    setError("");
    setNotice("");

    try {
      await apiRequest(`/projects/${id}/members`, {
        method: "POST",
        token,
        body: {
          ...lookup,
          role: memberRole
        }
      });
      setMemberLookup("");
      setNotice(tr("Приглашение отправлено. Участник появится после подтверждения.", "Invite sent. Member will appear after confirmation."));
      await load();
    } catch (err) {
      setError((err as Error).message);
    }
  }

  async function deleteMember(memberID: string, fullName: string, role: ProjectRole) {
    if (!token || !canManageMembers) return;
    const confirmed = await confirmAction({
      title: tr("Удалить участника из проекта?", "Remove member from project?"),
      message: tr(`Участник «${fullName}» будет исключён из проекта.`, `Member "${fullName}" will be removed from this project.`),
      confirmText: tr("Удалить", "Delete"),
      cancelText: tr("Отмена", "Cancel"),
      tone: "danger"
    });
    if (!confirmed) return;

    if (role === "PM" && pmCount <= 1) {
      setError(tr(`Нельзя удалить последнего «${projectRoleView(t, "PM").title}» из проекта.`, `You cannot remove the last "${projectRoleView(t, "PM").title}" from the project.`));
      return;
    }

    setError("");
    setNotice("");
    try {
      await apiRequest(`/projects/${id}/members/${memberID}`, { method: "DELETE", token });
      setNotice(tr("Участник удалён.", "Member removed."));
      await load();
    } catch (err) {
      setError((err as Error).message);
    }
  }

  async function revokeInvite(inviteId: string, recipient: string) {
    if (!token || !canManageMembers) return;
    const confirmed = await confirmAction({
      title: tr("Отозвать приглашение?", "Revoke invite?"),
      message: tr(
        `Приглашение для «${recipient}» будет отозвано.`,
        `The invite for "${recipient}" will be revoked.`
      ),
      confirmText: tr("Отозвать", "Revoke"),
      cancelText: tr("Отмена", "Cancel"),
      tone: "danger"
    });
    if (!confirmed) return;

    setError("");
    setNotice("");
    try {
      await apiRequest(`/projects/${id}/invites/${inviteId}`, { method: "DELETE", token });
      setNotice(tr("Приглашение отозвано.", "Invite revoked."));
      await load();
    } catch (err) {
      setError((err as Error).message);
    }
  }

  if (!ready || !project || !me) return null;

  return (
    <AppShell title={project.name} className="members-shell">
      <ProjectNav projectId={id} />

      {error && <p style={{ color: "#bc4749" }}>{error}</p>}
      {notice && <p style={{ color: "#2a9d8f" }}>{notice}</p>}

      <section className="card members-page-card" style={{ marginBottom: 14 }}>
        <div className="members-page-head">
          <div>
            <span className="shell-eyebrow">{tr("Участники", "Members")}</span>
            <h2 className="title">{tr("Команда проекта", "Project team")}</h2>
            <p className="subtle" style={{ margin: 0 }}>
              {tr(
                "Добавляйте зарегистрированных пользователей по email, user ID или никнейму и управляйте ролями в одном месте.",
                "Add registered users by email, user ID, or nickname and manage roles in one place."
              )}
            </p>
          </div>
          <div className="members-overview-pills">
            <span>{tr("Всего участников", "Total members")}: <strong>{members.length}</strong></span>
            <span>{tr("Администратор", "Administrator")}: <strong>{pmCount}</strong></span>
          </div>
        </div>

        {canManageMembers ? (
          <div className="members-add-card">
            <label className="members-add-lookup">
              {tr("Добавить участника", "Add member")}
              <input
                placeholder={tr("Email, user ID (UUID) или никнейм (@nickname)", "Email, user ID (UUID), or nickname (@nickname)")}
                value={memberLookup}
                onChange={(e) => setMemberLookup(e.target.value)}
              />
            </label>
            <label className="members-add-role">
              {tr("Роль", "Role")}
              <select value={memberRole} onChange={(e) => setMemberRole(e.target.value as ProjectRole)}>
                {roleOptions.map((opt) => (
                  <option key={opt.code} value={opt.code}>{opt.title}</option>
                ))}
              </select>
            </label>
            <div className="members-add-actions">
              <button type="button" className="fixed-width-button" onClick={addMember}>{tr("Добавить", "Add")}</button>
              <p className="subtle">{selectedMemberRoleView.subtitle}</p>
            </div>
          </div>
        ) : (
          <p className="subtle" style={{ marginTop: 0 }}>
            {tr(
              `У вас режим просмотра. Управление участниками доступно только роли «${projectRoleView(t, "PM").title}».`,
              `You have read-only access. Member management is available only for "${projectRoleView(t, "PM").title}".`
            )}
          </p>
        )}

        <div className="member-cards-grid">
          {members.map((member) => {
            const isSelf = member.user_id === me.id;
            const roleView = projectRoleView(t, member.role);
            return (
              <article key={member.id} className="member-card">
                <div className="member-card-topline">
                  <div>
                    <h3>{member.full_name}</h3>
                    <p>@{member.nickname}</p>
                  </div>
                  <div className="member-card-topline-side">
                    <span className={`member-role-badge role-${member.role.toLowerCase()}`}>{roleView.title}</span>
                    {member.default_currency_code ? (
                      <span className="member-currency-chip">{member.default_currency_code}</span>
                    ) : null}
                  </div>
                </div>
                <div className="member-card-meta">
                  <span>{tr("В проекте с", "Joined")}: {new Date(member.created_at).toLocaleDateString(locale === "en" ? "en-US" : "ru-RU")}</span>
                  {isSelf && <span>{tr("Это вы", "This is you")}</span>}
                </div>
                <div className="member-card-grid member-card-grid-wide">
                  <div>
                    <span>{tr("Email", "Email")}</span>
                    <strong>{member.email}</strong>
                  </div>
                </div>
                <p className="member-role-subtitle">{roleView.subtitle}</p>
                <div className="member-card-actions">
                  {canManageMembers ? (
                    <button
                      type="button"
                      className="warn"
                      disabled={isSelf && member.role === "PM" && pmCount <= 1}
                      onClick={() => deleteMember(member.id, member.full_name, member.role)}
                    >
                      {tr("Удалить", "Delete")}
                    </button>
                  ) : (
                    <span className="subtle">{tr("Только просмотр", "Read only")}</span>
                  )}
                </div>
              </article>
            );
          })}
          {!members.length && (
            <div className="ledger-empty-state">
              <strong>{tr("Участники пока не добавлены", "No members added yet")}</strong>
              <span>{tr("После приглашения и подтверждения участник появится здесь.", "After invite and confirmation, the member will appear here.")}</span>
            </div>
          )}
        </div>

        {canManageMembers ? (
          <div className="members-invites-block">
            <div className="members-invites-head">
              <div>
                <h3>{tr("Приглашения в ожидании", "Pending invites")}</h3>
                <p className="subtle members-invites-subtitle" style={{ margin: "4px 0 0" }}>
                  {tr(
                    "Здесь только активные приглашения. После принятия участник попадает в карточки команды выше.",
                    "Only active invites live here. Once accepted, the member moves into the team cards above."
                  )}
                </p>
              </div>
              <span>{pendingInvites.length}</span>
            </div>
            <div className="members-invites-list">
              {pendingInvites.map((invite) => {
                const roleView = projectRoleView(t, invite.invited_role);
                const recipientName = invite.invited_user_full_name || invite.invited_user_email || invite.invited_user_nickname || invite.invited_user_id || tr("Не указан", "Unknown");
                return (
                  <article key={invite.id} className="members-invite-card">
                    <div className="members-invite-main">
                      <strong>{recipientName}</strong>
                      <p>{roleView.title} · {tr("Ожидает подтверждения", "Waiting for confirmation")}</p>
                    </div>
                    <div className="members-invite-side">
                      <span>{tr("Истекает", "Expires")}: {new Date(invite.expires_at).toLocaleDateString(locale === "en" ? "en-US" : "ru-RU")}</span>
                      <button type="button" className="secondary" onClick={() => void revokeInvite(invite.id, String(recipientName))}>
                        {tr("Отозвать", "Revoke")}
                      </button>
                    </div>
                  </article>
                );
              })}
              {!pendingInvites.length && <p className="subtle" style={{ margin: 0 }}>{tr("Активных приглашений нет.", "There are no active invites.")}</p>}
            </div>
          </div>
        ) : null}
      </section>
    </AppShell>
  );
}
