import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../../api/client";
import type { Account, AccountType } from "../../api/types";
import { errorMessage, invalidateMoney } from "../../shared/api/query";
import { accountTypes } from "../../shared/constants";
import { currencyOptions } from "../../shared/currencies";
import { Button, Field, FormShell, Input, Select } from "../../shared/ui";

export function EditAccountForm({ account, onDone }: { account: Account; onDone: () => void }) {
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
  const currencies = currencyOptions();

  return (
    <FormShell title="Edit account" error={error} onSubmit={() => mutation.mutate()}>
      <Field label="Name"><Input required value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} /></Field>
      <Field label="Bank"><Input value={form.bank} onChange={(event) => setForm({ ...form, bank: event.target.value })} /></Field>
      <Field label="Type"><Select value={form.type} onChange={(event) => setForm({ ...form, type: event.target.value as AccountType })}>{accountTypes.map((type) => <option key={type}>{type}</option>)}</Select></Field>
      <Field label="Currency"><Select value={form.currency} onChange={(event) => setForm({ ...form, currency: event.target.value })}>{currencies.map((currency) => <option key={currency.code} value={currency.code}>{currency.label}</option>)}</Select></Field>
      <Field label="Opened"><Input type="date" value={form.opened_at} onChange={(event) => setForm({ ...form, opened_at: event.target.value })} /></Field>
      <label className="checkbox-field">
        <input type="checkbox" checked={form.is_active} onChange={(event) => setForm({ ...form, is_active: event.target.checked })} />
        <span>Active</span>
      </label>
      <Button disabled={mutation.isPending}>Save</Button>
    </FormShell>
  );
}
