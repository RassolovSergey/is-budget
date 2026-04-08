import type { Locale } from "./i18n";

export type ForecastScenarioKey = "poor" | "average" | "good";

export type ForecastScenarioView = {
  key: ForecastScenarioKey;
  label: string;
};

export function forecastScenarioViews(locale: Locale): ForecastScenarioView[] {
  return [
    {
      key: "poor",
      label: locale === "en" ? "Low scenario" : "Низкий сценарий"
    },
    {
      key: "average",
      label: locale === "en" ? "Base scenario" : "Базовый сценарий"
    },
    {
      key: "good",
      label: locale === "en" ? "Optimistic scenario" : "Оптимистичный сценарий"
    }
  ];
}
