import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Account, DashboardSummary, InterestRule } from "../../api/types";
import { AccountsView } from "./AccountsView";

const mocks = vi.hoisted(() => ({
  dashboardSummary: vi.fn(),
  interestRules: vi.fn(),
}));

vi.mock("../../api/client", () => ({
  ApiClientError: class ApiClientError extends Error {},
  api: {
    dashboardSummary: mocks.dashboardSummary,
    interestRules: mocks.interestRules,
  },
}));

const accounts: Account[] = [
  account("account-1", "Card"),
  account("account-2", "Savings"),
  account("account-3", "Term"),
];

const summary: DashboardSummary = {
  generated_at: "2026-05-19T00:00:00Z",
  accounts_count: 3,
  active_accounts_count: 3,
  balances: [],
  monthly_income: [],
  monthly_expense: [],
  monthly_interest_income: [],
  account_balances: [
    { account_id: "account-1", balance_minor: 10_000, transaction_count: 1, name: "Card", type: "card", currency: "RUB", is_active: true },
    { account_id: "account-2", balance_minor: 20_000, transaction_count: 1, name: "Savings", type: "savings", currency: "RUB", is_active: true },
    { account_id: "account-3", balance_minor: 30_000, transaction_count: 1, name: "Term", type: "term_deposit", currency: "RUB", is_active: true },
  ],
  recent_transactions: [],
  recent_transactions_limit: 5,
  recent_transactions_returned: 0,
};

const rules: InterestRule[] = [
  interestRule("rule-old", "account-1", 1_000, "2026-01-01"),
  interestRule("rule-new", "account-1", 1_500, "2026-02-01"),
  { ...interestRule("rule-inactive", "account-2", 2_000, "2026-02-01"), is_active: false },
];

function renderAccountsView(inputAccounts = accounts) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  render(
    <QueryClientProvider client={queryClient}>
      <AccountsView accounts={inputAccounts} onSelect={vi.fn()} />
    </QueryClientProvider>,
  );
}

describe("AccountsView", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.dashboardSummary.mockResolvedValue(summary);
    mocks.interestRules.mockResolvedValue(rules);
  });

  it("loads account interest rules once and renders active rates", async () => {
    renderAccountsView();

    expect(await screen.findByText("15.00%")).toBeInTheDocument();
    expect(screen.getAllByText("-")).toHaveLength(2);
    expect(mocks.interestRules).toHaveBeenCalledTimes(1);
    expect(mocks.interestRules).toHaveBeenCalledWith();
  });

  it("shows bounded loading and error states for rates", async () => {
    mocks.interestRules.mockReturnValueOnce(new Promise(() => {}));
    renderAccountsView();

    expect(await screen.findAllByText("Loading")).toHaveLength(3);
    expect(mocks.interestRules).toHaveBeenCalledTimes(1);

    mocks.interestRules.mockRejectedValueOnce(new Error("Rates unavailable"));
    renderAccountsView([accounts[0]]);

    expect(await screen.findByText("Rates unavailable")).toBeInTheDocument();
  });
});

function account(id: string, name: string): Account {
  return {
    id,
    name,
    bank: "Bank",
    type: "card",
    currency: "RUB",
    is_active: true,
    opened_at: "2026-05-19",
    created_at: "2026-05-19T00:00:00Z",
    updated_at: "2026-05-19T00:00:00Z",
  };
}

function interestRule(id: string, accountID: string, annualRateBps: number, startDate: string): InterestRule {
  return {
    id,
    account_id: accountID,
    annual_rate_bps: annualRateBps,
    accrual_frequency: "daily",
    capitalization_frequency: "none",
    day_count_convention: "actual_365",
    is_active: true,
    start_date: startDate,
  };
}
