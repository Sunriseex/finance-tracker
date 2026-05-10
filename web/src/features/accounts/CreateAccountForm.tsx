import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../../api/client";
import { parseMoneyToMinor } from "../../api/money";
import type { AccountType } from "../../api/types";
import { errorMessage, invalidateMoney } from "../../shared/api/query";
import { currencyOptions } from "../../shared/currencies";
import { accountTypes, today } from "../../shared/constants";
import { Button, Field, FormShell, Input, Select } from "../../shared/ui";

export function CreateAccountForm({ onDone }: { onDone: () => void }) {
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
    promoRate: "",
    promoEndDate: "",
    capitalization: "none",
  });
  const mutation = useMutation({
    mutationFn: async () => {
      const initial = parseMoneyToMinor(form.initial);
      const rate = Number(form.rate.replace(",", "."));
      const promoRate = Number(form.promoRate.replace(",", "."));

      if (Number.isNaN(rate) || rate < 0) {
        throw new Error("Annual rate must be a non-negative number");
      }

      if (Number.isNaN(promoRate) || promoRate < 0) {
        throw new Error("Promo rate must be a non-negative number");
      }

      if (rate <= 0 && (promoRate > 0 || form.promoEndDate)) {
        throw new Error("Annual rate is required when promo fields are set");
      }

      if (rate > 0 && ((promoRate > 0 && !form.promoEndDate) || (promoRate <= 0 && form.promoEndDate))) {
        throw new Error("Promo rate and promo end date must be set together");
      }

      const account = await api.createAccount({
        name: form.name,
        bank: form.bank,
        type: form.type,
        currency: form.currency,
        opened_at: form.opened_at,
      });

      if (initial > 0) {
        await api.createTransaction({
          account_id: account.id,
          type: "initial_balance",
          amount_minor: initial,
          description: "Initial balance",
          occurred_at: form.opened_at,
        });
      }

      if (rate > 0) {
        await api.createInterestRule(account.id, {
          annual_rate_bps: Math.round(rate * 100),
          promo_rate_bps: promoRate > 0 ? Math.round(promoRate * 100) : null,
          promo_end_date: promoRate > 0 ? form.promoEndDate : null,
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
  const currencies = currencyOptions();

  return (
    <FormShell title="Create account" error={error} onSubmit={() => mutation.mutate()}>
      <Field label="Name"><Input required value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} /></Field>
      <Field label="Bank"><Input value={form.bank} onChange={(event) => setForm({ ...form, bank: event.target.value })} /></Field>
      <Field label="Type"><Select value={form.type} onChange={(event) => setForm({ ...form, type: event.target.value as AccountType })}>{accountTypes.map((type) => <option key={type}>{type}</option>)}</Select></Field>
      <Field label="Currency"><Select value={form.currency} onChange={(event) => setForm({ ...form, currency: event.target.value })}>{currencies.map((currency) => <option key={currency.code} value={currency.code}>{currency.label}</option>)}</Select></Field>
      <Field label="Opened"><Input type="date" value={form.opened_at} onChange={(event) => setForm({ ...form, opened_at: event.target.value })} /></Field>
      <Field label="Initial balance"><Input inputMode="decimal" value={form.initial} onChange={(event) => setForm({ ...form, initial: event.target.value })} /></Field>
      <Field label="Annual rate %"><Input inputMode="decimal" value={form.rate} onChange={(event) => setForm({ ...form, rate: event.target.value })} /></Field>
      <Field label="Promo rate %"><Input inputMode="decimal" value={form.promoRate} onChange={(event) => setForm({ ...form, promoRate: event.target.value })} /></Field>
      <Field label="Promo end"><Input type="date" value={form.promoEndDate} onChange={(event) => setForm({ ...form, promoEndDate: event.target.value })} /></Field>
      <Field label="Capitalization"><Select value={form.capitalization} onChange={(event) => setForm({ ...form, capitalization: event.target.value })}><option>none</option><option>daily</option><option>monthly</option><option>end_of_term</option></Select></Field>
      <Button disabled={mutation.isPending}>Create</Button>
    </FormShell>
  );
}
