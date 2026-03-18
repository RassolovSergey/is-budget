"use client";

import { useLocale } from "./LocaleProvider";

export function LanguageToggle() {
  const { locale, setLocale, t } = useLocale();

  return (
    <div className="topbar-locale" aria-label={t("app.lang.label")}>
      <button
        type="button"
        className={`topbar-locale-item ${locale === "ru" ? "active" : ""}`}
        onClick={() => setLocale("ru")}
      >
        {t("app.lang.ru")}
      </button>
      <button
        type="button"
        className={`topbar-locale-item ${locale === "en" ? "active" : ""}`}
        onClick={() => setLocale("en")}
      >
        {t("app.lang.en")}
      </button>
    </div>
  );
}
