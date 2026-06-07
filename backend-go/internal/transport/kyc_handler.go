package transport

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/aspira/aspira-pay/internal/domain/kyc"
	"github.com/aspira/aspira-pay/internal/service"
)

// KYCHandler handles KYC-related HTTP endpoints.
type KYCHandler struct {
	svc *service.KYCService
}

// NewKYCHandler creates a new KYCHandler.
func NewKYCHandler(svc *service.KYCService) *KYCHandler {
	return &KYCHandler{svc: svc}
}

// SubmitKYC handles KYC profile submission.
func (h *KYCHandler) SubmitKYC(c *gin.Context) {
	userID := c.GetString("user_id")

	var req kyc.SubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	profile, err := h.svc.SubmitKYC(userID, req)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, profile)
}

// GetStatus returns the current user's KYC status.
func (h *KYCHandler) GetStatus(c *gin.Context) {
	userID := c.GetString("user_id")

	profile, err := h.svc.GetKYCStatus(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "KYC profile not found"})
		return
	}

	c.JSON(http.StatusOK, profile)
}

// ReviewKYC handles KYC review decisions (admin).
func (h *KYCHandler) ReviewKYC(c *gin.Context) {
	reviewerID := c.GetString("user_id")

	var req kyc.ReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.ReviewKYC(reviewerID, req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "reviewed"})
}

// ListPending returns KYC profiles awaiting review.
func (h *KYCHandler) ListPending(c *gin.Context) {
	page := 1
	pageSize := 20
	profiles, total, err := h.svc.ListPendingReviews(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"profiles": profiles, "total": total})
}
