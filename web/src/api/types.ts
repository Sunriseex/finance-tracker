export type AccountType = "cash" | "card" | "savings" | "term_deposit" | "broker" | "other";
export type TransactionType =
  | "initial_balance"
  | "income"
  | "expense"
  | "transfer_in"
  | "transfer_out"
  | "interest_income"
  | "adjustment";

export type Account = {
  id: string;
  legacy_id?: string;
  name: string;
  bank?: string;
  type: AccountType;
  currency: string;
  is_active: boolean;
  opened_at: string;
  created_at: string;
  updated_at: string;
};

export type Category = {
  id: string;
  slug: string;
  name: string;
  created_at: string;
  updated_at: string;
};

export type Transaction = {
  id: string;
  account_id: string;
  related_account_id?: string;
  type: TransactionType;
  amount_minor: number;
  category_id?: string;
  description?: string;
  occurred_at: string;
  created_at: string;
};

export type InterestRule = {
  id: string;
  account_id: string;
  annual_rate_bps: number;
  promo_rate_bps?: number;
  promo_end_date?: string;
  accrual_frequency: "daily" | "monthly" | "end_of_term";
  capitalization_frequency: "none" | "daily" | "monthly" | "end_of_term";
  day_count_convention: "actual_365" | "actual_366" | "actual_actual";
  is_active: boolean;
  start_date: string;
  end_date?: string;
};

export type Amount = {
  currency: string;
  amount_minor: number;
};

export type CurrencyRateTable = {
  base: string;
  date: string;
  provider: string;
  fetched_at: string;
  rates: Record<string, number>;
};

export type DashboardAccountBalance = {
  account_id: string;
  name: string;
  bank?: string;
  type: AccountType;
  currency: string;
  is_active: boolean;
  balance_minor: number;
  transaction_count: number;
};

export type DashboardSummary = {
  generated_at: string;
  accounts_count: number;
  active_accounts_count: number;
  balances: Amount[];
  monthly_income: Amount[];
  monthly_expense: Amount[];
  monthly_interest_income: Amount[];
  account_balances: DashboardAccountBalance[];
  recent_transactions: Transaction[];
  recent_transactions_limit: number;
  recent_transactions_returned: number;
};

export type DashboardCashflow = {
  buckets: {
    period: string;
    income: Amount[];
    expense: Amount[];
    net_cashflow: Amount[];
    transaction_count: number;
  }[];
};

export type DashboardInterestIncome = {
  total: Amount[];
  buckets: {
    period: string;
    interest_income: Amount[];
    transaction_count: number;
  }[];
};

export type AccountBalance = {
  account_id: string;
  balance_minor: number;
  transaction_count: number;
};

export type ApiError = {
  error?: {
    code?: string;
    message?: string;
    details?: unknown;
  };
};
