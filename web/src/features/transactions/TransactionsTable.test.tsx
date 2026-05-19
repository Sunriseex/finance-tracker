import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Account, Category, Transaction } from "../../api/types";
import { TransactionsTable } from "./TransactionsTable";

const mocks = vi.hoisted(() => ({
  deleteTransaction: vi.fn(),
}));

vi.mock("../../api/client", () => ({
  ApiClientError: class ApiClientError extends Error {},
  api: {
    deleteTransaction: mocks.deleteTransaction,
  },
}));

const accounts: Account[] = [
  {
    id: "account-1",
    name: "Card",
    type: "card",
    currency: "RUB",
    is_active: true,
    opened_at: "2026-05-17",
    created_at: "2026-05-17T00:00:00Z",
    updated_at: "2026-05-17T00:00:00Z",
  },
];

const categories: Category[] = [
  {
    id: "category-1",
    slug: "salary",
    name: "Salary",
    created_at: "2026-05-17T00:00:00Z",
    updated_at: "2026-05-17T00:00:00Z",
  },
];

const incomeTransaction: Transaction = {
  id: "transaction-1",
  account_id: "account-1",
  type: "income",
  amount_minor: 100_00,
  category_id: "category-1",
  description: "Salary",
  occurred_at: "2026-05-17T00:00:00Z",
  created_at: "2026-05-17T00:00:00Z",
};

function renderTransactionsTable(transactions: Transaction[] = [incomeTransaction]) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
  const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");

  render(
    <QueryClientProvider client={queryClient}>
      <TransactionsTable transactions={transactions} accounts={accounts} categories={categories} allowDelete />
    </QueryClientProvider>,
  );

  return { invalidateQueries };
}

describe("TransactionsTable", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    mocks.deleteTransaction.mockReset();
    mocks.deleteTransaction.mockResolvedValue(undefined);
  });

  it("does not delete when confirmation is cancelled", async () => {
    const user = userEvent.setup();
    vi.spyOn(window, "confirm").mockReturnValue(false);

    renderTransactionsTable();

    await user.click(screen.getByRole("button", { name: "Delete transaction" }));

    expect(window.confirm).toHaveBeenCalledWith("Delete this transaction?");
    expect(mocks.deleteTransaction).not.toHaveBeenCalled();
  });

  it("deletes confirmed transactions and invalidates money queries", async () => {
    const user = userEvent.setup();
    vi.spyOn(window, "confirm").mockReturnValue(true);
    const { invalidateQueries } = renderTransactionsTable();

    await user.click(screen.getByRole("button", { name: "Delete transaction" }));

    await waitFor(() => expect(mocks.deleteTransaction).toHaveBeenCalledWith("transaction-1"));
    await waitFor(() => expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: ["transactions"] }));
  });

  it("shows API errors after confirmed delete fails", async () => {
    const user = userEvent.setup();
    vi.spyOn(window, "confirm").mockReturnValue(true);
    mocks.deleteTransaction.mockRejectedValue(new Error("Delete failed"));

    renderTransactionsTable();

    await user.click(screen.getByRole("button", { name: "Delete transaction" }));

    expect(await screen.findByText("Delete failed")).toBeInTheDocument();
  });

  it("disables delete for transfer transactions", () => {
    renderTransactionsTable([
      {
        ...incomeTransaction,
        id: "transfer-transaction",
        type: "transfer_out",
        description: "Transfer",
      },
    ]);

    expect(screen.getByRole("button", { name: "Delete transaction" })).toBeDisabled();
    expect(mocks.deleteTransaction).not.toHaveBeenCalled();
  });
});
