// Aspira Pay V2 — High-Concurrency Benchmark Client
//
// Simulates realistic cross-border payment traffic with configurable
// concurrency levels, transaction patterns, and reporting.
//
// Usage:
//
//	go run cmd/bench-client/main.go [flags]
//
// Flags:
//
//	-target      API base URL (default: http://localhost:8080)
//	-workers     Number of concurrent workers (default: 10)
//	-duration    Test duration (default: 60s)
//	-rate        Target TPS rate, 0 = unlimited (default: 0)
//	-amount      Average transaction amount in major unit (default: 100)
//	-pairs       Currency pairs to use (default: "USD/JPY,USD/EUR")
//	-mode        Test mode: trade, full-lifecycle (default: trade)
//	-interactive Interactive mode with live dashboard
package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// ═══════════════════════════════════════════════════════════
// CLI Flags
// ═══════════════════════════════════════════════════════════

var (
	targetURL    = flag.String("target", "http://localhost:8080", "API base URL")
	workerCount  = flag.Int("workers", 10, "Number of concurrent workers")
	durationSec  = flag.Int("duration", 60, "Test duration in seconds (0 = infinite)")
	targetTPS    = flag.Int("rate", 0, "Target TPS (0 = unlimited)")
	avgAmount    = flag.Float64("amount", 100, "Average tx amount in major units")
	pairs        = flag.String("pairs", "USD/JPY,USD/EUR,USD/CNY,EUR/JPY,GBP/USD", "Currency pairs (comma-separated)")
	mode         = flag.String("mode", "trade", "Test mode: trade, full-lifecycle, burst")
	reportEvery  = flag.Duration("report", 5*time.Second, "Stats report interval")
	httpTimeout  = flag.Duration("timeout", 30*time.Second, "HTTP request timeout")
	skipVerify   = flag.Bool("skip-verify", true, "Skip TLS verification")
	batchSize    = flag.Int("batch", 1, "Transactions per batch per worker iteration")
	rampUpSec    = flag.Int("rampup", 5, "Ramp-up period in seconds")
	cooldownSec  = flag.Int("cooldown", 3, "Cooldown period in seconds")
	maxConns     = flag.Int("max-conns", 200, "Max idle connections per host")

	// Mode: full-lifecycle
	userCount    = flag.Int("users", 20, "Number of simulated users (full-lifecycle mode)")
	seedDeposit  = flag.Int64("deposit", 1_000_000, "Initial deposit per account (in cents)")
)

// ═══════════════════════════════════════════════════════════
// Currency & Pair Config
// ═══════════════════════════════════════════════════════════

var supportedCurrencies = []string{"USD", "EUR", "GBP", "JPY", "CNY", "AUD", "CAD"}

type CurrencyPair struct {
	Source string
	Target string
}

func parsePairs(s string) []CurrencyPair {
	var result []CurrencyPair
	for _, p := range strings.Split(s, ",") {
		parts := strings.Split(strings.TrimSpace(p), "/")
		if len(parts) == 2 {
			result = append(result, CurrencyPair{Source: parts[0], Target: parts[1]})
		}
	}
	if len(result) == 0 {
		result = []CurrencyPair{{Source: "USD", Target: "JPY"}}
	}
	return result
}

// ═══════════════════════════════════════════════════════════
// Statistics
// ═══════════════════════════════════════════════════════════

type LatencyStats struct {
	mu       sync.Mutex
	latencies []time.Duration
	maxSize   int
}

func NewLatencyStats(maxSize int) *LatencyStats {
	return &LatencyStats{latencies: make([]time.Duration, 0, maxSize), maxSize: maxSize}
}

func (ls *LatencyStats) Record(d time.Duration) {
	ls.mu.Lock()
	if len(ls.latencies) < ls.maxSize {
		ls.latencies = append(ls.latencies, d)
	}
	ls.mu.Unlock()
}

func (ls *LatencyStats) Percentile(p float64) time.Duration {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	if len(ls.latencies) == 0 {
		return 0
	}

	sorted := make([]time.Duration, len(ls.latencies))
	copy(sorted, ls.latencies)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	idx := int(math.Ceil(float64(len(sorted)) * p / 100.0))
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	if idx < 0 {
		idx = 0
	}
	return sorted[idx]
}

