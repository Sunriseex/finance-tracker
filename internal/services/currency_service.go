package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/shopspring/decimal"
)

const (
	defaultExchangeRateURL = "https://open.er-api.com/v6/latest"
	defaultExchangeRateTTL = 6 * time.Hour
)

type ExchangeRates struct {
	Base      string
	Date      string
	Provider  string
	FetchedAt time.Time
	Rates     map[string]decimal.Decimal
}

type ExchangeRateProvider interface {
	Latest(ctx context.Context, base string) (*ExchangeRates, error)
}

type CurrencyService struct {
	provider ExchangeRateProvider
}

func NewCurrencyService(provider ExchangeRateProvider) *CurrencyService {
	if provider == nil {
		provider = defaultExchangeRateProvider
	}
	return &CurrencyService{provider: provider}
}

func (s *CurrencyService) Latest(ctx context.Context, base string) (*ExchangeRates, error) {
	base = normalizeCurrency(base)
	if base == "" {
		return nil, validationError("base currency is required")
	}
	rates, err := s.provider.Latest(ctx, base)
	if err != nil {
		return nil, fmt.Errorf("load exchange rates: %w", err)
	}
	return rates, nil
}

func (s *CurrencyService) ConvertMinor(ctx context.Context, amountMinor int64, from, to string) (int64, decimal.Decimal, error) {
	from = normalizeCurrency(from)
	to = normalizeCurrency(to)
	if from == "" || to == "" {
		return 0, decimal.Zero, validationError("currency is required")
	}
	if from == to {
		return amountMinor, decimal.NewFromInt(1), nil
	}

	rates, err := s.Latest(ctx, from)
	if err != nil {
		return 0, decimal.Zero, err
	}

	rate, ok := rates.Rates[to]
	if !ok || !rate.IsPositive() {
		return 0, decimal.Zero, validationError("exchange rate not found for " + from + "/" + to)
	}

	converted := decimal.NewFromInt(amountMinor).Mul(rate).Round(0).IntPart()
	return converted, rate, nil
}

func normalizeCurrency(currency string) string {
	return strings.ToUpper(strings.TrimSpace(currency))
}

type HTTPExchangeRateProvider struct {
	client *http.Client
	base   string
	ttl    time.Duration
	now    func() time.Time

	mu    sync.Mutex
	cache map[string]cachedExchangeRates
}

type cachedExchangeRates struct {
	rates     *ExchangeRates
	expiresAt time.Time
}

var defaultExchangeRateProvider = NewHTTPExchangeRateProvider(nil, defaultExchangeRateURL, defaultExchangeRateTTL)

func NewHTTPExchangeRateProvider(client *http.Client, baseURL string, ttl time.Duration) *HTTPExchangeRateProvider {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultExchangeRateURL
	}
	if ttl <= 0 {
		ttl = defaultExchangeRateTTL
	}

	return &HTTPExchangeRateProvider{
		client: client,
		base:   strings.TrimRight(baseURL, "/"),
		ttl:    ttl,
		now:    time.Now,
		cache:  make(map[string]cachedExchangeRates),
	}
}

func (p *HTTPExchangeRateProvider) Latest(ctx context.Context, base string) (*ExchangeRates, error) {
	base = normalizeCurrency(base)
	if base == "" {
		return nil, validationError("base currency is required")
	}

	now := p.now()
	p.mu.Lock()
	if cached, ok := p.cache[base]; ok && now.Before(cached.expiresAt) {
		p.mu.Unlock()
		return cached.rates, nil
	}
	p.mu.Unlock()

	rates, err := p.fetch(ctx, base, now)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	p.cache[base] = cachedExchangeRates{rates: rates, expiresAt: now.Add(p.ttl)}
	p.mu.Unlock()
	return rates, nil
}

func (p *HTTPExchangeRateProvider) fetch(ctx context.Context, base string, now time.Time) (*ExchangeRates, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.base+"/"+base, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("build exchange rate request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch exchange rates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("fetch exchange rates: status %d", resp.StatusCode)
	}

	var payload exchangeRateAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode exchange rates: %w", err)
	}
	if payload.Result != "success" {
		return nil, fmt.Errorf("fetch exchange rates: provider result %q", payload.Result)
	}

	normalizedRates := make(map[string]decimal.Decimal, len(payload.Rates)+1)
	for currency, rate := range payload.Rates {
		normalizedRates[normalizeCurrency(currency)] = rate
	}
	normalizedRates[base] = decimal.NewFromInt(1)

	return &ExchangeRates{
		Base:      normalizeCurrency(payload.BaseCode),
		Date:      payload.TimeLastUpdateUTC,
		Provider:  "open.er-api.com",
		FetchedAt: now,
		Rates:     normalizedRates,
	}, nil
}

type exchangeRateAPIResponse struct {
	Result            string                     `json:"result"`
	BaseCode          string                     `json:"base_code"`
	TimeLastUpdateUTC string                     `json:"time_last_update_utc"`
	Rates             map[string]decimal.Decimal `json:"rates"`
}
