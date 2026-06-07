// Aspira Pay V2 — Progressive Stress Test
//
// Starts with 100 trading accounts and gradually scales to 1000
// based on real-time success rate and system health.
//
// Usage:
//
//	go run cmd/stress-test/main.go [flags]
//
// Flags:
//
//	-target       API base URL (default: http://localhost:8080)
//	-initial      初始账户数 (default: 100)
//	-max          最大账户数 (default: 1000)
//	-batch        每批新增账户数 (default: 50)
//	-scale-rate   成功率阈值，高于此值才扩容 (default: 0.90)
//	-scale-interval 扩容检查间隔 (default: 30s)
//	-duration     总运行时间，0=无限 (default: 0)
//	-rate         每个账户每秒交易数 (default: 1)
package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ═══════════════════════════════════════════════════════════
// CLI Flags
// ═══════════════════════════════════════════════════════════

var (
	targetURL      = flag.String("target", "http://localhost:8080", "API base URL")
	initialAccts   = flag.Int("initial", 100, "Initial number of accounts")
	maxAccts       = flag.Int("max", 1000, "Maximum number of accounts")
	batchSize      = flag.Int("batch", 50, "Accounts per scale-up batch")
	scaleRate      = flag.Float64("scale-rate", 0.90, "Success rate threshold to scale up")
	scaleInterval  = flag.Duration("scale-interval", 30*time.Second, "Interval between scale checks")
	durationSec    = flag.Int("duration", 0, "Total duration in seconds (0=infinite)")
	txPerAccount   = flag.Int("rate", 1, "Target transactions per second per account")
	reportInterval = flag.Duration("report", 10*time.Second, "Stats report interval")
	httpTimeout    = flag.Duration("timeout", 30*time.Second, "HTTP timeout")
	adminUser      = flag.String("admin", "admin", "Admin username")
	adminPass      = flag.String("pass", "admin123", "Admin password")
)

const (
	colorReset  = "\033[0m"
	colorBold   = "\033[1m"
	colorRed    = "\033[0;31m"
	colorGreen  = "\033[0;32m"
	colorYellow = "\033[1;33m"
	colorCyan   = "\033[0;36m"
	colorBlue   = "\033[0;34m"
	colorMagenta = "\033[0;35m"
)

// ═══════════════════════════════════════════════════════════
// Currency pairs for cross-border trading
// ═══════════════════════════════════════════════════════════

var currencyPairs = [][2]string{
	{"USD", "JPY"}, {"USD", "EUR"}, {"USD", "CNY"},
	{"EUR", "JPY"}, {"EUR", "USD"}, {"GBP", "USD"},
	{"GBP", "JPY"}, {"CNY", "USD"}, {"JPY", "USD"},
}

// ═══════════════════════════════════════════════════════════
// Trading Account
// ═══════════════════════════════════════════════════════════

type TradingAccount struct {
	ID       int
	Username string
	Email    string
	Password string
	UserID   string
	Token    string
	Currency string // primary currency
}

// ═══════════════════════════════════════════════════════════
// API Client
// ═══════════════════════════════════════════════════════════

type APIClient struct {
	client  *http.Client
	baseURL string
	token   string
}

func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
				MaxIdleConns:        500,
				MaxIdleConnsPerHost: 500,
				MaxConnsPerHost:     500,
			},
			Timeout: *httpTimeout,
		},
		baseURL: baseURL,
	}
}

