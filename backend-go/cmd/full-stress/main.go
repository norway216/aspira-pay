// Aspira Pay V2 — Full Stress Test: 500 accounts, multi-currency, cards, payments
//
// Usage:
//   go run cmd/full-stress/main.go -accounts=500 -duration=120 -workers=50
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
	"sync"
	"sync/atomic"
	"time"
)

var (
	targetURL   = flag.String("target", "http://localhost:8080", "API base URL")
	accountCount = flag.Int("accounts", 500, "Number of accounts")
	durationSec  = flag.Int("duration", 120, "Test duration in seconds")
	workerCount  = flag.Int("workers", 50, "Concurrent workers")
	reportEvery  = flag.Duration("report", 10*time.Second, "Report interval")
	httpTimeout  = flag.Duration("timeout", 30*time.Second, "HTTP timeout")
	adminUser    = flag.String("admin", "admin", "Admin username")
	adminPass    = flag.String("pass", "admin123", "Admin password")
	baseAmount   = flag.Int64("balance", 10_000_000_00, "Base balance in cents ($10M)")
)

const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[0;32m"
	colorYellow = "\033[1;33m"
	colorRed    = "\033[0;31m"
	colorCyan   = "\033[0;36m"
	colorBlue   = "\033[0;34m"
)

// Multi-currency balances per account (§8)
var currencies = []string{"USD", "EUR", "JPY", "GBP", "CHF", "SGD"}
var currencyPairs = [][2]string{
	{"USD", "EUR"}, {"USD", "JPY"}, {"USD", "GBP"}, {"USD", "CHF"}, {"USD", "SGD"},
	{"EUR", "USD"}, {"EUR", "JPY"}, {"EUR", "GBP"}, {"EUR", "CHF"},
	{"GBP", "USD"}, {"GBP", "EUR"}, {"GBP", "JPY"}, {"GBP", "CHF"},
	{"JPY", "USD"}, {"JPY", "EUR"}, {"CHF", "USD"}, {"CHF", "EUR"},
	{"SGD", "USD"}, {"SGD", "EUR"},
}

// FX rates (approximate, for reference only — actual rates from API)
var fxRates = map[string]float64{
	"USD": 1.0, "EUR": 0.92, "JPY": 156.0, "GBP": 0.79, "CHF": 0.90, "SGD": 1.35,
}

type Account struct {
	ID       int
	Username string
	Email    string
	UserID   string
	Token    string
	CardID   string
	CardToken string
}

type Stats struct {
	Total    atomic.Int64
	Success  atomic.Int64
	Failed   atomic.Int64
	Rejected atomic.Int64
	Payments atomic.Int64
	Cards    atomic.Int64
	lats     []time.Duration
	latMu    sync.Mutex
}

func (s *Stats) RecordSuccess(d time.Duration) {
	s.Total.Add(1); s.Success.Add(1); s.Payments.Add(1)
	s.latMu.Lock()
	if len(s.lats) < 100000 { s.lats = append(s.lats, d) }
	s.latMu.Unlock()
}
func (s *Stats) RecordFailed()  { s.Total.Add(1); s.Failed.Add(1) }
func (s *Stats) RecordRejected() { s.Total.Add(1); s.Rejected.Add(1) }
func (s *Stats) P50() time.Duration { return percentile(s.lats, &s.latMu, 50) }
func (s *Stats) P95() time.Duration { return percentile(s.lats, &s.latMu, 95) }
func (s *Stats) P99() time.Duration { return percentile(s.lats, &s.latMu, 99) }
func (s *Stats) Avg() time.Duration {
	s.latMu.Lock(); defer s.latMu.Unlock()
	if len(s.lats) == 0 { return 0 }
	var total time.Duration
	for _, d := range s.lats { total += d }
	return total / time.Duration(len(s.lats))
}

func percentile(lats []time.Duration, mu *sync.Mutex, p float64) time.Duration {
	mu.Lock(); defer mu.Unlock()
	if len(lats) == 0 { return 0 }
	sorted := make([]time.Duration, len(lats))
	copy(sorted, lats)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	idx := int(float64(len(sorted)) * p / 100.0)
	if idx >= len(sorted) { idx = len(sorted) - 1 }
	return sorted[idx]
}

