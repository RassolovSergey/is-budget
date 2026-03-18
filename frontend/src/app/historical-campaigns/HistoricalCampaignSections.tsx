"use client";

import type { ReactNode, RefObject } from "react";

import type {
  HistoricalCampaign,
  HistoricalCampaignDay,
  HistoricalCampaignExpense,
  CurrencyCode
} from "../../lib/types";
import type { CampaignForm, CumulativeRow, HistorySort } from "./types";

type Translator = (ru: string, en: string) => string;

type HistoricalToolbarProps = {
  tr: Translator;
  appLocale: string;
  stats: { campaigns: number; avgDuration: number };
  filterCategory: string;
  sortBy: HistorySort;
  onFilterCategoryChange: (value: string) => void;
  onSortChange: (value: HistorySort) => void;
  onApplyFilters: () => void;
  deletableCount: number;
  allDeletableSelected: boolean;
  onToggleSelectAll: () => void;
  selectedCount: number;
  deletingSelected: boolean;
  onDeleteSelected: () => void;
  importing: boolean;
  importInputRef: RefObject<HTMLInputElement | null>;
  onImportFile: (file: File) => void;
  onOpenCreate: () => void;
  formatAmount: (value: number, locale: string) => string;
};

type HistoricalCampaignListProps = {
  tr: Translator;
  appLocale: string;
  items: HistoricalCampaign[];
  selectedSet: Set<string>;
  defaultCategory: string;
  onToggleCampaign: (campaignId: string, checked: boolean) => void;
  onOpenEdit: (item: HistoricalCampaign) => void;
  onDeleteItem: (item: HistoricalCampaign) => void;
  formatAmount: (value: number, locale: string) => string;
  currencySymbol: (code: CurrencyCode) => string;
  formatDate: (dateISO: string, locale: string) => string;
};

type HistoricalCampaignEditorProps = {
  tr: Translator;
  appLocale: string;
  editingCampaignId: string | null;
  saving: boolean;
  form: CampaignForm;
  cumulativeRows: CumulativeRow[];
  selectedDayNumber: number;
  selectedDay: HistoricalCampaignDay | null;
  selectedDayIncome: number;
  selectedDayExpenseTotal: number;
  presetExpenseCategories: string[];
  dateLayout: string;
  currencySymbol: (code: CurrencyCode) => string;
  formatAmount: (value: number, locale: string) => string;
  onClose: () => void;
  onSubmit: (e: React.FormEvent) => void;
  onNameChange: (value: string) => void;
  onCategoryChange: (value: string) => void;
  onDurationChange: (value: string) => void;
  onStartedAtChange: (value: string) => void;
  onEndedAtChange: (value: string) => void;
  onCurrencyChange: (value: CurrencyCode) => void;
  onCumulativeIncomeChange: (dayNumber: number, value: string) => void;
  onSelectedDayChange: (dayNumber: number) => void;
  onAddPresetCategory: (category: string) => void;
  onExpenseCategoryChange: (index: number, value: string) => void;
  onExpenseAmountChange: (index: number, value: string) => void;
  onRemoveExpenseRow: (index: number) => void;
  onAddExpenseRow: () => void;
};

