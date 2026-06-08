// Aspira Pay V2 — Main Entry Point
// Cross-Border Payment Clearing & Transaction System
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aspira/aspira-pay/internal/config"
	"github.com/aspira/aspira-pay/internal/engine"
	"github.com/aspira/aspira-pay/internal/messaging"
	"github.com/aspira/aspira-pay/internal/observability"
	"github.com/aspira/aspira-pay/internal/repository"
	"github.com/aspira/aspira-pay/internal/service"
	"github.com/aspira/aspira-pay/internal/transport"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("═══════════════════════════════════════════════════════")
	log.Println("  Aspira Pay V2 — Cross-Border Payment System")
	log.Println("  Version: 2.0.0-sandbox")
	log.Println("═══════════════════════════════════════════════════════")

	// Load configuration
	cfgPath := "configs/config.yaml"
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Printf("Warning: cannot load config file %s: %v — using defaults", cfgPath, err)
		cfg = config.LoadEnv()
	}

	// Initialize logger
	logger := observability.NewLogger(cfg.Server.Mode)
	defer logger.Sync()

	log.Println("Configuration loaded")

	// Initialize database
	db, err := repository.New(cfg.Database)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer db.Close()
	log.Printf("PostgreSQL connected: %s:%d/%s", cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)

	// Initialize Redis (optional for Sandbox — no-op if unavailable)
	// redisClient := redis.NewClient(&redis.Options{Addr: cfg.Redis.Addr})

	// Initialize NATS messaging
	natsClient := messaging.NewClient(cfg.NATS)
	if err := natsClient.Connect(); err != nil {
		log.Printf("NATS: %v — continuing with local processing", err)
	}
	defer natsClient.Close()

	// Initialize JWT manager
	tokenExpirySeconds := int64(cfg.Auth.TokenExpiry.Seconds())
	refreshExpirySeconds := int64(cfg.Auth.RefreshExpiry.Seconds())
	jwtMgr := service.NewJWTManager(cfg.Auth.JWTSecret, tokenExpirySeconds, refreshExpirySeconds)

	// Initialize engine client
	engineClient := engine.NewClient(cfg.Engine)
	if err := engineClient.Connect(); err != nil {
		log.Printf("Engine: %v — using local fallback", err)
	}
	defer engineClient.Close()

	// Initialize services
	kycSvc := service.NewKYCService(db)
	riskSvc := service.NewRiskService(db)
	fxSvc := service.NewFXService(cfg.FX.APIURL)
	go fxSvc.RefreshLoop(cfg.FX.RefreshInterval)
	settlementSvc := service.NewSettlementService(db)
	chainSvc := service.NewChainService(db)
	userSvc := service.NewUserService(db, jwtMgr)

	paymentSvc := service.NewPaymentService(db, kycSvc, riskSvc, fxSvc, settlementSvc, chainSvc)

	// Card Payment Subsystem (§5.1)
	cardSvc := service.NewCardService(db, fxSvc)

	log.Println("All services initialized")

	// Setup router
	router := transport.SetupRouter(&transport.RouterConfig{
		DB:            db,
		UserSvc:       userSvc,
		KYCSvc:        kycSvc,
		RiskSvc:       riskSvc,
		FXSvc:         fxSvc,
		PaymentSvc:    paymentSvc,
		SettlementSvc: settlementSvc,
		ChainSvc:      chainSvc,
		CardSvc:       cardSvc,
		JWT:           jwtMgr,
	})

	// Setup HTTP server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start server in goroutine
	go func() {
		log.Printf("╔══════════════════════════════════════════════════════╗")
		log.Printf("║  API Server:  http://localhost%s                ║", addr)
		log.Printf("║  Health:      http://localhost%s/health         ║", addr)
		log.Printf("║  Metrics:     http://localhost%s/metrics        ║", addr)
		log.Printf("║  API Base:    http://localhost%s/api/v2         ║", addr)
		log.Printf("╚══════════════════════════════════════════════════════╝")

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}
