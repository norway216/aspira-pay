package transport

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/aspira/aspira-pay/internal/service"
)

// SettlementHandler handles ledger and settlement HTTP endpoints.
type SettlementHandler struct {
	svc *service.SettlementService
}

// NewSettlementHandler creates a new SettlementHandler.
func NewSettlementHandler(svc *service.SettlementService) *SettlementHandler {
	return &SettlementHandler{svc: svc}
}

// GetLedger returns all ledger entries for a payment.
func (h *SettlementHandler) GetLedger(c *gin.Context) {
	paymentID := c.Param("payment_id")

	summary, err := h.svc.GetLedgerForPayment(paymentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, summary)
}

// ListBatches returns paginated settlement batches.
func (h *SettlementHandler) ListBatches(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	batches, total, err := h.svc.ListBatches(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"batches": batches, "total": total})
}

// GetBatch retrieves a specific settlement batch.
func (h *SettlementHandler) GetBatch(c *gin.Context) {
	batchID := c.Param("id")

	batch, err := h.svc.GetBatch(batchID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "batch not found"})
		return
	}

	c.JSON(http.StatusOK, batch)
}
