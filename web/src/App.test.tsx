import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { App } from "./App";

const mocks = vi.hoisted(() => ({
  accounts: vi.fn(),
  categories: vi.fn(),
  profile: vi.fn(),
  dashboardSummary: vi.fn(),
  interestRules: vi.fn(),
  transactions: vi.fn(),
}));

vi.mock("./api/client", () => ({
  ApiClientError: class ApiClientError extends Error {
    status: number;

    constructor(message: string, status: number) {
      super(message);
      this.status = status;
    }
  },
  api: {
    accounts: mocks.accounts,
    categories: mocks.categories,
    profile: mocks.profile,
    dashboardSummary: mocks.dashboardSummary,
    interestRules: mocks.interestRules,
    transactions: mocks.transactions,
  },
  clearStoredSession: vi.fn(),
  getStoredToken: () => "token",
}));

vi.mock("./features/dashboard/DashboardView", () => ({
  DashboardView: () => <div>Dashboard mock</div>,
}));

function renderApp() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  render(
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>,
  );
}

describe("App query states", () => {
  beforeEach(() => {
    localStorage.setItem("capitalflow_theme", "light");
    vi.clearAllMocks();
    mocks.accounts.mockResolvedValue([]);
    mocks.categories.mockResolvedValue([]);
    mocks.profile.mockResolvedValue({
      user: { id: "user-1", email: "user@example.com", primary_currency: "RUB" },
    });
    mocks.dashboardSummary.mockResolvedValue({
      account_balances: [],
    });
    mocks.interestRules.mockResolvedValue([]);
    mocks.transactions.mockResolvedValue([]);
  });

  it("shows account loading state and disables account-dependent quick actions", async () => {
    const user = userEvent.setup();
    mocks.accounts.mockReturnValue(new Promise(() => {}));

    renderApp();

    expect(screen.getByTitle("Income")).toBeDisabled();
    expect(screen.getByTitle("Expense")).toBeDisabled();
    expect(screen.getByTitle("Transfer")).toBeDisabled();

    await user.click(screen.getByRole("button", { name: /Accounts/ }));

    expect(await screen.findByText("Loading accounts")).toBeInTheDocument();
  });

  it("shows transaction dependency errors instead of empty filters", async () => {
    const user = userEvent.setup();
    mocks.categories.mockRejectedValue(new Error("Categories unavailable"));

    renderApp();

    await user.click(screen.getByRole("button", { name: /Transactions/ }));

    expect(await screen.findByText("Categories unavailable")).toBeInTheDocument();
    expect(screen.getAllByRole("combobox")[1]).toBeDisabled();
  });

  it("shows profile loading state on settings", async () => {
    const user = userEvent.setup();
    mocks.profile.mockReturnValue(new Promise(() => {}));

    renderApp();

    await user.click(screen.getByRole("button", { name: /Settings/ }));

    expect(await screen.findByText("Loading profile")).toBeInTheDocument();
  });
});
