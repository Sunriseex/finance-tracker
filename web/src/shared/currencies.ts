const fallbackCurrencyCodes = [
  "AED", "ARS", "AUD", "BRL", "CAD", "CHF", "CNY", "EUR", "GBP", "HKD", "INR", "JPY", "KRW", "MXN", "RUB", "SGD", "TRY", "USD",
];

export function currencyOptions() {
  const codes = typeof Intl.supportedValuesOf === "function"
    ? Intl.supportedValuesOf("currency")
    : fallbackCurrencyCodes;

  return codes.map((code) => ({
    code,
    label: currencyLabel(code),
  }));
}

export function currencyLabel(code: string) {
  return `${code} - ${new Intl.DisplayNames(["en"], { type: "currency" }).of(code) ?? code}`;
}

