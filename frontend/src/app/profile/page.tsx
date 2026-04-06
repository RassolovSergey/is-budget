"use client";

import { FormEvent, useEffect, useRef, useState } from "react";

import { AppShell } from "../../components/AppShell";
import { useLocale } from "../../components/LocaleProvider";
import { apiAssetUrl, apiRequest } from "../../lib/api";
import { setToken } from "../../lib/session";
import type { CurrencyCode, LoginResponse, User } from "../../lib/types";
import { useAuthGuard } from "../../lib/useAuthGuard";

export default function ProfilePage() {
  const { locale } = useLocale();
  const { token, ready } = useAuthGuard();
  const [user, setUser] = useState<User | null>(null);

  const [firstName, setFirstName] = useState("");
  const [lastName, setLastName] = useState("");
  const [nickname, setNickname] = useState("");
  const [currency, setCurrency] = useState<CurrencyCode>("EUR");

  const [newEmail, setNewEmail] = useState("");
  const [emailPassword, setEmailPassword] = useState("");

  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [newPasswordRepeat, setNewPasswordRepeat] = useState("");

  const [avatarFile, setAvatarFile] = useState<File | null>(null);
  const [dragActive, setDragActive] = useState(false);

  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [saving, setSaving] = useState(false);
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const tr = (ru: string, en: string) => (locale === "en" ? en : ru);

  async function load() {
    if (!token) return;
    try {
      const me = await apiRequest<User>("/auth/me", { token });
      setUser(me);
      setFirstName(me.first_name || "");
      setLastName(me.last_name || "");
      setNickname(me.nickname || "");
      setCurrency(me.default_currency_code || "EUR");
      setNewEmail(me.email || "");
    } catch (err) {
      setError((err as Error).message);
    }
  }

  useEffect(() => {
    if (ready) void load();
  }, [ready]);

  function assignAvatar(nextFile: File | null) {
    if (!nextFile) return;
    setAvatarFile(nextFile);
    setNotice("");
    setError("");
  }

  async function saveProfile(e: FormEvent) {
    e.preventDefault();
    if (!token) return;
    setError("");
    setNotice("");
    setSaving(true);
    try {
      const updated = await apiRequest<User>("/auth/profile", {
        method: "PUT",
        token,
        body: {
          first_name: firstName,
          last_name: lastName,
          nickname,
          default_currency_code: currency
        }
      });
      setUser(updated);
      setNotice(tr("Профиль обновлён.", "Profile updated."));
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setSaving(false);
    }
  }

  async function saveEmail(e: FormEvent) {
    e.preventDefault();
    if (!token) return;
    setError("");
    setNotice("");
    setSaving(true);
    try {
      const result = await apiRequest<LoginResponse>("/auth/email", {
        method: "PUT",
        token,
        body: {
          new_email: newEmail,
          current_password: emailPassword
        }
      });
      setToken(result.token);
      setUser(result.user);
      setEmailPassword("");
      setNotice(tr("Email обновлён. Сессия обновлена.", "Email updated. Session refreshed."));
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setSaving(false);
    }
  }

  async function savePassword(e: FormEvent) {
    e.preventDefault();
    if (!token) return;
    setError("");
    setNotice("");
    if (newPassword !== newPasswordRepeat) {
      setError(tr("Новый пароль и подтверждение не совпадают.", "New password and confirmation do not match."));
      return;
    }
    setSaving(true);
    try {
      await apiRequest("/auth/password", {
        method: "PUT",
        token,
        body: {
          current_password: currentPassword,
          new_password: newPassword
        }
      });
      setCurrentPassword("");
      setNewPassword("");
      setNewPasswordRepeat("");
      setNotice(tr("Пароль обновлён.", "Password updated."));
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setSaving(false);
    }
  }

  async function uploadAvatar(e: FormEvent) {
    e.preventDefault();
    if (!token || !avatarFile) return;
    setError("");
    setNotice("");
    setSaving(true);
    try {
      const form = new FormData();
      form.append("avatar", avatarFile);
      const response = await fetch(`${process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api"}/auth/avatar`, {
        method: "POST",
        headers: {
          Authorization: `Bearer ${token}`
        },
        body: form
      });
      if (!response.ok) {
        let msg = `Request failed (${response.status})`;
        try {
          const payload = await response.json();
          if (payload.error) msg = payload.error;
        } catch {
          // noop
        }
        throw new Error(msg);
      }
      const updated = (await response.json()) as User;
      setUser(updated);
      setAvatarFile(null);
      setNotice(tr("Аватар обновлён.", "Avatar updated."));
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setSaving(false);
    }
  }

  async function deleteAvatar() {
    if (!token || !user || !user.avatar_path) return;
    setError("");
    setNotice("");
    setSaving(true);
    try {
      const updated = await apiRequest<User>("/auth/avatar", {
        method: "DELETE",
        token
      });
      setUser(updated);
      setAvatarFile(null);
      setNotice(tr("Аватар удалён.", "Avatar removed."));
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setSaving(false);
    }
  }

  if (!ready || !user) return null;

  return (
    <AppShell title={tr("Профиль", "Profile")} className="profile-page-shell">
      {error && <p style={{ color: "var(--danger)", margin: 0 }}>{error}</p>}
      {notice && <p style={{ color: "var(--ok)", margin: 0 }}>{notice}</p>}

      <div className="profile-dashboard-grid">
        <section className="card profile-hero-card">
          <div className="profile-hero-main">
            <div className="profile-sidebar-head">
              {user.avatar_path ? (
                <img
                  src={apiAssetUrl(user.avatar_path)}
                  alt={tr("аватар", "avatar")}
                  className="profile-avatar"
                />
              ) : (
                <div className="profile-avatar profile-avatar-fallback">
                  {user.first_name?.[0] || user.full_name?.[0] || "?"}
                </div>
              )}
              <div className="profile-identity">
                <span className="shell-eyebrow">{tr("Аккаунт", "Account")}</span>
                <h2>{user.full_name}</h2>
                <p>@{user.nickname}</p>
              </div>
            </div>

            <div className="profile-summary-row">
              <article className="profile-summary-item profile-summary-item-accent">
                <span>{tr("Валюта профиля", "Profile currency")}</span>
                <strong>{currency}</strong>
              </article>
              <article className="profile-summary-item">
                <span>Email</span>
                <strong>{user.email}</strong>
              </article>
              <article className="profile-summary-item">
                <span>{tr("User ID", "User ID")}</span>
                <strong className="profile-user-id">{user.id}</strong>
              </article>
            </div>
          </div>

          <form className="profile-avatar-uploader" onSubmit={uploadAvatar}>
            <div className="profile-avatar-uploader-head">
              <h3>{tr("Аватар профиля", "Profile avatar")}</h3>
              <p className="subtle">
                {tr(
                  "Перетащите изображение в область ниже или выберите файл вручную.",
                  "Drag an image into the zone below or choose a file manually."
                )}
              </p>
            </div>
            <input
              ref={fileInputRef}
              className="profile-avatar-input-hidden"
              type="file"
              accept=".jpg,.jpeg,.png,.webp"
              onChange={(e) => assignAvatar(e.target.files?.[0] || null)}
            />
            <button
              type="button"
              className={`profile-avatar-dropzone ${dragActive ? "drag-active" : ""}`}
              onClick={() => fileInputRef.current?.click()}
              onDragEnter={(e) => {
                e.preventDefault();
                setDragActive(true);
              }}
              onDragOver={(e) => {
                e.preventDefault();
                setDragActive(true);
              }}
              onDragLeave={(e) => {
                e.preventDefault();
                setDragActive(false);
              }}
              onDrop={(e) => {
                e.preventDefault();
                setDragActive(false);
                assignAvatar(e.dataTransfer.files?.[0] || null);
              }}
            >
              <span className="profile-avatar-dropzone-icon">+</span>
              <strong>{avatarFile ? avatarFile.name : tr("Перетащите файл или нажмите для выбора", "Drop file here or click to choose")}</strong>
              <span>{tr("PNG, JPG, WEBP до 5 МБ", "PNG, JPG, WEBP up to 5 MB")}</span>
            </button>
            <div className="profile-avatar-actions">
              <button type="submit" disabled={saving || !avatarFile}>
                {tr("Загрузить аватар", "Upload avatar")}
              </button>
              {user.avatar_path && (
                <button type="button" className="secondary" disabled={saving} onClick={() => void deleteAvatar()}>
                  {tr("Удалить аватар", "Remove avatar")}
                </button>
              )}
              {avatarFile && (
                <button type="button" className="secondary" onClick={() => setAvatarFile(null)}>
                  {tr("Очистить", "Clear")}
                </button>
              )}
            </div>
          </form>
        </section>

        <section className="card profile-main-card">
          <div className="profile-section-head">
            <div>
              <h2 className="title">{tr("Основные данные", "Basic data")}</h2>
              <p className="subtle" style={{ margin: 0 }}>
                {tr(
                  "Настройте отображаемое имя, никнейм и базовую валюту для выплат, аналитики и конвертации.",
                  "Set your display name, nickname, and base currency for payouts, analytics, and conversion."
                )}
              </p>
            </div>
          </div>

          <form className="profile-basic-form" onSubmit={saveProfile}>
            <label>
              {tr("Имя", "First name")}
              <input value={firstName} onChange={(e) => setFirstName(e.target.value)} required />
            </label>
            <label>
              {tr("Фамилия", "Last name")}
              <input value={lastName} onChange={(e) => setLastName(e.target.value)} required />
            </label>
            <label>
              {tr("Никнейм", "Nickname")}
              <input
                value={nickname}
                onChange={(e) => setNickname(e.target.value.toLowerCase())}
                pattern="[a-z0-9_]{3,32}"
                title={tr("Используйте 3-32 символа: a-z, 0-9 и _", "Use 3-32 chars: a-z, 0-9 and _")}
                required
              />
            </label>
            <label>
              {tr("Валюта по умолчанию", "Default currency")}
              <select value={currency} onChange={(e) => setCurrency(e.target.value as CurrencyCode)}>
                <option value="EUR">EUR (€)</option>
                <option value="USD">USD ($)</option>
                <option value="RUB">RUB (₽)</option>
              </select>
            </label>
            <div className="profile-form-actions">
              <button type="submit" disabled={saving}>
                {tr("Сохранить профиль", "Save profile")}
              </button>
            </div>
          </form>
        </section>

        <div className="profile-security-grid">
          <section className="card profile-main-card">
            <div className="profile-section-head">
              <div>
                <h2 className="title">{tr("Смена email", "Change email")}</h2>
                <p className="subtle" style={{ margin: 0 }}>
                  {tr("Подтвердите действие текущим паролем.", "Confirm the change with your current password.")}
                </p>
              </div>
            </div>
            <form className="grid" onSubmit={saveEmail}>
              <label>
                {tr("Новый email", "New email")}
                <input type="email" value={newEmail} onChange={(e) => setNewEmail(e.target.value)} required />
              </label>
              <label>
                {tr("Текущий пароль", "Current password")}
                <input type="password" value={emailPassword} onChange={(e) => setEmailPassword(e.target.value)} required />
              </label>
              <button type="submit" disabled={saving}>
                {tr("Обновить email", "Update email")}
              </button>
            </form>
          </section>

          <section className="card profile-main-card">
            <div className="profile-section-head">
              <div>
                <h2 className="title">{tr("Смена пароля", "Change password")}</h2>
                <p className="subtle" style={{ margin: 0 }}>
                  {tr("Используйте новый пароль длиной не менее 8 символов.", "Use a new password at least 8 characters long.")}
                </p>
              </div>
            </div>
            <form className="grid" onSubmit={savePassword}>
              <label>
                {tr("Текущий пароль", "Current password")}
                <input type="password" value={currentPassword} onChange={(e) => setCurrentPassword(e.target.value)} required />
              </label>
              <label>
                {tr("Новый пароль", "New password")}
                <input type="password" value={newPassword} onChange={(e) => setNewPassword(e.target.value)} required />
              </label>
              <label>
                {tr("Повторите новый пароль", "Repeat new password")}
                <input type="password" value={newPasswordRepeat} onChange={(e) => setNewPasswordRepeat(e.target.value)} required />
              </label>
              <button type="submit" disabled={saving}>
                {tr("Обновить пароль", "Update password")}
              </button>
            </form>
          </section>
        </div>
      </div>
    </AppShell>
  );
}
