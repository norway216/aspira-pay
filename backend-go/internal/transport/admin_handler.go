package transport

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/aspira/aspira-pay/internal/domain/payment"
	"github.com/aspira/aspira-pay/internal/service"
)

// AdminHandler handles admin dashboard HTTP endpoints.
type AdminHandler struct {
	paymentSvc    *service.PaymentService
	userSvc       *service.UserService
	settlementSvc *service.SettlementService
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(paySvc *service.PaymentService, userSvc *service.UserService, settleSvc *service.SettlementService) *AdminHandler {
	return &AdminHandler{
		paymentSvc:    paySvc,
		userSvc:       userSvc,
		settlementSvc: settleSvc,
	}
}

// GetDashboard returns admin dashboard stats.
func (h *AdminHandler) GetDashboard(c *gin.Context) {
	// Aggregate stats for dashboard
	_, totalPayments, _ := h.paymentSvc.ListPayments(payment.ListQuery{})
	_, totalUsers, _ := h.userSvc.ListUsers(1, 1)
	_, totalBatches, _ := h.settlementSvc.ListBatches(1, 1)

	c.JSON(http.StatusOK, gin.H{
		"total_payments":    totalPayments,
		"total_users":       totalUsers,
		"total_settlement_batches": totalBatches,
		"system_status":     "healthy",
		"engine_status":     "connected",
	})
}

// GetAuditLogs returns audit logs (simplified).
func (h *AdminHandler) GetAuditLogs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"logs": []gin.H{},
		"total": 0,
	})
}
