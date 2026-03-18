"use client";

import { FormEvent, useEffect, useMemo, useRef, useState } from "react";

import { AppShell } from "../../components/AppShell";
import { useConfirm } from "../../components/ConfirmProvider";
import { useLocale } from "../../components/LocaleProvider";
import { apiRequest } from "../../lib/api";
import type {
  CurrencyCode,
  HistoricalCampaign,
  HistoricalCampaignDay,
  HistoricalCampaignExpense,
  UpsertHistoricalCampaignRequest
} from "../../lib/types";
import { useAuthGuard } from "../../lib/useAuthGuard";
import {
  HistoricalCampaignEditor,
  HistoricalCampaignList,
  HistoricalCampaignToolbar
} from "./HistoricalCampaignSections";
import type { CampaignForm, CumulativeRow, HistorySort } from "./types";

const PRESET_EXPENSE_CATEGORIES_RU = ["Реклама", "Производство", "Логистика", "Комиссии", "Упаковка"];
const PRESET_EXPENSE_CATEGORIES_EN = ["Advertising", "Production", "Logistics", "Fees", "Packaging"];
const DATE_LAYOUT = "YYYY-MM-DD";

function currencySymbol(code: CurrencyCode): string {
  switch (code) {
    case "USD":
      return "$";
    case "RUB":
      return "₽";
    case "EUR":
    default:
      return "€";
  }
}

function formatAmount(value: number, locale: string): string {
  return value.toLocaleString(locale, { minimumFractionDigits: 0, maximumFractionDigits: 2 });
}

function round2(value: number): number {
  return Math.round(value * 100) / 100;
}

function clampDuration(value: number): number {
  if (!Number.isFinite(value)) return 3;
  return Math.min(60, Math.max(3, Math.floor(value)));
}

function parseAmountInput(value: string): number {
  const normalized = value.trim().replace(/\s+/g, "").replace(",", ".");
  if (!normalized) return 0;
  const parsed = Number.parseFloat(normalized);
  if (!Number.isFinite(parsed)) return 0;
  return round2(Math.max(0, parsed));
}

function normalizeCategory(category: string): string {
  return category.trim().toLowerCase();
}

function todayISO(): string {
  const now = new Date();
  const y = now.getFullYear();
  const m = String(now.getMonth() + 1).padStart(2, "0");
  const d = String(now.getDate()).padStart(2, "0");
  return `${y}-${m}-${d}`;
}

function addDaysISO(dateISO: string, days: number): string {
  const date = new Date(`${dateISO}T00:00:00Z`);
  if (Number.isNaN(date.getTime())) return dateISO;
  date.setUTCDate(date.getUTCDate() + days);
  return date.toISOString().slice(0, 10);
}

function parseCSVRows(rawText: string): string[][] {
  const text = rawText.replace(/^\uFEFF/, "");
  const firstLine = text.split(/\r?\n/, 1)[0] ?? "";
  const commaCount = (firstLine.match(/,/g) || []).length;
  const semicolonCount = (firstLine.match(/;/g) || []).length;
  const delimiter = semicolonCount > commaCount ? ";" : ",";

  const rows: string[][] = [];
  let row: string[] = [];
  let cell = "";
  let inQuotes = false;

  for (let i = 0; i < text.length; i += 1) {
    const ch = text[i];
    const next = text[i + 1];

    if (ch === "\"") {
      if (inQuotes && next === "\"") {
        cell += "\"";
        i += 1;
      } else {
        inQuotes = !inQuotes;
      }
      continue;
    }

    if (ch === delimiter && !inQuotes) {
      row.push(cell);
      cell = "";
      continue;
    }

    if ((ch === "\n" || ch === "\r") && !inQuotes) {
      if (ch === "\r" && next === "\n") {
        i += 1;
      }
      row.push(cell);
      cell = "";
      if (row.some((item) => item.trim() !== "")) {
        rows.push(row);
      }
      row = [];
      continue;
    }

    cell += ch;
  }

  if (cell !== "" || row.length > 0) {
    row.push(cell);
    if (row.some((item) => item.trim() !== "")) {
      rows.push(row);
    }
  }

  return rows;
}

