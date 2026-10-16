import { useCallback, useEffect, useMemo, useState } from "react";
import {
  createExpense,
  deleteExpense,
  getExpenses,
  getMembers,
  updateExpense,
} from "./api/client";
import ConfirmDialog from "./components/ConfirmDialog";
import ExpenseModal from "./components/ExpenseModal";
import {
  categories,
  type Category,
  type Expense,
  type ExpenseFilters,
  type ExpenseInput,
  type ExpensePage,
  type FamilyMember,
} from "./types";

const emptyFilters: ExpenseFilters = {
  userId: "",
  category: "",
  dateFrom: "",
  dateTo: "",
};

const categoryClass: Record<Category, string> = {
  "еда": "food",
  "транспорт": "transport",
  "жильё": "housing",
  "развлечения": "entertainment",
  "здоровье": "health",
  "покупки": "shopping",
  "другое": "other",
};

export default function App() {
  const [members, setMembers] = useState<FamilyMember[]>([]);
  const [page, setPage] = useState<ExpensePage>({ items: [], total: 0, limit: 100, offset: 0 });
  const [filters, setFilters] = useState<ExpenseFilters>(emptyFilters);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [modalExpense, setModalExpense] = useState<Expense | null | undefined>(undefined);
  const [deleteTarget, setDeleteTarget] = useState<Expense | null>(null);
  const [saving, setSaving] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [toast, setToast] = useState("");

  const loadExpenses = useCallback(async (nextFilters: ExpenseFilters) => {
    setLoading(true);
    setError("");
    try {
      setPage(await getExpenses(nextFilters));
    } catch (requestError) {
      setError(errorMessage(requestError));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void getMembers()
      .then(setMembers)
      .catch((requestError) => setError(errorMessage(requestError)));
  }, []);

  useEffect(() => {
    void loadExpenses(filters);
  }, [filters, loadExpenses]);

  useEffect(() => {
    if (!toast) return;
    const timeout = window.setTimeout(() => setToast(""), 3500);
    return () => window.clearTimeout(timeout);
  }, [toast]);

  const memberNames = useMemo(
    () => new Map(members.map((member) => [member.telegram_user_id, member.display_name])),
    [members],
  );

  const visibleAmount = useMemo(
    () => page.items.reduce((sum, expense) => sum + Number(expense.amount), 0),
    [page.items],
  );

  async function saveExpense(input: ExpenseInput) {
    setSaving(true);
    try {
      if (modalExpense) {
        await updateExpense(modalExpense.id, input);
        setToast("Расход обновлён");
      } else {
        await createExpense(input);
        setToast("Расход добавлен");
      }
      setModalExpense(undefined);
      await loadExpenses(filters);
    } catch (requestError) {
      setError(errorMessage(requestError));
      throw requestError;
    } finally {
      setSaving(false);
    }
  }

  async function confirmDelete() {
    if (!deleteTarget) return;
    setDeleting(true);
    try {
      await deleteExpense(deleteTarget.id);
      setDeleteTarget(null);
      setToast("Расход удалён");
      await loadExpenses(filters);
    } catch (requestError) {
      const message = errorMessage(requestError);
      setError(message);
      setToast(`Не удалось удалить расход: ${message}`);
    } finally {
      setDeleting(false);
    }
  }

  const noMembers = members.length === 0;

  return (
    <div className="app-shell">
      <header className="topbar">
        <div className="brand">
          <span className="brand__mark" aria-hidden="true">₽</span>
          <div>
            <strong>Домашний счёт</strong>
            <span>семейные расходы</span>
          </div>
        </div>
        <div className="topbar__status">
          <span className="status-dot" />
          Панель главы семьи
        </div>
      </header>

      <main>
        <section className="hero">
          <div>
            <p className="eyebrow">Общий обзор</p>
            <h1>Расходы семьи</h1>
            <p className="hero__subtitle">Общий журнал расходов всех членов семьи — от сообщения боту до семейного отчёта.</p>
          </div>
          <button
            className="button button--primary button--with-icon"
            type="button"
            onClick={() => setModalExpense(null)}
            disabled={noMembers}
            title={noMembers ? "Сначала член семьи должен написать Telegram-боту" : undefined}
          >
            <PlusIcon />
            Добавить расход
          </button>
        </section>

        <section className="stats-grid" aria-label="Сводка">
          <article className="stat-card stat-card--accent">
            <span>Сумма в выборке</span>
            <strong>{formatCurrency(visibleAmount)}</strong>
            <small>{page.total > page.items.length ? `по первым ${page.items.length} записям` : "по текущим фильтрам"}</small>
          </article>
          <article className="stat-card">
            <span>Найдено расходов</span>
            <strong>{page.total}</strong>
            <small>записей в базе</small>
          </article>
          <article className="stat-card">
            <span>Членов семьи</span>
            <strong>{members.length}</strong>
            <small>писали Telegram-боту</small>
          </article>
        </section>

        <section className="panel filters-panel" aria-label="Фильтры расходов">
          <div className="filters-panel__heading">
            <div>
              <p className="eyebrow">Фильтры</p>
              <h2>Найти нужные расходы</h2>
            </div>
            {Object.values(filters).some(Boolean) && (
              <button className="text-button" type="button" onClick={() => setFilters(emptyFilters)}>
                Сбросить всё
              </button>
            )}
          </div>
          <div className="filters-grid">
            <label className="field">
              <span>Член семьи</span>
              <select value={filters.userId} onChange={(event) => setFilters({ ...filters, userId: event.target.value })}>
                <option value="">Все</option>
                {members.map((member) => (
                  <option key={member.telegram_user_id} value={member.telegram_user_id}>{member.display_name}</option>
                ))}
              </select>
            </label>
            <label className="field">
              <span>Категория</span>
              <select value={filters.category} onChange={(event) => setFilters({ ...filters, category: event.target.value })}>
                <option value="">Все категории</option>
                {categories.map((category) => <option key={category} value={category}>{category}</option>)}
              </select>
            </label>
            <label className="field">
              <span>С даты</span>
              <input type="date" value={filters.dateFrom} onChange={(event) => setFilters({ ...filters, dateFrom: event.target.value })} />
            </label>
            <label className="field">
              <span>По дату</span>
              <input type="date" value={filters.dateTo} onChange={(event) => setFilters({ ...filters, dateTo: event.target.value })} />
            </label>
          </div>
        </section>

        {error && (
          <div className="error-banner" role="alert">
            <span>Не удалось выполнить запрос: {error}</span>
            <button type="button" onClick={() => { setError(""); void loadExpenses(filters); }}>Повторить</button>
          </div>
        )}

        <section className="panel expenses-panel">
          <div className="expenses-panel__heading">
            <div>
              <p className="eyebrow">Журнал</p>
              <h2>Последние расходы</h2>
            </div>
            <span className="result-count">{page.total} записей</span>
          </div>

          {loading ? (
            <LoadingState />
          ) : page.items.length === 0 ? (
            <EmptyState noMembers={noMembers} onCreate={() => setModalExpense(null)} />
          ) : (
            <>
              <div className="table-wrap">
                <table>
                  <thead>
                    <tr>
                      <th>Дата</th>
                      <th>Член семьи</th>
                      <th>Описание</th>
                      <th>Категория</th>
                      <th className="amount-cell">Сумма</th>
                      <th><span className="sr-only">Действия</span></th>
                    </tr>
                  </thead>
                  <tbody>
                    {page.items.map((expense) => (
                      <tr key={expense.id}>
                        <td className="date-cell">{formatDate(expense.spent_at)}</td>
                        <td>
                          <span className="member-cell">
                            <span className="avatar">{initials(memberNames.get(expense.user_id) ?? String(expense.user_id))}</span>
                            {memberNames.get(expense.user_id) ?? `ID ${expense.user_id}`}
                          </span>
                        </td>
                        <td className="comment-cell">{expense.comment || "Без комментария"}</td>
                        <td><CategoryBadge category={expense.category} /></td>
                        <td className="amount-cell">{formatCurrency(Number(expense.amount))}</td>
                        <td>
                          <div className="row-actions">
                            <button className="icon-button" type="button" onClick={() => setModalExpense(expense)} aria-label={`Редактировать ${expense.comment || expense.category}`}>
                              <EditIcon />
                            </button>
                            <button className="icon-button icon-button--danger" type="button" onClick={() => setDeleteTarget(expense)} aria-label={`Удалить ${expense.comment || expense.category}`}>
                              <TrashIcon />
                            </button>
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>

              <div className="expense-cards">
                {page.items.map((expense) => (
                  <article className="expense-card" key={expense.id}>
                    <div className="expense-card__top">
                      <CategoryBadge category={expense.category} />
                      <strong>{formatCurrency(Number(expense.amount))}</strong>
                    </div>
                    <h3>{expense.comment || "Без комментария"}</h3>
                    <p>{memberNames.get(expense.user_id) ?? `ID ${expense.user_id}`} · {formatDate(expense.spent_at)}</p>
                    <div className="expense-card__actions">
                      <button className="text-button" type="button" onClick={() => setModalExpense(expense)}>Изменить</button>
                      <button className="text-button text-button--danger" type="button" onClick={() => setDeleteTarget(expense)}>Удалить</button>
                    </div>
                  </article>
                ))}
              </div>
            </>
          )}
        </section>
      </main>

      {modalExpense !== undefined && (
        <ExpenseModal
          expense={modalExpense}
          members={members}
          saving={saving}
          onClose={() => setModalExpense(undefined)}
          onSubmit={saveExpense}
        />
      )}

      {deleteTarget && (
        <ConfirmDialog
          expense={deleteTarget}
          deleting={deleting}
          onCancel={() => setDeleteTarget(null)}
          onConfirm={confirmDelete}
        />
      )}

      <div className={`toast ${toast ? "toast--visible" : ""}`} role="status" aria-live="polite">
        {toast}
      </div>
    </div>
  );
}

function CategoryBadge({ category }: { category: Category }) {
  return <span className={`category-badge category-badge--${categoryClass[category]}`}>{category}</span>;
}

function EmptyState({ noMembers, onCreate }: { noMembers: boolean; onCreate: () => void }) {
  return (
    <div className="empty-state">
      <span className="empty-state__icon" aria-hidden="true">₽</span>
      <h3>{noMembers ? "Семья пока не подключена" : "Расходов не найдено"}</h3>
      <p>
        {noMembers
          ? "Попросите члена семьи написать Telegram-боту — после первого сообщения он появится здесь."
          : "Измените фильтры или добавьте первую запись вручную."}
      </p>
      {!noMembers && <button className="button button--secondary" type="button" onClick={onCreate}>Добавить расход</button>}
    </div>
  );
}

function LoadingState() {
  return (
    <div className="loading-state" aria-label="Загрузка расходов">
      {[1, 2, 3, 4].map((item) => <span key={item} />)}
    </div>
  );
}

function formatCurrency(value: number): string {
  return new Intl.NumberFormat("ru-RU", {
    style: "currency",
    currency: "RUB",
    maximumFractionDigits: 2,
  }).format(value);
}

function formatDate(value: string): string {
  return new Intl.DateTimeFormat("ru-RU", { day: "2-digit", month: "short", year: "numeric" })
    .format(new Date(`${value}T00:00:00`));
}

function initials(name: string): string {
  return name.split(/\s+/).filter(Boolean).slice(0, 2).map((part) => part[0]).join("").toUpperCase();
}

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : "неизвестная ошибка";
}

function PlusIcon() {
  return <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 5v14M5 12h14" /></svg>;
}

function EditIcon() {
  return <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 20h4l11-11a2.8 2.8 0 00-4-4L4 16v4zM13.5 6.5l4 4" /></svg>;
}

function TrashIcon() {
  return <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 7h16M9 7V4h6v3M7 7l1 13h8l1-13M10 11v5M14 11v5" /></svg>;
}
