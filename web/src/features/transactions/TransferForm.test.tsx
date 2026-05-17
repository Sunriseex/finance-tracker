import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Account } from "../../api/types";
import { TransferForm } from "./TransferForm";

const mocks = vi.hoisted(() => ({
  createTransfer: vi.fn(),
  currencyRates: vi.fn(),
}));

vi.mock("../../api/client", () => ({
  ApiClientError: class ApiClientError extends Error {},
  api: {
    createTransfer: mocks.createTransfer,
    currencyRates: mocks.currencyRates,
  },
}));

const sameCurrencyAccounts: Account[] = [
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
  {
    id: "account-2",
    name: "Savings",
    type: "savings",
    currency: "RUB",
    is_active: true,
    opened_at: "2026-05-17",
    created_at: "2026-05-17T00:00:00Z",
    updated_at: "2026-05-17T00:00:00Z",
  },
];

const crossCurrencyAccounts: Account[] = [
  sameCurrencyAccounts[0],
  {
    ...sameCurrencyAccounts[1],
    currency: "USD",
  },
];

function renderTransferForm(accounts: Account[], onDone = vi.fn()) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  render(
    <QueryClientProvider client={queryClient}>
      <TransferForm accounts={accounts} onDone={onDone} />
    </QueryClientProvider>,
  );
}

describe("TransferForm", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("does not call the API when amount is invalid", async () => {
    const user = userEvent.setup();

    renderTransferForm(sameCurrencyAccounts);

    await user.type(screen.getByLabelText("Amount"), "Infinity");
    await user.click(screen.getByRole("button", { name: "Create" }));

    await screen.findByText("Amount must be a number with up to 2 decimal places");
    await waitFor(() => expect(mocks.createTransfer).not.toHaveBeenCalled());
  });

  it("disables submit while exchange rate is loading", async () => {
    mocks.currencyRates.mockReturnValue(new Promise(() => {}));

    renderTransferForm(crossCurrencyAccounts);

    expect(await screen.findByText("Loading rate")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Create" })).toBeDisabled();
    expect(mocks.currencyRates).toHaveBeenCalledWith("RUB");
  });

  it("shows rate error and does not call createTransfer", async () => {
    const user = userEvent.setup();
    mocks.currencyRates.mockRejectedValue(new Error("Rate provider unavailable"));

    renderTransferForm(crossCurrencyAccounts);

    expect(await screen.findByText("Rate provider unavailable")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Create" })).toBeDisabled();

    await user.type(screen.getByLabelText("Amount"), "10");
    await user.click(screen.getByRole("button", { name: "Create" }));

    expect(mocks.createTransfer).not.toHaveBeenCalled();
  });

  it("submits selected accounts and parsed amount after rate loads", async () => {
    const user = userEvent.setup();
    const onDone = vi.fn();
    mocks.currencyRates.mockResolvedValue({
      base: "RUB",
      date: "2026-05-18",
      provider: "test",
      rates: { USD: 0.01 },
    });
    mocks.createTransfer.mockResolvedValue({});

    renderTransferForm(crossCurrencyAccounts, onDone);

    await user.type(screen.getByLabelText("Amount"), "123.45");
    await screen.findByText("RUB to USD");
    await waitFor(() => expect(screen.getByRole("button", { name: "Create" })).toBeEnabled());
    await user.click(screen.getByRole("button", { name: "Create" }));

    await waitFor(() => {
      expect(mocks.createTransfer).toHaveBeenCalledWith({
        from_account_id: "account-1",
        to_account_id: "account-2",
        amount_minor: 12345,
        description: "",
      });
    });
    expect(onDone).toHaveBeenCalled();
  });
});
