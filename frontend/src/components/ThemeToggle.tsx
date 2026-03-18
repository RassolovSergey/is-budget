"use client";

import { useLocale } from "./LocaleProvider";
import { useTheme } from "./ThemeProvider";

export function ThemeToggle() {
  const { locale } = useLocale();
  const { theme, toggleTheme } = useTheme();
  const switchToDark = theme === "light";
  const label = switchToDark
    ? (locale === "en" ? "Enable dark theme" : "Включить тёмную тему")
    : (locale === "en" ? "Enable light theme" : "Включить светлую тему");

  return (
    <button
      type="button"
      className="secondary theme-toggle"
      onClick={toggleTheme}
      aria-label={label}
      title={label}
    >
      {switchToDark ? (
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path d="M20 14.5A8.5 8.5 0 1 1 9.5 4a7 7 0 0 0 10.5 10.5Z" />
        </svg>
      ) : (
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <circle cx="12" cy="12" r="4" />
          <path d="M12 2v2.5M12 19.5V22M4.9 4.9l1.8 1.8M17.3 17.3l1.8 1.8M2 12h2.5M19.5 12H22M4.9 19.1l1.8-1.8M17.3 6.7l1.8-1.8" />
        </svg>
      )}
    </button>
  );
}
