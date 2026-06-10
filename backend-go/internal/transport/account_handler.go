package transport

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/aspira/aspira-pay/internal/repository"
	"github.com/aspira/aspira-pay/internal/service"
)

// AccountHandler handles account balance HTTP endpoints.
type AccountHandler struct {
	db  *repository.DB
	fx  *service.FXService
}

// NewAccountHandler creates a new AccountHandler.
func NewAccountHandler(db *repository.DB, fx *service.FXService) *AccountHandler {
	return &AccountHandler{db: db, fx: fx}
}

// GetMyAccounts returns the current user's accounts with balances
// converted to their default currency.
func (h *AccountHandler) GetMyAccounts(c *gin.Context) {
	userID := c.GetString("user_id")

	// Get user for default currency
	u, err := h.db.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	accounts, err := h.db.GetAccountsByUser(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	defaultCur := u.GetDefaultCurrency()

	type AccountDisplay struct {
		AccountID        string `json:"account_id"`
		Currency         string `json:"currency"`
		AvailableBalance int64  `json:"available_balance"`
		FrozenBalance    int64  `json:"frozen_balance"`
		SettledBalance   int64  `json:"settled_balance"`
		DefaultAmount    int64  `json:"default_amount"`    // Converted to user's default currency
		DefaultCurrency  string `json:"default_currency"`
	}

	result := make([]AccountDisplay, 0, len(accounts))
	for _, a := range accounts {
		// Convert available balance to user's default currency
		defaultAmount := a.AvailableBalance
		if a.Currency != defaultCur {
			// Convert via USD: source → USD → target
			usdCents := h.fx.ConvertToUSD(a.Currency, a.AvailableBalance)
			defaultAmount = h.fx.ConvertFromUSD(defaultCur, usdCents)
		}

		result = append(result, AccountDisplay{
			AccountID:        a.AccountID,
			Currency:         a.Currency,
			AvailableBalance: a.AvailableBalance,
			FrozenBalance:    a.FrozenBalance,
			SettledBalance:   a.SettledBalance,
			DefaultAmount:    defaultAmount,
			DefaultCurrency:  defaultCur,
		})
	}

	// Also include the total in user's default currency
	totalUSD := int64(0)
	for _, a := range accounts {
		totalUSD += h.fx.ConvertToUSD(a.Currency, a.AvailableBalance)
	}
	totalDefault := h.fx.ConvertFromUSD(defaultCur, totalUSD)

	c.JSON(http.StatusOK, gin.H{
		"accounts":        result,
		"default_currency": defaultCur,
		"total_default":    totalDefault,
		"total_usd":        totalUSD,
		// Live FX rates for reference
		"fx_rates": map[string]string{
			"JPY/USD": h.fx.GetUSDRate("JPY"),
			"EUR/USD": h.fx.GetUSDRate("EUR"),
			"CNY/USD": h.fx.GetUSDRate("CNY"),
			"GBP/USD": h.fx.GetUSDRate("GBP"),
		},
	})
}

// BadgeAmount formats an amount for display.
func BadgeAmount(amount int64, currency string) string {
	decimals := 2
	if currency == "JPY" || currency == "KRW" {
		decimals = 0
	}
	if decimals == 0 {
		return currency + " " + strconv.FormatInt(amount, 10)
	}
	major := float64(amount) / 100.0
	return currency + " " + strconv.FormatFloat(major, 'f', 2, 64)
}

// GetTotalUSDBalance returns the total balance across all accounts converted to USD.
func (h *AccountHandler) GetTotalUSDBalance(c *gin.Context) {
	userID := c.GetString("user_id")
	accounts, err := h.db.GetAccountsByUser(userID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if len(accounts) == 0 {
		c.JSON(200, gin.H{"total_usd": 0, "has_accounts": false, "currencies": 0})
		return
	}

	var totalUSD int64
	currencyCount := 0
	for _, a := range accounts {
		if a.AvailableBalance <= 0 { continue }
		currencyCount++
		if a.Currency == "USD" {
			totalUSD += a.AvailableBalance
		} else {
			usdAmount := h.fx.ConvertToUSD(a.Currency, a.AvailableBalance)
			totalUSD += usdAmount
		}
	}

	c.JSON(200, gin.H{
		"total_usd":   totalUSD,
		"has_accounts": true,
		"currencies":   currencyCount,
	})
}