func (c *APIClient) do(method, path string, body, result interface{}, extraHeaders ...string) (int, error) {
	var bodyReader io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(data)
	}
	req, _ := http.NewRequest(method, c.baseURL+path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	// Apply extra headers: key1, value1, key2, value2, ...
	for i := 0; i+1 < len(extraHeaders); i += 2 {
		req.Header.Set(extraHeaders[i], extraHeaders[i+1])
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if result != nil && resp.StatusCode < 400 {
		json.NewDecoder(resp.Body).Decode(result)
	}
	return resp.StatusCode, nil
}

func (c *APIClient) Login(user, pass string) (string, error) {
	var r map[string]interface{}
	status, err := c.do("POST", "/api/v2/auth/login", map[string]string{
		"username": user, "password": pass,
	}, &r)
	if err != nil || status >= 400 {
		return "", fmt.Errorf("login failed: %d", status)
	}
	t, _ := r["token"].(string)
	return t, nil
}

func (c *APIClient) Register(user, email, pass string) (string, error) {
	var r map[string]interface{}
	status, err := c.do("POST", "/api/v2/auth/register", map[string]string{
		"username": user, "email": email, "password": pass,
	}, &r)
	if err != nil || status >= 400 {
		return "", fmt.Errorf("register HTTP %d", status)
	}
	uid, _ := r["user_id"].(string)
	return uid, nil
}

func (c *APIClient) GetMe() (string, error) {
	var r map[string]interface{}
	_, err := c.do("GET", "/api/v2/users/me", nil, &r)
	if err != nil {
		return "", err
	}
	uid, _ := r["user_id"].(string)
	return uid, nil
}

func (c *APIClient) SubmitKYC(name, nationality string) error {
	status, err := c.do("POST", "/api/v2/kyc/submit", map[string]string{
		"full_name":       name,
		"nationality":     nationality,
		"document_type":   "passport",
		"document_number": fmt.Sprintf("PP%09d", rand.Intn(999999999)),
		"address":         fmt.Sprintf("Test Address %d", rand.Intn(9999)),
	}, nil)
	if err != nil || status >= 400 {
		return fmt.Errorf("kyc HTTP %d", status)
	}
	return nil
}

func (c *APIClient) ApproveKYC(userID string) error {
	status, err := c.do("PUT", "/api/v2/kyc/review", map[string]string{
		"user_id":    userID,
		"action":     "APPROVED",
		"risk_level": "LOW",
	}, nil)
	return firstErr(err, status >= 400, fmt.Errorf("approve HTTP %d", status))
}

func (c *APIClient) CreatePayment(senderID, receiverID, srcCurrency, tgtCurrency string, amount int64) (int, error) {
	// Generate unique idempotency key per request
	// Architecture doc §9.1: Every request must include idempotency key
	idempotencyKey := fmt.Sprintf("stress_%d_%d", time.Now().UnixNano(), rand.Int63())
	status, err := c.do("POST", "/api/v2/payments", map[string]interface{}{
		"sender_user_id":   senderID,
		"receiver_user_id": receiverID,
		"source_currency":  srcCurrency,
		"target_currency":  tgtCurrency,
		"source_amount":    amount,
		"purpose":          "stress_test",
		"country_from":     "US",
		"country_to":       "JP",
	}, nil,
		"Idempotency-Key", idempotencyKey)
	return status, err
}

func firstErr(errs ...interface{}) error {
	for _, e := range errs {
		switch v := e.(type) {
		case error:
			if v != nil {
				return v
			}
		case bool:
			if v {
				return errs[len(errs)-1].(error)
			}
		}
	}
	return nil
}

// ═══════════════════════════════════════════════════════════
// Account Factory — creates and provisions accounts
// ═══════════════════════════════════════════════════════════

type AccountFactory struct {
	client     *APIClient
	adminToken string
	dbPrefix   string
}

func NewAccountFactory(client *APIClient, adminToken string) *AccountFactory {
	return &AccountFactory{client: client, adminToken: adminToken}
}

// CreateBatch creates a batch of trading accounts with KYC + balances.
func (f *AccountFactory) CreateBatch(startID, count int) ([]TradingAccount, error) {
	accounts := make([]TradingAccount, count)
	var wg sync.WaitGroup
	var mu sync.Mutex
	sem := make(chan struct{}, 10)
	created := int32(0)
	failed := int32(0)

	fmt.Printf("  Creating %d accounts (ID %d-%d)...\n", count, startID, startID+count-1)

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			id := startID + idx
			username := fmt.Sprintf("stress_%d", id)
			email := fmt.Sprintf("stress_%d@test.aspira.io", id)
			password := "stress123"
			currency := currencyPairs[id%len(currencyPairs)][0]

			// Step 1: Register
			userID, err := f.client.Register(username, email, password)
			if err != nil {
				// Try to get existing user
				tok, _ := f.client.Login(username, password)
				if tok != "" {
					f.client.token = tok
					userID, _ = f.client.GetMe()
				}
			}
			if userID == "" {
				atomic.AddInt32(&failed, 1)
				return
			}

			// Step 2: Login + KYC
			tok, err := f.client.Login(username, password)
			if err != nil {
				atomic.AddInt32(&failed, 1)
				return
			}
			f.client.token = tok
			f.client.SubmitKYC(fmt.Sprintf("Stress User %d", id), "US")

			atomic.AddInt32(&created, 1)
			mu.Lock()
			accounts[idx] = TradingAccount{
				ID: id, Username: username, Email: email,
				Password: password, UserID: userID, Token: tok, Currency: currency,
			}
			mu.Unlock()
		}(i)
	}
	wg.Wait()

	fmt.Printf("  Registered: %d created, %d failed\n", created, failed)

	// Step 3: Approve all KYCs as admin
	f.client.token = f.adminToken
	fmt.Printf("  Approving KYC...\n")
	for i := range accounts {
		if accounts[i].UserID != "" {
			f.client.ApproveKYC(accounts[i].UserID)
		}
	}

	// Step 4: Seed balances via direct SQL
	f.seedBalances(accounts)

	return accounts, nil
}

