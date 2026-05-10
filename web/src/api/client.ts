import type {
  Account,
  AccountBalance,
  AccountType,
  Category,
  DashboardCashflow,
  DashboardInterestIncome,
  DashboardSummary,
  CurrencyRateTable,
  InterestRule,
  Transaction,
  TransactionType,
} from "./types";

const tokenKey = "capitalflow_api_token";
const apiBaseKey = "capitalflow_api_base";
const legacyTokenKey = "finance_tracker_api_token";
const legacyApiBaseKey = "finance_tracker_api_base";
const legacyDefaultApiBase = "http://localhost:8080/api";
const defaultApiBase = "/api";

export function getStoredToken() {
  return localStorage.getItem(tokenKey) ?? localStorage.getItem(legacyTokenKey) ?? "";
}

export function setStoredToken(token: string) {
  localStorage.setItem(tokenKey, token.trim());
}

export function getStoredApiBase() {
  const stored = (localStorage.getItem(apiBaseKey) ?? localStorage.getItem(legacyApiBaseKey))?.replace(/\/$/, "");
  if (!stored || stored === legacyDefaultApiBase) {
    return defaultApiBase;
  }
  return stored;
}

export function setStoredApiBase(base: string) {
  const normalized = base.trim().replace(/\/$/, "");
  localStorage.setItem(apiBaseKey, normalized || defaultApiBase);
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

  let response: Response;
  try {
    response = await fetch(`${getStoredApiBase()}${path}`, { ...init, headers });
  } catch (err) {
    throw new ApiClientError(err instanceof Error ? err.message : "API request failed", 0, "network_error");
  }
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
  currencyRates: (base = "RUB") => apiFetch<CurrencyRateTable>(`/currency-rates?base=${encodeURIComponent(base)}`),
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
    apiFetch<{ out: Transaction; in: Transaction; exchange_rate: string }>("/transfers", { method: "POST", body: JSON.stringify(input) }),
  createInterestRule: (
    accountId: string,
    input: {
      annual_rate_bps: number;
      promo_rate_bps?: number | null;
      promo_end_date?: string | null;
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
