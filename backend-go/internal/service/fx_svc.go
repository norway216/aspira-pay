package service

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/aspira/aspira-pay/internal/domain/fx"
	pkgerrors "github.com/aspira/aspira-pay/pkg/errors"
	"github.com/aspira/aspira-pay/pkg/idgen"
)

// FXService provides FX quote generation with live rates.
// All rates are USD-based: we store XXX/USD, and compute cross-rates through USD.
// Settlement always happens in USD.
type FXService struct {
	mu         sync.RWMutex
	usdRates   map[string]string // currency -> USD rate (e.g., "JPY" -> "156.0" means 1 USD = 156 JPY)
	lastFetch  time.Time
	fetchURL   string
	httpClient *http.Client
}

// frankfurterResponse matches the Frankfurter API JSON format.
type frankfurterResponse struct {
	Amount float64            `json:"amount"`
	Base   string             `json:"base"`
	Date   string             `json:"date"`
	Rates  map[string]float64 `json:"rates"`
}

// NewFXService creates a new FXService and fetches initial rates.
func NewFXService(fetchURL string) *FXService {
	svc := &FXService{
		usdRates:   fx.DefaultUSDRates(), // Fallback defaults
		fetchURL:   fetchURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
	// Fetch live rates on startup
	go svc.fetchLiveRates()
	return svc
}

// RefreshLoop periodically fetches live rates.
func (s *FXService) RefreshLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		s.fetchLiveRates()
	}
}

// fetchLiveRates fetches USD-based rates from the public API.
func (s *FXService) fetchLiveRates() {
	if s.fetchURL == "" {
		return
	}

	resp, err := s.httpClient.Get(s.fetchURL)
	if err != nil {
		fmt.Printf("[FX] Failed to fetch live rates: %v — using cached/defaults\n", err)
		return
	}
	defer resp.Body.Close()

	var data frankfurterResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		fmt.Printf("[FX] Failed to parse live rates: %v\n", err)
		return
	}

	if data.Base != "USD" || len(data.Rates) == 0 {
		fmt.Printf("[FX] Unexpected API response: base=%s rates=%d\n", data.Base, len(data.Rates))
		return
	}

	s.mu.Lock()
	for currency, rate := range data.Rates {
		// Store as string with 12 decimal places for precision
		s.usdRates[currency] = fmt.Sprintf("%.12f", rate)
	}
	s.lastFetch = time.Now()
	s.mu.Unlock()

	fmt.Printf("[FX] Live rates updated: %d currencies (base=USD, date=%s)\n", len(data.Rates), data.Date)
}

// GetQuote generates an FX quote. All conversions go through USD.
// source_amount × (USD/source) × (target/USD) = target_amount
func (s *FXService) GetQuote(req fx.QuoteRequest) (*fx.QuoteResponse, error) {
	rateStr, err := s.getPairRate(req.SourceCurrency, req.TargetCurrency)
	if err != nil {
		return nil, err
	}

	// Parse rate using big.Rat for precision
	rateRat := new(big.Rat)
	if _, ok := rateRat.SetString(rateStr); !ok {
		return nil, fmt.Errorf("invalid rate: %s", rateStr)
	}

	// Calculate target amount
	sourceRat := new(big.Rat).SetInt64(req.SourceAmount)
	targetRat := new(big.Rat).Mul(sourceRat, rateRat)
	num := targetRat.Num()
	denom := targetRat.Denom()
	targetAmount := new(big.Int).Div(num, denom)

	// Calculate fee in USD equivalent
	usdAmount := s.convertToUSD(req.SourceCurrency, req.SourceAmount)
	feeBps := fx.FeeBasisPointsUSD()
	feeUSDCents := usdAmount * feeBps / 10000

	quoteID := idgen.QuoteID()
	expiresAt := time.Now().Unix() + fx.QuoteTTLSeconds

	return &fx.QuoteResponse{
		QuoteID:        quoteID,
		SourceCurrency: req.SourceCurrency,
		TargetCurrency: req.TargetCurrency,
		Rate:           rateStr,
		SourceAmount:   req.SourceAmount,
		TargetAmount:   targetAmount.Int64(),
		FeeAmount:      feeUSDCents,
		ExpiresAt:      expiresAt,
	}, nil
}

