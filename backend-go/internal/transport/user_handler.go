package transport

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/aspira/aspira-pay/internal/domain/user"
	"github.com/aspira/aspira-pay/internal/service"
)

// UserHandler handles user-related HTTP endpoints.
type UserHandler struct {
	svc *service.UserService
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

// Register handles user registration.
func (h *UserHandler) Register(c *gin.Context) {
	var req user.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	u, err := h.svc.Register(req)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"user_id":    u.UserID,
		"username":   u.Username,
		"email":      u.Email,
		"status":     u.Status,
		"risk_level": u.RiskLevel,
	})
}

// Login handles user login.
func (h *UserHandler) Login(c *gin.Context) {
	var req user.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.svc.Login(req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Refresh handles token refresh.
func (h *UserHandler) Refresh(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "refresh not implemented in Sandbox"})
}

// GetMe returns the current user's profile.
func (h *UserHandler) GetMe(c *gin.Context) {
	userID := c.GetString("user_id")
	u, err := h.svc.GetUser(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, u)
}

// GetUser returns a user by ID.
func (h *UserHandler) GetUser(c *gin.Context) {
	userID := c.Param("id")
	u, err := h.svc.GetUser(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, u)
}

// ListUsers returns a paginated user list.
func (h *UserHandler) ListUsers(c *gin.Context) {
	page := 1
	pageSize := 20
	users, total, err := h.svc.ListUsers(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"users": users, "total": total, "page": page, "page_size": pageSize})
}

// UpdateStatus updates a user's status.
func (h *UserHandler) UpdateStatus(c *gin.Context) {
	userID := c.Param("id")
	var req struct {
		Status user.UserStatus `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.UpdateStatus(userID, req.Status); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}