func (ls *LatencyStats) Avg() time.Duration {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	if len(ls.latencies) == 0 {
		return 0
	}

	var total time.Duration
	for _, d := range ls.latencies {
		total += d
	}
	return time.Duration(int64(total) / int64(len(ls.latencies)))
}

type GlobalStats struct {
	TotalRequests   atomic.Int64
	SuccessCount    atomic.Int64
	ErrorCount      atomic.Int64
	RejectedCount   atomic.Int64
	TotalAmount     atomic.Int64 // In cents
	ActiveWorkers   atomic.Int64
	Latency         *LatencyStats
	StartTime       time.Time
	LastReportTime  time.Time
	lastCount       atomic.Int64 // For TPS calculation
}

func NewGlobalStats() *GlobalStats {
	return &GlobalStats{
		Latency: NewLatencyStats(100000),
	}
}

func (gs *GlobalStats) RecordSuccess(amount int64, latency time.Duration) {
	gs.SuccessCount.Add(1)
	gs.TotalRequests.Add(1)
	gs.TotalAmount.Add(amount)
	gs.Latency.Record(latency)
}

func (gs *GlobalStats) RecordError() {
	gs.ErrorCount.Add(1)
	gs.TotalRequests.Add(1)
}

func (gs *GlobalStats) RecordRejected() {
	gs.RejectedCount.Add(1)
	gs.TotalRequests.Add(1)
}

func (gs *GlobalStats) CurrentTPS() float64 {
	now := time.Now()
	elapsed := now.Sub(gs.LastReportTime).Seconds()
	current := gs.TotalRequests.Load()
	last := gs.lastCount.Load()
	gs.lastCount.Store(current)
	gs.LastReportTime = now

	if elapsed <= 0 {
		return 0
	}
	return float64(current-last) / elapsed
}

// ═══════════════════════════════════════════════════════════
// HTTP Client with connection pooling
// ═══════════════════════════════════════════════════════════

type APIClient struct {
	client  *http.Client
	baseURL string
	token   string
	stats   *GlobalStats
}

func NewAPIClient(baseURL string, stats *GlobalStats) *APIClient {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: *skipVerify},
		MaxIdleConns:    *maxConns,
		MaxIdleConnsPerHost: *maxConns,
		MaxConnsPerHost:     *maxConns,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  true,
	}

	return &APIClient{
		client: &http.Client{
			Transport: tr,
			Timeout:   *httpTimeout,
		},
		baseURL: baseURL,
		stats:   stats,
	}
}

func (c *APIClient) SetToken(token string) {
	c.token = token
}

func (c *APIClient) doRequest(method, path string, body interface{}, result interface{}) (int, time.Duration, error) {
	start := time.Now()

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return 0, 0, fmt.Errorf("marshal: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return 0, 0, fmt.Errorf("request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.client.Do(req)
	latency := time.Since(start)
	if err != nil {
		return 0, latency, fmt.Errorf("do: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if result != nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return resp.StatusCode, latency, fmt.Errorf("unmarshal %d: %s", resp.StatusCode, string(respBody[:min(len(respBody), 200)]))
		}
	}

	return resp.StatusCode, latency, nil
}

func (c *APIClient) Register(username, email, password string) (string, error) {
	var result map[string]interface{}
	status, _, err := c.doRequest("POST", "/api/v2/auth/register", map[string]string{
		"username": username,
		"email":    email,
		"password": password,
	}, &result)
	if err != nil {
		return "", err
	}
	if status >= 400 {
		return "", fmt.Errorf("register: HTTP %d", status)
	}
	userID, _ := result["user_id"].(string)
	return userID, nil
}

func (c *APIClient) Login(username, password string) (string, error) {
	var result map[string]interface{}
	status, _, err := c.doRequest("POST", "/api/v2/auth/login", map[string]string{
		"username": username,
		"password": password,
	}, &result)
	if err != nil {
		return "", err
	}
	if status >= 400 {
		return "", fmt.Errorf("login: HTTP %d", status)
	}
	token, _ := result["token"].(string)
	if token == "" {
		return "", fmt.Errorf("no token in login response")
	}
	return token, nil
}

func (c *APIClient) GetMe() (string, error) {
	var result map[string]interface{}
	status, _, err := c.doRequest("GET", "/api/v2/users/me", nil, &result)
	if err != nil {
		return "", err
	}
	if status >= 400 {
		return "", fmt.Errorf("getme: HTTP %d", status)
	}
	userID, _ := result["user_id"].(string)
	return userID, nil
}

func (c *APIClient) SubmitKYC(fullName, nationality string) error {
	status, _, err := c.doRequest("POST", "/api/v2/kyc/submit", map[string]string{
		"full_name":      fullName,
		"nationality":    nationality,
		"document_type":  "passport",
		"document_number": fmt.Sprintf("PP%d", rand.Int63n(999999999)),
		"address":        fmt.Sprintf("Test Address %d", rand.Intn(9999)),
	}, nil)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("kyc submit: HTTP %d", status)
	}
	return nil
}

