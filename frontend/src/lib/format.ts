import type { CurrencyCode } from "./types";
import { LOCALE_STORAGE_KEY, type Locale, toIntlLocale } from "./i18n";

function getCurrentLocale(): Locale {
  if (typeof window === "undefined") return "ru";
  const saved = window.localStorage.getItem(LOCALE_STORAGE_KEY);
  return saved === "en" ? "en" : "ru";
}

export function money(value: number, currency: CurrencyCode = "EUR"): string {
  return new Intl.NumberFormat(toIntlLocale(getCurrentLocale()), {
    style: "currency",
    currency,
    maximumFractionDigits: 2
  }).format(value || 0);
}

export function currencySymbol(currency: CurrencyCode = "EUR"): string {
  switch (currency) {
    case "USD":
      return "$";
    case "RUB":
      return "₽";
    case "EUR":
    default:
      return "€";
  }
}

export function shortDate(value: string): string {
  return new Date(value).toLocaleDateString(toIntlLocale(getCurrentLocale()));
}

export function todayISO(): string {
  return new Date().toISOString().slice(0, 10);
}