// convertToUSD converts any currency amount to USD cents.
func (s *FXService) convertToUSD(currency string, amount int64) int64 {
	if currency == "USD" {
		return amount
	}

	rateStr := s.getUSDRate(currency)
	if rateStr == "" {
		return 0
	}

	// For XXX/USD: USD = XXX_amount / rate
	// e.g., 15600 JPY / 156.0 = 100 USD
	rateRat := new(big.Rat)
	rateRat.SetString(rateStr)

	amountRat := new(big.Rat).SetInt64(amount)
	usdRat := new(big.Rat).Quo(amountRat, rateRat)

	num := usdRat.Num()
	denom := usdRat.Denom()
	return new(big.Int).Div(num, denom).Int64()
}

// getUSDRate returns the USD rate for a currency (XXX/USD).
func (s *FXService) getUSDRate(currency string) string {
	if currency == "USD" {
		return "1.000000000000"
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	rate, ok := s.usdRates[currency]
	if !ok {
		// Try fallback
		if fb, ok2 := fx.DefaultUSDRates()[currency]; ok2 {
			return fb
		}
		return ""
	}
	return rate
}

// getPairRate computes cross-rate through USD.
// EUR/JPY = (EUR/USD) × (JPY/USD)⁻¹ = EUR_USD / JPY_USD
// Source→Target: amount_in_source × (1/src_usd) × (tgt_usd) = target
func (s *FXService) getPairRate(source, target string) (string, error) {
	if source == target {
		return "1.000000000000", nil
	}

	sourceUSD := s.getUSDRate(source)
	targetUSD := s.getUSDRate(target)

	if sourceUSD == "" {
		return "", pkgerrors.InvalidInput(fmt.Sprintf("unsupported source currency: %s", source))
	}
	if targetUSD == "" {
		return "", pkgerrors.InvalidInput(fmt.Sprintf("unsupported target currency: %s", target))
	}

	// Cross-rate: rate = target_USD / source_USD
	// e.g., EUR→JPY: rate = 156.0 / 0.92 = 169.57
	srcRat := new(big.Rat)
	srcRat.SetString(sourceUSD)
	tgtRat := new(big.Rat)
	tgtRat.SetString(targetUSD)

	crossRat := new(big.Rat).Quo(tgtRat, srcRat)
	return crossRat.FloatString(12), nil
}

// GetRate returns the current cross-rate for a currency pair.
func (s *FXService) GetRate(source, target string) (string, error) {
	return s.getPairRate(source, target)
}

// GetUSDRate returns the USD rate for a single currency.
func (s *FXService) GetUSDRate(currency string) string {
	return s.getUSDRate(currency)
}

// ListRates returns all available FX rates.
func (s *FXService) ListRates() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]string)
	for k, v := range s.usdRates {
		result["USD/"+k] = v
		result[k+"/USD"] = fmt.Sprintf("%.12f", 1.0/mustParseFloat(v))
	}
	return result
}

// ConvertToUSD is the public method to convert any amount to USD.
func (s *FXService) ConvertToUSD(currency string, amount int64) int64 {
	return s.convertToUSD(currency, amount)
}

// ConvertFromUSD converts USD to any target currency.
func (s *FXService) ConvertFromUSD(targetCurrency string, usdCents int64) int64 {
	if targetCurrency == "USD" {
		return usdCents
	}

	rateStr := s.getUSDRate(targetCurrency)
	if rateStr == "" {
		return 0
	}

	rateRat := new(big.Rat)
	rateRat.SetString(rateStr)

	usdRat := new(big.Rat).SetInt64(usdCents)
	targetRat := new(big.Rat).Mul(usdRat, rateRat)

	num := targetRat.Num()
	denom := targetRat.Denom()
	return new(big.Int).Div(num, denom).Int64()
}

func mustParseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	if f == 0 {
		return 1.0
	}
	return f
}