func (f *AccountFactory) seedBalances(accounts []TradingAccount) {
	for _, a := range accounts {
		if a.UserID == "" {
			continue
		}
		// Best-effort seed for primary and secondary currencies
		for _, cur := range []string{"USD", "EUR", "JPY", "CNY", "GBP"} {
			sql := fmt.Sprintf(
				`INSERT INTO accounts (account_id, user_id, currency, available_balance, status)
				 VALUES ('acc_%s_%s', '%s', '%s', 50000000, 'NORMAL')
				 ON CONFLICT (account_id) DO UPDATE SET available_balance = 50000000`,
				a.UserID[:8], strings.ToLower(cur), a.UserID, cur,
			)
			exec.Command("docker", "exec", "aspira-pay-postgres",
				"psql", "-U", "aspirapay", "-d", "aspirapay",
				"-c", sql).Run()
		}
	}
}

// ═══════════════════════════════════════════════════════════
// Health Monitor — tracks success rate and decides when to scale
// ═══════════════════════════════════════════════════════════

type HealthMonitor struct {
	mu           sync.Mutex
	window       []bool // true=success, false=failure
	windowSize   int
	pos          int
	totalSuccess int64
	totalFailed  int64
}

func NewHealthMonitor(windowSize int) *HealthMonitor {
	return &HealthMonitor{
		window:     make([]bool, windowSize),
		windowSize: windowSize,
	}
}

func (h *HealthMonitor) Record(success bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if success {
		atomic.AddInt64(&h.totalSuccess, 1)
	} else {
		atomic.AddInt64(&h.totalFailed, 1)
	}
	h.window[h.pos%h.windowSize] = success
	h.pos++
}

func (h *HealthMonitor) RecentRate() float64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.pos == 0 {
		return 1.0
	}
	success := 0
	samples := h.pos
	if samples > h.windowSize {
		samples = h.windowSize
	}
	start := h.pos - samples
	if start < 0 {
		start = 0
	}
	for i := start; i < h.pos; i++ {
		if h.window[i%h.windowSize] {
			success++
		}
	}
	if samples == 0 {
		return 1.0
	}
	return float64(success) / float64(samples)
}

func (h *HealthMonitor) TotalSuccess() int64 { return h.totalSuccess }
func (h *HealthMonitor) TotalFailed() int64  { return h.totalFailed }

// ═══════════════════════════════════════════════════════════
// Global Stats
// ═══════════════════════════════════════════════════════════

type GlobalStats struct {
	TxTotal    atomic.Int64
	TxSuccess  atomic.Int64
	TxFailed   atomic.Int64
	ActiveAccts atomic.Int64
	latencies  []time.Duration
	latMu      sync.Mutex
}

func (s *GlobalStats) RecordSuccess(lat time.Duration) {
	s.TxTotal.Add(1)
	s.TxSuccess.Add(1)
	s.latMu.Lock()
	if len(s.latencies) < 10000 {
		s.latencies = append(s.latencies, lat)
	}
	s.latMu.Unlock()
}

func (s *GlobalStats) RecordFailed() {
	s.TxTotal.Add(1)
	s.TxFailed.Add(1)
}

