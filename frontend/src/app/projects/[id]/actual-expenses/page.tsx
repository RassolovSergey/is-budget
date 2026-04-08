"use client";

import { FormEvent, useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";

import { AppShell } from "../../../../components/AppShell";
import { useConfirm } from "../../../../components/ConfirmProvider";
import { useLocale } from "../../../../components/LocaleProvider";
import { ProjectNav } from "../../../../components/ProjectNav";
import { apiRequest } from "../../../../lib/api";
import { currencySymbol, money, todayISO } from "../../../../lib/format";
import type { ActualExpense, Project } from "../../../../lib/types";
import { useAuthGuard } from "../../../../lib/useAuthGuard";

type ActualSort = "paid_at" | "created_at";

type FormState = {
  id?: string;
  title: string;
  category: string;
  amount: string;
  paid_at: string;
  note: string;
};

const initialForm: FormState = {
  title: "",
  category: "",
  amount: "",
  paid_at: todayISO(),
  note: ""
};

function toMillis(value: string | undefined): number {
  if (!value) return 0;
  const ts = new Date(value).getTime();
  return Number.isNaN(ts) ? 0 : ts;
}

export default function ActualExpensesPage() {
  const { id } = useParams<{ id: string }>();
  const confirmAction = useConfirm();
  const { locale } = useLocale();
  const { token, ready } = useAuthGuard();
  const [project, setProject] = useState<Project | null>(null);
  const [items, setItems] = useState<ActualExpense[]>([]);
  const [form, setForm] = useState<FormState>(initialForm);
  const [sortBy, setSortBy] = useState<ActualSort>("paid_at");
  const [error, setError] = useState("");

  const sortedItems = [...items].sort((a, b) => {
    if (sortBy === "created_at") {
      return toMillis(b.created_at) - toMillis(a.created_at);
    }
    return toMillis(a.paid_at) - toMillis(b.paid_at);
  });

  async function load() {
    if (!token) return;
    try {
      const [projectResp, expensesResp] = await Promise.all([
        apiRequest<Project>(`/projects/${id}`, { token }),
        apiRequest<ActualExpense[]>(`/projects/${id}/actual-expenses?from=2000-01-01&to=2100-12-31`, { token })
      ]);
      setProject(projectResp);
      setItems(expensesResp);
    } catch (err) {
      setError((err as Error).message);
    }
  }

  useEffect(() => {
    if (ready) void load();
  }, [ready, id, token]);

  const tr = (ru: string, en: string) => (locale === "en" ? en : ru);
  const legacyReadOnly = true;

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    if (legacyReadOnly) {
      setError(tr("Этот legacy-раздел доступен только для просмотра. Новые списания теперь настраиваются через «Финансовый конвейер».", "This legacy section is read-only. Configure new deductions through Finance Pipeline."));
      return;
    }
    setError("");
    const amount = Number(form.amount);
    if (form.amount.trim() === "" || Number.isNaN(amount) || amount < 0) {
      setError(tr("Сумма должна быть числом не меньше 0.", "Amount must be a number greater than or equal to 0."));
      return;
    }
    const body = {
      title: form.title,
      category: form.category,
      amount,
      paid_at: form.paid_at,
      note: form.note
    };
    try {
      if (form.id) {
        await apiRequest(`/projects/${id}/actual-expenses/${form.id}`, { method: "PUT", token, body });
      } else {
        await apiRequest(`/projects/${id}/actual-expenses`, { method: "POST", token, body });
      }
      setForm(initialForm);
      await load();
    } catch (err) {
      setError((err as Error).message);
    }
  }

  async function onDelete(expenseId: string) {
    if (legacyReadOnly) {
      setError(tr("Этот legacy-раздел доступен только для просмотра. Новые списания теперь настраиваются через «Финансовый конвейер».", "This legacy section is read-only. Configure new deductions through Finance Pipeline."));
      return;
    }
    const confirmed = await confirmAction({
      title: tr("Удалить фактический расход?", "Delete actual expense?"),
      message: tr(
        "Запись о фактическом расходе будет удалена без возможности восстановления.",
        "This expense record will be permanently deleted."
      ),
      confirmText: tr("Удалить", "Delete"),
      cancelText: tr("Отмена", "Cancel"),
      tone: "danger"
    });
    if (!confirmed) return;

    try {
      await apiRequest(`/projects/${id}/actual-expenses/${expenseId}`, { method: "DELETE", token });
      await load();
    } catch (err) {
      setError((err as Error).message);
    }
  }

  if (!ready || !project) return null;
  const projectCurrency = project.currency_code;
  const canFinanceWrite = false;
  const totalSpent = items.reduce((sum, item) => sum + item.amount, 0);
  const categoryCount = new Set(items.map((item) => item.category.trim().toLowerCase()).filter(Boolean)).size;
  const nearestPayment = sortedItems[0]?.paid_at?.slice(0, 10) || "";
  const averageExpense = items.length ? totalSpent / items.length : 0;
  const biggestExpense = items.reduce<ActualExpense | null>((largest, item) => (!largest || item.amount > largest.amount ? item : largest), null);

  return (
    <AppShell title={project.name} className="actual-expenses-shell">
      <ProjectNav projectId={id} />
      {error && <p style={{ color: "#bc4749" }}>{error}</p>}

      <section className="card legacy-route-note">
        <strong>{tr("Legacy-раздел расходов", "Legacy expense section")}</strong>
        <p>
          {tr(
            "Эта страница сохранена для совместимости со старыми данными. Основную настройку денежного потока и новых списаний теперь ведите через «Финансовый конвейер».",
            "This page is kept for compatibility with older data. Use Finance Pipeline as the main place for configuring money flow and new deductions."
          )}
        </p>
        <Link href={`/projects/${id}/finance-pipeline`} className="inline-link">
          {tr("Открыть финансовый конвейер", "Open Finance Pipeline")}
        </Link>
      </section>

      <section className="card expenses-overview-card">
        <div className="expenses-overview-head">
          <div>
            <span className="shell-eyebrow">{tr("Журнал", "Ledger")}</span>
            <h2 className="title">{tr("Фактические расходы проекта", "Actual project expenses")}</h2>
          </div>
          <p className="subtle" style={{ margin: 0 }}>
            {tr(
              "Все оплаченные траты проекта собраны в одном месте: с категориями, суммами и пояснениями по каждому платежу.",
              "All real project spend is collected here with categories, amounts, and context for every payment."
            )}
          </p>
        </div>
        <div className="expenses-overview-grid expenses-overview-grid-rich">
          <article className="expenses-overview-item expenses-overview-item-accent">
            <span>{tr("Потрачено всего", "Total spent")}</span>
            <strong>{money(totalSpent, projectCurrency)}</strong>
            <p>{tr("Сумма подтверждённых фактических расходов проекта", "Sum of confirmed actual project expenses")}</p>
          </article>
          <article className="expenses-overview-item">
            <span>{tr("Средний платёж", "Average payment")}</span>
            <strong>{money(averageExpense, projectCurrency)}</strong>
            <p>{tr("Средний размер одной расходной операции", "Average size of a single expense entry")}</p>
          </article>
          <article className="expenses-overview-item">
            <span>{tr("Категорий расходов", "Expense categories")}</span>
            <strong>{categoryCount}</strong>
            <p>{tr("Сколько направлений трат уже зафиксировано", "How many spend categories are already tracked")}</p>
          </article>
          <article className="expenses-overview-item">
            <span>{tr("Крупнейший расход", "Largest expense")}</span>
            <strong>{biggestExpense ? money(biggestExpense.amount, projectCurrency) : "—"}</strong>
            <p>{biggestExpense ? biggestExpense.title : tr("Пока нет записей", "No entries yet")}</p>
          </article>
          <article className="expenses-overview-item">
            <span>{tr("Ближайшая запись", "Nearest ledger date")}</span>
            <strong>{nearestPayment || "—"}</strong>
            <p>{tr("Дата ближайшего платежа в текущей сортировке", "Nearest payment date in the current sort order")}</p>
          </article>
        </div>
      </section>

      <div className="grid two actual-expenses-top-grid">
        <section className="card finance-form-card">
          <div className="finance-card-head">
            <div>
              <span className="shell-eyebrow">{tr("Расход", "Expense")}</span>
              <h2 className="title">{form.id ? tr("Редактировать фактический расход", "Edit actual expense") : tr("Добавить фактический расход", "Add actual expense")}</h2>
            </div>
            <p className="subtle" style={{ margin: 0 }}>
              {tr(
                "Записывайте реальные траты проекта с категорией, суммой и пояснением.",
                "Record real project spending with category, amount, and context."
              )}
            </p>
          </div>
          <form className="grid finance-form-grid" onSubmit={onSubmit}>
            <label>
              {tr("Название расхода", "Expense title")}
              <input placeholder={tr("Например: Meta Ads / печать образцов", "For example: Meta Ads / prototype print") } value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })} required disabled={!canFinanceWrite} />
            </label>
            <label>
              {tr("Категория", "Category")}
              <input placeholder={tr("Реклама, производство, логистика...", "Advertising, production, logistics...") } value={form.category} onChange={(e) => setForm({ ...form, category: e.target.value })} required disabled={!canFinanceWrite} />
            </label>
            <label>
              {tr("Сумма", "Amount")}
              <div className="input-with-suffix">
                <input
                  type="number"
                  min="0"
                  step="1"
                  placeholder="0"
                  value={form.amount}
                  onChange={(e) => setForm({ ...form, amount: e.target.value })}
                  disabled={!canFinanceWrite}
                  required
                />
                <span className="input-suffix">{currencySymbol(projectCurrency)}</span>
              </div>
            </label>
            <label>
              {tr("Дата оплаты", "Payment date")}
              <input type="date" value={form.paid_at} onChange={(e) => setForm({ ...form, paid_at: e.target.value })} required disabled={!canFinanceWrite} />
            </label>
            <label className="finance-form-span-full">
              {tr("Примечание", "Note")}
              <textarea placeholder={tr("Коротко опишите, за что именно был платёж.", "Briefly describe what this payment was for.")} rows={4} value={form.note} onChange={(e) => setForm({ ...form, note: e.target.value })} disabled={!canFinanceWrite} />
            </label>
            <div className="finance-form-actions">
              <button disabled={!canFinanceWrite}>{form.id ? tr("Обновить", "Update") : tr("Создать", "Create")}</button>
              {form.id && (
                <button className="secondary" type="button" onClick={() => setForm(initialForm)} disabled={!canFinanceWrite}>
                  {tr("Отмена", "Cancel")}
                </button>
              )}
            </div>
            {!canFinanceWrite && (
              <p className="subtle" style={{ margin: 0 }}>
                {tr(
                  "У вас только просмотр. Изменение фактических расходов доступно ролям PM/ANALYST.",
                  "You have read-only access. Editing actual expenses is available for PM/ANALYST roles."
                )}
              </p>
            )}
          </form>
        </section>

        <section className="card ledger-card">
          <div className="section-head">
            <div>
              <h2 className="title">{tr("Журнал фактических расходов", "Actual expense ledger")}</h2>
              <p className="subtle" style={{ margin: 0 }}>{tr("Просматривайте траты как отдельные финансовые записи, а не как перегруженную таблицу.", "Review spending as individual finance records instead of a dense table.")}</p>
            </div>
            <label className="sort-control">
              {tr("Сортировка", "Sorting")}
              <select value={sortBy} onChange={(e) => setSortBy(e.target.value as ActualSort)}>
                <option value="paid_at">{tr("По сроку оплаты", "By payment date")}</option>
                <option value="created_at">{tr("По дате создания", "By created date")}</option>
              </select>
            </label>
          </div>

          <div className="ledger-list">
            {sortedItems.map((item) => (
              <article key={item.id} className="ledger-item-card">
                <div className="ledger-item-main">
                  <div className="ledger-item-topline">
                    <div>
                      <h3>{item.title}</h3>
                      <div className="ledger-item-tags">
                        <span>{item.category}</span>
                        <span>{item.paid_at.slice(0, 10)}</span>
                      </div>
                    </div>
                    <div className="ledger-amount-block">
                      <strong>{money(item.amount, projectCurrency)}</strong>
                      <span>{tr("Фактический расход", "Actual expense")}</span>
                    </div>
                  </div>
                  {item.note && <p className="ledger-item-note">{item.note}</p>}
                </div>
                <div className="ledger-item-actions">
                  <button
                    className="secondary"
                    disabled={!canFinanceWrite}
                    onClick={() =>
                      setForm({
                        id: item.id,
                        title: item.title,
                        category: item.category,
                        amount: String(item.amount),
                        paid_at: item.paid_at.slice(0, 10),
                        note: item.note
                      })
                    }
                  >
                    {tr("Редактировать", "Edit")}
                  </button>
                  <button className="warn" onClick={() => onDelete(item.id)} disabled={!canFinanceWrite}>{tr("Удалить", "Delete")}</button>
                </div>
              </article>
            ))}
            {!items.length && (
              <div className="ledger-empty-state">
                <strong>{tr("Пока нет фактических расходов", "No actual expenses yet")}</strong>
                <span>{tr("Добавьте первый расход слева, и он появится здесь в журнале.", "Add your first expense on the left and it will appear here in the ledger.")}</span>
              </div>
            )}
          </div>
        </section>
      </div>
    </AppShell>
  );
}
