package transport

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/aspira/aspira-pay/internal/domain/payment"
	"github.com/aspira/aspira-pay/internal/service"
)

// PaymentHandler handles payment-related HTTP endpoints.
type PaymentHandler struct {
	svc *service.PaymentService
}

// NewPaymentHandler creates a new PaymentHandler.
func NewPaymentHandler(svc *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{svc: svc}
}

// CreatePayment handles payment order creation.
func (h *PaymentHandler) CreatePayment(c *gin.Context) {
	var req payment.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.svc.CreatePayment(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// GetPayment retrieves a payment by ID.
func (h *PaymentHandler) GetPayment(c *gin.Context) {
	paymentID := c.Param("id")

	order, err := h.svc.GetPayment(paymentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "payment not found"})
		return
	}

	c.JSON(http.StatusOK, order)
}

// ListPayments returns a filtered list of payments.
func (h *PaymentHandler) ListPayments(c *gin.Context) {
	var q payment.ListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		q.Page = 1
		q.PageSize = 20
	}
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 || q.PageSize > 100 {
		q.PageSize = 20
	}

	orders, total, err := h.svc.ListPayments(q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"orders": orders, "total": total, "page": q.Page, "page_size": q.PageSize})
}

// RefundPayment processes a refund.
func (h *PaymentHandler) RefundPayment(c *gin.Context) {
	paymentID := c.Param("id")

	if err := h.svc.RefundPayment(paymentID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "refunded"})
}
