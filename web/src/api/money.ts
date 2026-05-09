import type { Amount, Transaction, TransactionType } from "./types";

export function formatMoney(minor: number, currency = "RUB") {
  return new Intl.NumberFormat("ru-RU", { style: "currency", currency }).format(minor / 100);
}

export function parseMoneyToMinor(value: string) {
  const normalized = value.trim().replace(",", ".");
  if (!normalized) {
    return 0;
  }
  return Math.round(Number(normalized) * 100);
}

export function amountFor(amounts: Amount[] = [], currency = "RUB") {
  return amounts.find((amount) => amount.currency === currency)?.amount_minor ?? 0;
}

export function signedAmount(transaction: Transaction) {
  switch (transaction.type) {
    case "expense":
    case "transfer_out":
      return -transaction.amount_minor;
    default:
      return transaction.amount_minor;
  }
}

export function transactionTypeLabel(type: TransactionType) {
  return {
    initial_balance: "Initial",
    income: "Income",
    expense: "Expense",
    transfer_in: "Transfer in",
    transfer_out: "Transfer out",
    interest_income: "Interest",
    adjustment: "Adjustment",
  }[type];
}