function headerIndex(headers: string[], candidates: string[]): number {
  const normalized = headers.map((h) => h.trim().toLowerCase());
  for (const candidate of candidates) {
    const idx = normalized.indexOf(candidate.trim().toLowerCase());
    if (idx >= 0) return idx;
  }
  return -1;
}

function parseDateFromKickstarter(value: string): string {
  const raw = value.trim().replace(/^"|"$/g, "");
  const m = raw.match(/^(\d{4})[\/.-](\d{2})[\/.-](\d{2})/);
  if (m) {
    return `${m[1]}-${m[2]}-${m[3]}`;
  }

  const parsed = new Date(raw);
  if (!Number.isNaN(parsed.getTime())) {
    return parsed.toISOString().slice(0, 10);
  }
  return "";
}

function parseMoneyWithCurrency(value: string): { amount: number; currency: CurrencyCode | "" } {
  const raw = value.trim();
  if (!raw) return { amount: 0, currency: "" };

  const upper = raw.toUpperCase();
  let currency: CurrencyCode | "" = "";
  if (upper.includes("EUR") || raw.includes("€")) currency = "EUR";
  if (upper.includes("USD") || raw.includes("$")) currency = "USD";
  if (upper.includes("RUB") || raw.includes("₽")) currency = "RUB";

  let cleaned = raw.replace(/\s+/g, "");
  cleaned = cleaned.replace(/[^\d,.\-]/g, "");

  if (cleaned.includes(",") && cleaned.includes(".")) {
    cleaned = cleaned.replace(/,/g, "");
  } else if (cleaned.includes(",") && !cleaned.includes(".")) {
    cleaned = cleaned.replace(/,/g, ".");
  }

  const parsed = Number.parseFloat(cleaned);
  if (!Number.isFinite(parsed)) {
    return { amount: 0, currency };
  }
  return { amount: round2(parsed), currency };
}

function alignDurationToKickstarterWindow(spanDays: number): number {
  // Kickstarter campaigns are often configured with fixed windows.
  // If the CSV misses the last zero-income day, nudge duration to nearest common window (+1 day tolerance).
  const commonWindows = [7, 10, 14, 21, 30, 45, 60];
  for (const windowDays of commonWindows) {
    if (spanDays <= windowDays && windowDays-spanDays <= 1) {
      return windowDays;
    }
  }
  return spanDays;
}

function campaignNameFromFileName(fileName: string): string {
  const noExt = fileName.replace(/\.[^/.]+$/, "");
  return noExt.replace(/\s*-\s*All rewards.*$/i, "").trim() || "Imported campaign";
}