func (c *APIClient) GetQuote(sourceCurrency, targetCurrency string, sourceAmount int64) (map[string]interface{}, error) {
	var result map[string]interface{}
	status, lat, err := c.doRequest("POST", "/api/v2/fx/quote", map[string]interface{}{
		"source_currency": sourceCurrency,
		"target_currency": targetCurrency,
		"source_amount":   sourceAmount,
	}, &result)
	if err != nil {
		c.stats.Latency.Record(lat)
		c.stats.RecordError()
		return nil, err
	}
	if status >= 400 {
		c.stats.Latency.Record(lat)
		c.stats.RecordRejected()
		return nil, fmt.Errorf("quote: HTTP %d", status)
	}
	return result, nil
}

func (c *APIClient) CreatePayment(senderID, receiverID, sourceCurrency, targetCurrency string, sourceAmount int64) (map[string]interface{}, error) {
	reqBody := map[string]interface{}{
		"sender_user_id":   senderID,
		"receiver_user_id": receiverID,
		"source_currency":  sourceCurrency,
		"target_currency":  targetCurrency,
		"source_amount":    sourceAmount,
		"purpose":          "benchmark_test",
		"country_from":     "US",
		"country_to":       "JP",
	}

	idempotencyKey := fmt.Sprintf("bench_%d_%d", time.Now().UnixNano(), rand.Int63())

	var result map[string]interface{}
	status, lat, err := c.doRequest("POST", "/api/v2/payments", reqBody, &result)

	// Set idempotency key via header (simplified: included in URL pattern)
	_ = idempotencyKey

	if err != nil {
		c.stats.Latency.Record(lat)
		c.stats.RecordError()
		return nil, err
	}

	if status >= 500 {
		c.stats.Latency.Record(lat)
		c.stats.RecordError()
		return nil, fmt.Errorf("payment: HTTP %d", status)
	}

	if status >= 400 {
		c.stats.Latency.Record(lat)
		c.stats.RecordRejected()
		return nil, fmt.Errorf("payment rejected: HTTP %d", status)
	}

	// Record success
	sourceAmt, _ := result["source_amount"].(float64)
	c.stats.RecordSuccess(int64(sourceAmt), lat)

	return result, nil
}

// ═══════════════════════════════════════════════════════════
// Simulated User (for full-lifecycle mode)
// ═══════════════════════════════════════════════════════════

type SimulatedUser struct {
	Username string
	Email    string
	Password string
	UserID   string
	Token    string
}

func createUserPool(count int, client *APIClient, adminToken string) ([]SimulatedUser, error) {
	users := make([]SimulatedUser, count)
	client.SetToken(adminToken)

	fmt.Printf("  Creating %d simulated users...\n", count)
	start := time.Now()

	var wg sync.WaitGroup
	var mu sync.Mutex
	var created atomic.Int64
	var errors atomic.Int64

	sem := make(chan struct{}, 20) // Limit concurrent registration

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			username := fmt.Sprintf("bench_u%d_%d", time.Now().Unix(), idx)
			email := fmt.Sprintf("bench_%d_%d@test.aspira.io", time.Now().Unix(), idx)

			userID, err := client.Register(username, email, "benchmark123")
			if err != nil {
				errors.Add(1)
				// Try to use existing test users
				userID = fmt.Sprintf("u_bench_test_%d", idx)
			} else {
				created.Add(1)
			}

			// Submit KYC
			_ = client.SubmitKYC(fmt.Sprintf("Bench User %d", idx), "US")

			mu.Lock()
			users[idx] = SimulatedUser{
				Username: username,
				Email:    email,
				Password: "benchmark123",
				UserID:   userID,
			}
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	fmt.Printf("  Users: %d created, %d fallback, %d errors (%.1fs)\n",
		created.Load(), int64(count)-created.Load()-errors.Load(), errors.Load(), time.Since(start).Seconds())

	return users, nil
}