type Client struct {
	client  *http.Client
	baseURL string
	token   string
}

func NewClient() *Client {
	return &Client{
		client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				MaxIdleConns: 500, MaxIdleConnsPerHost: 500, MaxConnsPerHost: 500,
			},
			Timeout: *httpTimeout,
		},
		baseURL: *targetURL,
	}
}

func (c *Client) do(method, path string, body, result interface{}, extraHeaders ...string) (int, time.Duration, error) {
	start := time.Now()
	var bodyReader io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(data)
	}
	req, _ := http.NewRequest(method, c.baseURL+path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" { req.Header.Set("Authorization", "Bearer "+c.token) }
	for i := 0; i+1 < len(extraHeaders); i += 2 { req.Header.Set(extraHeaders[i], extraHeaders[i+1]) }
	resp, err := c.client.Do(req)
	lat := time.Since(start)
	if err != nil { return 0, lat, err }
	defer resp.Body.Close()
	if result != nil && resp.StatusCode < 400 {
		json.NewDecoder(resp.Body).Decode(result)
	}
	return resp.StatusCode, lat, nil
}

func (c *Client) Register(user, email, pass string) (string, error) {
	var r map[string]interface{}
	status, _, err := c.do("POST", "/api/v2/auth/register", map[string]string{
		"username": user, "email": email, "password": pass,
		"full_name": fmt.Sprintf("Stress User %s", user),
		"nationality": "US", "date_of_birth": "1990-01-01",
	}, &r)
	if err != nil || status >= 400 { return "", fmt.Errorf("register %d", status) }
	return r["user_id"].(string), nil
}

func (c *Client) Login(user, pass string) (string, error) {
	var r map[string]interface{}
	status, _, err := c.do("POST", "/api/v2/auth/login", map[string]string{
		"username": user, "password": pass,
	}, &r)
	if err != nil || status >= 400 { return "", fmt.Errorf("login %d", status) }
	return r["token"].(string), nil
}

func (c *Client) CreateCard() (string, string, error) {
	var r map[string]interface{}
	status, _, err := c.do("POST", "/api/v2/cards/virtual", map[string]string{
		"card_network": "VISA", "default_currency": "USD",
	}, &r)
	if err != nil || status >= 400 { return "", "", fmt.Errorf("card %d", status) }
	return r["card_id"].(string), r["card_token"].(string), nil
}

func (c *Client) MakePayment(senderID, receiverID, srcCur, tgtCur string, amount int64) (int, time.Duration, error) {
	status, lat, err := c.do("POST", "/api/v2/payments", map[string]interface{}{
		"sender_user_id": senderID, "receiver_user_id": receiverID,
		"source_currency": srcCur, "target_currency": tgtCur,
		"source_amount": amount, "purpose": "stress_test",
		"country_from": "US", "country_to": "JP",
	}, nil, "Idempotency-Key", fmt.Sprintf("stress_%d_%d", time.Now().UnixNano(), rand.Int63()))
	return status, lat, err
}

