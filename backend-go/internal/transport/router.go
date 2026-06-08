// Package transport provides HTTP handlers and router configuration.
package transport

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/aspira/aspira-pay/internal/middleware"
	"github.com/aspira/aspira-pay/internal/repository"
	"github.com/aspira/aspira-pay/internal/service"
)

// RouterConfig holds all dependencies for the HTTP router.
type RouterConfig struct {
	DB            *repository.DB
	UserSvc       *service.UserService
	KYCSvc        *service.KYCService
	RiskSvc       *service.RiskService
	FXSvc         *service.FXService
	PaymentSvc    *service.PaymentService
	SettlementSvc *service.SettlementService
	ChainSvc      *service.ChainService
	CardSvc       *service.CardService
	AdminSvc      *service.AdminService
	JWT           *service.JWTManager
}

// SetupRouter creates and configures the Gin router with all routes.
func SetupRouter(cfg *RouterConfig) *gin.Engine {
	r := gin.Default()

	// Global middleware
	r.Use(middleware.CORS())
	r.Use(middleware.AuditLog())
	r.Use(middleware.Recovery())
	r.Use(middleware.RateLimit(100000, 60)) // High limit for Sandbox benchmarking

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "aspira-pay"})
	})

	// Prometheus metrics
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API v2 routes
	api := r.Group("/api/v2")
	{
		// Public routes (no auth required)
		auth := api.Group("/auth")
		{
			auth.POST("/register", NewUserHandler(cfg.UserSvc).Register)
			auth.POST("/login", NewUserHandler(cfg.UserSvc).Login)
			auth.POST("/refresh", NewUserHandler(cfg.UserSvc).Refresh)
		}

		// Protected routes (require JWT)
		protected := api.Group("")
		protected.Use(middleware.AuthRequired(cfg.JWT))
		{
			// Users
			userH := NewUserHandler(cfg.UserSvc)
			protected.GET("/users/me", userH.GetMe)
			protected.GET("/users/:id", userH.GetUser)
			protected.GET("/users", userH.ListUsers)
			protected.PUT("/users/:id/status", userH.UpdateStatus)
			protected.POST("/users/me/pin", userH.SetPIN)

			// KYC
			kycH := NewKYCHandler(cfg.KYCSvc)
			protected.POST("/kyc/submit", kycH.SubmitKYC)
			protected.GET("/kyc/status", kycH.GetStatus)
			protected.PUT("/kyc/review", kycH.ReviewKYC)
			protected.GET("/kyc/pending", kycH.ListPending)

			// Payments (with idempotency)
			payH := NewPaymentHandler(cfg.PaymentSvc)
			protected.POST("/payments", middleware.IdempotencyMiddleware(), payH.CreatePayment)
			protected.GET("/payments", payH.ListPayments)
			protected.GET("/payments/:id", payH.GetPayment)
			protected.POST("/payments/:id/refund", payH.RefundPayment)

			// FX Quotes
			fxH := NewFXHandler(cfg.FXSvc)
			protected.POST("/fx/quote", fxH.GetQuote)
			protected.GET("/fx/quotes/:id", fxH.GetQuoteByID)
			protected.GET("/fx/rates", fxH.ListRates)

			// Ledger & Settlement
			settleH := NewSettlementHandler(cfg.SettlementSvc)
			protected.GET("/ledger/:payment_id", settleH.GetLedger)
			protected.GET("/settlement/batches", settleH.ListBatches)
			protected.GET("/settlement/batches/:id", settleH.GetBatch)

			// Blockchain Audit
			chainH := NewChainHandler(cfg.ChainSvc)
			protected.GET("/chain/blocks", chainH.ListBlocks)
			protected.GET("/chain/blocks/:height", chainH.GetBlock)
			protected.GET("/chain/audit/:payment_id", chainH.GetAuditTrail)
			// §15: Merkle proof verification API
			protected.GET("/chain/verify/:payment_id", chainH.VerifyAudit)
			protected.GET("/chain/batches/:batch_id", chainH.GetBatch)

			// Accounts (with FX conversion to user preferred currency)
			acctH := NewAccountHandler(cfg.DB, cfg.FXSvc)
			protected.GET("/accounts", acctH.GetMyAccounts)

			// Card Payment Subsystem
			if cfg.CardSvc != nil {
				cardH := NewCardHandler(cfg.CardSvc)
				protected.POST("/cards/virtual", cardH.CreateVirtualCard)
			protected.POST("/cards/apply", cardH.ApplyForCard)
			protected.POST("/cards/:card_id/cancel", cardH.CancelCard)
				protected.GET("/cards", cardH.ListCards)
				protected.GET("/cards/:card_id", cardH.GetCard)
				protected.POST("/cards/:card_id/freeze", cardH.FreezeCard)
				protected.POST("/cards/:card_id/unfreeze", cardH.UnfreezeCard)
				protected.POST("/cards/:card_id/quote-spend", cardH.SpendQuote)
				protected.GET("/cards/:card_id/transactions", cardH.GetCardTransactions)
				protected.POST("/internal/card-authorizations", cardH.AuthorizeCard)
			}

			// Admin Dashboard
			adminH := NewAdminHandler(cfg.DB, cfg.PaymentSvc, cfg.UserSvc, cfg.SettlementSvc)
			protected.GET("/admin/dashboard", adminH.GetDashboard)
			protected.GET("/admin/audit-logs", adminH.GetAuditLogs)
			// V2 Admin: audit logs, card review (§14)
			if cfg.AdminSvc != nil {
				adminV2 := NewAdminV2Handler(cfg.AdminSvc)
				protected.GET("/admin/v2/audit-logs", adminV2.GetAuditLogs)
				protected.GET("/admin/v2/pending-cards", adminV2.ListPendingCardApps)
				protected.POST("/admin/v2/review-card", adminV2.ReviewCardApplication)
			}
		}
	}

	return r
}
