import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Plus } from "lucide-react";
import { api } from "../../api/client";
import type { Account, Category } from "../../api/types";
import { errorMessage } from "../../shared/api/query";
import { transactionTypes } from "../../shared/constants";
import { Button, Dialog, Empty, Input, Panel, Select } from "../../shared/ui";
import { TransactionForm } from "./TransactionForm";
import { TransactionsTable } from "./TransactionsTable";

export function TransactionsView({
  accounts,
  categories,
  accountsLoading = false,
  accountsError = null,
  categoriesLoading = false,
  categoriesError = null,
}: {
  accounts: Account[];
  categories: Category[];
  accountsLoading?: boolean;
  accountsError?: unknown;
  categoriesLoading?: boolean;
  categoriesError?: unknown;
}) {
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

  const disabledCreate = accountsLoading || Boolean(accountsError) || accounts.length === 0;

  return (
    <Panel
      title="Transactions"
      action={<Button onClick={() => setCreateOpen(true)} disabled={disabledCreate}><Plus size={16} /> Adjustment</Button>}
    >
      {accountsLoading ? <Empty>Loading accounts</Empty> : null}
      {accountsError ? <div className="error inline-error">{errorMessage(accountsError)}</div> : null}
      {categoriesLoading ? <Empty>Loading categories</Empty> : null}
      {categoriesError ? <div className="error inline-error">{errorMessage(categoriesError)}</div> : null}
      {transactions.isLoading ? <Empty>Loading transactions</Empty> : null}
      {transactions.error ? <div className="error inline-error">{errorMessage(transactions.error)}</div> : null}
      <div className="filters">
        <Select value={accountId} disabled={accountsLoading || Boolean(accountsError)} onChange={(event) => setAccountId(event.target.value)}>
          <option value="">All accounts</option>
          {accounts.map((account) => <option key={account.id} value={account.id}>{account.name}</option>)}
        </Select>
        <Select value={categoryId} disabled={categoriesLoading || Boolean(categoriesError)} onChange={(event) => setCategoryId(event.target.value)}>
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
      {!transactions.isLoading && !transactions.error ? (
        <TransactionsTable transactions={filtered} accounts={accounts} categories={categories} allowDelete />
      ) : null}
      {createOpen ? (
        <Dialog title="Create adjustment" onClose={() => setCreateOpen(false)}>
          <TransactionForm accounts={accounts} categories={categories} fixedType="adjustment" onDone={() => setCreateOpen(false)} />
        </Dialog>
      ) : null}
    </Panel>
  );
}