function parseKickstarterCsvToForm(
  fileName: string,
  fileText: string,
  categoryFallback: string,
  locale: "ru" | "en"
): CampaignForm {
  const tr = (ru: string, en: string) => (locale === "en" ? en : ru);
  const rows = parseCSVRows(fileText);
  if (rows.length < 2) {
    throw new Error(tr("CSV пустой или некорректный.", "CSV is empty or invalid."));
  }

  const headers = rows[0];
  const idxPledgeAmount = headerIndex(headers, ["Pledge Amount"]);
  const idxPledgedAt = headerIndex(headers, ["Pledged At (UTC)", "Pledged At"]);
  const idxRefunded = headerIndex(headers, ["Pledge Total Refunded"]);
  const idxPmRefunded = headerIndex(headers, ["Pledge Manager Total Refunded"]);
  const idxPledgedStatus = headerIndex(headers, ["Pledged Status"]);

  if (idxPledgeAmount < 0 || idxPledgedAt < 0) {
    throw new Error(tr("Не найдены обязательные колонки: Pledge Amount и/или Pledged At (UTC).", "Required columns are missing: Pledge Amount and/or Pledged At (UTC)."));
  }

  const byDate = new Map<string, number>();
  let detectedCurrency: CurrencyCode | "" = "";

  for (let i = 1; i < rows.length; i += 1) {
    const row = rows[i];
    if (!row || row.length === 0) continue;

    const status = idxPledgedStatus >= 0 ? (row[idxPledgedStatus] || "").trim().toLowerCase() : "";
    if (status && status !== "collected") {
      continue;
    }

    const day = parseDateFromKickstarter(row[idxPledgedAt] || "");
    if (!day) continue;

    const pledge = parseMoneyWithCurrency(row[idxPledgeAmount] || "");
    const refunded = idxRefunded >= 0 ? parseMoneyWithCurrency(row[idxRefunded] || "") : { amount: 0, currency: "" as const };
    const pmRefunded = idxPmRefunded >= 0 ? parseMoneyWithCurrency(row[idxPmRefunded] || "") : { amount: 0, currency: "" as const };

    if (!detectedCurrency) {
      detectedCurrency = pledge.currency || refunded.currency || pmRefunded.currency;
    }

    const netAmount = round2(pledge.amount - refunded.amount - pmRefunded.amount);
    if (!Number.isFinite(netAmount)) continue;

    byDate.set(day, round2((byDate.get(day) || 0) + netAmount));
  }

  const sortedDates = Array.from(byDate.keys()).sort();
  if (sortedDates.length === 0) {
    throw new Error(tr("В файле нет подходящих платежей (Pledged Status = collected).", "No eligible payments were found in file (Pledged Status = collected)."));
  }

  const startedAt = sortedDates[0];
  const endedAtRaw = sortedDates[sortedDates.length - 1];
  const diff = dateDiffInclusive(startedAt, endedAtRaw);
  if (!diff) {
    throw new Error(tr("Не удалось определить период кампании по датам.", "Failed to determine campaign period from dates."));
  }
  if (diff > 60) {
    throw new Error(tr("Длительность кампании в файле больше 60 дней. Сократите период вручную.", "Campaign duration in file is longer than 60 days. Shorten the period manually."));
  }

  const duration = clampDuration(alignDurationToKickstarterWindow(diff));
  const endedAt = addDaysISO(startedAt, duration - 1);

  const days: HistoricalCampaignDay[] = [];
  let cumulative = 0;
  for (let dayNumber = 1; dayNumber <= duration; dayNumber += 1) {
    const date = addDaysISO(startedAt, dayNumber - 1);
    const dayAmount = byDate.get(date) || 0;
    cumulative = round2(cumulative + dayAmount);
    cumulative = Math.max(0, cumulative);

    days.push({
      day_number: dayNumber,
      income_amount: cumulative,
      expenses: []
    });
  }

  return {
    name: campaignNameFromFileName(fileName),
    category: categoryFallback.trim() || tr("Общее", "General"),
    duration_days: duration,
    currency_code: detectedCurrency || "EUR",
    started_at: startedAt,
    ended_at: endedAt,
    days
  };
}

function dateDiffInclusive(startedAt: string, endedAt: string): number | null {
  const start = new Date(`${startedAt}T00:00:00Z`).getTime();
  const end = new Date(`${endedAt}T00:00:00Z`).getTime();
  if (!Number.isFinite(start) || !Number.isFinite(end) || end < start) return null;
  return Math.floor((end - start) / (24 * 60 * 60 * 1000)) + 1;
}

function dateToMillis(dateISO: string): number {
  const value = new Date(`${dateISO}T00:00:00Z`).getTime();
  return Number.isFinite(value) ? value : 0;
}

function formatDate(dateISO: string, locale: string): string {
  const dt = new Date(`${dateISO}T00:00:00Z`);
  if (Number.isNaN(dt.getTime())) return dateISO;
  return dt.toLocaleDateString(locale);
}

