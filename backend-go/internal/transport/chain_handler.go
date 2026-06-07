package transport

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/aspira/aspira-pay/internal/service"
)

// ChainHandler handles blockchain audit HTTP endpoints.
type ChainHandler struct {
	svc *service.ChainService
}

// NewChainHandler creates a new ChainHandler.
func NewChainHandler(svc *service.ChainService) *ChainHandler {
	return &ChainHandler{svc: svc}
}

// ListBlocks returns paginated chain blocks.
func (h *ChainHandler) ListBlocks(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	blocks, total, err := h.svc.ListBlocks(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"blocks": blocks, "total": total})
}

// GetBlock retrieves a specific chain block.
func (h *ChainHandler) GetBlock(c *gin.Context) {
	height, err := strconv.ParseInt(c.Param("height"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid block height"})
		return
	}

	block, err := h.svc.GetBlock(height)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "block not found"})
		return
	}

	c.JSON(http.StatusOK, block)
}

// GetAuditTrail returns the complete chain audit trail for a payment.
func (h *ChainHandler) GetAuditTrail(c *gin.Context) {
	paymentID := c.Param("payment_id")

	trail, err := h.svc.GetAuditTrail(paymentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, trail)
}
