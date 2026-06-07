package transport

import (
	"github.com/aspira/aspira-pay/internal/service"
)

// RiskHandler handles risk/AML-related HTTP endpoints.
// Risk checks are embedded in the payment flow; separate endpoints are for admin review.
type RiskHandler struct {
	svc *service.RiskService
}

// NewRiskHandler creates a new RiskHandler.
func NewRiskHandler(svc *service.RiskService) *RiskHandler {
	return &RiskHandler{svc: svc}
}
