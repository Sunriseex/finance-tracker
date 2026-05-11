// Generated from docs/openapi.yaml. Do not edit by hand.

export type ErrorResponse = {
  "error": {
  "code": string;
  "message": string;
  "details"?: Record<string, unknown>;
};
};

export type AccountType = "cash" | "card" | "savings" | "term_deposit" | "broker" | "other";

export type TransactionType = "initial_balance" | "income" | "expense" | "transfer_in" | "transfer_out" | "interest_income" | "adjustment";

export type AccrualFrequency = "daily" | "monthly" | "end_of_term";

export type CapitalizationFrequency = "none" | "daily" | "monthly" | "end_of_term";

export type DayCountConvention = "actual_365" | "actual_366" | "actual_actual";

export type Amount = {
  "currency": string;
  "amount_minor": number;
};

export type Account = {
  "id": string;
  "legacy_id"?: string | null;
  "name": string;
  "bank"?: string;
  "type": AccountType;
  "currency": string;
  "is_active": boolean;
  "opened_at": string;
  "created_at": string;
  "updated_at": string;
};

export type CreateAccountRequest = {
  "name": string;
  "bank": string;
  "type": AccountType;
  "currency": string;
  "opened_at": string;
};

export type UpdateAccountRequest = {
  "name"?: string;
  "bank"?: string;
  "type"?: AccountType;
  "currency"?: string;
  "opened_at"?: string;
  "is_active"?: boolean;
};

export type AccountBalance = {
  "account_id": string;
  "balance_minor": number;
  "transaction_count": number;
};

export type Category = {
  "id": string;
  "slug": string;
  "name": string;
  "created_at": string;
  "updated_at": string;
};

export type Transaction = {
  "id": string;
  "account_id": string;
  "related_account_id"?: string | null;
  "type": TransactionType;
  "amount_minor": number;
  "category_id"?: string | null;
  "description"?: string;
  "occurred_at": string;
  "created_at": string;
};

export type CreateTransactionRequest = {
  "account_id": string;
  "related_account_id"?: string | null;
  "type": TransactionType;
  "amount_minor": number;
  "category_id"?: string | null;
  "description": string;
  "occurred_at": string;
};

export type CreateTransferRequest = {
  "from_account_id": string;
  "to_account_id": string;
  "amount_minor": number;
  "description": string;
};

export type TransferResponse = {
  "out": Transaction;
  "in": Transaction;
  "exchange_rate": string;
};

export type CurrencyRateTable = {
  "base": string;
  "date": string;
  "provider": string;
  "fetched_at": string;
  "rates": Record<string, number>;
};

export type InterestRule = {
  "id": string;
  "account_id": string;
  "annual_rate_bps": number;
  "promo_rate_bps"?: number | null;
  "promo_end_date"?: string | null;
  "accrual_frequency": AccrualFrequency;
  "capitalization_frequency": CapitalizationFrequency;
  "day_count_convention": DayCountConvention;
  "is_active": boolean;
  "start_date": string;
  "end_date"?: string | null;
};

export type CreateInterestRuleRequest = {
  "annual_rate_bps": number;
  "promo_rate_bps"?: number | null;
  "promo_end_date"?: string | null;
  "accrual_frequency": AccrualFrequency;
  "capitalization_frequency": CapitalizationFrequency;
  "day_count_convention": DayCountConvention;
  "start_date": string;
  "end_date"?: string | null;
};

export type UpdateInterestRuleRequest = {
  "annual_rate_bps"?: number;
  "promo_rate_bps"?: number | null;
  "promo_end_date"?: string | null;
  "accrual_frequency"?: AccrualFrequency;
  "capitalization_frequency"?: CapitalizationFrequency;
  "day_count_convention"?: DayCountConvention;
  "is_active"?: boolean;
  "start_date"?: string;
  "end_date"?: string | null;
};

export type AccrueInterestRequest = {
  "rule_id"?: string;
  "date"?: string;
};

export type RecalculateInterestRequest = {
  "rule_id"?: string;
  "from_date"?: string;
  "to_date"?: string;
};

export type RecalculateInterestResponse = {
  "account_id": string;
  "rule_id": string;
  "from_date": string;
  "to_date": string;
  "deleted_accruals": number;
  "created_accruals": number;
  "skipped_days": number;
  "total_amount_minor": number;
};

export type DashboardAccountBalance = AccountBalance & {
  "name": string;
  "bank"?: string;
  "type": AccountType;
  "currency": string;
  "is_active": boolean;
};

export type DashboardSummary = {
  "generated_at": string;
  "accounts_count": number;
  "active_accounts_count": number;
  "balances": Amount[];
  "monthly_income": Amount[];
  "monthly_expense": Amount[];
  "monthly_interest_income": Amount[];
  "account_balances": DashboardAccountBalance[];
  "recent_transactions": Transaction[];
  "recent_transactions_limit": number;
  "recent_transactions_returned": number;
};

export type DashboardNetWorth = {
  "generated_at": string;
  "balances": Amount[];
  "account_balances": DashboardAccountBalance[];
};

export type DashboardCashflowBucket = {
  "period": string;
  "income": Amount[];
  "expense": Amount[];
  "net_cashflow": Amount[];
  "transaction_count": number;
};

export type DashboardCashflow = {
  "generated_at": string;
  "months": number;
  "buckets": DashboardCashflowBucket[];
};

export type DashboardInterestIncomeBucket = {
  "period": string;
  "interest_income": Amount[];
  "transaction_count": number;
};

export type DashboardInterestIncome = {
  "generated_at": string;
  "months": number;
  "total": Amount[];
  "buckets": DashboardInterestIncomeBucket[];
};

export type AccrueInterestResponse = {
  "skipped": false;
  "transaction": Transaction;
  "accrual": InterestAccrual;
};

export type AccrueInterestSkippedResponse = {
  "skipped": true;
};

export type InterestAccrual = {
  "id": string;
  "account_id": string;
  "rule_id": string;
  "transaction_id": string;
  "accrual_date": string;
  "amount_minor": number;
  "balance_minor": number;
  "annual_rate_bps": number;
  "created_at": string;
};

