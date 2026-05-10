import { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { api } from "../../api/client";
import type { Profile } from "../../api/types";
import { errorMessage } from "../../shared/api/query";
import { currencyOptions } from "../../shared/currencies";
import { Button, Field, Input, Panel, Select } from "../../shared/ui";

export function SettingsView({ profile }: { profile?: Profile }) {
  const queryClient = useQueryClient();
  const [primaryCurrency, setPrimaryCurrency] = useState(profile?.user.primary_currency ?? "RUB");
  const [error, setError] = useState("");
  const [saved, setSaved] = useState(false);
  const currencies = currencyOptions();

  async function save() {
    setError("");
    setSaved(false);
    try {
      await api.updateProfile({ primary_currency: primaryCurrency });
      await queryClient.invalidateQueries({ queryKey: ["profile"] });
      await queryClient.invalidateQueries({ queryKey: ["dashboard"] });
      setSaved(true);
    } catch (err) {
      setError(errorMessage(err));
    }
  }

  return (
    <div className="grid settings-grid">
      <Panel title="Profile">
        <form className="form compact-form" onSubmit={(event) => { event.preventDefault(); void save(); }}>
          <Field label="Email">
            <Input value={profile?.user.email ?? ""} readOnly />
          </Field>
          <Field label="Primary currency">
            <Select value={primaryCurrency} onChange={(event) => { setPrimaryCurrency(event.target.value); setSaved(false); }}>
              {currencies.map((currency) => (
                <option key={currency.code} value={currency.code}>{currency.label}</option>
              ))}
            </Select>
          </Field>
          {error ? <div className="error">{error}</div> : null}
          {saved ? <div className="success">Saved</div> : null}
          <Button>Save settings</Button>
        </form>
      </Panel>
    </div>
  );
}
