import type { Expense } from "../types";

interface ConfirmDialogProps {
  expense: Expense;
  deleting: boolean;
  onCancel: () => void;
  onConfirm: () => Promise<void>;
}

export default function ConfirmDialog({
  expense,
  deleting,
  onCancel,
  onConfirm,
}: ConfirmDialogProps) {
  return (
    <div className="modal-backdrop" onMouseDown={(event) => {
      if (event.target === event.currentTarget && !deleting) onCancel();
    }}>
      <section className="confirm-dialog" role="alertdialog" aria-modal="true" aria-labelledby="confirm-title">
        <div className="confirm-dialog__icon" aria-hidden="true">!</div>
        <h2 id="confirm-title">Удалить расход?</h2>
        <p>
          Запись «{expense.comment || expense.category}» на сумму {formatAmount(expense.amount)} ₽ будет удалена безвозвратно.
        </p>
        <div className="modal__actions">
          <button className="button button--secondary" type="button" onClick={onCancel} disabled={deleting}>
            Отмена
          </button>
          <button className="button button--danger" type="button" onClick={() => void onConfirm()} disabled={deleting}>
            {deleting ? "Удаляем…" : "Удалить"}
          </button>
        </div>
      </section>
    </div>
  );
}

function formatAmount(amount: string | number): string {
  return new Intl.NumberFormat("ru-RU", { maximumFractionDigits: 2 }).format(Number(amount));
}
