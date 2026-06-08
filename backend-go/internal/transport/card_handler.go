package transport

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/aspira/aspira-pay/internal/domain/card"
	"github.com/aspira/aspira-pay/internal/domain/user"
	"github.com/aspira/aspira-pay/internal/service"
)

// CardHandler handles card-related HTTP endpoints (§16).
type CardHandler struct {
	svc *service.CardService
}

func NewCardHandler(svc *service.CardService) *CardHandler {
	return &CardHandler{svc: svc}
}

// CreateVirtualCard issues a new virtual card (§16.1).
// Uses the authenticated user's ID as the owner (security: cannot create cards for others).
func (h *CardHandler) CreateVirtualCard(c *gin.Context) {
	var req card.CreateCardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Force owner to be the authenticated user
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	req.OwnerID = userID
	req.OwnerType = "CUSTOMER"

	resp, err := h.svc.CreateVirtualCard(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// GetCard returns card details.
func (h *CardHandler) GetCard(c *gin.Context) {
	cardID := c.Param("card_id")

	cd, err := h.svc.GetCard(cardID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "card not found"})
		return
	}

	c.JSON(http.StatusOK, cd)
}

// ListCards returns all cards for the authenticated user.
func (h *CardHandler) ListCards(c *gin.Context) {
	ownerID, _ := c.Get("user_id")
	cards, err := h.svc.ListCards(ownerID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"cards": cards})
}

// SpendQuote returns a fee estimate before payment (§16.2).
func (h *CardHandler) SpendQuote(c *gin.Context) {
	cardID := c.Param("card_id")
	var req card.SpendQuoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	quote, err := h.svc.SpendQuote(cardID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, quote)
}

// AuthorizeCard handles card authorization (§10.2, §16.3).
func (h *CardHandler) AuthorizeCard(c *gin.Context) {
	var req struct {
		CardToken           string `json:"card_token" binding:"required"`
		NetworkAuthID       string `json:"network_auth_id"`
		TransactionAmount   int64  `json:"transaction_amount" binding:"required"`
		TransactionCurrency string `json:"transaction_currency" binding:"required"`
		MerchantName        string `json:"merchant_name"`
		MerchantCountry     string `json:"merchant_country"`
		MCC                 string `json:"merchant_category_code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	auth, err := h.svc.AuthorizeCardPayment(req.CardToken, req.NetworkAuthID, card.SpendQuoteRequest{
		TransactionAmount:   req.TransactionAmount,
		TransactionCurrency: req.TransactionCurrency,
		MerchantCountry:     req.MerchantCountry,
		MCC:                 req.MCC,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, auth)
}

// FreezeCard freezes a card.
func (h *CardHandler) FreezeCard(c *gin.Context) {
	if err := h.svc.FreezeCard(c.Param("card_id")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "FROZEN"})
}

// UnfreezeCard unfreezes a card.
func (h *CardHandler) UnfreezeCard(c *gin.Context) {
	if err := h.svc.UnfreezeCard(c.Param("card_id")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ACTIVE"})
}

// ApplyForCard handles KYC-based card application for regular users.
func (h *CardHandler) ApplyForCard(c *gin.Context) {
	userID := c.GetString("user_id")
	var req user.CardApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.svc.ApplyForCard(userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, resp)
}

// CancelCard cancels a card (owner only).
func (h *CardHandler) CancelCard(c *gin.Context) {
	userID := c.GetString("user_id")
	if err := h.svc.CancelCard(userID, c.Param("card_id")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "CANCELLED"})
}

// GetCardTransactions returns the transaction history for a card.
func (h *CardHandler) GetCardTransactions(c *gin.Context) {
	cardID := c.Param("card_id")
	page := 1
	pageSize := 20
	txs, total, err := h.svc.GetCardTransactions(cardID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"transactions": txs, "total": total})
}
