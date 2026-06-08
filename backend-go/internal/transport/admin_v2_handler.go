package transport

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/aspira/aspira-pay/internal/service"
)

type AdminV2Handler struct {
	adminSvc *service.AdminService
}

func NewAdminV2Handler(adminSvc *service.AdminService) *AdminV2Handler {
	return &AdminV2Handler{adminSvc: adminSvc}
}

// GetAuditLogs returns admin operation audit logs (§14.2).
func (h *AdminV2Handler) GetAuditLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	logs, total, err := h.adminSvc.GetAuditLogs("", "", page, pageSize)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"logs": logs, "total": total})
}

// ReviewCardApplication approves/rejects a card application (§7.2).
func (h *AdminV2Handler) ReviewCardApplication(c *gin.Context) {
	adminID := c.GetString("user_id")
	var req struct {
		CardID   string `json:"card_id" binding:"required"`
		Decision string `json:"decision" binding:"required"`
		Notes    string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(400, gin.H{"error": err.Error()}); return }
	if err := h.adminSvc.ReviewCardApplication(adminID, req.CardID, req.Decision, req.Notes, c.ClientIP()); err != nil {
		c.JSON(400, gin.H{"error": err.Error()}); return
	}
	c.JSON(200, gin.H{"status": "reviewed"})
}

// ListPendingCardApps returns cards awaiting admin review.
func (h *AdminV2Handler) ListPendingCardApps(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	apps, total, err := h.adminSvc.PendingApps(page, pageSize)
	if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
	c.JSON(200, gin.H{"applications": apps, "total": total})
}
