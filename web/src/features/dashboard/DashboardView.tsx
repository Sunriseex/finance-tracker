import { useQuery } from "@tanstack/react-query";
import {
  Area,
  Bar,
  CartesianGrid,
  ComposedChart,
  Legend,
  Line,
  LineChart,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { api } from "../../api/client";
import { amountFor, formatMoney } from "../../api/money";
import { errorMessage } from "../../shared/api/query";
import { ChartShell, Empty, Panel } from "../../shared/ui";
import { TransactionsTable } from "../transactions/TransactionsTable";

export function DashboardView({ onOpenAccount }: { onOpenAccount: (id: string) => void }) {
  const summary = useQuery({ queryKey: ["dashboard", "summary"], queryFn: api.dashboardSummary });
  const cashflow = useQuery({ queryKey: ["dashboard", "cashflow"], queryFn: api.dashboardCashflow });
  const interest = useQuery({ queryKey: ["dashboard", "interest"], queryFn: api.dashboardInterestIncome });
  const data = summary.data;

  const balances = data?.account_balances ?? [];
  const currencyTotals = data?.balances ?? [];
  const primaryCurrency = currencyTotals[0]?.currency ?? balances[0]?.currency ?? "RUB";
  const chartCurrency = primaryCurrency;

  const chartData = (cashflow.data?.buckets ?? []).map((bucket) => ({
    period: bucket.period,
    income: amountFor(bucket.income, chartCurrency),
    expense: amountFor(bucket.expense, chartCurrency),
    net: amountFor(bucket.net_cashflow, chartCurrency),
  }));

  const interestData = (interest.data?.buckets ?? []).map((bucket) => ({
    period: bucket.period,
    interest: amountFor(bucket.interest_income, chartCurrency),
  }));

  const positiveTotalsByCurrency = new Map(
    currencyTotals.map((amount) => [amount.currency, Math.max(amount.amount_minor, 0)]),
  );

  const allocation = balances
    .filter((account) => account.balance_minor > 0)
    .sort((a, b) => {
      if (a.currency !== b.currency) {
        return a.currency.localeCompare(b.currency);
      }

      return b.balance_minor - a.balance_minor;
    })
    .slice(0, 6)
    .map((account) => {
      const currencyTotal = positiveTotalsByCurrency.get(account.currency) ?? 0;

      return {
        ...account,
        share: currencyTotal > 0 ? Math.round((account.balance_minor / currencyTotal) * 100) : 0,
      };
    });

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
          <p className="eyebrow">Portfolio value by currency</p>
          <div className="hero-totals">
            {currencyTotals.length ? (
              currencyTotals.map((amount) => (
                <strong key={amount.currency}>
                  {formatMoney(amount.amount_minor, amount.currency)}
                </strong>
              ))
            ) : (
              <strong>{formatMoney(0, primaryCurrency)}</strong>
            )}
          </div>
          <span>
            {data?.active_accounts_count ?? 0} active accounts across {currencyTotals.length || 1} currency
          </span>
        </div>

        <div className={monthlyNet < 0 ? "hero-delta negative" : "hero-delta"}>
          <span>Net this month</span>
          <strong>{formatMoney(monthlyNet, primaryCurrency)}</strong>
        </div>
      </section>

      <div className="metric-strip">
        {currencyTotals.map((amount) => (
          <div className="metric" key={amount.currency}>
            <span>Total {amount.currency}</span>
            <strong>{formatMoney(amount.amount_minor, amount.currency)}</strong>
          </div>
        ))}

        <div className="metric">
          <span>Accounts</span>
          <strong>
            {data?.active_accounts_count ?? 0}/{data?.accounts_count ?? 0}
          </strong>
        </div>

        <div className="metric">
          <span>Income this month</span>
          <strong>{formatMoney(amountFor(data?.monthly_income, primaryCurrency), primaryCurrency)}</strong>
        </div>

        <div className="metric">
          <span>Expense this month</span>
          <strong>{formatMoney(amountFor(data?.monthly_expense, primaryCurrency), primaryCurrency)}</strong>
        </div>

        <div className="metric">
          <span>Interest this month</span>
          <strong>{formatMoney(amountFor(data?.monthly_interest_income, primaryCurrency), primaryCurrency)}</strong>
        </div>
      </div>

      <div className="dashboard-main">
        <Panel title={`Cashflow trend (${chartCurrency})`}>
          <ChartShell size="large">
            <ComposedChart data={chartData} margin={{ top: 8, right: 18, bottom: 0, left: 0 }}>
              <defs>
                <linearGradient id="netFlow" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#315f8d" stopOpacity={0.22} />
                  <stop offset="95%" stopColor="#315f8d" stopOpacity={0.02} />
                </linearGradient>
              </defs>

              <CartesianGrid stroke="var(--chart-grid)" vertical={false} />
              <XAxis dataKey="period" axisLine={false} tickLine={false} />
              <YAxis
                axisLine={false}
                tickLine={false}
                width={70}
                tickFormatter={(value) => formatCompactMoney(Number(value))}
              />
              <Tooltip formatter={(value) => formatMoney(Number(value), chartCurrency)} />
              <Legend />

              <Area
                type="monotone"
                dataKey="net"
                name="Net"
                stroke="#315f8d"
                fill="url(#netFlow)"
                strokeWidth={2}
              />
              <Bar dataKey="income" name="Income" fill="#24735a" radius={[4, 4, 0, 0]} />
              <Bar dataKey="expense" name="Expense" fill="#a23b3b" radius={[4, 4, 0, 0]} />
              <Line
                type="monotone"
                dataKey="net"
                name="Net line"
                stroke="#1f2937"
                strokeWidth={2}
                dot={false}
              />
            </ComposedChart>
          </ChartShell>
        </Panel>

        <Panel title="Allocation">
          <div className="allocation-list">
            {allocation.map((account) => (
              <button
                className="allocation-row"
                key={account.account_id}
                onClick={() => onOpenAccount(account.account_id)}
              >
                <span>
                  <strong>{account.name}</strong>
                  <small>{account.bank || account.type}</small>
                </span>

                <span className="allocation-value">
                  {formatMoney(account.balance_minor, account.currency)}
                </span>

                <span className="allocation-bar">
                  <i style={{ width: `${account.share}%` }} />
                </span>

                <em>
                  {account.share}% {account.currency}
                </em>
              </button>
            ))}

            {!allocation.length ? <Empty>No positive balances</Empty> : null}
          </div>
        </Panel>
      </div>

      <Panel title={`Cashflow (${chartCurrency})`}>
        <ChartShell>
          <ComposedChart data={chartData} margin={{ top: 8, right: 14, bottom: 0, left: 0 }}>
            <CartesianGrid stroke="var(--chart-grid)" vertical={false} />
            <XAxis dataKey="period" axisLine={false} tickLine={false} />
            <YAxis
              axisLine={false}
              tickLine={false}
              width={70}
              tickFormatter={(value) => formatCompactMoney(Number(value))}
            />
            <Tooltip formatter={(value) => formatMoney(Number(value), chartCurrency)} />
            <Bar dataKey="income" fill="#24735a" radius={[4, 4, 0, 0]} />
            <Bar dataKey="expense" fill="#a23b3b" radius={[4, 4, 0, 0]} />
          </ComposedChart>
        </ChartShell>
      </Panel>

      <Panel title={`Interest income (${chartCurrency})`}>
        <ChartShell>
          <LineChart data={interestData} margin={{ top: 8, right: 14, bottom: 0, left: 0 }}>
            <CartesianGrid stroke="var(--chart-grid)" vertical={false} />
            <XAxis dataKey="period" axisLine={false} tickLine={false} />
            <YAxis
              axisLine={false}
              tickLine={false}
              width={70}
              tickFormatter={(value) => formatCompactMoney(Number(value))}
            />
            <Tooltip formatter={(value) => formatMoney(Number(value), chartCurrency)} />
            <Line
              type="monotone"
              dataKey="interest"
              stroke="#8a6f2a"
              strokeWidth={3}
              dot={{ r: 3 }}
              activeDot={{ r: 5 }}
            />
          </LineChart>
        </ChartShell>
      </Panel>

      <Panel title="Account balances">
        <div className="table-wrap">
          <table>
            <tbody>
              {balances.map((account) => (
                <tr key={account.account_id} onClick={() => onOpenAccount(account.account_id)}>
                  <td>{account.name}</td>
                  <td>{account.bank || "-"}</td>
                  <td>{account.type}</td>
                  <td className="amount">
                    {formatMoney(account.balance_minor, account.currency)}
                  </td>
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