import type {
  Account,
  AccountBalance,
  AccountType,
  Category,
  DashboardCashflow,
  DashboardInterestIncome,
  DashboardSummary,
  InterestRule,
  Transaction,
  TransactionType,
} from "./types";

const tokenKey = "finance_tracker_api_token";
const apiBaseKey = "finance_tracker_api_base";
const defaultApiBase = "http://localhost:8080/api";

export function getStoredToken() {
  return localStorage.getItem(tokenKey) ?? "";
}

export function setStoredToken(token: string) {
  localStorage.setItem(tokenKey, token.trim());
}

export function getStoredApiBase() {
  return localStorage.getItem(apiBaseKey) ?? defaultApiBase;
}

export function setStoredApiBase(base: string) {
  localStorage.setItem(apiBaseKey, base.trim().replace(/\/$/, ""));
}

export class ApiClientError extends Error {
  status: number;
  code?: string;

  constructor(
    message: string,
    status: number,
    code?: string,
  ) {
    super(message);
    this.status = status;
    this.code = code;
  }
}

async function apiFetch<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  const token = getStoredToken();
  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }
  if (init.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  const response = await fetch(`${getStoredApiBase()}${path}`, { ...init, headers });
  if (response.status === 204) {
    return undefined as T;
  }

  const payload = await response.json().catch(() => null);
  if (!response.ok) {
    const err = payload?.error;
    throw new ApiClientError(err?.message ?? response.statusText, response.status, err?.code);
  }

  return payload as T;
}

export const api = {
  dashboardSummary: () => apiFetch<DashboardSummary>("/dashboard/summary"),
  dashboardCashflow: () => apiFetch<DashboardCashflow>("/dashboard/cashflow?months=6"),
  dashboardInterestIncome: () => apiFetch<DashboardInterestIncome>("/dashboard/interest-income?months=6"),
  accounts: () => apiFetch<Account[]>("/accounts"),
  account: (id: string) => apiFetch<Account>(`/accounts/${id}`),
  accountBalance: (id: string) => apiFetch<AccountBalance>(`/accounts/${id}/balance`),
  transactions: (accountId?: string) =>
    apiFetch<Transaction[]>(accountId ? `/transactions?account_id=${accountId}` : "/transactions"),
  categories: () => apiFetch<Category[]>("/categories"),
  interestRules: (accountId: string) => apiFetch<InterestRule[]>(`/accounts/${accountId}/interest-rules`),
  createAccount: (input: { name: string; bank: string; type: AccountType; currency: string; opened_at: string }) =>
    apiFetch<Account>("/accounts", { method: "POST", body: JSON.stringify(input) }),
  updateAccount: (
    id: string,
    input: { name: string; bank: string; type: AccountType; currency: string; opened_at: string; is_active: boolean },
  ) => apiFetch<Account>(`/accounts/${id}`, { method: "PATCH", body: JSON.stringify(input) }),
  archiveAccount: (id: string) =>
    apiFetch<void>(`/accounts/${id}/archive`, { method: "POST" }),
  createTransaction: (input: {
    account_id: string;
    type: TransactionType;
    amount_minor: number;
    category_id?: string | null;
    description: string;
    occurred_at: string;
  }) => apiFetch<Transaction>("/transactions", { method: "POST", body: JSON.stringify(input) }),
  deleteTransaction: (id: string) =>
    apiFetch<void>(`/transactions/${id}`, { method: "DELETE" }),
  createTransfer: (input: { from_account_id: string; to_account_id: string; amount_minor: number; description: string }) =>
    apiFetch<{ out: Transaction; in: Transaction }>("/transfers", { method: "POST", body: JSON.stringify(input) }),
  createInterestRule: (
    accountId: string,
    input: {
      annual_rate_bps: number;
      accrual_frequency: "daily";
      capitalization_frequency: "none" | "daily" | "monthly" | "end_of_term";
      day_count_convention: "actual_365";
      start_date: string;
      end_date?: string | null;
    },
  ) => apiFetch<InterestRule>(`/accounts/${accountId}/interest-rules`, { method: "POST", body: JSON.stringify(input) }),
  accrueInterest: (accountId: string, date: string) =>
    apiFetch<unknown>(`/accounts/${accountId}/accrue-interest`, {
      method: "POST",
      body: JSON.stringify({ date }),
    }),
};
