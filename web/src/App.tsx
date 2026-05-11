import { useEffect, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  ArrowDownLeft,
  ArrowRightLeft,
  ArrowUpRight,
  Landmark,
  LogIn,
  LogOut,
  Moon,
  Plus,
  Settings,
  ShieldCheck,
  Sun,
  Wallet,
} from "lucide-react";
import { ApiClientError, api, clearStoredSession, getStoredToken } from "./api/client";
import { AccountDetails } from "./features/accounts/AccountDetails";
import { AccountsView } from "./features/accounts/AccountsView";
import { CreateAccountForm } from "./features/accounts/CreateAccountForm";
import { DashboardView } from "./features/dashboard/DashboardView";
import { SettingsView } from "./features/settings/SettingsView";
import { TransactionForm } from "./features/transactions/TransactionForm";
import { TransactionsView } from "./features/transactions/TransactionsView";
import { TransferForm } from "./features/transactions/TransferForm";
import type { QuickAction, Theme, View } from "./shared/constants";
import { themeStorageKey } from "./shared/constants";
import { currencyOptions } from "./shared/currencies";
import { Button, Field, IconButton, Input, Select } from "./shared/ui";

export function App() {
  const queryClient = useQueryClient();

  const [hasSession, setHasSession] = useState(() => Boolean(getStoredToken()));
  const [sessionNonce, setSessionNonce] = useState(0);
  const [view, setView] = useState<View>("dashboard");
  const [selectedAccountId, setSelectedAccountId] = useState("");
  const [quickAction, setQuickAction] = useState<QuickAction>(null);
  const [authOpen, setAuthOpen] = useState(false);
  const [theme, setTheme] = useState<Theme>(() => storedTheme());

  const accounts = useQuery({
    queryKey: ["accounts", sessionNonce],
    queryFn: api.accounts,
    enabled: hasSession,
  });

  const categories = useQuery({
    queryKey: ["categories", sessionNonce],
    queryFn: api.categories,
    enabled: hasSession,
  });

  const profile = useQuery({
    queryKey: ["profile", sessionNonce],
    queryFn: api.profile,
    enabled: hasSession,
    retry: false,
  });

  const selectedAccount = accounts.data?.find((account) => account.id === selectedAccountId);
  const primaryCurrency = profile.data?.user.primary_currency ?? "RUB";
  const sessionInvalid = profile.error instanceof ApiClientError && profile.error.status === 401;

  useEffect(() => {
    document.documentElement.dataset.theme = theme;
    localStorage.setItem(themeStorageKey, theme);
  }, [theme]);

  useEffect(() => {
    if (sessionInvalid) {
      clearStoredSession();
    }
  }, [sessionInvalid]);

  function handleAuthenticated() {
    queryClient.clear();
    setSessionNonce((nonce) => nonce + 1);
    setHasSession(true);
  }

  function handleLogout() {
    clearStoredSession();
    queryClient.clear();
    setSelectedAccountId("");
    setQuickAction(null);
    setAuthOpen(false);
    setSessionNonce((nonce) => nonce + 1);
    setHasSession(false);
  }

  if (!hasSession || sessionInvalid) {
    return <AuthScreen onAuthenticated={handleAuthenticated} theme={theme} setTheme={setTheme} />;
  }

  return (
    <div className="app">
      <aside className="sidebar">
        <div className="brand">
          <Wallet size={22} />
          <span>CapitalFlow</span>
        </div>

        <nav>
          <button className={view === "dashboard" ? "active" : ""} onClick={() => setView("dashboard")}>
            <Landmark size={16} /> Dashboard
          </button>

          <button className={view === "accounts" ? "active" : ""} onClick={() => setView("accounts")}>
            <Wallet size={16} /> Accounts
          </button>

          <button className={view === "transactions" ? "active" : ""} onClick={() => setView("transactions")}>
            <ArrowRightLeft size={16} /> Transactions
          </button>

          <button className={view === "settings" ? "active" : ""} onClick={() => setView("settings")}>
            <Settings size={16} /> Settings
          </button>
        </nav>

        <Button className="muted-button" onClick={() => setAuthOpen((open) => !open)}>
          <Settings size={16} /> Session
        </Button>

        {authOpen ? <SessionPanel onLogout={handleLogout} /> : null}
      </aside>

      <main>
        <header className="topbar">
          <div>
            <p className="eyebrow">v0.5 MVP</p>
            <h1>{selectedAccount ? selectedAccount.name : titleForView(view)}</h1>
          </div>

          <div className="quick-actions">
            <IconButton
              title={theme === "dark" ? "Light theme" : "Dark theme"}
              onClick={() => setTheme((current) => (current === "dark" ? "light" : "dark"))}
            >
              {theme === "dark" ? <Sun size={18} /> : <Moon size={18} />}
            </IconButton>

            <IconButton title="Income" onClick={() => setQuickAction("income")}>
              <ArrowDownLeft size={18} />
            </IconButton>

            <IconButton title="Expense" onClick={() => setQuickAction("expense")}>
              <ArrowUpRight size={18} />
            </IconButton>

            <IconButton title="Transfer" onClick={() => setQuickAction("transfer")}>
              <ArrowRightLeft size={18} />
            </IconButton>

            <IconButton title="Create account" onClick={() => setQuickAction("account")}>
              <Plus size={18} />
            </IconButton>
          </div>
        </header>

        {view === "dashboard" ? (
          <DashboardView
            key={primaryCurrency}
            primaryCurrency={primaryCurrency}
            onOpenAccount={(id) => {
              setSelectedAccountId(id);
              setView("accounts");
            }}
          />
        ) : null}

        {view === "accounts" ? (
          selectedAccount ? (
            <AccountDetails account={selectedAccount} onBack={() => setSelectedAccountId("")} />
          ) : (
            <AccountsView accounts={accounts.data ?? []} onSelect={setSelectedAccountId} />
          )
        ) : null}

        {view === "transactions" ? (
          <TransactionsView accounts={accounts.data ?? []} categories={categories.data ?? []} />
        ) : null}

        {view === "settings" ? <SettingsView profile={profile.data} /> : null}
      </main>

      {quickAction ? (
        <div className="modal-backdrop" onClick={() => setQuickAction(null)}>
          <div className="modal" onClick={(event) => event.stopPropagation()}>
            {quickAction === "account" ? <CreateAccountForm onDone={() => setQuickAction(null)} /> : null}

            {quickAction === "transfer" ? (
              <TransferForm accounts={accounts.data ?? []} onDone={() => setQuickAction(null)} />
            ) : null}

            {quickAction === "income" || quickAction === "expense" ? (
              <TransactionForm
                accounts={accounts.data ?? []}
                categories={categories.data ?? []}
                fixedType={quickAction}
                onDone={() => setQuickAction(null)}
              />
            ) : null}
          </div>
        </div>
      ) : null}
    </div>
  );
}

