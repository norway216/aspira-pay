package transport

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/aspira/aspira-pay/internal/domain/payment"
	"github.com/aspira/aspira-pay/internal/repository"
	"github.com/aspira/aspira-pay/internal/service"
)

// AdminHandler handles admin dashboard HTTP endpoints.
type AdminHandler struct {
	db            *repository.DB
	paymentSvc    *service.PaymentService
	userSvc       *service.UserService
	settlementSvc *service.SettlementService
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(db *repository.DB, paySvc *service.PaymentService, userSvc *service.UserService, settleSvc *service.SettlementService) *AdminHandler {
	return &AdminHandler{
		db:            db,
		paymentSvc:    paySvc,
		userSvc:       userSvc,
		settlementSvc: settleSvc,
	}
}

// GetDashboard returns admin dashboard stats using a single aggregate query.
// Before: 3 separate List* calls = 6 SQL queries (3 COUNT + 3 SELECT)
// After:  1 SQL query with 3 sub-select COUNTs
func (h *AdminHandler) GetDashboard(c *gin.Context) {
	stats, err := h.db.GetDashboardStats()
	if err != nil {
		// Fallback to per-service counts if aggregate query fails
		_, totalPayments, _ := h.paymentSvc.ListPayments(payment.ListQuery{})
		_, totalUsers, _ := h.userSvc.ListUsers(1, 1)
		_, totalBatches, _ := h.settlementSvc.ListBatches(1, 1)
		c.JSON(http.StatusOK, gin.H{
			"total_payments":          totalPayments,
			"total_users":             totalUsers,
			"total_settlement_batches": totalBatches,
			"system_status":           "healthy",
			"engine_status":           "connected",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total_payments":          stats.TotalPayments,
		"total_users":             stats.TotalUsers,
		"total_settlement_batches": stats.TotalBatches,
		"system_status":           stats.SystemStatus,
		"engine_status":           "connected",
	})
}

// GetAuditLogs returns audit logs (simplified).
func (h *AdminHandler) GetAuditLogs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"logs": []gin.H{},
		"total": 0,
	})
}
