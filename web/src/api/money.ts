import type { Amount, CurrencyRateTable, Transaction, TransactionType } from "./types";

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

export function amountFor(amounts: Amount[] | null | undefined, currency = "RUB") {
  return amounts?.find((amount) => amount.currency === currency)?.amount_minor ?? 0;
}

export function convertMinor(amountMinor: number, from: string, to: string, table?: CurrencyRateTable) {
  if (from === to) {
    return amountMinor;
  }
  if (!table || table.base !== to) {
    return 0;
  }
  const rate = table.rates[from];
  if (!rate || rate <= 0) {
    return 0;
  }
  return Math.round(amountMinor / rate);
}

export function sumConverted(amounts: Amount[] | null | undefined, to: string, table?: CurrencyRateTable) {
  return (amounts ?? []).reduce((total, amount) => total + convertMinor(amount.amount_minor, amount.currency, to, table), 0);
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
