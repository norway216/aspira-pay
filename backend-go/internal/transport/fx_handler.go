package transport

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/aspira/aspira-pay/internal/domain/fx"
	"github.com/aspira/aspira-pay/internal/service"
)

// FXHandler handles FX quote HTTP endpoints.
type FXHandler struct {
	svc *service.FXService
}

// NewFXHandler creates a new FXHandler.
func NewFXHandler(svc *service.FXService) *FXHandler {
	return &FXHandler{svc: svc}
}

// GetQuote generates a new FX quote.
func (h *FXHandler) GetQuote(c *gin.Context) {
	var req fx.QuoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	quote, err := h.svc.GetQuote(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, quote)
}

// GetQuoteByID retrieves a quote by ID.
func (h *FXHandler) GetQuoteByID(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "quote retrieval by ID not implemented"})
}

// ListRates returns all available FX rates.
func (h *FXHandler) ListRates(c *gin.Context) {
	rates := h.svc.ListRates()
	c.JSON(http.StatusOK, gin.H{"rates": rates})
}
