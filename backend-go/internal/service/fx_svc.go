package service

import (
	"fmt"
	"math/big"
	"time"

	"github.com/aspira/aspira-pay/internal/domain/fx"
	pkgerrors "github.com/aspira/aspira-pay/pkg/errors"
	"github.com/aspira/aspira-pay/pkg/idgen"
)

// FXService provides FX quote generation and rate management.
type FXService struct {
	rates map[string]string // currency pair -> rate string (NUMERIC)
}

// NewFXService creates a new FXService with default Sandbox rates.
func NewFXService() *FXService {
	return &FXService{
		rates: fx.DefaultRates(),
	}
}

// GetQuote generates an FX quote for a currency pair and amount.
// All monetary calculations use integer arithmetic with NUMERIC string rates.
func (s *FXService) GetQuote(req fx.QuoteRequest) (*fx.QuoteResponse, error) {
	pair := req.SourceCurrency + "/" + req.TargetCurrency

	rateStr, ok := s.rates[pair]
	if !ok {
		return nil, pkgerrors.InvalidInput(fmt.Sprintf("unsupported currency pair: %s", pair))
	}

	// Parse rate using big.Rat for precision
	rateRat := new(big.Rat)
	if _, ok := rateRat.SetString(rateStr); !ok {
		return nil, fmt.Errorf("invalid rate: %s", rateStr)
	}

	// Calculate target amount: source_amount * rate
	sourceRat := new(big.Rat).SetInt64(req.SourceAmount)
	targetRat := new(big.Rat).Mul(sourceRat, rateRat)

	// Truncate to integer (smallest currency unit)
	// big.Rat.Num() returns the numerator (no arguments), Denom() returns denominator
	num := targetRat.Num()
	denom := targetRat.Denom()
	targetAmount := new(big.Int).Div(num, denom) // Integer division (floor)

	// Calculate fee
	feeBps := fx.FeeBasisPoints(req.SourceCurrency, req.TargetCurrency)
	feeAmount := req.SourceAmount * feeBps / 10000

	quoteID := idgen.QuoteID()
	expiresAt := time.Now().Unix() + fx.QuoteTTLSeconds

	quote := &fx.QuoteResponse{
		QuoteID:        quoteID,
		SourceCurrency: req.SourceCurrency,
		TargetCurrency: req.TargetCurrency,
		Rate:           rateStr,
		SourceAmount:   req.SourceAmount,
		TargetAmount:   targetAmount.Int64(),
		FeeAmount:      feeAmount,
		ExpiresAt:      expiresAt,
	}

	return quote, nil
}

// GetRate returns the current rate for a currency pair.
func (s *FXService) GetRate(source, target string) (string, error) {
	pair := source + "/" + target
	rate, ok := s.rates[pair]
	if !ok {
		return "", pkgerrors.NotFound(fmt.Sprintf("rate not found for %s", pair))
	}
	return rate, nil
}

// ListRates returns all available FX rates.
func (s *FXService) ListRates() map[string]string {
	result := make(map[string]string)
	for k, v := range s.rates {
		result[k] = v
	}
	return result
}

// CalculateTargetAmount computes target amount from source amount and rate.
func CalculateTargetAmount(sourceAmount int64, rateStr string) (int64, error) {
	rateRat := new(big.Rat)
	if _, ok := rateRat.SetString(rateStr); !ok {
		return 0, fmt.Errorf("invalid rate: %s", rateStr)
	}

	sourceRat := new(big.Rat).SetInt64(sourceAmount)
	targetRat := new(big.Rat).Mul(sourceRat, rateRat)

	num := targetRat.Num()
	denom := targetRat.Denom()
	targetAmount := new(big.Int).Div(num, denom)

	return targetAmount.Int64(), nil
}