function AuthScreen({
  onAuthenticated,
  theme,
  setTheme,
}: {
  onAuthenticated: () => void;
  theme: Theme;
  setTheme: (theme: Theme) => void;
}) {
  const status = useQuery({ queryKey: ["auth-status"], queryFn: api.authStatus, retry: false });
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [primaryCurrency, setPrimaryCurrency] = useState("RUB");
  const [error, setError] = useState("");

  const setupRequired = status.data?.setup_required;
  const isSetup = setupRequired === true;

  async function submit() {
    setError("");

    try {
      if (isSetup) {
        await api.setup({ email, password, primary_currency: primaryCurrency });
      } else {
        await api.login({ email, password });
      }

      onAuthenticated();
    } catch (err) {
      setError(errorText(err));
    }
  }

  return (
    <div className="auth-page">
      <section className="auth-info">
        <div className="auth-info-brand">
          <Wallet size={24} />
          <span>CapitalFlow</span>
        </div>

        <div className="auth-preview">
          <div className="auth-preview-card auth-preview-hero">
            <span>Portfolio value</span>
            <strong>₽ 1,284,500</strong>
            <small>+₽ 42,800 this month</small>
          </div>

          <div className="auth-preview-metrics">
            <div className="auth-preview-card">
              <span>Income</span>
              <strong>₽ 180,000</strong>
            </div>

            <div className="auth-preview-card">
              <span>Expenses</span>
              <strong>₽ 92,400</strong>
            </div>
          </div>

          <div className="auth-preview-card">
            <div className="auth-preview-chart">
              <i style={{ height: "42%" }} />
              <i style={{ height: "58%" }} />
              <i style={{ height: "48%" }} />
              <i style={{ height: "74%" }} />
              <i style={{ height: "66%" }} />
              <i style={{ height: "84%" }} />
            </div>
          </div>

          <div className="auth-preview-card auth-preview-list">
            <div className="auth-preview-row">
              <span>Savings</span>
              <strong>62%</strong>
            </div>

            <div className="auth-preview-row">
              <span>Broker</span>
              <strong>24%</strong>
            </div>

            <div className="auth-preview-row">
              <span>Cash</span>
              <strong>14%</strong>
            </div>
          </div>
        </div>

        <div className="auth-info-panel">
          <ShieldCheck size={18} />
          <span>{isSetup ? "First local user setup" : "Private local session"}</span>
        </div>
      </section>

      <form
        className="auth-card"
        onSubmit={(event) => {
          event.preventDefault();
          void submit();
        }}
      >
        <div className="auth-card-header">
          <div>
            <p className="eyebrow">{isSetup ? "Registration" : "Login"}</p>
            <h1>{isSetup ? "Create your first user" : "Welcome back"}</h1>
          </div>

          <IconButton
            title={theme === "dark" ? "Light theme" : "Dark theme"}
            onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
            type="button"
          >
            {theme === "dark" ? <Sun size={18} /> : <Moon size={18} />}
          </IconButton>
        </div>

        {isSetup ? (
          <div className="setup-notice">
            <ShieldCheck size={18} />
            <span>This is the first launch. Create the local owner account and choose the base budget currency.</span>
          </div>
        ) : null}

        <Field label="Email">
          <Input type="email" value={email} onChange={(event) => setEmail(event.target.value)} autoComplete="email" />
        </Field>

        <Field label="Password">
          <Input
            type="password"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            autoComplete={isSetup ? "new-password" : "current-password"}
          />
        </Field>

        {isSetup ? (
          <Field label="Primary currency">
            <Select value={primaryCurrency} onChange={(event) => setPrimaryCurrency(event.target.value)}>
              {currencyOptions().map((currency) => (
                <option key={currency.code} value={currency.code}>
                  {currency.label}
                </option>
              ))}
            </Select>
          </Field>
        ) : null}

        {error ? <div className="error">{error}</div> : null}

        <Button className="primary-button" disabled={status.isLoading}>
          {isSetup ? <ShieldCheck size={16} /> : <LogIn size={16} />}
          {isSetup ? "Create account" : "Login"}
        </Button>
      </form>
    </div>
  );
}

function SessionPanel({ onLogout }: { onLogout: () => void }) {
  return (
    <div className="auth-panel">
      <Button
        className="muted-button"
        type="button"
        onClick={() => {
          void api.logout().finally(() => {
            onLogout();
          });
        }}
      >
        <LogOut size={16} /> Logout
      </Button>
    </div>
  );
}

function errorText(err: unknown) {
  if (err instanceof ApiClientError) {
    return err.message;
  }

  if (err instanceof Error) {
    return err.message;
  }

  return "Request failed";
}

function titleForView(view: View) {
  return {
    dashboard: "Dashboard",
    accounts: "Accounts",
    transactions: "Transactions",
    settings: "Settings",
  }[view];
}

function storedTheme(): Theme {
  const stored = localStorage.getItem(themeStorageKey);

  if (stored === "dark" || stored === "light") {
    return stored;
  }

  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}