func displayReport(stats *Stats, startTime time.Time, phase string, accounts, cards int) {
	elapsed := time.Since(startTime).Seconds()
	total := stats.Total.Load()
	success := stats.Success.Load()
	failed := stats.Failed.Load()
	rejected := stats.Rejected.Load()
	tps := float64(total) / elapsed
	rate := 0.0
	if total > 0 { rate = float64(success) / float64(total) * 100 }

	fmt.Printf("\n%s╔══════════════════════════════════════════════════════════════╗%s\n", colorCyan, colorReset)
	fmt.Printf("%s║  Full Stress Test  |  %s  |  %6.1fs  ║%s\n", colorCyan, phase, elapsed, colorReset)
	fmt.Printf("%s╠══════════════════════════════════════════════════════════════╣%s\n", colorCyan, colorReset)
	fmt.Printf("  Accounts: %d  |  Cards: %d  |  Workers: %d\n", accounts, cards, *workerCount)
	fmt.Printf("  Tx: %d total  |  OK: %d  |  Fail: %d  |  Reject: %d  |  Rate: %s%.1f%%%s\n",
		total, success, failed, rejected, colorGreen, rate, colorReset)
	fmt.Printf("  TPS: %s%.0f%s  |  P50: %v  |  P95: %v  |  P99: %v  |  Avg: %v\n",
		colorYellow, tps, colorReset, stats.P50(), stats.P95(), stats.P99(), stats.Avg())
	if failed > 0 || rejected > 0 {
		fmt.Printf("  %sErrors: %.1f%%  Rejected: %.1f%%%s\n", colorRed,
			float64(failed)/float64(total)*100, float64(rejected)/float64(total)*100, colorReset)
	}
	fmt.Printf("%s╚══════════════════════════════════════════════════════════════╝%s\n", colorCyan, colorReset)
}

func seedBalance(userID string, currency string, amount int64) {
	exec.Command("docker", "exec", "aspira-pay-postgres", "psql", "-U", "aspirapay", "-d", "aspirapay", "-c",
		fmt.Sprintf(`INSERT INTO accounts (account_id, user_id, currency, available_balance, status)
			VALUES ('acc_%s_%s', '%s', '%s', %d, 'NORMAL')
			ON CONFLICT (user_id, currency) DO UPDATE SET available_balance = %d`,
			userID[:8], currency, userID, currency, amount, amount)).Run()
}

