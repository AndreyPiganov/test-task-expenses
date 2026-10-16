import { useEffect, useState, type FormEvent } from "react";
import type {
  Category,
  Expense,
  ExpenseInput,
  FamilyMember,
} from "../types";
import { categories } from "../types";

interface ExpenseModalProps {
  expense: Expense | null;
  members: FamilyMember[];
  saving: boolean;
  onClose: () => void;
  onSubmit: (input: ExpenseInput) => Promise<void>;
}

interface FormState {
  userId: string;
  amount: string;
  category: Category;
  spentAt: string;
  comment: string;
}

function today(): string {
  const date = new Date();
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

function initialState(expense: Expense | null): FormState {
  return {
    userId: expense ? String(expense.user_id) : "",
    amount: expense ? String(expense.amount) : "",
    category: expense?.category ?? "еда",
    spentAt: expense?.spent_at ?? today(),
    comment: expense?.comment ?? "",
  };
}

export default function ExpenseModal({
  expense,
  members,
  saving,
  onClose,
  onSubmit,
}: ExpenseModalProps) {
  const [form, setForm] = useState(() => initialState(expense));
  const [validationError, setValidationError] = useState("");

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape" && !saving) onClose();
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [onClose, saving]);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setValidationError("");

    const amount = Number(form.amount.replace(",", "."));
    if (!form.userId) {
      setValidationError("Выберите члена семьи.");
      return;
    }
    if (!Number.isFinite(amount) || amount <= 0) {
      setValidationError("Введите положительную сумму.");
      return;
    }
    if (!/^\d+(?:[.,]\d{1,2})?$/.test(form.amount.trim())) {
      setValidationError("В сумме допустимо не больше двух знаков после запятой.");
      return;
    }

    try {
      await onSubmit({
        user_id: Number(form.userId),
        amount: form.amount.replace(",", "."),
        category: form.category,
        spent_at: form.spentAt,
        comment: form.comment.trim(),
      });
    } catch (requestError) {
      setValidationError(
        requestError instanceof Error
          ? requestError.message
          : "Не удалось сохранить расход. Попробуйте ещё раз.",
      );
    }
  }

  return (
    <div className="modal-backdrop" onMouseDown={(event) => {
      if (event.target === event.currentTarget && !saving) onClose();
    }}>
      <section className="modal" role="dialog" aria-modal="true" aria-labelledby="expense-modal-title">
        <div className="modal__header">
          <div>
            <p className="eyebrow">{expense ? "Редактирование" : "Новая запись"}</p>
            <h2 id="expense-modal-title">{expense ? "Изменить расход" : "Добавить расход"}</h2>
          </div>
          <button className="icon-button" type="button" onClick={onClose} disabled={saving} aria-label="Закрыть">
            <CloseIcon />
          </button>
        </div>

        <form onSubmit={submit} className="expense-form">
          <label className="field field--wide">
            <span>Член семьи</span>
            <select
              value={form.userId}
              onChange={(event) => setForm({ ...form, userId: event.target.value })}
              autoFocus
              required
            >
              <option value="" disabled>Выберите человека</option>
              {members.map((member) => (
                <option key={member.telegram_user_id} value={member.telegram_user_id}>
                  {member.display_name}
                </option>
              ))}
            </select>
            {!expense && (
              <small className="field__hint">
                Выберите члена семьи, которому принадлежит этот расход.
              </small>
            )}
          </label>

          <label className="field">
            <span>Сумма, ₽</span>
            <input
              inputMode="decimal"
              placeholder="0,00"
              value={form.amount}
              onChange={(event) => setForm({ ...form, amount: event.target.value })}
              required
            />
          </label>

          <label className="field">
            <span>Дата</span>
            <input
              type="date"
              value={form.spentAt}
              onChange={(event) => setForm({ ...form, spentAt: event.target.value })}
              required
            />
          </label>

          <label className="field field--wide">
            <span>Категория</span>
            <select
              value={form.category}
              onChange={(event) => setForm({ ...form, category: event.target.value as Category })}
            >
              {categories.map((category) => (
                <option key={category} value={category}>{category}</option>
              ))}
            </select>
          </label>

          <label className="field field--wide">
            <span>Комментарий</span>
            <textarea
              rows={3}
              maxLength={1000}
              placeholder="Например, продукты на неделю"
              value={form.comment}
              onChange={(event) => setForm({ ...form, comment: event.target.value })}
            />
          </label>

          {validationError && <p className="form-error" role="alert">{validationError}</p>}

          <div className="modal__actions field--wide">
            <button className="button button--secondary" type="button" onClick={onClose} disabled={saving}>
              Отмена
            </button>
            <button className="button button--primary" type="submit" disabled={saving}>
              {saving ? "Сохраняем…" : expense ? "Сохранить" : "Добавить"}
            </button>
          </div>
        </form>
      </section>
    </div>
  );
}

function CloseIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="M6 6l12 12M18 6L6 18" />
    </svg>
  );
}