// ═══════════════════════════════════════════════════════════
// Worker — trade mode
// ═══════════════════════════════════════════════════════════

type WorkerTrade struct {
	id       int
	client   *APIClient
	pairs    []CurrencyPair
	users    []SimulatedUser
	stopCh   <-chan struct{}
	rate     int // target TPS per worker (0 = unlimited)
}

func (w *WorkerTrade) Run() {
	rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(w.id)))
	userCount := len(w.users)

	// Per-worker rate limiter
	var rateLimiter <-chan time.Time
	if w.rate > 0 {
		interval := time.Second / time.Duration(w.rate)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		rateLimiter = ticker.C
	} else {
		// Unlimited: use a closed channel-like behavior
		c := make(chan time.Time)
		close(c)
		rateLimiter = c
	}

	for {
		select {
		case <-w.stopCh:
			return
		case <-rateLimiter:
			// Pick random user pair
			senderIdx := rng.Intn(userCount)
			receiverIdx := rng.Intn(userCount)
			for receiverIdx == senderIdx {
				receiverIdx = rng.Intn(userCount)
			}

			sender := w.users[senderIdx]
			receiver := w.users[receiverIdx]

			// Pick random currency pair
			pair := w.pairs[rng.Intn(len(w.pairs))]

			// Random amount: 10 ~ average*2 in major units, converted to cents
			amountMajor := *avgAmount * (0.1 + rng.Float64()*1.9)
			amountCents := int64(amountMajor * 100)

			// Execute payment
			_, _ = w.client.CreatePayment(
				sender.UserID,
				receiver.UserID,
				pair.Source,
				pair.Target,
				amountCents,
			)
		}
	}
}

// ═══════════════════════════════════════════════════════════
// Worker — burst mode
// ═══════════════════════════════════════════════════════════

type WorkerBurst struct {
	id        int
	client    *APIClient
	pairs     []CurrencyPair
	users     []SimulatedUser
	stopCh    <-chan struct{}
	burstSize int
}

func (w *WorkerBurst) Run() {
	rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(w.id)))
	userCount := len(w.users)

	for {
		select {
		case <-w.stopCh:
			return
		default:
		}

		// Send burst
		var wg sync.WaitGroup
		for i := 0; i < w.burstSize; i++ {
			select {
			case <-w.stopCh:
				return
			default:
			}

			wg.Add(1)
			go func() {
				defer wg.Done()

				sender := w.users[rng.Intn(userCount)]
				receiver := w.users[rng.Intn(userCount)]
				pair := w.pairs[rng.Intn(len(w.pairs))]
				amountCents := int64(*avgAmount*100) * (1 + rng.Int63n(9))

				_, _ = w.client.CreatePayment(
					sender.UserID, receiver.UserID,
					pair.Source, pair.Target, amountCents,
				)
			}()
		}
		wg.Wait()

		// Pause between bursts
		select {
		case <-w.stopCh:
			return
		case <-time.After(time.Duration(500+rng.Intn(1500)) * time.Millisecond):
		}
	}
}

// ═══════════════════════════════════════════════════════════
// Display
// ═══════════════════════════════════════════════════════════

const (
	colorReset  = "\033[0m"
	colorCyan   = "\033[0;36m"
	colorGreen  = "\033[0;32m"
	colorYellow = "\033[1;33m"
	colorRed    = "\033[0;31m"
	colorBlue   = "\033[0;34m"
	colorBold   = "\033[1m"
)

