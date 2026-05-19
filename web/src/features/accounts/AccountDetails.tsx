import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Archive, BadgePercent, Pencil } from "lucide-react";
import { CartesianGrid, Line, LineChart, Tooltip, XAxis, YAxis } from "recharts";
import { api } from "../../api/client";
import { formatMoney, signedAmount } from "../../api/money";
import type { Account, InterestRule, Transaction } from "../../api/types";
import { errorMessage, invalidateMoney } from "../../shared/api/query";
import { today } from "../../shared/constants";
import { dateLabel } from "../../shared/date";
import { Button, ChartShell, Dialog, Empty, Panel } from "../../shared/ui";
import { TransactionsTable } from "../transactions/TransactionsTable";
import { EditAccountForm } from "./EditAccountForm";

export function AccountDetails({ account, onBack }: { account: Account; onBack: () => void }) {
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
        <Dialog title="Edit account" onClose={() => setEditOpen(false)}>
          <EditAccountForm account={account} onDone={() => setEditOpen(false)} />
        </Dialog>
      ) : null}
    </div>
  );
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

function runningBalance(transactions: Transaction[]) {
  let balance = 0;
  return [...transactions]
    .sort((a, b) => a.occurred_at.localeCompare(b.occurred_at))
    .map((transaction) => {
      balance += signedAmount(transaction);
      return { date: transaction.occurred_at.slice(0, 10), balance };
    });
}

