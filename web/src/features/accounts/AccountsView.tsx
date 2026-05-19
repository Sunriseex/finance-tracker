import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { api } from "../../api/client";
import { formatMoney } from "../../api/money";
import type { Account, InterestRule } from "../../api/types";
import { accountTypes } from "../../shared/constants";
import { errorMessage } from "../../shared/api/query";
import { Button, Empty, Panel, Select } from "../../shared/ui";

export function AccountsView({
  accounts,
  isLoading = false,
  error = null,
  onSelect,
}: {
  accounts: Account[];
  isLoading?: boolean;
  error?: unknown;
  onSelect: (id: string) => void;
}) {
  const [type, setType] = useState("");
  const summary = useQuery({ queryKey: ["dashboard", "summary"], queryFn: api.dashboardSummary });
  const rules = useQuery({ queryKey: ["interest-rules"], queryFn: () => api.interestRules() });
  const balances = new Map((summary.data?.account_balances ?? []).map((account) => [account.account_id, account]));
  const activeRules = activeRulesByAccount(rules.data ?? []);
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
      {isLoading ? <Empty>Loading accounts</Empty> : null}
      {error ? <div className="error inline-error">{errorMessage(error)}</div> : null}
      {!isLoading && !error && !filtered.length ? <Empty>No accounts</Empty> : null}
      <div className="table-wrap">
        <table>
          <thead>
            <tr><th>Name</th><th>Bank</th><th>Type</th><th>Balance</th><th>Rate</th><th>Status</th><th></th></tr>
          </thead>
          <tbody>
            {!isLoading && !error ? filtered.map((account) => (
              <tr key={account.id}>
                <td>{account.name}</td>
                <td>{account.bank || "-"}</td>
                <td>{account.type}</td>
                <td className="amount">{formatMoney(balances.get(account.id)?.balance_minor ?? 0, account.currency)}</td>
                <td><AccountRate rule={activeRules.get(account.id)} isLoading={rules.isLoading} error={rules.error} /></td>
                <td>{account.is_active ? "active" : "archived"}</td>
                <td><Button onClick={() => onSelect(account.id)}>Open</Button></td>
              </tr>
            )) : null}
          </tbody>
        </table>
      </div>
    </Panel>
  );
}

function AccountRate({ rule, isLoading, error }: { rule?: InterestRule; isLoading: boolean; error: unknown }) {
  if (isLoading) {
    return <span>Loading</span>;
  }
  if (error) {
    return <span className="error-text">{errorMessage(error)}</span>;
  }
  if (!rule) {
    return <span>-</span>;
  }
  return <span>{(rule.annual_rate_bps / 100).toFixed(2)}%</span>;
}

function activeRulesByAccount(rules: InterestRule[]) {
  const activeRules = new Map<string, InterestRule>();
  for (const rule of rules) {
    if (!rule.is_active) {
      continue;
    }
    const current = activeRules.get(rule.account_id);
    if (!current || rule.start_date.localeCompare(current.start_date) > 0) {
      activeRules.set(rule.account_id, rule);
    }
  }
  return activeRules;
}

