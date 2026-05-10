import { useEffect, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { ArrowDownLeft, ArrowRightLeft, ArrowUpRight, Landmark, LogOut, Moon, Plus, Settings, Sun, Wallet } from "lucide-react";
import { ApiClientError, api, getStoredApiBase, getStoredToken, setStoredApiBase } from "./api/client";
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
  const [hasSession, setHasSession] = useState(() => Boolean(getStoredToken()));
  const [view, setView] = useState<View>("dashboard");
  const [selectedAccountId, setSelectedAccountId] = useState("");
  const [quickAction, setQuickAction] = useState<QuickAction>(null);
  const [authOpen, setAuthOpen] = useState(false);
  const [theme, setTheme] = useState<Theme>(() => storedTheme());
  const accounts = useQuery({ queryKey: ["accounts"], queryFn: api.accounts, enabled: hasSession });
  const categories = useQuery({ queryKey: ["categories"], queryFn: api.categories, enabled: hasSession });
  const profile = useQuery({ queryKey: ["profile"], queryFn: api.profile, enabled: hasSession });

  const selectedAccount = accounts.data?.find((account) => account.id === selectedAccountId);
  const primaryCurrency = profile.data?.user.primary_currency ?? "RUB";

  useEffect(() => {
    document.documentElement.dataset.theme = theme;
    localStorage.setItem(themeStorageKey, theme);
  }, [theme]);

  if (!hasSession) {
    return <AuthScreen onAuthenticated={() => setHasSession(true)} theme={theme} setTheme={setTheme} />;
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
        {authOpen ? <SessionPanel onLogout={() => setHasSession(false)} /> : null}
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
              onClick={() => setTheme((current) => current === "dark" ? "light" : "dark")}
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

        {view === "dashboard" ? <DashboardView key={primaryCurrency} primaryCurrency={primaryCurrency} onOpenAccount={(id) => { setSelectedAccountId(id); setView("accounts"); }} /> : null}
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
  const [mode, setMode] = useState<"setup" | "login">("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [primaryCurrency, setPrimaryCurrency] = useState("RUB");
  const [apiBase, setApiBase] = useState(getStoredApiBase());
  const [error, setError] = useState("");
  const setupRequired = status.data?.setup_required;
  const activeMode = setupRequired ? "setup" : mode;

  async function submit() {
    setError("");
    setStoredApiBase(apiBase);
    try {
      if (activeMode === "setup") {
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
      <form className="auth-card" onSubmit={(event) => { event.preventDefault(); void submit(); }}>
        <div className="auth-card-header">
          <div className="brand auth-brand">
            <Wallet size={22} />
            <span>CapitalFlow</span>
          </div>
          <IconButton
            title={theme === "dark" ? "Light theme" : "Dark theme"}
            onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
            type="button"
          >
            {theme === "dark" ? <Sun size={18} /> : <Moon size={18} />}
          </IconButton>
        </div>
        {setupRequired === false ? (
          <div className="segmented">
            <button type="button" className={mode === "login" ? "active" : ""} onClick={() => setMode("login")}>
              Login
            </button>
            <button type="button" className={mode === "setup" ? "active" : ""} onClick={() => setMode("setup")}>
              Setup
            </button>
          </div>
        ) : null}
        <Field label="API base">
          <Input value={apiBase} onChange={(event) => setApiBase(event.target.value)} />
        </Field>
        <Field label="Email">
          <Input type="email" value={email} onChange={(event) => setEmail(event.target.value)} autoComplete="email" />
        </Field>
        <Field label="Password">
          <Input type="password" value={password} onChange={(event) => setPassword(event.target.value)} autoComplete={activeMode === "setup" ? "new-password" : "current-password"} />
        </Field>
        {activeMode === "setup" ? (
          <Field label="Primary currency">
            <Select value={primaryCurrency} onChange={(event) => setPrimaryCurrency(event.target.value)}>
              {currencyOptions().map((currency) => (
                <option key={currency.code} value={currency.code}>{currency.label}</option>
              ))}
            </Select>
          </Field>
        ) : null}
        {error ? <div className="error">{error}</div> : null}
        <Button disabled={status.isLoading}>{activeMode === "setup" ? "Create user" : "Login"}</Button>
      </form>
    </div>
  );
}

function SessionPanel({ onLogout }: { onLogout: () => void }) {
  const queryClient = useQueryClient();
  const [apiBase, setApiBase] = useState(getStoredApiBase());

  return (
    <form className="auth-panel" onSubmit={(event) => { event.preventDefault(); setStoredApiBase(apiBase); location.reload(); }}>
      <Field label="API base">
        <Input value={apiBase} onChange={(event) => setApiBase(event.target.value)} />
      </Field>
      <Button>Save</Button>
      <Button
        className="muted-button"
        type="button"
        onClick={() => {
          void api.logout().finally(() => {
            queryClient.clear();
            onLogout();
          });
        }}
      >
        <LogOut size={16} /> Logout
      </Button>
    </form>
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
