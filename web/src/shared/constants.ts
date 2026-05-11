import type { AccountType, TransactionType } from "../api/types";

export type View = "dashboard" | "accounts" | "transactions" | "settings";
export type QuickAction = "income" | "expense" | "transfer" | "account" | null;
export type Theme = "light" | "dark";

export const today = new Date().toISOString().slice(0, 10);
export const accountTypes: AccountType[] = ["cash", "card", "savings", "term_deposit", "broker", "other"];
export const transactionTypes: TransactionType[] = ["income", "expense", "adjustment"];
export const themeStorageKey = "capitalflow_theme";