function cloneExpenses(expenses: HistoricalCampaignExpense[]): HistoricalCampaignExpense[] {
  return expenses.map((expense) => ({
    category: expense.category,
    amount: expense.amount
  }));
}

function buildDays(duration: number, sourceDays: HistoricalCampaignDay[] = []): HistoricalCampaignDay[] {
  const safeDuration = clampDuration(duration);
  const sourceMap = new Map<number, HistoricalCampaignDay>();
  for (const source of sourceDays) {
    sourceMap.set(source.day_number, source);
  }

  const days: HistoricalCampaignDay[] = [];
  for (let day = 1; day <= safeDuration; day += 1) {
    const source = sourceMap.get(day);
    days.push({
      day_number: day,
      income_amount: source?.income_amount ?? 0,
      expenses: cloneExpenses(source?.expenses ?? [])
    });
  }
  return days;
}

function emptyCampaignForm(defaultDuration: number, defaultCategory: string): CampaignForm {
  const duration = clampDuration(defaultDuration || 10);
  const startedAt = todayISO();
  return {
    name: "",
    category: defaultCategory || "General",
    duration_days: duration,
    currency_code: "EUR",
    started_at: startedAt,
    ended_at: addDaysISO(startedAt, duration - 1),
    days: buildDays(duration)
  };
}

function cumulativeFromDaily(days: HistoricalCampaignDay[], durationDays: number): HistoricalCampaignDay[] {
  const sorted = [...days].sort((a, b) => a.day_number - b.day_number);
  let cumulative = 0;
  const sourceMap = new Map<number, HistoricalCampaignDay>();

  for (const day of sorted) {
    cumulative = round2(cumulative + day.income_amount);
    sourceMap.set(day.day_number, {
      day_number: day.day_number,
      income_amount: cumulative,
      expenses: cloneExpenses(day.expenses ?? [])
    });
  }

  return buildDays(durationDays, Array.from(sourceMap.values()));
}

function toDailyForPayload(days: HistoricalCampaignDay[]): HistoricalCampaignDay[] {
  const sorted = [...days].sort((a, b) => a.day_number - b.day_number);
  let prevCumulative = 0;

  return sorted.map((day) => {
    const cumulative = round2(Math.max(0, day.income_amount));
    const dayIncome = round2(cumulative - prevCumulative);
    prevCumulative = cumulative;

    return {
      day_number: day.day_number,
      income_amount: dayIncome,
      expenses: (day.expenses ?? [])
        .map((expense) => ({
          category: expense.category.trim(),
          amount: round2(Math.max(0, expense.amount))
        }))
        .filter((expense) => expense.category.length > 0)
    };
  });
}

function validateCumulative(days: HistoricalCampaignDay[], locale: "ru" | "en"): string | null {
  const tr = (ru: string, en: string) => (locale === "en" ? en : ru);
  for (const day of [...days].sort((a, b) => a.day_number - b.day_number)) {
    const current = Math.max(0, day.income_amount);
    if (!Number.isFinite(current)) {
      return tr(
        `Некорректное значение накопленного дохода: день ${day.day_number}.`,
        `Invalid cumulative income value: day ${day.day_number}.`
      );
    }
  }
  return null;
}

