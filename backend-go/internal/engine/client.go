// Package engine provides the Go client for the C++ Trading & Clearing Engine.
package engine

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/aspira/aspira-pay/internal/config"
)

// Client communicates with the C++ engine via TCP.
// In Sandbox mode without a running engine, it uses local fallback execution.
type Client struct {
	addr      string
	enabled   bool
	timeout   time.Duration
	conn      net.Conn
	mu        sync.Mutex
	connected bool
}

// NewClient creates a new engine client.
func NewClient(cfg config.EngineConfig) *Client {
	return &Client{
		addr:    cfg.Addr,
		enabled: cfg.Enabled,
		timeout: cfg.Timeout,
	}
}

// Connect establishes a TCP connection to the C++ engine.
func (c *Client) Connect() error {
	if !c.enabled {
		log.Println("Engine: disabled, using local fallback execution")
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	conn, err := net.DialTimeout("tcp", c.addr, c.timeout)
	if err != nil {
		c.connected = false
		return fmt.Errorf("engine: cannot connect to %s: %w", c.addr, err)
	}

	c.conn = conn
	c.connected = true
	log.Printf("Engine: connected to %s", c.addr)
	return nil
}

// IsEnabled returns true if the engine integration is enabled.
func (c *Client) IsEnabled() bool { return c.enabled }

// IsConnected returns true if currently connected to the engine.
func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}

// SendCommand sends a payment command to the C++ engine.
// Returns the engine's response or an error.
func (c *Client) SendCommand(cmd *Command) (*Response, error) {
	if !c.enabled || !c.IsConnected() {
		// Sandbox fallback: execute locally
		return c.executeLocal(cmd), nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := json.Marshal(cmd)
	if err != nil {
		return nil, fmt.Errorf("engine: cannot marshal command: %w", err)
	}

	c.conn.SetWriteDeadline(time.Now().Add(c.timeout))
	if _, err := c.conn.Write(append(data, '\n')); err != nil {
		c.connected = false
		return nil, fmt.Errorf("engine: write failed: %w", err)
	}

	// Read response
	c.conn.SetReadDeadline(time.Now().Add(c.timeout))
	buf := make([]byte, 4096)
	n, err := c.conn.Read(buf)
	if err != nil {
		c.connected = false
		return nil, fmt.Errorf("engine: read failed: %w", err)
	}

	var resp Response
	if err := json.Unmarshal(buf[:n], &resp); err != nil {
		return nil, fmt.Errorf("engine: cannot unmarshal response: %w", err)
	}

	return &resp, nil
}

// executeLocal simulates engine execution for Sandbox without C++ engine.
func (c *Client) executeLocal(cmd *Command) *Response {
	log.Printf("Engine [local]: executing command type=%s payment_id=%s", cmd.CommandType, cmd.PaymentID)
	return &Response{
		SequenceID: cmd.SequenceID,
		PaymentID:  cmd.PaymentID,
		Result:     "EXECUTED",
		EventID:    fmt.Sprintf("evt_local_%d", time.Now().UnixNano()),
	}
}

// Close closes the engine connection.
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		c.conn.Close()
	}
	c.connected = false
}