func main() {
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	stats := &Stats{}
	client := NewClient()

	fmt.Printf("%s╔══════════════════════════════════════════════════════════╗%s\n", colorCyan, colorReset)
	fmt.Printf("%s║  Aspira Pay V2 — 500-Account Full Stress Test           ║%s\n", colorCyan, colorReset)
	fmt.Printf("%s╚══════════════════════════════════════════════════════════╝%s\n", colorCyan, colorReset)
	fmt.Printf("  Accounts: %d  |  Workers: %d  |  Duration: %ds\n", *accountCount, *workerCount, *durationSec)
	fmt.Printf("  Balance: $%.0fM per currency per account\n", float64(*baseAmount)/100_000_000)
	fmt.Printf("  Currencies: %v\n\n", currencies)

	// ── Phase 1: Admin Auth ──────────────────────────
	fmt.Printf("%s═══ Phase 1: Admin Auth ═══%s\n", colorBlue, colorReset)
	_, err := client.Login(*adminUser, *adminPass)
	if err != nil {
		fmt.Printf("  %sAdmin login failed: %v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}
	fmt.Println("  ✓ Admin authenticated")

	// ── Phase 2: Create Accounts ─────────────────────
	fmt.Printf("\n%s═══ Phase 2: Creating %d Accounts ═══%s\n", colorBlue, *accountCount, colorReset)
	startTime := time.Now()

	accounts := make([]Account, *accountCount)
	var wg sync.WaitGroup
	sem := make(chan struct{}, 30)
	var created atomic.Int64

	for i := 0; i < *accountCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			username := fmt.Sprintf("stress5_%d", idx)
			email := fmt.Sprintf("stress5_%d@test.aspira.io", idx)

			// Register
			client.token = ""
			userID, err := client.Register(username, email, "stress123")
			if err != nil {
				// Try login if already exists
				tok, err2 := client.Login(username, "stress123")
				if err2 == nil {
					client.token = tok
					var r map[string]interface{}
					client.do("GET", "/api/v2/users/me", nil, &r)
					userID, _ = r["user_id"].(string)
				}
			}
			if userID == "" { return }

			// Login
			tok, _ := client.Login(username, "stress123")
			client.token = tok

			// Seed multi-currency balances
			for _, cur := range currencies {
				amt := *baseAmount
				if cur == "JPY" {
					amt = int64(float64(*baseAmount) * fxRates["JPY"])
				} else if cur != "USD" {
					amt = int64(float64(*baseAmount) / fxRates[cur])
				}
				seedBalance(userID, cur, amt)
			}

			// Create virtual card
			cardID, cardToken, _ := client.CreateCard()
			stats.Cards.Add(1)
			created.Add(1)

			accounts[idx] = Account{
				ID: idx, Username: username, Email: email,
				UserID: userID, Token: tok, CardID: cardID, CardToken: cardToken,
			}

			if idx%100 == 0 && idx > 0 {
				fmt.Printf("  Created %d/%d accounts...\n", idx, *accountCount)
			}
		}(i)
	}
	wg.Wait()

	// Filter valid accounts
	var valid []Account
	for _, a := range accounts {
		if a.UserID != "" { valid = append(valid, a) }
	}
	fmt.Printf("  ✓ %d valid accounts created (%.1fs)\n", len(valid), time.Since(startTime).Seconds())

	if len(valid) < 2 {
		fmt.Printf("  %sNot enough accounts to start trading%s\n", colorRed, colorReset)
		os.Exit(1)
	}

	// ── Phase 3: Trading ────────────────────────────
	fmt.Printf("\n%s═══ Phase 3: Trading ═══%s\n", colorBlue, colorReset)
	tradeStart := time.Now()
	stopCh := make(chan struct{})

	// Start workers
	for w := 0; w < *workerCount; w++ {
		go func(workerID int) {
			rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(workerID)))
			for {
				select {
				case <-stopCh: return
				default:
				}
				sender := valid[rng.Intn(len(valid))]
				receiver := valid[rng.Intn(len(valid))]
				for receiver.UserID == sender.UserID && len(valid) > 1 {
					receiver = valid[rng.Intn(len(valid))]
				}
				pair := currencyPairs[rng.Intn(len(currencyPairs))]
				amount := int64(1000 + rng.Intn(499000)) // $10 ~ $4,990

				client.token = sender.Token
				status, lat, err := client.MakePayment(sender.UserID, receiver.UserID, pair[0], pair[1], amount)
				if err != nil {
					stats.RecordFailed()
				} else if status >= 400 {
					stats.RecordRejected()
				} else {
					stats.RecordSuccess(lat)
				}
			}
		}(w)
	}

	// Report loop
	reportDone := make(chan struct{})
	go func() {
		ticker := time.NewTicker(*reportEvery)
		defer ticker.Stop()
		for {
			select {
			case <-stopCh: close(reportDone); return
			case <-ticker.C:
				displayReport(stats, tradeStart, "RUNNING", len(valid), int(stats.Cards.Load()))
			}
		}
	}()

	// Duration timer
	if *durationSec > 0 {
		time.Sleep(time.Duration(*durationSec) * time.Second)
	}
	close(stopCh)
	<-reportDone

	// ── Final Report ────────────────────────────────
	displayReport(stats, tradeStart, "FINAL", len(valid), int(stats.Cards.Load()))

	total := stats.Total.Load()
	success := stats.Success.Load()
	elapsed := time.Since(tradeStart).Seconds()
	fmt.Printf("\n%s╔══════════════════════════════════════════════════════════╗%s\n", colorGreen, colorReset)
	fmt.Printf("%s║  TEST COMPLETE                                         ║%s\n", colorGreen, colorReset)
	fmt.Printf("%s╠══════════════════════════════════════════════════════════╣%s\n", colorGreen, colorReset)
	fmt.Printf("  Accounts: %d  |  Cards: %d  |  Duration: %.0fs\n", len(valid), stats.Cards.Load(), elapsed)
	fmt.Printf("  Total Tx: %d  |  OK: %d (%.1f%%)  |  TPS: %.0f\n",
		total, success, float64(success)/float64(total)*100, float64(total)/elapsed)
	fmt.Printf("  P50: %v  |  P95: %v  |  P99: %v  |  Avg: %v\n",
		stats.P50(), stats.P95(), stats.P99(), stats.Avg())
	fmt.Printf("%s╚══════════════════════════════════════════════════════════╝%s\n", colorGreen, colorReset)
}
