import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { DashboardCashflow, DashboardInterestIncome, DashboardSummary } from "../../api/types";
import { DashboardView } from "./DashboardView";

const mocks = vi.hoisted(() => ({
  dashboardSummary: vi.fn(),
  dashboardCashflow: vi.fn(),
  dashboardInterestIncome: vi.fn(),
  currencyRates: vi.fn(),
}));

vi.mock("../../api/client", () => ({
  ApiClientError: class ApiClientError extends Error {},
  api: {
    dashboardSummary: mocks.dashboardSummary,
    dashboardCashflow: mocks.dashboardCashflow,
    dashboardInterestIncome: mocks.dashboardInterestIncome,
    currencyRates: mocks.currencyRates,
  },
}));

const summary: DashboardSummary = {
  generated_at: "2026-05-19T00:00:00Z",
  accounts_count: 1,
  active_accounts_count: 1,
  balances: [{ currency: "RUB", amount_minor: 0 }],
  monthly_income: [],
  monthly_expense: [],
  monthly_interest_income: [],
  account_balances: [
    {
      account_id: "account-1",
      balance_minor: 0,
      transaction_count: 0,
      name: "Card",
      bank: "Bank",
      type: "card",
      currency: "RUB",
      is_active: true,
    },
  ],
  recent_transactions: [],
  recent_transactions_limit: 5,
  recent_transactions_returned: 0,
};

const cashflow: DashboardCashflow = {
  generated_at: "2026-05-19T00:00:00Z",
  months: 6,
  buckets: [],
};

const interest: DashboardInterestIncome = {
  generated_at: "2026-05-19T00:00:00Z",
  months: 6,
  total: [],
  buckets: [],
};

function renderDashboardView({
  onOpenAccount = vi.fn<(id: string) => void>(),
  primaryCurrency = "RUB",
}: {
  onOpenAccount?: (id: string) => void;
  primaryCurrency?: string;
} = {}) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  render(
    <QueryClientProvider client={queryClient}>
      <DashboardView primaryCurrency={primaryCurrency} onOpenAccount={onOpenAccount} />
    </QueryClientProvider>,
  );

  return { onOpenAccount };
}

describe("DashboardView", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.dashboardSummary.mockResolvedValue(summary);
    mocks.dashboardCashflow.mockResolvedValue(cashflow);
    mocks.dashboardInterestIncome.mockResolvedValue(interest);
    mocks.currencyRates.mockResolvedValue({
      base: "RUB",
      date: "2026-05-19",
      provider: "test",
      rates: {},
    });
  });

  it("shows loading and summary API errors", async () => {
    mocks.dashboardSummary.mockReturnValueOnce(new Promise(() => {}));
    renderDashboardView();

    expect(screen.getByText("Loading dashboard")).toBeInTheDocument();

    mocks.dashboardSummary.mockRejectedValueOnce(new Error("Dashboard unavailable"));
    renderDashboardView();

    expect(await screen.findByText("Dashboard unavailable")).toBeInTheDocument();
  });

  it("renders empty balance and transaction states", async () => {
    mocks.dashboardSummary.mockResolvedValueOnce({
      ...summary,
      accounts_count: 0,
      active_accounts_count: 0,
      balances: [],
      account_balances: [],
      recent_transactions: [],
      recent_transactions_returned: 0,
    } satisfies DashboardSummary);

    renderDashboardView();

    expect(await screen.findByText("0 active accounts across 1 currency")).toBeInTheDocument();
    expect(screen.getByText("No positive balances")).toBeInTheDocument();
    expect(screen.getByText("No transactions")).toBeInTheDocument();
  });

  it("switches dashboard currency and reloads conversion rates", async () => {
    const user = userEvent.setup();
    mocks.dashboardSummary.mockResolvedValueOnce({
      ...summary,
      balances: [
        { currency: "RUB", amount_minor: 100_000 },
        { currency: "USD", amount_minor: 100_00 },
      ],
    } satisfies DashboardSummary);

    renderDashboardView();

    await screen.findByRole("button", { name: "USD" });
    expect(mocks.currencyRates).toHaveBeenCalledWith("RUB");

    await user.click(screen.getByRole("button", { name: "USD" }));

    expect(await screen.findByText("Cashflow (USD)")).toBeInTheDocument();
    await waitFor(() => expect(mocks.currencyRates).toHaveBeenCalledWith("USD"));
  });

  it("shows currency rate errors", async () => {
    mocks.currencyRates.mockRejectedValueOnce(new Error("Rate provider unavailable"));

    renderDashboardView();

    expect(await screen.findByText("Rate provider unavailable")).toBeInTheDocument();
  });

  it("opens account details from a keyboard-accessible account balance action", async () => {
    const user = userEvent.setup();
    const onOpenAccount = vi.fn<(id: string) => void>();
    renderDashboardView({ onOpenAccount });

    const action = await screen.findByRole("button", { name: "Open Card account" });
    action.focus();

    await user.keyboard("{Enter}");
    expect(onOpenAccount).toHaveBeenCalledWith("account-1");

    onOpenAccount.mockClear();
    await user.click(action);
    expect(onOpenAccount).toHaveBeenCalledWith("account-1");
  });
});