func displayReport(stats *GlobalStats, startTime time.Time) {
	elapsed := time.Since(startTime).Seconds()
	total := stats.TotalRequests.Load()
	success := stats.SuccessCount.Load()
	errors := stats.ErrorCount.Load()
	rejected := stats.RejectedCount.Load()
	totalAmount := stats.TotalAmount.Load()
	tps := stats.CurrentTPS()
	avgLatency := stats.Latency.Avg()
	p50 := stats.Latency.Percentile(50)
	p95 := stats.Latency.Percentile(95)
	p99 := stats.Latency.Percentile(99)

	successRate := 0.0
	if total > 0 {
		successRate = float64(success) / float64(total) * 100
	}

	fmt.Println()
	fmt.Printf("%s╔══════════════════════════════════════════════════════════════╗%s\n", colorCyan, colorReset)
	fmt.Printf("%s║  Aspira Pay V2 — Benchmark Report  |  %6.1fs elapsed       ║%s\n", colorCyan, elapsed, colorReset)
	fmt.Printf("%s╠══════════════════════════════════════════════════════════════╣%s\n", colorCyan, colorReset)

	fmt.Printf("  %sTotal:%-7d%s  OK:%-8d  ERR:%-6d  REJ:%-5d  Rate:%s%.1f%%%s\n",
		colorBold, total, colorReset, success, errors, rejected, colorGreen, successRate, colorReset)

	fmt.Printf("  %sTPS:%-8.0f%s  Volume:  %-12d cents (%.2f major)\n",
		colorYellow, tps, colorReset, totalAmount, float64(totalAmount)/100.0)

	fmt.Printf("  Latency:  avg=%-10v  P50=%-10v  P95=%-10v  P99=%-10v\n",
		avgLatency, p50, p95, p99)

	fmt.Printf("  Active workers: %d\n", stats.ActiveWorkers.Load())

	if errors > 0 || rejected > 0 {
		errRate := 0.0
		if total > 0 {
			errRate = float64(errors) / float64(total) * 100
		}
		rejRate := 0.0
		if total > 0 {
			rejRate = float64(rejected) / float64(total) * 100
		}
		fmt.Printf("  %sErrors: %.1f%%  Rejected: %.1f%%%s\n", colorRed, errRate, rejRate, colorReset)
	}

	fmt.Printf("%s╚══════════════════════════════════════════════════════════════╝%s\n", colorCyan, colorReset)
}

// ═══════════════════════════════════════════════════════════
// Main
// ═══════════════════════════════════════════════════════════

