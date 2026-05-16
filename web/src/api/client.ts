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
  Profile,
  Transaction,
  TransactionType,
} from "./types";

const tokenKey = "capitalflow_api_token";
const apiBaseKey = "capitalflow_api_base";
const legacyTokenKey = "finance_tracker_api_token";
const legacyApiBaseKey = "finance_tracker_api_base";
const legacyRefreshTokenKey = "capitalflow_refresh_token";
const legacyDefaultApiBase = "http://localhost:8080/api";
const defaultApiBase = "/api/v1";

export function getStoredToken() {
  return localStorage.getItem(tokenKey) ?? localStorage.getItem(legacyTokenKey) ?? "";
}

export function setStoredToken(token: string) {
  localStorage.setItem(tokenKey, token.trim());
}

export function clearStoredSession() {
  localStorage.removeItem(tokenKey);
  localStorage.removeItem(legacyRefreshTokenKey);
  localStorage.removeItem(legacyTokenKey);
}

export function getStoredApiBase() {
  const stored = localStorage.getItem(apiBaseKey) ?? localStorage.getItem(legacyApiBaseKey);
  return normalizeApiBase(stored ?? "");
}

export function setStoredApiBase(base: string) {
  localStorage.setItem(apiBaseKey, normalizeApiBase(base));
}

function normalizeApiBase(base: string) {
  const normalized = base.trim().replace(/\/$/, "");

  if (!normalized || normalized === legacyDefaultApiBase) {
    return defaultApiBase;
  }

  if (normalized === "/api") {
    return defaultApiBase;
  }

  if (normalized.endsWith("/api")) {
    return `${normalized.slice(0, -4)}/api/v1`;
  }

  return normalized;
}

function getAuthBase() {
  const apiBase = getStoredApiBase();

  if (apiBase.endsWith("/api/v1")) {
    return apiBase.slice(0, -7);
  }

  if (apiBase.endsWith("/api")) {
    return apiBase.slice(0, -4);
  }

  return apiBase;
}

export class ApiClientError extends Error {
  status: number;
  code?: string;

  constructor(message: string, status: number, code?: string) {
    super(message);
    this.status = status;
    this.code = code;
  }
}

async function apiFetch<T>(path: string, init: RequestInit = {}): Promise<T> {
  return apiFetchWithAuth<T>(path, init, true);
}

async function apiFetchWithAuth<T>(path: string, init: RequestInit = {}, allowRefresh: boolean): Promise<T> {
  const headers = new Headers(init.headers);
  const token = getStoredToken();

  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }

  if (init.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  if (isMutation(init.method) && !headers.has("Idempotency-Key")) {
    headers.set("Idempotency-Key", newIdempotencyKey());
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

  if (response.status === 401 && allowRefresh) {
    await refreshSession();
    return apiFetchWithAuth<T>(path, init, false);
  }

  if (!response.ok) {
    const err = payload?.error;
    throw new ApiClientError(err?.message ?? response.statusText, response.status, err?.code);
  }

  return payload as T;
}

function isMutation(method?: string) {
  const normalized = (method ?? "GET").toUpperCase();
  return normalized === "POST" || normalized === "PATCH" || normalized === "DELETE";
}

function newIdempotencyKey() {
  return globalThis.crypto?.randomUUID?.() ?? `${Date.now()}-${Math.random()}`;
}

async function authFetch<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);

  if (init.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  let response: Response;
  try {
    response = await fetch(`${getAuthBase()}${path}`, {
      ...init,
      headers,
      credentials: "include",
    });
  } catch (err) {
    throw new ApiClientError(err instanceof Error ? err.message : "API request failed", 0, "network_error");
  }

  if (response.status === 204) {
    return undefined as T;
  }

  const contentType = response.headers.get("Content-Type") ?? "";
  const payload = contentType.includes("application/json") ? await response.json().catch(() => null) : null;

  if (!response.ok) {
    const err = payload?.error;
    throw new ApiClientError(err?.message ?? response.statusText, response.status, err?.code);
  }

  if (payload == null) {
    throw new ApiClientError("Invalid API response", response.status, "invalid_response");
  }

  return payload as T;
}

let refreshSessionPromise: Promise<AuthResponse> | null = null;

type AuthResponse = {
  user: { id: string; email: string; primary_currency: string };
  access_token: string;
  access_expires_at: string;
};

function storeSession(session: AuthResponse) {
  setStoredToken(session.access_token);
  localStorage.removeItem(legacyRefreshTokenKey);
  return session;
}

async function refreshSession() {
  if (refreshSessionPromise) {
    return refreshSessionPromise;
  }

  refreshSessionPromise = authFetch<AuthResponse>("/auth/refresh", {
    method: "POST",
  })
    .then(storeSession)
    .catch((err) => {
      clearStoredSession();

      if (err instanceof ApiClientError) {
        throw new ApiClientError("Login required", 401, "unauthorized");
      }

      throw err;
    })
    .finally(() => {
      refreshSessionPromise = null;
    });

  return refreshSessionPromise;
}

export const api = {
  authStatus: () => authFetch<{ setup_required: boolean }>("/auth/status"),

  setup: async (input: { email: string; password: string; primary_currency: string }) =>
    storeSession(await authFetch<AuthResponse>("/auth/setup", { method: "POST", body: JSON.stringify(input) })),

  login: async (input: { email: string; password: string }) =>
    storeSession(await authFetch<AuthResponse>("/auth/login", { method: "POST", body: JSON.stringify(input) })),

  logout: async () => {
    await authFetch<void>("/auth/logout", {
      method: "POST",
    });
    clearStoredSession();
  },

  profile: () => apiFetch<Profile>("/settings/profile"),

  updateProfile: (input: { primary_currency: string }) =>
    apiFetch<Profile>("/settings/profile", { method: "PATCH", body: JSON.stringify(input) }),

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

  archiveAccount: (id: string) => apiFetch<void>(`/accounts/${id}/archive`, { method: "POST" }),

  createTransaction: (input: {
    account_id: string;
    type: TransactionType;
    amount_minor: number;
    category_id?: string | null;
    description: string;
    occurred_at: string;
  }) => apiFetch<Transaction>("/transactions", { method: "POST", body: JSON.stringify(input) }),

  deleteTransaction: (id: string) => apiFetch<void>(`/transactions/${id}`, { method: "DELETE" }),

  createTransfer: (input: { from_account_id: string; to_account_id: string; amount_minor: number; description: string }) =>
    apiFetch<{ out: Transaction; in: Transaction; exchange_rate: string }>("/transfers", {
      method: "POST",
      body: JSON.stringify(input),
    }),

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