export default function HistoricalCampaignsPage() {
  const { locale } = useLocale();
  const { token, ready } = useAuthGuard();
  const confirmAction = useConfirm();
  const importInputRef = useRef<HTMLInputElement | null>(null);
  const [items, setItems] = useState<HistoricalCampaign[]>([]);

  const tr = (ru: string, en: string) => (locale === "en" ? en : ru);
  const appLocale = locale === "en" ? "en-US" : "ru-RU";
  const defaultCategory = tr("Общее", "General");
  const presetExpenseCategories = locale === "en" ? PRESET_EXPENSE_CATEGORIES_EN : PRESET_EXPENSE_CATEGORIES_RU;

  const [filterCategory, setFilterCategory] = useState("");
  const [isEditorOpen, setIsEditorOpen] = useState(false);
  const [editingCampaignId, setEditingCampaignId] = useState<string | null>(null);
  const [form, setForm] = useState<CampaignForm>(emptyCampaignForm(10, defaultCategory));
  const [selectedDayNumber, setSelectedDayNumber] = useState(1);
  const [sortBy, setSortBy] = useState<HistorySort>("created_desc");
  const [selectedCampaignIds, setSelectedCampaignIds] = useState<string[]>([]);

  const [saving, setSaving] = useState(false);
  const [importing, setImporting] = useState(false);
  const [deletingSelected, setDeletingSelected] = useState(false);
  const [error, setError] = useState("");

  async function loadItems(authToken: string, category = filterCategory) {
    const qs = category.trim() ? `?category=${encodeURIComponent(category.trim())}` : "";
    const response = await apiRequest<HistoricalCampaign[]>(`/historical-campaigns${qs}`, { token: authToken });
    setItems(response);
  }

  async function load() {
    if (!token) return;
    setError("");
    try {
      let category = filterCategory;
      if (!category) {
        const initialCategory = new URLSearchParams(window.location.search).get("category")?.trim() || "";
        if (initialCategory) {
          category = initialCategory;
          setFilterCategory(initialCategory);
          setForm((prev) => ({ ...prev, category: initialCategory }));
        }
      }
      await loadItems(token, category);
    } catch (err) {
      setError((err as Error).message);
    }
  }

  useEffect(() => {
    if (ready) load();
  }, [ready]);

  useEffect(() => {
    if (!form.days.length) {
      setSelectedDayNumber(1);
      return;
    }
    if (selectedDayNumber < 1 || selectedDayNumber > form.days.length) {
      setSelectedDayNumber(1);
    }
  }, [form.days.length, selectedDayNumber]);

  const cumulativeRows = useMemo<CumulativeRow[]>(() => {
    let prev = 0;
    return form.days.map((day) => {
      const cumulative = Math.max(0, day.income_amount);
      const dayIncome = round2(cumulative - prev);
      prev = cumulative;
      return {
        day_number: day.day_number,
        cumulative_income: cumulative,
        day_income: dayIncome
      };
    });
  }, [form.days]);

  const selectedDay = form.days.find((day) => day.day_number === selectedDayNumber) ?? null;

  const selectedDayIncome =
    cumulativeRows.find((row) => row.day_number === selectedDayNumber)?.day_income ?? 0;

  const selectedDayExpenseTotal = (selectedDay?.expenses ?? []).reduce((sum, expense) => sum + expense.amount, 0);

  const stats = useMemo(() => {
    const campaigns = items.length;
    if (!campaigns) {
      return {
        campaigns: 0,
        avgDuration: 0
      };
    }

    let totalDuration = 0;
    for (const item of items) {
      totalDuration += item.duration_days;
    }

    return {
      campaigns,
      avgDuration: totalDuration / campaigns
    };
  }, [items]);

  const sortedItems = useMemo(() => {
    const withTotals = items.map((item) => ({
      item,
      totalIncome: item.days.reduce((sum, day) => sum + day.income_amount, 0)
    }));

    withTotals.sort((a, b) => {
      switch (sortBy) {
        case "created_asc":
          return new Date(a.item.created_at).getTime() - new Date(b.item.created_at).getTime();
        case "period_desc":
          return dateToMillis(b.item.ended_at) - dateToMillis(a.item.ended_at);
        case "period_asc":
          return dateToMillis(a.item.started_at) - dateToMillis(b.item.started_at);
        case "income_desc":
          return b.totalIncome - a.totalIncome;
        case "income_asc":
          return a.totalIncome - b.totalIncome;
        case "created_desc":
        default:
          return new Date(b.item.created_at).getTime() - new Date(a.item.created_at).getTime();
      }
    });

    return withTotals.map((entry) => entry.item);
  }, [items, sortBy]);

  const deletableItems = useMemo(
    () => sortedItems.filter((item) => item.source !== "completed_project"),
    [sortedItems]
  );
  const selectedSet = useMemo(() => new Set(selectedCampaignIds), [selectedCampaignIds]);
  const allDeletableSelected = deletableItems.length > 0 && deletableItems.every((item) => selectedSet.has(item.id));

  useEffect(() => {
    const validIds = new Set(items.filter((item) => item.source !== "completed_project").map((item) => item.id));
    setSelectedCampaignIds((prev) => prev.filter((id) => validIds.has(id)));
  }, [items]);

  function openCreate() {
    setEditingCampaignId(null);
    setForm(emptyCampaignForm(10, filterCategory.trim() || defaultCategory));
    setSelectedDayNumber(1);
    setError("");
    setIsEditorOpen(true);
  }

  async function importFromCSV(file: File) {
    if (!file) return;
    setImporting(true);
    setError("");
    try {
      const text = await file.text();
      const imported = parseKickstarterCsvToForm(file.name, text, filterCategory.trim() || defaultCategory, locale);
      setEditingCampaignId(null);
      setForm(imported);
      setSelectedDayNumber(1);
      setIsEditorOpen(true);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setImporting(false);
    }
  }

  function openEdit(item: HistoricalCampaign) {
    const duration = clampDuration(item.duration_days);
    setEditingCampaignId(item.id);
    setForm({
      name: item.name,
      category: item.category || defaultCategory,
      duration_days: duration,
      currency_code: item.currency_code,
      started_at: item.started_at,
      ended_at: item.ended_at,
      days: cumulativeFromDaily(item.days, duration)
    });
    setSelectedDayNumber(1);
    setError("");
    setIsEditorOpen(true);
  }

  function closeEditor() {
    if (saving) return;
    setIsEditorOpen(false);
    setEditingCampaignId(null);
  }

  function updateDuration(value: string) {
    const duration = clampDuration(Number.parseInt(value || "0", 10));
    setForm((prev) => ({
      ...prev,
      duration_days: duration,
      ended_at: addDaysISO(prev.started_at, duration - 1),
      days: buildDays(duration, prev.days)
    }));
  }

  function updateStartedAt(value: string) {
    setForm((prev) => ({
      ...prev,
      started_at: value,
      ended_at: addDaysISO(value, prev.duration_days - 1)
    }));
  }

  function updateEndedAt(value: string) {
    const diff = dateDiffInclusive(form.started_at, value);
    if (diff != null && diff >= 3 && diff <= 60) {
      const duration = clampDuration(diff);
      setForm((prev) => ({
        ...prev,
        ended_at: value,
        duration_days: duration,
        days: buildDays(duration, prev.days)
      }));
      return;
    }
    setForm((prev) => ({ ...prev, ended_at: value }));
  }

  function updateCumulativeIncome(dayNumber: number, value: string) {
    const amount = parseAmountInput(value);
    setForm((prev) => ({
      ...prev,
      days: prev.days.map((day) =>
        day.day_number === dayNumber
          ? {
              ...day,
              income_amount: amount
            }
          : day
      )
    }));
  }

  function updateSelectedDayExpenses(mutator: (current: HistoricalCampaignExpense[]) => HistoricalCampaignExpense[]) {
    setForm((prev) => ({
      ...prev,
      days: prev.days.map((day) => {
        if (day.day_number !== selectedDayNumber) return day;
        return {
          ...day,
          expenses: mutator(cloneExpenses(day.expenses ?? []))
        };
      })
    }));
  }

  function addExpenseRow(category = "") {
    updateSelectedDayExpenses((current) => [...current, { category, amount: 0 }]);
  }

  function addPresetCategory(category: string) {
    const normalized = normalizeCategory(category);
    updateSelectedDayExpenses((current) => {
      const exists = current.some((expense) => normalizeCategory(expense.category) === normalized);
      if (exists) return current;
      return [...current, { category, amount: 0 }];
    });
  }

  function updateExpenseCategory(index: number, category: string) {
    updateSelectedDayExpenses((current) => {
      if (!current[index]) return current;
      current[index] = { ...current[index], category };
      return current;
    });
  }

  function updateExpenseAmount(index: number, value: string) {
    const amount = parseAmountInput(value);
    updateSelectedDayExpenses((current) => {
      if (!current[index]) return current;
      current[index] = { ...current[index], amount };
      return current;
    });
  }

  function removeExpenseRow(index: number) {
    updateSelectedDayExpenses((current) => current.filter((_, i) => i !== index));
  }

  async function submit(e: FormEvent) {
    e.preventDefault();
    if (!token) return;

    const name = form.name.trim();
    const category = form.category.trim();
    if (!name) {
      setError(tr("Укажите название кампании.", "Please provide campaign name."));
      return;
    }
    if (category.length < 2 || category.length > 80) {
      setError(tr("Категория должна быть длиной от 2 до 80 символов.", "Category length must be from 2 to 80 characters."));
      return;
    }

    const validationError = validateCumulative(form.days, locale);
    if (validationError) {
      setError(validationError);
      return;
    }

    const payload: UpsertHistoricalCampaignRequest = {
      name,
      category,
      duration_days: clampDuration(form.duration_days),
      currency_code: form.currency_code,
      started_at: form.started_at,
      ended_at: form.ended_at,
      days: toDailyForPayload(form.days)
    };

    setSaving(true);
    setError("");
    try {
      if (editingCampaignId) {
        await apiRequest<HistoricalCampaign>(`/historical-campaigns/${editingCampaignId}`, {
          method: "PUT",
          token,
          body: payload
        });
      } else {
        await apiRequest<HistoricalCampaign>("/historical-campaigns", {
          method: "POST",
          token,
          body: payload
        });
      }

      await loadItems(token, filterCategory);
      setIsEditorOpen(false);
      setEditingCampaignId(null);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setSaving(false);
    }
  }

  async function deleteItem(item: HistoricalCampaign) {
    if (!token) return;
    const confirmed = await confirmAction({
      title: tr("Удалить кампанию?", "Delete campaign?"),
      message: tr(`Кампания «${item.name}» будет удалена из истории.`, `Campaign "${item.name}" will be removed from history.`),
      confirmText: tr("Удалить", "Delete"),
      cancelText: tr("Отмена", "Cancel"),
      tone: "danger"
    });
    if (!confirmed) return;

    try {
      await apiRequest<void>(`/historical-campaigns/${item.id}`, {
        method: "DELETE",
        token
      });
      setSelectedCampaignIds((prev) => prev.filter((id) => id !== item.id));
      await loadItems(token, filterCategory);
    } catch (err) {
      setError((err as Error).message);
    }
  }

  function toggleCampaignSelection(campaignId: string, checked: boolean) {
    setSelectedCampaignIds((prev) => {
      if (checked) {
        return prev.includes(campaignId) ? prev : [...prev, campaignId];
      }
      return prev.filter((id) => id !== campaignId);
    });
  }

  function toggleSelectAllCampaigns() {
    if (!deletableItems.length) return;
    if (allDeletableSelected) {
      setSelectedCampaignIds([]);
      return;
    }
    setSelectedCampaignIds(deletableItems.map((item) => item.id));
  }

  async function deleteSelectedCampaigns() {
    if (!token || selectedCampaignIds.length === 0) return;

    const confirmed = await confirmAction({
      title: tr("Удалить выбранные кампании?", "Delete selected campaigns?"),
      message: tr(
        `Будет удалено записей: ${selectedCampaignIds.length}. Это действие нельзя отменить.`,
        `${selectedCampaignIds.length} records will be deleted. This action cannot be undone.`
      ),
      confirmText: tr("Удалить", "Delete"),
      cancelText: tr("Отмена", "Cancel"),
      tone: "danger"
    });
    if (!confirmed) return;

    setDeletingSelected(true);
    setError("");
    try {
      const results = await Promise.allSettled(
        selectedCampaignIds.map((campaignId) =>
          apiRequest<void>(`/historical-campaigns/${campaignId}`, {
            method: "DELETE",
            token
          })
        )
      );

      const failedIds = results
        .map((result, idx) => ({ result, id: selectedCampaignIds[idx] }))
        .filter((entry) => entry.result.status === "rejected")
        .map((entry) => entry.id);

      await loadItems(token, filterCategory);
      setSelectedCampaignIds(failedIds);

      if (failedIds.length > 0) {
        setError(tr(`Не удалось удалить записей: ${failedIds.length}. Проверьте доступ и повторите.`, `Failed to delete records: ${failedIds.length}. Check access and retry.`));
      }
    } finally {
      setDeletingSelected(false);
    }
  }

  if (!ready) return null;

  return (
    <AppShell title={tr("История кампаний", "Campaign history")} className="historical-campaigns-shell">
      {error && <p style={{ color: "#bc4749" }}>{error}</p>}

      <HistoricalCampaignToolbar
        tr={tr}
        appLocale={appLocale}
        stats={stats}
        filterCategory={filterCategory}
        sortBy={sortBy}
        onFilterCategoryChange={setFilterCategory}
        onSortChange={setSortBy}
        onApplyFilters={() => token && void loadItems(token, filterCategory)}
        deletableCount={deletableItems.length}
        allDeletableSelected={allDeletableSelected}
        onToggleSelectAll={toggleSelectAllCampaigns}
        selectedCount={selectedCampaignIds.length}
        deletingSelected={deletingSelected}
        onDeleteSelected={deleteSelectedCampaigns}
        importing={importing}
        importInputRef={importInputRef}
        onImportFile={importFromCSV}
        onOpenCreate={openCreate}
        formatAmount={formatAmount}
      />

      <HistoricalCampaignList
        tr={tr}
        appLocale={appLocale}
        items={sortedItems}
        selectedSet={selectedSet}
        defaultCategory={defaultCategory}
        onToggleCampaign={toggleCampaignSelection}
        onOpenEdit={openEdit}
        onDeleteItem={deleteItem}
        formatAmount={formatAmount}
        currencySymbol={currencySymbol}
        formatDate={formatDate}
      />

      {isEditorOpen && (
        <HistoricalCampaignEditor
          tr={tr}
          appLocale={appLocale}
          editingCampaignId={editingCampaignId}
          saving={saving}
          form={form}
          cumulativeRows={cumulativeRows}
          selectedDayNumber={selectedDayNumber}
          selectedDay={selectedDay}
          selectedDayIncome={selectedDayIncome}
          selectedDayExpenseTotal={selectedDayExpenseTotal}
          presetExpenseCategories={presetExpenseCategories}
          dateLayout={DATE_LAYOUT}
          currencySymbol={currencySymbol}
          formatAmount={formatAmount}
          onClose={closeEditor}
          onSubmit={submit}
          onNameChange={(value) => setForm((prev) => ({ ...prev, name: value }))}
          onCategoryChange={(value) => setForm((prev) => ({ ...prev, category: value }))}
          onDurationChange={updateDuration}
          onStartedAtChange={updateStartedAt}
          onEndedAtChange={updateEndedAt}
          onCurrencyChange={(value) => setForm((prev) => ({ ...prev, currency_code: value }))}
          onCumulativeIncomeChange={updateCumulativeIncome}
          onSelectedDayChange={setSelectedDayNumber}
          onAddPresetCategory={addPresetCategory}
          onExpenseCategoryChange={updateExpenseCategory}
          onExpenseAmountChange={updateExpenseAmount}
          onRemoveExpenseRow={removeExpenseRow}
          onAddExpenseRow={() => addExpenseRow()}
        />
      )}
    </AppShell>
  );
}