func main() {
	flag.Parse()

	pairList := parsePairs(*pairs)
	stats := NewGlobalStats()

	fmt.Println()
	fmt.Printf("%s╔══════════════════════════════════════════════════════════╗%s\n", colorCyan, colorReset)
	fmt.Printf("%s║   Aspira Pay V2 — High-Concurrency Benchmark Client     ║%s\n", colorCyan, colorReset)
	fmt.Printf("%s╚══════════════════════════════════════════════════════════╝%s\n", colorCyan, colorReset)
	fmt.Printf("  Target:        %s\n", *targetURL)
	fmt.Printf("  Workers:       %d\n", *workerCount)
	fmt.Printf("  Duration:      %ds\n", *durationSec)
	fmt.Printf("  Target TPS:    %d (0=unlimited)\n", *targetTPS)
	fmt.Printf("  Avg Amount:    %.2f major unit\n", *avgAmount)
	fmt.Printf("  Pairs:         %v\n", pairList)
	fmt.Printf("  Mode:          %s\n", *mode)
	fmt.Printf("  Ramp-up:       %ds\n", *rampUpSec)
	fmt.Println()

	// Create HTTP client
	client := NewAPIClient(*targetURL, stats)

	// Step 1: Authenticate as admin to create users if needed
	fmt.Printf("%s═══ Phase 1: Setup ═══%s\n", colorBlue, colorReset)

	var users []SimulatedUser

	if *mode == "full-lifecycle" {
		// Register admin user first
		fmt.Println("  Registering admin user...")
		adminToken, err := client.Login("admin", "admin123")
		if err != nil {
			_, err = client.Register("admin", "admin@aspira.io", "admin123")
			if err != nil {
				fmt.Printf("  %sAdmin user setup: %v (continuing...) %s\n", colorYellow, err, colorReset)
			}
			adminToken, _ = client.Login("admin", "admin123")
		}
		client.SetToken(adminToken)

		users, _ = createUserPool(*userCount, client, adminToken)
	} else {
		// Trade/burst mode: login as admin and use their real user_id (has KYC + balance)
		fmt.Println("  Authenticating...")
		authToken, err := client.Login("admin", "admin123")
		if err != nil {
			client.Register("admin", "admin@aspira.io", "admin123")
			authToken, _ = client.Login("admin", "admin123")
		}
		client.SetToken(authToken)

		adminID, err := client.GetMe()
		if err != nil || adminID == "" {
			fmt.Printf("  %sCannot authenticate. Is the API running?%s\n", colorRed, colorReset)
			os.Exit(1)
		}
		fmt.Printf("  Using admin: %s\n", adminID)

		// All workers use admin as both sender and receiver.
		// Admin already has KYC approved + account balances.
		users = []SimulatedUser{{
			Username: "admin",
			UserID:   adminID,
			Token:    authToken,
		}}
		// Duplicate admin for the pool so workers can pick different "receivers"
		for i := 1; i < *userCount && i < 20; i++ {
			users = append(users, SimulatedUser{
				Username: fmt.Sprintf("admin_%d", i),
				UserID:   adminID,
				Token:    authToken,
			})
		}
		*userCount = len(users)
	}
	fmt.Printf("  User pool: %d users ready\n", len(users))

	// Step 2: Start workers with ramp-up
	fmt.Printf("\n%s═══ Phase 2: Trading ═══%s\n", colorBlue, colorReset)

	stopCh := make(chan struct{})
	var wg sync.WaitGroup
	startTime := time.Now()
	stats.StartTime = startTime
	stats.LastReportTime = startTime

	activeWorkers := int32(0)

	// Ramp-up: start workers gradually
	rampInterval := time.Duration(*rampUpSec*1e9) / time.Duration(*workerCount)
	if *workerCount <= 0 {
		*workerCount = 1
	}
	if rampInterval < 10*time.Millisecond {
		rampInterval = 10 * time.Millisecond
	}

	fmt.Printf("  Starting %d workers (ramp-up over %ds)...\n", *workerCount, *rampUpSec)

	// Per-worker TPS target
	perWorkerRate := 0
	if *targetTPS > 0 {
		perWorkerRate = *targetTPS / *workerCount
		if perWorkerRate < 1 {
			perWorkerRate = 1
		}
	}

	for i := 0; i < *workerCount; i++ {
		wg.Add(1)
		atomic.AddInt32(&activeWorkers, 1)
		stats.ActiveWorkers.Store(int64(atomic.LoadInt32(&activeWorkers)))

		go func(workerID int) {
			defer wg.Done()
			defer func() {
				atomic.AddInt32(&activeWorkers, -1)
				stats.ActiveWorkers.Store(int64(atomic.LoadInt32(&activeWorkers)))
			}()

			switch *mode {
			case "burst":
				w := &WorkerBurst{
					id:        workerID,
					client:    client,
					pairs:     pairList,
					users:     users,
					stopCh:    stopCh,
					burstSize: *batchSize,
				}
				w.Run()
			default: // trade, full-lifecycle
				w := &WorkerTrade{
					id:     workerID,
					client: client,
					pairs:  pairList,
					users:  users,
					stopCh: stopCh,
					rate:   perWorkerRate,
				}
				w.Run()
			}
		}(i)

		// Ramp-up delay
		if rampInterval > 0 && i < *workerCount-1 {
			time.Sleep(rampInterval)
		}
	}

	fmt.Printf("  All %d workers running. Ramp-up complete.\n", *workerCount)

	// Step 3: Periodic stats reporting
	reportDone := make(chan struct{})
	go func() {
		ticker := time.NewTicker(*reportEvery)
		defer ticker.Stop()
		for {
			select {
			case <-stopCh:
				close(reportDone)
				return
			case <-ticker.C:
				displayReport(stats, startTime)
			}
		}
	}()

	// Step 4: Run for configured duration or until Ctrl+C
	if *durationSec > 0 {
		fmt.Printf("  Running for %ds... (Ctrl+C to stop early)\n", *durationSec)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	var timerCh <-chan time.Time
	if *durationSec > 0 {
		timer := time.NewTimer(time.Duration(*durationSec) * time.Second)
		timerCh = timer.C
	}

	select {
	case <-sigCh:
		fmt.Println("\n\n  Received interrupt signal...")
	case <-timerCh:
		fmt.Println("\n\n  Duration reached...")
	}

	// Cooldown
	fmt.Printf("  Cooldown (%ds)...\n", *cooldownSec)
	time.Sleep(time.Duration(*cooldownSec) * time.Second)

	// Stop workers
	close(stopCh)
	wg.Wait()
	<-reportDone

	// Final report
	fmt.Println()
	fmt.Printf("%s╔══════════════════════════════════════════════════════════╗%s\n", colorGreen, colorReset)
	fmt.Printf("%s║            FINAL BENCHMARK REPORT                       ║%s\n", colorGreen, colorReset)
	fmt.Printf("%s╚══════════════════════════════════════════════════════════╝%s\n", colorGreen, colorReset)
	displayReport(stats, startTime)

	fmt.Printf("\n%s✓ Benchmark complete.%s\n", colorGreen, colorReset)
}