func (s *GlobalStats) P50() time.Duration { return s.percentile(50) }
func (s *GlobalStats) P95() time.Duration { return s.percentile(95) }
func (s *GlobalStats) P99() time.Duration { return s.percentile(99) }

func (s *GlobalStats) percentile(p float64) time.Duration {
	s.latMu.Lock()
	defer s.latMu.Unlock()
	if len(s.latencies) == 0 {
		return 0
	}
	sorted := make([]time.Duration, len(s.latencies))
	copy(sorted, s.latencies)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	idx := int(float64(len(sorted)) * p / 100.0)
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

// ═══════════════════════════════════════════════════════════
// Worker — one per account, makes periodic trades
// ═══════════════════════════════════════════════════════════

func runWorker(acct TradingAccount, pool []TradingAccount, client *APIClient,
	stats *GlobalStats, health *HealthMonitor, stopCh <-chan struct{}, txRate int) {

	rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(acct.ID)))
	interval := time.Second / time.Duration(txRate)
	if interval < 10*time.Millisecond {
		interval = 10 * time.Millisecond
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			// Pick a random receiver (different from sender)
			receiver := pool[rng.Intn(len(pool))]
			for receiver.UserID == acct.UserID && len(pool) > 1 {
				receiver = pool[rng.Intn(len(pool))]
			}

			// Pick random currency pair
			pair := currencyPairs[rng.Intn(len(currencyPairs))]

			// Random amount: $10 ~ $500
			amount := int64(1000 + rng.Intn(49000))

			start := time.Now()
			status, err := client.CreatePayment(
				acct.UserID, receiver.UserID,
				pair[0], pair[1], amount,
			)
			lat := time.Since(start)

			success := err == nil && status < 400
			if success {
				stats.RecordSuccess(lat)
			} else {
				stats.RecordFailed()
			}
			health.Record(success)
		}
	}
}

// ═══════════════════════════════════════════════════════════
// Display
// ═══════════════════════════════════════════════════════════

func displayReport(stats *GlobalStats, health *HealthMonitor, activeCount, targetCount int, elapsed time.Duration, phase string) {
	total := stats.TxTotal.Load()
	success := stats.TxSuccess.Load()
	failed := stats.TxFailed.Load()
	rate := 0.0
	if total > 0 {
		rate = float64(success) / float64(total) * 100
	}
	tps := float64(total) / elapsed.Seconds()
	healthRate := health.RecentRate() * 100

	statusColor := colorGreen
	if healthRate < 80 {
		statusColor = colorRed
	} else if healthRate < 95 {
		statusColor = colorYellow
	}

	fmt.Printf("%s╔══════════════════════════════════════════════════════════════╗%s\n", colorCyan, colorReset)
	fmt.Printf("%s║  Stress Test Report  |  %s Phase%s  |  %6.1fs elapsed       ║%s\n",
		colorCyan, colorMagenta, phase, elapsed.Seconds(), colorReset)
	fmt.Printf("%s╠══════════════════════════════════════════════════════════════╣%s\n", colorCyan, colorReset)
	fmt.Printf("  %sAccounts:%s  Active=%-5d  Target=%-5d\n", colorBold, colorReset, activeCount, targetCount)
	fmt.Printf("  %sTx:%s       Total=%-8d  OK=%-8d  Fail=%-6d  Rate=%s%.1f%%%s\n",
		colorBold, colorReset, total, success, failed, statusColor, rate, colorReset)
	fmt.Printf("  %sHealth:%s    Recent=%.1f%%  (scale threshold: %.0f%%)\n",
		colorBold, colorReset, healthRate, *scaleRate*100)
	fmt.Printf("  TPS:      %.0f  |  P50=%-10v  P95=%-10v  P99=%-10v\n",
		tps, stats.P50(), stats.P95(), stats.P99())
	fmt.Printf("%s╚══════════════════════════════════════════════════════════════╝%s\n", colorCyan, colorReset)
}

// ═══════════════════════════════════════════════════════════
// Scaler — manages gradual account scaling
// ═══════════════════════════════════════════════════════════

type Scaler struct {
	factory       *AccountFactory
	pool          []TradingAccount
	poolMu        sync.RWMutex
	nextID        int
	targetCount   int
	maxCount      int
	batchSz       int
	health        *HealthMonitor
	stats         *GlobalStats
	scaleThreshold float64
}

