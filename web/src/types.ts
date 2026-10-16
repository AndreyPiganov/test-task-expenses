export const categories = [
  "еда",
  "транспорт",
  "жильё",
  "развлечения",
  "здоровье",
  "покупки",
  "другое",
] as const;

export type Category = (typeof categories)[number];

export interface FamilyMember {
  telegram_user_id: number;
  username?: string;
  display_name: string;
  created_at: string;
  updated_at: string;
}

export interface Expense {
  id: string;
  user_id: number;
  amount: string | number;
  category: Category;
  spent_at: string;
  comment?: string;
  created_at: string;
  updated_at: string;
}

export interface ExpensePage {
  items: Expense[];
  total: number;
  limit: number;
  offset: number;
}

export interface ExpenseFilters {
  userId: string;
  category: string;
  dateFrom: string;
  dateTo: string;
}

export interface ExpenseInput {
  user_id: number;
  amount: string;
  category: Category;
  spent_at: string;
  comment: string;
}
