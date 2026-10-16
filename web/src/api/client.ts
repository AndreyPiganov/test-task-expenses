import type {
  Expense,
  ExpenseFilters,
  ExpenseInput,
  ExpensePage,
  FamilyMember,
} from "../types";

interface MembersResponse {
  items: FamilyMember[];
}

interface APIError {
  error?: string;
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...init?.headers,
    },
  });

  if (!response.ok) {
    let message = `Ошибка сервера (${response.status})`;
    try {
      const payload = (await response.json()) as APIError;
      if (payload.error) message = payload.error;
    } catch {
      // The generic message above is enough for a non-JSON response.
    }
    throw new Error(message);
  }

  if (response.status === 204) return undefined as T;
  return (await response.json()) as T;
}

export async function getMembers(): Promise<FamilyMember[]> {
  const response = await request<MembersResponse>("/api/v1/members");
  return response.items;
}

export function getExpenses(filters: ExpenseFilters): Promise<ExpensePage> {
  const query = new URLSearchParams({ limit: "100", offset: "0" });
  if (filters.userId) query.set("user_id", filters.userId);
  if (filters.category) query.set("category", filters.category);
  if (filters.dateFrom) query.set("date_from", filters.dateFrom);
  if (filters.dateTo) query.set("date_to", filters.dateTo);

  return request<ExpensePage>(`/api/v1/expenses?${query.toString()}`);
}

export function createExpense(input: ExpenseInput): Promise<Expense> {
  return request<Expense>("/api/v1/expenses", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function updateExpense(id: string, input: ExpenseInput): Promise<Expense> {
  return request<Expense>(`/api/v1/expenses/${id}`, {
    method: "PATCH",
    body: JSON.stringify(input),
  });
}

export function deleteExpense(id: string): Promise<void> {
  return request<void>(`/api/v1/expenses/${id}`, { method: "DELETE" });
}