func NewScaler(factory *AccountFactory, initial []TradingAccount, max, batch int, health *HealthMonitor, stats *GlobalStats, threshold float64) *Scaler {
	return &Scaler{
		factory:        factory,
		pool:           initial,
		nextID:         len(initial),
		targetCount:    len(initial),
		maxCount:       max,
		batchSz:        batch,
		health:         health,
		stats:          stats,
		scaleThreshold: threshold,
	}
}

func (s *Scaler) ActiveCount() int {
	s.poolMu.RLock()
	defer s.poolMu.RUnlock()
	return len(s.pool)
}

func (s *Scaler) GetPool() []TradingAccount {
	s.poolMu.RLock()
	defer s.poolMu.RUnlock()
	pool := make([]TradingAccount, len(s.pool))
	copy(pool, s.pool)
	return pool
}

func (s *Scaler) ShouldScaleUp() bool {
	return s.ActiveCount() < s.maxCount && s.health.RecentRate() >= s.scaleThreshold
}

func (s *Scaler) ScaleUp() bool {
	if !s.ShouldScaleUp() {
		return false
	}

	s.poolMu.Lock()
	defer s.poolMu.Unlock()

	count := s.batchSz
	if s.nextID+count > s.maxCount {
		count = s.maxCount - s.nextID
	}
	if count <= 0 {
		return false
	}

	fmt.Printf("\n%s═══ Scaling Up: +%d accounts (%d → %d) ═══%s\n",
		colorBlue, count, s.nextID, s.nextID+count, colorReset)

	newAccounts, err := s.factory.CreateBatch(s.nextID, count)
	if err != nil {
		fmt.Printf("  %sScale-up failed: %v%s\n", colorRed, err, colorReset)
		return false
	}

	valid := 0
	for _, a := range newAccounts {
		if a.UserID != "" {
			s.pool = append(s.pool, a)
			s.nextID++
			valid++
		}
	}
	s.targetCount = len(s.pool)
	fmt.Printf("  %sScale-up complete: +%d valid accounts, pool=%d%s\n",
		colorGreen, valid, len(s.pool), colorReset)
	return true
}

// ═══════════════════════════════════════════════════════════
// Main
// ═══════════════════════════════════════════════════════════

