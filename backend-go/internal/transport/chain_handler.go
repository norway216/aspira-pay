package transport

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/aspira/aspira-pay/internal/service"
)

// ChainHandler handles blockchain audit HTTP endpoints.
// Architecture doc §15: Audit verification API.
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

// VerifyAudit performs full Merkle proof verification for a payment.
// Architecture doc §15.1: GET /api/v2/audit/payments/{payment_id}
func (h *ChainHandler) VerifyAudit(c *gin.Context) {
	paymentID := c.Param("payment_id")

	result, err := h.svc.VerifyPaymentAudit(paymentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetBatch returns a specific chain batch.
func (h *ChainHandler) GetBatch(c *gin.Context) {
	batchID := c.Param("batch_id")

	batch, err := h.svc.GetBatch(batchID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "batch not found"})
		return
	}

	c.JSON(http.StatusOK, batch)
}
