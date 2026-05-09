import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "../../api/client";
import { formatMoney, parseMoneyToMinor } from "../../api/money";
import type { Account } from "../../api/types";
import { errorMessage, invalidateMoney } from "../../shared/api/query";
import { Button, Empty, Field, FormShell, Input, Select } from "../../shared/ui";

export function TransferForm({ accounts, onDone }: { accounts: Account[]; onDone: () => void }) {
  const queryClient = useQueryClient();
  const [error, setError] = useState("");
  const [form, setForm] = useState({
    from_account_id: accounts[0]?.id ?? "",
    to_account_id: accounts[1]?.id ?? "",
    amount: "",
    description: "",
  });
  const fromAccount = accounts.find((account) => account.id === form.from_account_id);
  const toAccount = accounts.find((account) => account.id === form.to_account_id);
  const amountMinor = parseMoneyToMinor(form.amount);
  const rates = useQuery({
    queryKey: ["currency-rates", fromAccount?.currency],
    queryFn: () => api.currencyRates(fromAccount?.currency ?? "RUB"),
    enabled: Boolean(fromAccount?.currency && toAccount?.currency && fromAccount.currency !== toAccount.currency),
    staleTime: 1000 * 60 * 60,
  });
  const rate = toAccount?.currency ? rates.data?.rates[toAccount.currency] : undefined;
  const convertedMinor = amountMinor > 0 && rate ? Math.round(amountMinor * rate) : 0;
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
      {fromAccount && toAccount && fromAccount.currency !== toAccount.currency ? (
        <div className="conversion-preview">
          <span>{fromAccount.currency} to {toAccount.currency}</span>
          {rates.isLoading ? <strong>Loading rate</strong> : null}
          {rate ? (
            <strong>
              {formatMoney(amountMinor, fromAccount.currency)} = {formatMoney(convertedMinor, toAccount.currency)}
            </strong>
          ) : null}
          {rates.error ? <Empty>{errorMessage(rates.error)}</Empty> : null}
        </div>
      ) : null}
      <Field label="Description"><Input value={form.description} onChange={(event) => setForm({ ...form, description: event.target.value })} /></Field>
      <Button disabled={mutation.isPending}>Create</Button>
    </FormShell>
  );
}
