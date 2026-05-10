import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Trash2 } from "lucide-react";
import { api } from "../../api/client";
import { formatMoney, signedAmount, transactionTypeLabel } from "../../api/money";
import type { Account, Category, Transaction } from "../../api/types";
import { errorMessage, invalidateMoney } from "../../shared/api/query";
import { dateLabel } from "../../shared/date";
import { Empty, IconButton } from "../../shared/ui";

export function TransactionsTable({
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
  const accountCurrencies = new Map(accounts.map((account) => [account.id, account.currency]));
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
              <td className={signedAmount(transaction) < 0 ? "amount danger" : "amount"}>
                {formatMoney(signedAmount(transaction), accountCurrencies.get(transaction.account_id) ?? "RUB")}
              </td>
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