func main() {
	flag.Parse()

	fmt.Printf("%s╔══════════════════════════════════════════════════════════╗%s\n", colorCyan, colorReset)
	fmt.Printf("%s║   Aspira Pay V2 — Progressive Stress Test               ║%s\n", colorCyan, colorReset)
	fmt.Printf("%s╚══════════════════════════════════════════════════════════╝%s\n", colorCyan, colorReset)
	fmt.Printf("  Target:       %s\n", *targetURL)
	fmt.Printf("  Accounts:     %d initial → %d max (batch: %d)\n", *initialAccts, *maxAccts, *batchSize)
	fmt.Printf("  Scale-up:     when success rate ≥ %.0f%%\n", *scaleRate*100)
	fmt.Printf("  Check every:  %v\n", *scaleInterval)
	fmt.Printf("  Tx/account/s: %d\n", *txPerAccount)
	fmt.Println()

	// Initialize
	client := NewAPIClient(*targetURL)
	stats := &GlobalStats{}
	health := NewHealthMonitor(1000)

	// Phase 1: Authenticate admin
	fmt.Printf("%s═══ Phase 1: Admin Auth ═══%s\n", colorBlue, colorReset)
	adminToken, err := client.Login(*adminUser, *adminPass)
	if err != nil {
		fmt.Printf("  %sAdmin login failed. Registering...%s\n", colorYellow, colorReset)
		client.Register(*adminUser, "admin@aspira.io", *adminPass)
		adminToken, _ = client.Login(*adminUser, *adminPass)
	}
	client.token = adminToken
	adminID, _ := client.GetMe()
	fmt.Printf("  Admin: %s\n", adminID)

	// Phase 2: Create initial 100 accounts
	fmt.Printf("\n%s═══ Phase 2: Initial Setup (%d accounts) ═══%s\n", colorBlue, *initialAccts, colorReset)
	factory := NewAccountFactory(client, adminToken)
	initialPool, err := factory.CreateBatch(0, *initialAccts)
	if err != nil {
		fmt.Printf("  %sFailed to create initial accounts: %v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}
	validCount := 0
	for _, a := range initialPool {
		if a.UserID != "" {
			validCount++
		}
	}
	fmt.Printf("  %sInitial pool: %d valid accounts ready%s\n", colorGreen, validCount, colorReset)

	// Filter valid accounts
	var validPool []TradingAccount
	for _, a := range initialPool {
		if a.UserID != "" {
			validPool = append(validPool, a)
		}
	}

	// Create scaler
	scaler := NewScaler(factory, validPool, *maxAccts, *batchSize, health, stats, *scaleRate)

	// Phase 3: Start trading
	fmt.Printf("\n%s═══ Phase 3: Trading + Progressive Scale-Up ═══%s\n", colorBlue, colorReset)

	stopCh := make(chan struct{})
	var wg sync.WaitGroup
	startTime := time.Now()

	// Start workers for initial accounts
	pool := scaler.GetPool()
	for i := range pool {
		wg.Add(1)
		stats.ActiveAccts.Add(1)
		go func(acct TradingAccount) {
			defer wg.Done()
			defer stats.ActiveAccts.Add(-1)
			runWorker(acct, pool, client, stats, health, stopCh, *txPerAccount)
		}(pool[i])
	}
	fmt.Printf("  Started %d workers\n", len(pool))

	// Phase 4: Scale-up loop
	scaleTicker := time.NewTicker(*scaleInterval)
	defer scaleTicker.Stop()

	// Report ticker
	reportTicker := time.NewTicker(*reportInterval)
	defer reportTicker.Stop()

	// Duration timer
	var durationTimer <-chan time.Time
	if *durationSec > 0 {
		durationTimer = time.NewTimer(time.Duration(*durationSec) * time.Second).C
	}

	// Main loop: monitor health and scale up
	scalePhase := "WARMUP"
	scaleUpCount := 0

	mainLoop:
	for {
		select {
		case <-stopCh:
			break mainLoop

		case <-durationTimer:
			fmt.Printf("\n  %sDuration reached — completing current cycle...%s\n", colorYellow, colorReset)
			break mainLoop

		case <-scaleTicker.C:
			currentCount := scaler.ActiveCount()
			recentRate := health.RecentRate()

			if recentRate >= *scaleRate && currentCount < *maxAccts {
				scalePhase = "SCALING"
				if scaler.ScaleUp() {
					scaleUpCount++
					// Start workers for new accounts
					pool = scaler.GetPool()
					for i := currentCount; i < len(pool); i++ {
						wg.Add(1)
						stats.ActiveAccts.Add(1)
						go func(acct TradingAccount) {
							defer wg.Done()
							defer stats.ActiveAccts.Add(-1)
							runWorker(acct, pool, client, stats, health, stopCh, *txPerAccount)
						}(pool[i])
					}
				}
			} else if currentCount >= *maxAccts {
				scalePhase = "STEADY"
			} else {
				scalePhase = "STABILIZING"
			}

		case <-reportTicker.C:
			displayReport(stats, health, scaler.ActiveCount(), *maxAccts, time.Since(startTime), scalePhase)
		}
	}

	// Final report
	fmt.Println()
	displayReport(stats, health, scaler.ActiveCount(), *maxAccts, time.Since(startTime), "FINAL")

	// Cleanup
	close(stopCh)
	wg.Wait()

	fmt.Printf("\n%s╔══════════════════════════════════════════════════════════╗%s\n", colorGreen, colorReset)
	fmt.Printf("%s║  Stress Test Complete                                  ║%s\n", colorGreen, colorReset)
	fmt.Printf("%s╠══════════════════════════════════════════════════════════╣%s\n", colorGreen, colorReset)
	fmt.Printf("  Final accounts: %d (%d scale-ups)\n", scaler.ActiveCount(), scaleUpCount)
	fmt.Printf("  Total TX:       %d (OK=%d Fail=%d)\n",
		stats.TxTotal.Load(), stats.TxSuccess.Load(), stats.TxFailed.Load())
	fmt.Printf("%s╚══════════════════════════════════════════════════════════╝%s\n", colorGreen, colorReset)
}

func init() {
	rand.Seed(time.Now().UnixNano())
}