export function HistoricalCampaignToolbar({
  tr,
  appLocale,
  stats,
  filterCategory,
  sortBy,
  onFilterCategoryChange,
  onSortChange,
  onApplyFilters,
  deletableCount,
  allDeletableSelected,
  onToggleSelectAll,
  selectedCount,
  deletingSelected,
  onDeleteSelected,
  importing,
  importInputRef,
  onImportFile,
  onOpenCreate,
  formatAmount
}: HistoricalToolbarProps) {
  return (
    <section className="card history-toolbar">
      <div className="history-toolbar-hero">
        <div>
          <span className="shell-eyebrow">{tr("Архив", "Archive")}</span>
          <h2 className="title">{tr("История кампаний", "Campaign history")}</h2>
          <p className="subtle" style={{ margin: 0 }}>
            {tr(
              "Здесь живёт база завершённых кампаний: ручные записи и импортированные CSV, на которых строится аналитика по категориям.",
              "This is your completed-campaign archive: manual entries and imported CSVs used for category forecasting."
            )}
          </p>
        </div>
        <div className="history-toolbar-quickstats">
          <article className="history-quickstat-card history-quickstat-card-accent">
            <span>{tr("Кампаний в архиве", "Campaigns in archive")}</span>
            <strong>{stats.campaigns}</strong>
          </article>
          <article className="history-quickstat-card">
            <span>{tr("Средняя длительность", "Average duration")}</span>
            <strong>{formatAmount(stats.avgDuration, appLocale)} {tr("дн.", "days")}</strong>
          </article>
          <article className="history-quickstat-card">
            <span>{tr("Фильтр категории", "Category filter")}</span>
            <strong>{filterCategory.trim() || tr("Все", "All")}</strong>
          </article>
        </div>
      </div>
      <div className="history-toolbar-panels">
        <div className="history-filter-panel">
          <label className="sort-control">
            {tr("Категория", "Category")}
            <input
              value={filterCategory}
              onChange={(e) => onFilterCategoryChange(e.target.value)}
              placeholder={tr("Например: 3D модельные наборы", "Example: 3D model sets")}
            />
          </label>
          <label className="sort-control">
            {tr("Сортировка", "Sorting")}
            <select value={sortBy} onChange={(e) => onSortChange(e.target.value as HistorySort)}>
              <option value="created_desc">{tr("По добавлению: новые сверху", "By added date: newest first")}</option>
              <option value="created_asc">{tr("По добавлению: старые сверху", "By added date: oldest first")}</option>
              <option value="period_desc">{tr("По периоду: недавние кампании", "By campaign period: recent first")}</option>
              <option value="period_asc">{tr("По периоду: ранние кампании", "By campaign period: earliest first")}</option>
              <option value="income_desc">{tr("По сборам: больше сверху", "By collected amount: highest first")}</option>
              <option value="income_asc">{tr("По сборам: меньше сверху", "By collected amount: lowest first")}</option>
            </select>
          </label>
          <button type="button" className="secondary" onClick={onApplyFilters}>
            {tr("Применить", "Apply")}
          </button>
        </div>
        <div className="history-toolbar-action-groups">
          <div className="history-toolbar-actions history-toolbar-actions-bulk">
            <button type="button" className="secondary" onClick={onToggleSelectAll} disabled={!deletableCount}>
              {allDeletableSelected ? tr("Снять выбор", "Clear selection") : tr("Выбрать все", "Select all")}
            </button>
            <button type="button" className="warn" onClick={onDeleteSelected} disabled={selectedCount === 0 || deletingSelected}>
              {deletingSelected
                ? tr("Удаление...", "Deleting...")
                : tr(`Удалить выбранные (${selectedCount})`, `Delete selected (${selectedCount})`)}
            </button>
          </div>
          <div className="history-toolbar-actions history-toolbar-actions-primary">
            <button type="button" className="secondary" onClick={() => importInputRef.current?.click()} disabled={importing}>
              {importing ? tr("Импорт...", "Importing...") : tr("Импорт CSV", "Import CSV")}
            </button>
            <input
              ref={importInputRef as RefObject<HTMLInputElement>}
              type="file"
              accept=".csv,text/csv"
              style={{ display: "none" }}
              onChange={(e) => {
                const file = e.target.files?.[0];
                if (file) onImportFile(file);
                e.currentTarget.value = "";
              }}
            />
            <button type="button" onClick={onOpenCreate}>{tr("Добавить кампанию", "Add campaign")}</button>
          </div>
        </div>
      </div>
      <p className="subtle history-selected-counter" style={{ margin: 0 }}>
        {tr("Выбрано для удаления", "Selected for deletion")}: {selectedCount}
      </p>
    </section>
  );
}

