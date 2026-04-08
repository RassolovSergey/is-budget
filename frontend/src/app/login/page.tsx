"use client";

import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";

import { LanguageToggle } from "../../components/LanguageToggle";
import { useLocale } from "../../components/LocaleProvider";
import { ThemeToggle } from "../../components/ThemeToggle";
import { apiRequest } from "../../lib/api";
import { setToken } from "../../lib/session";
import type { LoginResponse } from "../../lib/types";

function PasswordField({
  label,
  value,
  onChange,
  placeholder,
  required = false,
  minLength
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  required?: boolean;
  minLength?: number;
}) {
  const [visible, setVisible] = useState(false);

  return (
    <label>
      {label}
      <div className="password-field">
        <input
          type={visible ? "text" : "password"}
          value={value}
          placeholder={placeholder}
          minLength={minLength}
          onChange={(e) => onChange(e.target.value)}
          required={required}
        />
        <button
          type="button"
          className="password-toggle"
          onClick={() => setVisible((prev) => !prev)}
          aria-label={visible ? "Hide password" : "Show password"}
        >
          {visible ? "🙈" : "👁"}
        </button>
      </div>
    </label>
  );
}

export default function LoginPage() {
  const router = useRouter();
  const { t, locale } = useLocale();
  const [mode, setMode] = useState<"login" | "register">("login");
  const [email, setEmail] = useState("pm@example.com");
  const [password, setPassword] = useState("password");
  const [regFirstName, setRegFirstName] = useState("");
  const [regLastName, setRegLastName] = useState("");
  const [regNickname, setRegNickname] = useState("");
  const [regEmail, setRegEmail] = useState("");
  const [regPassword, setRegPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const tr = (ru: string, en: string) => (locale === "en" ? en : ru);

  function resolveNextPath(): string {
    if (typeof window === "undefined") {
      return "/projects";
    }
    const next = new URLSearchParams(window.location.search).get("next");
    if (!next || !next.startsWith("/") || next.startsWith("//")) {
      return "/projects";
    }
    return next;
  }

  async function submitLogin(e: FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);
    const normalizedEmail = email.trim();
    setEmail(normalizedEmail);
    try {
      const result = await apiRequest<LoginResponse>("/auth/login", {
        method: "POST",
        body: { email: normalizedEmail, password }
      });
      setToken(result.token);
      router.push(resolveNextPath());
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }

  async function submitRegister(e: FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      const result = await apiRequest<LoginResponse>("/auth/register", {
        method: "POST",
        body: {
          first_name: regFirstName,
          last_name: regLastName,
          nickname: regNickname,
          email: regEmail,
          password: regPassword
        }
      });
      setToken(result.token);
      router.push(resolveNextPath());
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="page auth-page">
      <div className="auth-shell">
        <section className="auth-spotlight">
          <div className="auth-floating-controls">
            <LanguageToggle />
            <ThemeToggle />
          </div>
          <div className="auth-brand">
            <span className="shell-eyebrow">IS Budget</span>
            <h1>{tr("Управление финансовой логикой проекта в одном окне", "One workspace for project finance operations")}</h1>
            <p className="subtle">
              {tr(
                "Доходы, расходы, налоги, роли команды, прогнозы и распределение прибыли — в едином рабочем контуре без лишнего шума.",
                "Income, expenses, taxes, team roles, forecasts, and profit sharing — in one operational workspace without clutter."
              )}
            </p>
          </div>
          <div className="card auth-showcase">
            <div className="auth-showcase-metric">
              <span>{tr("Операционный контур", "Operations layer")}</span>
              <strong>{tr("Доходы, налоги, выплаты", "Income, taxes, payouts")}</strong>
            </div>
            <div className="auth-showcase-metric">
              <span>{tr("Команда", "Team")}</span>
              <strong>{tr("Роли, прозрачность, контроль", "Roles, transparency, control")}</strong>
            </div>
            <div className="auth-showcase-metric">
              <span>{tr("Аналитика", "Analytics")}</span>
              <strong>{tr("Прогноз и история кампаний", "Forecasting and campaign history")}</strong>
            </div>
            {mode === "login" && <p className="subtle auth-test-users">{t("login.test_users")}</p>}
          </div>
        </section>

        <section className="auth-panel">
          <div className="card auth-card auth-card-elevated">
            <div className="auth-card-head">
              <div>
                <span className="shell-eyebrow">{mode === "login" ? tr("Вход", "Sign in") : tr("Регистрация", "Register")}</span>
                <h2 className="title auth-form-title">
                  {mode === "login" ? t("login.login_title") : tr("Регистрация в IS Budget", "Create your IS Budget account")}
                </h2>
                <p className="subtle" style={{ margin: 0 }}>
                  {mode === "login"
                    ? tr("Войдите и продолжайте работу с проектами, историей и прогнозами.", "Sign in to continue working with projects, history, and forecasts.")
                    : tr("Создайте аккаунт и сразу настройте личную валюту и доступ к проектам.", "Create an account and immediately configure your personal currency and project access.")}
                </p>
              </div>
            </div>

            {mode === "login" ? (
              <form className="grid auth-form-grid" onSubmit={submitLogin}>
                <label>
                  {t("login.email")}
                  <input
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    onBlur={() => setEmail((prev) => prev.trim())}
                  />
                </label>
                <PasswordField label={t("login.password")} value={password} onChange={setPassword} />
                {error && <p style={{ color: "#bc4749", margin: 0 }}>{error}</p>}
                <button disabled={loading}>{loading ? t("login.submit_login_loading") : t("login.submit_login")}</button>
                <div className="auth-mode-switch">
                  <span className="subtle">{t("login.switch_to_register")}</span>
                  <div className="auth-toggle">
                    <button
                      type="button"
                      className=""
                      onClick={() => {
                        setMode("login");
                        setError("");
                      }}
                    >
                      {t("login.mode_login")}
                    </button>
                    <button
                      type="button"
                      className="secondary"
                      onClick={() => {
                        setMode("register");
                        setError("");
                      }}
                    >
                      {t("login.mode_register")}
                    </button>
                  </div>
                </div>
              </form>
            ) : (
              <form className="grid auth-form-grid" onSubmit={submitRegister}>
                <div className="grid two auth-pair-grid" style={{ gap: 12 }}>
                  <label>
                    {t("login.first_name")}
                    <input value={regFirstName} onChange={(e) => setRegFirstName(e.target.value)} required />
                  </label>
                  <label>
                    {t("login.last_name")}
                    <input value={regLastName} onChange={(e) => setRegLastName(e.target.value)} required />
                  </label>
                </div>
                <label>
                  {t("login.nickname")}
                  <input
                    value={regNickname}
                    onChange={(e) => setRegNickname(e.target.value.toLowerCase())}
                    pattern="[a-z0-9_]{3,32}"
                    title={t("login.nickname_title")}
                    required
                  />
                </label>
                <label>
                  {t("login.email")}
                  <input type="email" value={regEmail} onChange={(e) => setRegEmail(e.target.value)} required />
                </label>
                <PasswordField
                  label={t("login.password")}
                  value={regPassword}
                  onChange={setRegPassword}
                  minLength={6}
                  required
                />
                {error && <p style={{ color: "#bc4749", margin: 0 }}>{error}</p>}
                <button disabled={loading}>{loading ? t("login.submit_register_loading") : t("login.submit_register")}</button>
                <div className="auth-mode-switch">
                  <span className="subtle">{t("login.switch_to_login")}</span>
                  <div className="auth-toggle">
                    <button
                      type="button"
                      className="secondary"
                      onClick={() => {
                        setMode("login");
                        setError("");
                      }}
                    >
                      {t("login.mode_login")}
                    </button>
                    <button
                      type="button"
                      className=""
                      onClick={() => {
                        setMode("register");
                        setError("");
                      }}
                    >
                      {t("login.mode_register")}
                    </button>
                  </div>
                </div>
              </form>
            )}
          </div>
        </section>
      </div>
    </div>
  );
}
