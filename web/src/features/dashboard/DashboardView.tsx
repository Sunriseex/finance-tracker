import { useState } from "react";
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
import { convertMinor, formatMoney, sumConverted } from "../../api/money";
import type { Account } from "../../api/types";
import { errorMessage } from "../../shared/api/query";
import { ChartShell, Empty, Panel } from "../../shared/ui";
import { TransactionsTable } from "../transactions/TransactionsTable";

export function DashboardView({ primaryCurrency, onOpenAccount }: { primaryCurrency: string; onOpenAccount: (id: string) => void }) {
  const summary = useQuery({ queryKey: ["dashboard", "summary"], queryFn: api.dashboardSummary });
  const cashflow = useQuery({ queryKey: ["dashboard", "cashflow"], queryFn: api.dashboardCashflow });
  const interest = useQuery({ queryKey: ["dashboard", "interest"], queryFn: api.dashboardInterestIncome });
  const [selectedCurrency, setSelectedCurrency] = useState(primaryCurrency);
  const data = summary.data;

  const balances = data?.account_balances ?? [];
  const currencyTotals = data?.balances ?? [];
  const seenCurrencies = new Set<string>([selectedCurrency]);
  for (const amount of currencyTotals) {
    seenCurrencies.add(amount.currency);
  }
  for (const account of balances) {
    seenCurrencies.add(account.currency);
  }
  const currencies = [...seenCurrencies].sort();
  const rates = useQuery({
    queryKey: ["currency-rates", selectedCurrency],
    queryFn: () => api.currencyRates(selectedCurrency),
    enabled: Boolean(selectedCurrency),
    staleTime: 1000 * 60 * 60,
  });
  const rateTable = rates.data?.base === selectedCurrency ? rates.data : undefined;
  const portfolioValue = sumConverted(currencyTotals, selectedCurrency, rateTable);
  const conversionStatus = rates.error
    ? errorMessage(rates.error)
    : rateTable
      ? `${rateTable.provider}, ${rateTable.date}`
      : "Loading rates";

  const chartData = (cashflow.data?.buckets ?? []).map((bucket) => ({
    period: bucket.period,
    income: sumConverted(bucket.income, selectedCurrency, rateTable),
    expense: sumConverted(bucket.expense, selectedCurrency, rateTable),
    net: sumConverted(bucket.net_cashflow, selectedCurrency, rateTable),
  }));

  const interestData = (interest.data?.buckets ?? []).map((bucket) => ({
    period: bucket.period,
    interest: sumConverted(bucket.interest_income, selectedCurrency, rateTable),
  }));

  const allocation = balances
    .filter((account) => account.balance_minor > 0)
    .map((account) => ({
      ...account,
      converted_balance_minor: convertMinor(account.balance_minor, account.currency, selectedCurrency, rateTable),
    }))
    .sort((a, b) => b.converted_balance_minor - a.converted_balance_minor)
    .slice(0, 6)
    .map((account) => ({
      ...account,
      share: portfolioValue > 0 ? Math.round((account.converted_balance_minor / portfolioValue) * 100) : 0,
    }));

  const monthlyNet =
    sumConverted(data?.monthly_income, selectedCurrency, rateTable) -
    sumConverted(data?.monthly_expense, selectedCurrency, rateTable);
  const recentAccounts = balances.map((account): Account => ({
    id: account.account_id,
    name: account.name,
    bank: account.bank,
    type: account.type,
    currency: account.currency,
    is_active: account.is_active,
    opened_at: "",
    created_at: "",
    updated_at: "",
  }));

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
          <div className="hero-totals">
            <strong>{formatMoney(portfolioValue, selectedCurrency)}</strong>
          </div>
          <span>
            {data?.active_accounts_count ?? 0} active accounts across {currencies.length || 1} currency
          </span>
        </div>

        <div className={monthlyNet < 0 ? "hero-delta negative" : "hero-delta"}>
          <span>Net this month</span>
          <strong>{formatMoney(monthlyNet, selectedCurrency)}</strong>
        </div>
      </section>

      <div className="currency-tabs" role="tablist" aria-label="Dashboard currency">
        {currencies.map((currency) => (
          <button
            key={currency}
            className={currency === selectedCurrency ? "active" : ""}
            onClick={() => setSelectedCurrency(currency)}
          >
            {currency}
          </button>
        ))}
      </div>

      <div className="metric-strip">
        <div className="metric primary-metric">
          <span>Main currency</span>
          <strong>{selectedCurrency}</strong>
          <small>{conversionStatus}</small>
        </div>

        {currencyTotals.map((amount) => (
          <div className="metric" key={amount.currency}>
            <span>Total {amount.currency}</span>
            <strong>{formatMoney(amount.amount_minor, amount.currency)}</strong>
            {amount.currency !== selectedCurrency ? (
              <small>{formatMoney(convertMinor(amount.amount_minor, amount.currency, selectedCurrency, rateTable), selectedCurrency)}</small>
            ) : null}
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
          <strong>{formatMoney(sumConverted(data?.monthly_income, selectedCurrency, rateTable), selectedCurrency)}</strong>
        </div>

        <div className="metric">
          <span>Expense this month</span>
          <strong>{formatMoney(sumConverted(data?.monthly_expense, selectedCurrency, rateTable), selectedCurrency)}</strong>
        </div>

        <div className="metric">
          <span>Interest this month</span>
          <strong>{formatMoney(sumConverted(data?.monthly_interest_income, selectedCurrency, rateTable), selectedCurrency)}</strong>
        </div>
      </div>

      <div className="dashboard-main">
        <Panel title={`Cashflow trend (${selectedCurrency})`}>
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
              <Tooltip formatter={(value) => formatMoney(Number(value), selectedCurrency)} />
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
                {account.currency !== selectedCurrency ? (
                  <small className="allocation-converted">
                    {formatMoney(account.converted_balance_minor, selectedCurrency)}
                  </small>
                ) : null}

                <span className="allocation-bar">
                  <i style={{ width: `${account.share}%` }} />
                </span>

                <em>{account.share}%</em>
              </button>
            ))}

            {!allocation.length ? <Empty>No positive balances</Empty> : null}
          </div>
        </Panel>
      </div>

      <Panel title={`Cashflow (${selectedCurrency})`}>
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
            <Tooltip formatter={(value) => formatMoney(Number(value), selectedCurrency)} />
            <Bar dataKey="income" fill="#24735a" radius={[4, 4, 0, 0]} />
            <Bar dataKey="expense" fill="#a23b3b" radius={[4, 4, 0, 0]} />
          </ComposedChart>
        </ChartShell>
      </Panel>

      <Panel title={`Interest income (${selectedCurrency})`}>
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
            <Tooltip formatter={(value) => formatMoney(Number(value), selectedCurrency)} />
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
                  <td className="amount stacked-amount">
                    <strong>{formatMoney(account.balance_minor, account.currency)}</strong>
                    {account.currency !== selectedCurrency ? (
                      <small>
                        {formatMoney(
                          convertMinor(account.balance_minor, account.currency, selectedCurrency, rateTable),
                          selectedCurrency,
                        )}
                      </small>
                    ) : null}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Panel>

      <Panel title="Recent transactions">
        <TransactionsTable transactions={data?.recent_transactions ?? []} accounts={recentAccounts} categories={[]} compact />
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