export function HistoricalCampaignList({
  tr,
  appLocale,
  items,
  selectedSet,
  defaultCategory,
  onToggleCampaign,
  onOpenEdit,
  onDeleteItem,
  formatAmount,
  currencySymbol,
  formatDate
}: HistoricalCampaignListProps) {
  return (
    <div className="history-campaign-list">
      {items.length === 0 && (
        <article className="card history-campaign-empty">
          <p className="subtle" style={{ margin: 0 }}>
            {tr("История пока пуста. Нажмите «Добавить кампанию», чтобы создать первую запись.", "History is empty. Click Add campaign to create the first record.")}
          </p>
        </article>
      )}
      {items.map((item) => {
        const isReadonly = item.source === "completed_project";
        const isSelected = selectedSet.has(item.id);
        const totalIncome = item.days.reduce((sum, day) => sum + day.income_amount, 0);
        const dayOne = item.days.find((day) => day.day_number === 1)?.income_amount ?? 0;
        const totalExpenses = item.days.reduce(
          (sum, day) => sum + day.expenses.reduce((daySum, expense) => daySum + expense.amount, 0),
          0
        );

        return (
          <article key={item.id} className="card history-campaign-card">
            <div className="history-campaign-head">
              <div className="history-campaign-head-main">
                {!isReadonly && (
                  <input
                    type="checkbox"
                    className="history-select-checkbox"
                    checked={isSelected}
                    onChange={(e) => onToggleCampaign(item.id, e.target.checked)}
                    aria-label={tr(`Выбрать кампанию ${item.name}`, `Select campaign ${item.name}`)}
                  />
                )}
                <h3>{item.name}</h3>
              </div>
              <div className="history-badges">
                <span>{isReadonly ? tr("Из завершенного проекта", "From completed project") : tr("Ручная запись", "Manual entry")}</span>
                <span>{item.category || defaultCategory}</span>
                <span>{item.duration_days} {tr("дн.", "days")}</span>
                <span>{item.currency_code}</span>
              </div>
            </div>
            <p className="subtle history-campaign-period" style={{ margin: 0 }}>
              {tr("Период", "Period")}: {formatDate(item.started_at, appLocale)} - {formatDate(item.ended_at, appLocale)}
            </p>
            <p className="subtle history-campaign-created" style={{ margin: 0 }}>
              {tr("Добавлено", "Added")}: {new Date(item.created_at).toLocaleDateString(appLocale)}
            </p>

            <div className="history-inline-kpis">
              <div>
                <p>{tr("Собрано", "Collected")}</p>
                <strong>{formatAmount(totalIncome, appLocale)} {currencySymbol(item.currency_code)}</strong>
              </div>
              <div>
                <p>{tr("День 1", "Day 1")}</p>
                <strong>{formatAmount(dayOne, appLocale)} {currencySymbol(item.currency_code)}</strong>
              </div>
              <div>
                <p>{tr("Расходы", "Expenses")}</p>
                <strong>{formatAmount(totalExpenses, appLocale)} {currencySymbol(item.currency_code)}</strong>
              </div>
            </div>

            <div className="history-card-actions">
              {isReadonly ? (
                <p className="subtle" style={{ margin: 0 }}>
                  {tr("Эта запись формируется автоматически из завершенного проекта.", "This record is generated automatically from a completed project.")}
                </p>
              ) : (
                <>
                  <button type="button" className="secondary" onClick={() => onOpenEdit(item)}>
                    {tr("Редактировать", "Edit")}
                  </button>
                  <button type="button" className="warn" onClick={() => onDeleteItem(item)}>
                    {tr("Удалить", "Delete")}
                  </button>
                </>
              )}
            </div>
          </article>
        );
      })}
    </div>
  );
}

