import { useMemo, useState } from "react";
import type { ReactElement, ReactNode } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { QueryClient } from "@tanstack/react-query";
import {
  ArrowDownLeft,
  ArrowRightLeft,
  ArrowUpRight,
  Archive,
  BadgePercent,
  Landmark,
  Pencil,
  Plus,
  Settings,
  Trash2,
  Wallet,
} from "lucide-react";
import {
  Area,
  Bar,
  CartesianGrid,
  ComposedChart,
  Legend,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { api, ApiClientError, getStoredApiBase, getStoredToken, setStoredApiBase, setStoredToken } from "./api/client";
import { amountFor, formatMoney, parseMoneyToMinor, signedAmount, transactionTypeLabel } from "./api/money";
import type { Account, AccountType, Category, InterestRule, Transaction, TransactionType } from "./api/types";
import { Button, Empty, Field, IconButton, Input, Panel, Select } from "./components/ui";

type View = "dashboard" | "accounts" | "transactions";
type QuickAction = "income" | "expense" | "transfer" | "account" | null;

const today = new Date().toISOString().slice(0, 10);
const accountTypes: AccountType[] = ["cash", "card", "savings", "term_deposit", "broker", "other"];
const transactionTypes: TransactionType[] = ["income", "expense", "adjustment"];

export function App() {
  const [view, setView] = useState<View>("dashboard");
  const [selectedAccountId, setSelectedAccountId] = useState("");
  const [quickAction, setQuickAction] = useState<QuickAction>(null);
  const [authOpen, setAuthOpen] = useState(false);
  const accounts = useQuery({ queryKey: ["accounts"], queryFn: api.accounts });
  const categories = useQuery({ queryKey: ["categories"], queryFn: api.categories });

  const selectedAccount = accounts.data?.find((account) => account.id === selectedAccountId);

  return (
    <div className="app">
      <aside className="sidebar">
        <div className="brand">
          <Wallet size={22} />
          <span>Finance Tracker</span>
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
        </nav>
        <Button className="muted-button" onClick={() => setAuthOpen((open) => !open)}>
          <Settings size={16} /> API
        </Button>
        {authOpen ? <AuthPanel /> : null}
      </aside>

      <main>
        <header className="topbar">
          <div>
            <p className="eyebrow">v0.5 MVP</p>
            <h1>{selectedAccount ? selectedAccount.name : titleForView(view)}</h1>
          </div>
          <div className="quick-actions">
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

        {view === "dashboard" ? <DashboardView onOpenAccount={(id) => { setSelectedAccountId(id); setView("accounts"); }} /> : null}
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

function AuthPanel() {
  const [token, setToken] = useState(getStoredToken());
  const [apiBase, setApiBase] = useState(getStoredApiBase());

  return (
    <form className="auth-panel" onSubmit={(event) => { event.preventDefault(); setStoredToken(token); setStoredApiBase(apiBase); location.reload(); }}>
      <Field label="API base">
        <Input value={apiBase} onChange={(event) => setApiBase(event.target.value)} />
      </Field>
      <Field label="Bearer token">
        <Input type="password" value={token} onChange={(event) => setToken(event.target.value)} />
      </Field>
      <Button>Save</Button>
    </form>
  );
}

function DashboardView({ onOpenAccount }: { onOpenAccount: (id: string) => void }) {
  const summary = useQuery({ queryKey: ["dashboard", "summary"], queryFn: api.dashboardSummary });
  const cashflow = useQuery({ queryKey: ["dashboard", "cashflow"], queryFn: api.dashboardCashflow });
  const interest = useQuery({ queryKey: ["dashboard", "interest"], queryFn: api.dashboardInterestIncome });
  const data = summary.data;

  const chartData = (cashflow.data?.buckets ?? []).map((bucket) => ({
    period: bucket.period,
    income: amountFor(bucket.income),
    expense: amountFor(bucket.expense),
    net: amountFor(bucket.net_cashflow),
  }));
  const interestData = (interest.data?.buckets ?? []).map((bucket) => ({
    period: bucket.period,
    interest: amountFor(bucket.interest_income),
  }));
  const balances = data?.account_balances ?? [];
  const primaryCurrency = balances[0]?.currency ?? "RUB";
  const portfolioValue = balances.reduce((sum, account) => sum + Math.max(account.balance_minor, 0), 0);
  const allocation = balances
    .filter((account) => account.balance_minor > 0)
    .sort((a, b) => b.balance_minor - a.balance_minor)
    .slice(0, 6)
    .map((account) => ({
      ...account,
      share: portfolioValue > 0 ? Math.round((account.balance_minor / portfolioValue) * 100) : 0,
    }));
  const monthlyNet = amountFor(data?.monthly_income, primaryCurrency) - amountFor(data?.monthly_expense, primaryCurrency);

  if (summary.isLoading) {
    return <Empty>Loading dashboard</Empty>;
  }
  if (summary.error) {
    return <Empty>{errorMessage(summary.error)}</Empty>;
  }

  return (
    <div className="grid">
      <section className="portfolio-hero">
        <div>
          <p className="eyebrow">Portfolio value</p>
          <strong>{formatMoney(portfolioValue, primaryCurrency)}</strong>
          <span>{data?.active_accounts_count ?? 0} active accounts across {(data?.balances ?? []).length || 1} currency</span>
        </div>
        <div className={monthlyNet < 0 ? "hero-delta negative" : "hero-delta"}>
          <span>Net this month</span>
          <strong>{formatMoney(monthlyNet, primaryCurrency)}</strong>
        </div>
      </section>

      <div className="metric-strip">
        {(data?.balances ?? []).map((amount) => (
          <div className="metric" key={amount.currency}>
            <span>Total {amount.currency}</span>
            <strong>{formatMoney(amount.amount_minor, amount.currency)}</strong>
          </div>
        ))}
        <div className="metric">
          <span>Accounts</span>
          <strong>{data?.active_accounts_count ?? 0}/{data?.accounts_count ?? 0}</strong>
        </div>
        <div className="metric">
          <span>Income this month</span>
          <strong>{formatMoney(amountFor(data?.monthly_income))}</strong>
        </div>
        <div className="metric">
          <span>Expense this month</span>
          <strong>{formatMoney(amountFor(data?.monthly_expense))}</strong>
        </div>
        <div className="metric">
          <span>Interest this month</span>
          <strong>{formatMoney(amountFor(data?.monthly_interest_income))}</strong>
        </div>
      </div>

      <div className="dashboard-main">
        <Panel title="Cashflow trend">
          <ChartShell size="large">
            <ComposedChart data={chartData} margin={{ top: 8, right: 18, bottom: 0, left: 0 }}>
              <defs>
                <linearGradient id="netFlow" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#315f8d" stopOpacity={0.22} />
                  <stop offset="95%" stopColor="#315f8d" stopOpacity={0.02} />
                </linearGradient>
              </defs>
              <CartesianGrid stroke="#e2e9e5" vertical={false} />
              <XAxis dataKey="period" axisLine={false} tickLine={false} />
              <YAxis axisLine={false} tickLine={false} width={70} tickFormatter={(value) => formatCompactMoney(Number(value))} />
              <Tooltip formatter={(value) => formatMoney(Number(value), primaryCurrency)} />
              <Legend />
              <Area type="monotone" dataKey="net" name="Net" stroke="#315f8d" fill="url(#netFlow)" strokeWidth={2} />
              <Bar dataKey="income" name="Income" fill="#24735a" radius={[4, 4, 0, 0]} />
              <Bar dataKey="expense" name="Expense" fill="#a23b3b" radius={[4, 4, 0, 0]} />
              <Line type="monotone" dataKey="net" name="Net line" stroke="#1f2937" strokeWidth={2} dot={false} />
            </ComposedChart>
          </ChartShell>
        </Panel>

        <Panel title="Allocation">
          <div className="allocation-list">
            {allocation.map((account) => (
              <button className="allocation-row" key={account.account_id} onClick={() => onOpenAccount(account.account_id)}>
                <span>
                  <strong>{account.name}</strong>
                  <small>{account.bank || account.type}</small>
                </span>
                <span className="allocation-value">{formatMoney(account.balance_minor, account.currency)}</span>
                <span className="allocation-bar"><i style={{ width: `${account.share}%` }} /></span>
                <em>{account.share}%</em>
              </button>
            ))}
            {!allocation.length ? <Empty>No positive balances</Empty> : null}
          </div>
        </Panel>
      </div>

      <Panel title="Cashflow">
        <ChartShell>
          <ComposedChart data={chartData} margin={{ top: 8, right: 14, bottom: 0, left: 0 }}>
            <CartesianGrid stroke="#e2e9e5" vertical={false} />
            <XAxis dataKey="period" axisLine={false} tickLine={false} />
            <YAxis axisLine={false} tickLine={false} width={70} tickFormatter={(value) => formatCompactMoney(Number(value))} />
            <Tooltip formatter={(value) => formatMoney(Number(value), primaryCurrency)} />
            <Bar dataKey="income" fill="#24735a" radius={[4, 4, 0, 0]} />
            <Bar dataKey="expense" fill="#a23b3b" radius={[4, 4, 0, 0]} />
          </ComposedChart>
        </ChartShell>
      </Panel>

      <Panel title="Interest income">
        <ChartShell>
          <LineChart data={interestData} margin={{ top: 8, right: 14, bottom: 0, left: 0 }}>
            <CartesianGrid stroke="#e2e9e5" vertical={false} />
            <XAxis dataKey="period" axisLine={false} tickLine={false} />
            <YAxis axisLine={false} tickLine={false} width={70} tickFormatter={(value) => formatCompactMoney(Number(value))} />
            <Tooltip formatter={(value) => formatMoney(Number(value), primaryCurrency)} />
            <Line type="monotone" dataKey="interest" stroke="#8a6f2a" strokeWidth={3} dot={{ r: 3 }} activeDot={{ r: 5 }} />
          </LineChart>
        </ChartShell>
      </Panel>

      <Panel title="Account balances">
        <div className="table-wrap">
          <table>
            <tbody>
              {(data?.account_balances ?? []).map((account) => (
                <tr key={account.account_id} onClick={() => onOpenAccount(account.account_id)}>
                  <td>{account.name}</td>
                  <td>{account.bank || "-"}</td>
                  <td>{account.type}</td>
                  <td className="amount">{formatMoney(account.balance_minor, account.currency)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Panel>

      <Panel title="Recent transactions">
        <TransactionsTable transactions={data?.recent_transactions ?? []} accounts={[]} categories={[]} compact />
      </Panel>
    </div>
  );
}

function AccountsView({ accounts, onSelect }: { accounts: Account[]; onSelect: (id: string) => void }) {
  const [type, setType] = useState("");
  const summary = useQuery({ queryKey: ["dashboard", "summary"], queryFn: api.dashboardSummary });
  const balances = new Map((summary.data?.account_balances ?? []).map((account) => [account.account_id, account]));
  const filtered = accounts.filter((account) => !type || account.type === type);

  return (
    <Panel
      title="Accounts"
      action={
        <Select value={type} onChange={(event) => setType(event.target.value)}>
          <option value="">All types</option>
          {accountTypes.map((accountType) => <option key={accountType}>{accountType}</option>)}
        </Select>
      }
    >
      <div className="table-wrap">
        <table>
          <thead>
            <tr><th>Name</th><th>Bank</th><th>Type</th><th>Balance</th><th>Rate</th><th>Status</th><th></th></tr>
          </thead>
          <tbody>
            {filtered.map((account) => (
              <tr key={account.id}>
                <td>{account.name}</td>
                <td>{account.bank || "-"}</td>
                <td>{account.type}</td>
                <td className="amount">{formatMoney(balances.get(account.id)?.balance_minor ?? 0, account.currency)}</td>
                <td><AccountRate accountId={account.id} /></td>
                <td>{account.is_active ? "active" : "archived"}</td>
                <td><Button onClick={() => onSelect(account.id)}>Open</Button></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </Panel>
  );
}

function AccountRate({ accountId }: { accountId: string }) {
  const rules = useQuery({ queryKey: ["interest-rules", accountId], queryFn: () => api.interestRules(accountId) });
  const activeRule = rules.data?.find((rule) => rule.is_active);
  if (!activeRule) {
    return <span>-</span>;
  }
  return <span>{(activeRule.annual_rate_bps / 100).toFixed(2)}%</span>;
}

function AccountDetails({ account, onBack }: { account: Account; onBack: () => void }) {
  const queryClient = useQueryClient();
  const [editOpen, setEditOpen] = useState(false);
  const [actionError, setActionError] = useState("");
  const transactions = useQuery({ queryKey: ["transactions", account.id], queryFn: () => api.transactions(account.id) });
  const balance = useQuery({ queryKey: ["balance", account.id], queryFn: () => api.accountBalance(account.id) });
  const rules = useQuery({ queryKey: ["interest-rules", account.id], queryFn: () => api.interestRules(account.id) });
  const accrue = useMutation({
    mutationFn: () => api.accrueInterest(account.id, today),
    onSuccess: () => invalidateMoney(queryClient),
  });
  const archive = useMutation({
    mutationFn: () => api.archiveAccount(account.id),
    onSuccess: () => {
      setActionError("");
      invalidateMoney(queryClient);
    },
    onError: (err) => setActionError(errorMessage(err)),
  });
  const running = useMemo(() => runningBalance(transactions.data ?? []), [transactions.data]);

  return (
    <div className="grid">
      <Panel
        title="Account summary"
        action={
          <div className="panel-actions">
            <Button onClick={() => setEditOpen(true)}><Pencil size={16} /> Edit</Button>
            <Button onClick={() => archive.mutate()} disabled={archive.isPending || !account.is_active}><Archive size={16} /> Archive</Button>
            <Button onClick={onBack}>Back</Button>
          </div>
        }
      >
        {actionError ? <div className="error inline-error">{actionError}</div> : null}
        <div className="summary-grid">
          <div><span>Balance</span><strong>{formatMoney(balance.data?.balance_minor ?? 0, account.currency)}</strong></div>
          <div><span>Bank</span><strong>{account.bank || "-"}</strong></div>
          <div><span>Status</span><strong>{account.is_active ? "active" : "archived"}</strong></div>
          <div><span>Opened</span><strong>{dateLabel(account.opened_at)}</strong></div>
        </div>
      </Panel>

      <Panel title="Running balance">
        <ChartShell>
          <LineChart data={running}>
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis dataKey="date" />
            <YAxis />
            <Tooltip formatter={(value) => formatMoney(Number(value), account.currency)} />
            <Line type="monotone" dataKey="balance" stroke="#3b6ea8" strokeWidth={2} />
          </LineChart>
        </ChartShell>
      </Panel>

      <Panel
        title="Interest rules"
        action={<Button onClick={() => accrue.mutate()} disabled={accrue.isPending}><BadgePercent size={16} /> Accrue</Button>}
      >
        <div className="rule-list">
          {(rules.data ?? []).map((rule) => <RuleRow key={rule.id} rule={rule} />)}
          {!rules.data?.length ? <Empty>No interest rules</Empty> : null}
        </div>
      </Panel>

      <Panel title="Transactions">
        <TransactionsTable transactions={transactions.data ?? []} accounts={[account]} categories={[]} allowDelete />
      </Panel>

      {editOpen ? (
        <div className="modal-backdrop" onClick={() => setEditOpen(false)}>
          <div className="modal" onClick={(event) => event.stopPropagation()}>
            <EditAccountForm account={account} onDone={() => setEditOpen(false)} />
          </div>
        </div>
      ) : null}
    </div>
  );
}

function TransactionsView({ accounts, categories }: { accounts: Account[]; categories: Category[] }) {
  const transactions = useQuery({ queryKey: ["transactions"], queryFn: () => api.transactions() });
  const [createOpen, setCreateOpen] = useState(false);
  const [accountId, setAccountId] = useState("");
  const [categoryId, setCategoryId] = useState("");
  const [type, setType] = useState("");
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  const filtered = (transactions.data ?? []).filter((transaction) => {
    const day = transaction.occurred_at.slice(0, 10);
    return (!accountId || transaction.account_id === accountId) &&
      (!categoryId || transaction.category_id === categoryId) &&
      (!type || transaction.type === type) &&
      (!from || day >= from) &&
      (!to || day <= to);
  });

  return (
    <Panel title="Transactions" action={<Button onClick={() => setCreateOpen(true)}><Plus size={16} /> Adjustment</Button>}>
      <div className="filters">
        <Select value={accountId} onChange={(event) => setAccountId(event.target.value)}>
          <option value="">All accounts</option>
          {accounts.map((account) => <option key={account.id} value={account.id}>{account.name}</option>)}
        </Select>
        <Select value={categoryId} onChange={(event) => setCategoryId(event.target.value)}>
          <option value="">All categories</option>
          {categories.map((category) => <option key={category.id} value={category.id}>{category.name}</option>)}
        </Select>
        <Select value={type} onChange={(event) => setType(event.target.value)}>
          <option value="">All types</option>
          {transactionTypes.map((transactionType) => <option key={transactionType}>{transactionType}</option>)}
        </Select>
        <Input type="date" value={from} onChange={(event) => setFrom(event.target.value)} />
        <Input type="date" value={to} onChange={(event) => setTo(event.target.value)} />
      </div>
      <TransactionsTable transactions={filtered} accounts={accounts} categories={categories} allowDelete />
      {createOpen ? (
        <div className="modal-backdrop" onClick={() => setCreateOpen(false)}>
          <div className="modal" onClick={(event) => event.stopPropagation()}>
            <TransactionForm accounts={accounts} categories={categories} fixedType="adjustment" onDone={() => setCreateOpen(false)} />
          </div>
        </div>
      ) : null}
    </Panel>
  );
}

function CreateAccountForm({ onDone }: { onDone: () => void }) {
  const queryClient = useQueryClient();
  const [error, setError] = useState("");
  const [form, setForm] = useState({
    name: "",
    bank: "",
    type: "card" as AccountType,
    currency: "RUB",
    opened_at: today,
    initial: "",
    rate: "",
    capitalization: "none",
  });
  const mutation = useMutation({
    mutationFn: async () => {
      const account = await api.createAccount({
        name: form.name,
        bank: form.bank,
        type: form.type,
        currency: form.currency,
        opened_at: form.opened_at,
      });
      const initial = parseMoneyToMinor(form.initial);
      if (initial > 0) {
        await api.createTransaction({
          account_id: account.id,
          type: "initial_balance",
          amount_minor: initial,
          description: "Initial balance",
          occurred_at: form.opened_at,
        });
      }
      const rate = Number(form.rate.replace(",", "."));
      if (rate > 0) {
        await api.createInterestRule(account.id, {
          annual_rate_bps: Math.round(rate * 100),
          accrual_frequency: "daily",
          capitalization_frequency: form.capitalization as "none" | "daily" | "monthly" | "end_of_term",
          day_count_convention: "actual_365",
          start_date: form.opened_at,
        });
      }
      return account;
    },
    onSuccess: () => {
      invalidateMoney(queryClient);
      onDone();
    },
    onError: (err) => setError(errorMessage(err)),
  });

  return (
    <FormShell title="Create account" error={error} onSubmit={() => mutation.mutate()}>
      <Field label="Name"><Input required value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} /></Field>
      <Field label="Bank"><Input value={form.bank} onChange={(event) => setForm({ ...form, bank: event.target.value })} /></Field>
      <Field label="Type"><Select value={form.type} onChange={(event) => setForm({ ...form, type: event.target.value as AccountType })}>{accountTypes.map((type) => <option key={type}>{type}</option>)}</Select></Field>
      <Field label="Currency"><Input value={form.currency} maxLength={3} onChange={(event) => setForm({ ...form, currency: event.target.value.toUpperCase() })} /></Field>
      <Field label="Opened"><Input type="date" value={form.opened_at} onChange={(event) => setForm({ ...form, opened_at: event.target.value })} /></Field>
      <Field label="Initial balance"><Input inputMode="decimal" value={form.initial} onChange={(event) => setForm({ ...form, initial: event.target.value })} /></Field>
      <Field label="Annual rate %"><Input inputMode="decimal" value={form.rate} onChange={(event) => setForm({ ...form, rate: event.target.value })} /></Field>
      <Field label="Capitalization"><Select value={form.capitalization} onChange={(event) => setForm({ ...form, capitalization: event.target.value })}><option>none</option><option>daily</option><option>monthly</option><option>end_of_term</option></Select></Field>
      <Button disabled={mutation.isPending}>Create</Button>
    </FormShell>
  );
}

function EditAccountForm({ account, onDone }: { account: Account; onDone: () => void }) {
  const queryClient = useQueryClient();
  const [error, setError] = useState("");
  const [form, setForm] = useState({
    name: account.name,
    bank: account.bank ?? "",
    type: account.type,
    currency: account.currency,
    opened_at: account.opened_at.slice(0, 10),
    is_active: account.is_active,
  });
  const mutation = useMutation({
    mutationFn: () => api.updateAccount(account.id, form),
    onSuccess: () => {
      invalidateMoney(queryClient);
      onDone();
    },
    onError: (err) => setError(errorMessage(err)),
  });

  return (
    <FormShell title="Edit account" error={error} onSubmit={() => mutation.mutate()}>
      <Field label="Name"><Input required value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} /></Field>
      <Field label="Bank"><Input value={form.bank} onChange={(event) => setForm({ ...form, bank: event.target.value })} /></Field>
      <Field label="Type"><Select value={form.type} onChange={(event) => setForm({ ...form, type: event.target.value as AccountType })}>{accountTypes.map((type) => <option key={type}>{type}</option>)}</Select></Field>
      <Field label="Currency"><Input value={form.currency} maxLength={3} onChange={(event) => setForm({ ...form, currency: event.target.value.toUpperCase() })} /></Field>
      <Field label="Opened"><Input type="date" value={form.opened_at} onChange={(event) => setForm({ ...form, opened_at: event.target.value })} /></Field>
      <label className="checkbox-field">
        <input type="checkbox" checked={form.is_active} onChange={(event) => setForm({ ...form, is_active: event.target.checked })} />
        <span>Active</span>
      </label>
      <Button disabled={mutation.isPending}>Save</Button>
    </FormShell>
  );
}

function TransactionForm({ accounts, categories, fixedType, onDone }: { accounts: Account[]; categories: Category[]; fixedType?: TransactionType; onDone: () => void }) {
  const queryClient = useQueryClient();
  const [error, setError] = useState("");
  const [form, setForm] = useState({
    account_id: accounts[0]?.id ?? "",
    type: fixedType ?? "income",
    amount: "",
    category_id: "",
    description: "",
    occurred_at: today,
  });
  const mutation = useMutation({
    mutationFn: () => api.createTransaction({
      account_id: form.account_id,
      type: form.type as TransactionType,
      amount_minor: parseMoneyToMinor(form.amount),
      category_id: form.category_id || null,
      description: form.description,
      occurred_at: form.occurred_at,
    }),
    onSuccess: () => {
      invalidateMoney(queryClient);
      onDone();
    },
    onError: (err) => setError(errorMessage(err)),
  });

  return (
    <FormShell title={`Create ${form.type}`} error={error} onSubmit={() => mutation.mutate()}>
      <Field label="Account"><Select value={form.account_id} onChange={(event) => setForm({ ...form, account_id: event.target.value })}>{accounts.map((account) => <option key={account.id} value={account.id}>{account.name}</option>)}</Select></Field>
      {!fixedType ? <Field label="Type"><Select value={form.type} onChange={(event) => setForm({ ...form, type: event.target.value as TransactionType })}>{transactionTypes.map((type) => <option key={type}>{type}</option>)}</Select></Field> : null}
      <Field label="Amount"><Input required inputMode="decimal" value={form.amount} onChange={(event) => setForm({ ...form, amount: event.target.value })} /></Field>
      <Field label="Category"><Select value={form.category_id} onChange={(event) => setForm({ ...form, category_id: event.target.value })}><option value="">None</option>{categories.map((category) => <option key={category.id} value={category.id}>{category.name}</option>)}</Select></Field>
      <Field label="Date"><Input type="date" value={form.occurred_at} onChange={(event) => setForm({ ...form, occurred_at: event.target.value })} /></Field>
      <Field label="Description"><Input value={form.description} onChange={(event) => setForm({ ...form, description: event.target.value })} /></Field>
      <Button disabled={mutation.isPending}>Create</Button>
    </FormShell>
  );
}

function TransferForm({ accounts, onDone }: { accounts: Account[]; onDone: () => void }) {
  const queryClient = useQueryClient();
  const [error, setError] = useState("");
  const [form, setForm] = useState({
    from_account_id: accounts[0]?.id ?? "",
    to_account_id: accounts[1]?.id ?? "",
    amount: "",
    description: "",
  });
  const mutation = useMutation({
    mutationFn: () => api.createTransfer({
      from_account_id: form.from_account_id,
      to_account_id: form.to_account_id,
      amount_minor: parseMoneyToMinor(form.amount),
      description: form.description,
    }),
    onSuccess: () => {
      invalidateMoney(queryClient);
      onDone();
    },
    onError: (err) => setError(errorMessage(err)),
  });

  return (
    <FormShell title="Create transfer" error={error} onSubmit={() => mutation.mutate()}>
      <Field label="From"><Select value={form.from_account_id} onChange={(event) => setForm({ ...form, from_account_id: event.target.value })}>{accounts.map((account) => <option key={account.id} value={account.id}>{account.name}</option>)}</Select></Field>
      <Field label="To"><Select value={form.to_account_id} onChange={(event) => setForm({ ...form, to_account_id: event.target.value })}>{accounts.map((account) => <option key={account.id} value={account.id}>{account.name}</option>)}</Select></Field>
      <Field label="Amount"><Input required inputMode="decimal" value={form.amount} onChange={(event) => setForm({ ...form, amount: event.target.value })} /></Field>
      <Field label="Description"><Input value={form.description} onChange={(event) => setForm({ ...form, description: event.target.value })} /></Field>
      <Button disabled={mutation.isPending}>Create</Button>
    </FormShell>
  );
}

function FormShell({ title, error, onSubmit, children }: { title: string; error: string; onSubmit: () => void; children: ReactNode }) {
  return (
    <form className="form" onSubmit={(event) => { event.preventDefault(); onSubmit(); }}>
      <h2>{title}</h2>
      {error ? <div className="error">{error}</div> : null}
      {children}
    </form>
  );
}

function TransactionsTable({
  transactions,
  accounts,
  categories,
  compact = false,
  allowDelete = false,
}: {
  transactions: Transaction[];
  accounts: Account[];
  categories: Category[];
  compact?: boolean;
  allowDelete?: boolean;
}) {
  const queryClient = useQueryClient();
  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.deleteTransaction(id),
    onSuccess: () => invalidateMoney(queryClient),
  });
  const accountNames = new Map(accounts.map((account) => [account.id, account.name]));
  const categoryNames = new Map(categories.map((category) => [category.id, category.name]));

  if (!transactions.length) {
    return <Empty>No transactions</Empty>;
  }

  return (
    <div className="table-wrap">
      {deleteMutation.error ? <div className="error inline-error">{errorMessage(deleteMutation.error)}</div> : null}
      <table>
        <thead>
          <tr><th>Date</th><th>Type</th>{compact ? null : <th>Account</th>}{compact ? null : <th>Category</th>}<th>Description</th><th>Amount</th>{allowDelete ? <th></th> : null}</tr>
        </thead>
        <tbody>
          {transactions.map((transaction) => (
            <tr key={transaction.id}>
              <td>{dateLabel(transaction.occurred_at)}</td>
              <td>{transactionTypeLabel(transaction.type)}</td>
              {compact ? null : <td>{accountNames.get(transaction.account_id) ?? transaction.account_id.slice(0, 8)}</td>}
              {compact ? null : <td>{transaction.category_id ? categoryNames.get(transaction.category_id) ?? transaction.category_id.slice(0, 8) : "-"}</td>}
              <td>{transaction.description || "-"}</td>
              <td className={signedAmount(transaction) < 0 ? "amount danger" : "amount"}>{formatMoney(signedAmount(transaction))}</td>
              {allowDelete ? (
                <td>
                  <IconButton
                    title="Delete transaction"
                    disabled={deleteMutation.isPending || isTransferTransaction(transaction)}
                    onClick={() => {
                      if (window.confirm("Delete this transaction?")) {
                        deleteMutation.mutate(transaction.id);
                      }
                    }}
                  >
                    <Trash2 size={16} />
                  </IconButton>
                </td>
              ) : null}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function isTransferTransaction(transaction: Transaction) {
  return transaction.type === "transfer_in" || transaction.type === "transfer_out";
}

function RuleRow({ rule }: { rule: InterestRule }) {
  const rate = (rule.annual_rate_bps / 100).toFixed(2);
  return (
    <div className="rule-row">
      <strong>{rate}%</strong>
      <span>{rule.accrual_frequency}</span>
      <span>{rule.capitalization_frequency}</span>
      <span>{rule.is_active ? "active" : "inactive"}</span>
    </div>
  );
}

function ChartShell({ children, size = "regular" }: { children: ReactElement; size?: "regular" | "large" }) {
  return (
    <div className={`chart chart-${size}`}>
      <ResponsiveContainer width="100%" height="100%">
        {children}
      </ResponsiveContainer>
    </div>
  );
}

function formatCompactMoney(value: number) {
  const abs = Math.abs(value);
  if (abs >= 1_000_000) {
    return `${Math.round(value / 1_000_000)}M`;
  }
  if (abs >= 1_000) {
    return `${Math.round(value / 1_000)}K`;
  }
  return `${value}`;
}

function runningBalance(transactions: Transaction[]) {
  let balance = 0;
  return [...transactions]
    .sort((a, b) => a.occurred_at.localeCompare(b.occurred_at))
    .map((transaction) => {
      balance += signedAmount(transaction);
      return { date: transaction.occurred_at.slice(0, 10), balance };
    });
}

function invalidateMoney(queryClient: QueryClient) {
  void queryClient.invalidateQueries({ queryKey: ["accounts"] });
  void queryClient.invalidateQueries({ queryKey: ["transactions"] });
  void queryClient.invalidateQueries({ queryKey: ["dashboard"] });
  void queryClient.invalidateQueries({ queryKey: ["balance"] });
  void queryClient.invalidateQueries({ queryKey: ["interest-rules"] });
}

function dateLabel(date: string) {
  return new Date(date).toLocaleDateString("ru-RU");
}

function errorMessage(err: unknown) {
  if (err instanceof ApiClientError) {
    return `${err.code ? `${err.code}: ` : ""}${err.message}`;
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
  }[view];
}