export function HistoricalCampaignEditor({
  tr,
  appLocale,
  editingCampaignId,
  saving,
  form,
  cumulativeRows,
  selectedDayNumber,
  selectedDay,
  selectedDayIncome,
  selectedDayExpenseTotal,
  presetExpenseCategories,
  dateLayout,
  currencySymbol,
  formatAmount,
  onClose,
  onSubmit,
  onNameChange,
  onCategoryChange,
  onDurationChange,
  onStartedAtChange,
  onEndedAtChange,
  onCurrencyChange,
  onCumulativeIncomeChange,
  onSelectedDayChange,
  onAddPresetCategory,
  onExpenseCategoryChange,
  onExpenseAmountChange,
  onRemoveExpenseRow,
  onAddExpenseRow
}: HistoricalCampaignEditorProps) {
  return (
    <div className="confirm-overlay" role="presentation" onClick={onClose}>
      <section className="card history-editor" role="dialog" aria-modal="true" onClick={(event) => event.stopPropagation()}>
        <div className="history-editor-header">
          <h2 className="title" style={{ marginBottom: 0 }}>
            {editingCampaignId ? tr("Редактирование кампании", "Edit campaign") : tr("Новая кампания", "New campaign")}
          </h2>
          <button type="button" className="secondary" onClick={onClose}>{tr("Закрыть", "Close")}</button>
        </div>

        <form onSubmit={onSubmit} className="history-editor-form">
          <div className="history-editor-layout">
            <div className="history-editor-main">
              <div className="history-editor-fields">
                <label>
                  {tr("Название", "Name")}
                  <input value={form.name} onChange={(e) => onNameChange(e.target.value)} placeholder={tr("Например: Kickstarter Wave 1", "For example: Kickstarter Wave 1")} />
                </label>
                <label>
                  {tr("Категория", "Category")}
                  <input value={form.category} onChange={(e) => onCategoryChange(e.target.value)} placeholder={tr("Например: 3D модельные наборы", "For example: 3D model sets")} />
                </label>
                <label>
                  {tr("Длительность (3-60)", "Duration (3-60)")}
                  <input type="number" min={3} max={60} step={1} value={form.duration_days} onChange={(e) => onDurationChange(e.target.value)} />
                </label>
                <label>
                  {tr("Начало кампании", "Campaign start")}
                  <input type="date" value={form.started_at} onChange={(e) => onStartedAtChange(e.target.value)} />
                </label>
                <label>
                  {tr("Конец кампании", "Campaign end")}
                  <input type="date" value={form.ended_at} onChange={(e) => onEndedAtChange(e.target.value)} />
                </label>
                <label>
                  {tr("Валюта", "Currency")}
                  <select value={form.currency_code} onChange={(e) => onCurrencyChange(e.target.value as CurrencyCode)}>
                    <option value="EUR">EUR (€)</option>
                    <option value="USD">USD ($)</option>
                    <option value="RUB">RUB (₽)</option>
                  </select>
                </label>
              </div>
              <p className="subtle" style={{ margin: 0 }}>
                {tr(
                  `Формат даты: ${dateLayout}. При изменении конца кампании длительность пересчитается автоматически, если период в диапазоне 3-60 дней.`,
                  `Date format: ${dateLayout}. When campaign end is changed, duration is recalculated automatically if period stays within 3-60 days.`
                )}
              </p>

              <div className="history-editor-table-wrap">
                <table className="history-income-table">
                  <thead>
                    <tr>
                      <th>{tr("День", "Day")}</th>
                      <th>{tr("Накопленный доход", "Cumulative income")}</th>
                      <th>{tr("Доход за день", "Income for day")}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {cumulativeRows.map((row) => (
                      <tr key={row.day_number}>
                        <td>{tr("День", "Day")} {row.day_number}</td>
                        <td>
                          <div className="input-with-suffix">
                            <input
                              type="number"
                              min={0}
                              step={1}
                              value={row.cumulative_income === 0 ? "" : row.cumulative_income}
                              placeholder="0"
                              onChange={(e) => onCumulativeIncomeChange(row.day_number, e.target.value)}
                            />
                            <span className="input-suffix">{currencySymbol(form.currency_code)}</span>
                          </div>
                        </td>
                        <td>
                          {formatAmount(row.day_income, appLocale)} {currencySymbol(form.currency_code)}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>

            <aside className="history-editor-side">
              <div className="history-day-header">
                <label>
                  {tr("День расходов", "Expense day")}
                  <select value={selectedDayNumber} onChange={(e) => onSelectedDayChange(Number.parseInt(e.target.value, 10))}>
                    {form.days.map((day) => (
                      <option key={day.day_number} value={day.day_number}>
                        {tr("День", "Day")} {day.day_number}
                      </option>
                    ))}
                  </select>
                </label>

                <div className="history-side-kpis">
                  <p>{tr("Доход за день", "Income for day")}: <b>{formatAmount(selectedDayIncome, appLocale)} {currencySymbol(form.currency_code)}</b></p>
                  <p>{tr("Расходы дня", "Day expenses")}: <b>{formatAmount(selectedDayExpenseTotal, appLocale)} {currencySymbol(form.currency_code)}</b></p>
                </div>
              </div>

              <div className="history-quick-cats">
                {presetExpenseCategories.map((category) => (
                  <button key={category} type="button" className="secondary" onClick={() => onAddPresetCategory(category)}>
                    + {category}
                  </button>
                ))}
              </div>

              <div className="history-expenses-list">
                {(selectedDay?.expenses ?? []).map((expense, idx) => (
                  <div key={`${selectedDayNumber}-${idx}`} className="history-expense-row">
                    <input value={expense.category} onChange={(e) => onExpenseCategoryChange(idx, e.target.value)} placeholder={tr("Категория", "Category")} />
                    <div className="input-with-suffix">
                      <input
                        type="number"
                        min={0}
                        step={1}
                        value={expense.amount === 0 ? "" : expense.amount}
                        placeholder="0"
                        onChange={(e) => onExpenseAmountChange(idx, e.target.value)}
                      />
                      <span className="input-suffix">{currencySymbol(form.currency_code)}</span>
                    </div>
                    <button type="button" className="secondary" onClick={() => onRemoveExpenseRow(idx)}>
                      {tr("Удалить", "Delete")}
                    </button>
                  </div>
                ))}
              </div>

              <button type="button" className="secondary" onClick={onAddExpenseRow}>
                {tr("Добавить категорию", "Add category")}
              </button>
            </aside>
          </div>

          <div className="history-editor-actions">
            <button type="submit" disabled={saving}>{saving ? tr("Сохранение...", "Saving...") : tr("Сохранить", "Save")}</button>
            <button type="button" className="secondary" onClick={onClose} disabled={saving}>{tr("Отмена", "Cancel")}</button>
          </div>
        </form>
      </section>
    </div>
  );
}
