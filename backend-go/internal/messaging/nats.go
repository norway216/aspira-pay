// Package messaging provides NATS JetStream integration for event-driven messaging.
package messaging

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/nats-io/nats.go"

	"github.com/aspira/aspira-pay/internal/config"
)

// Client wraps a NATS connection for publishing and subscribing.
type Client struct {
	nc     *nats.Conn
	js     nats.JetStreamContext
	stream string
	cfg    config.NATSConfig
}

// NewClient creates a new NATS messaging client.
func NewClient(cfg config.NATSConfig) *Client {
	return &Client{cfg: cfg, stream: cfg.Stream}
}

// Connect establishes connection to NATS and creates the JetStream if needed.
func (c *Client) Connect() error {
	if !c.cfg.Enabled {
		log.Println("NATS: disabled, using local synchronous processing")
		return nil
	}

	nc, err := nats.Connect(c.cfg.URL)
	if err != nil {
		return fmt.Errorf("nats: cannot connect to %s: %w", c.cfg.URL, err)
	}
	c.nc = nc

	js, err := nc.JetStream()
	if err != nil {
		return fmt.Errorf("nats: cannot get JetStream context: %w", err)
	}
	c.js = js

	// Create stream if not exists
	_, err = js.StreamInfo(c.stream)
	if err != nil {
		_, err = js.AddStream(&nats.StreamConfig{
			Name:     c.stream,
			Subjects: []string{c.stream + ".*"},
			Storage:  nats.FileStorage,
		})
		if err != nil {
			log.Printf("NATS: cannot create stream %s (may already exist): %v", c.stream, err)
		}
	}

	log.Printf("NATS: connected to %s, stream: %s", c.cfg.URL, c.stream)
	return nil
}

// Publish publishes an event to the specified topic.
func (c *Client) Publish(topic string, payload interface{}) error {
	if !c.cfg.Enabled || c.nc == nil {
		log.Printf("NATS [disabled]: would publish to %s", topic)
		return nil
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("nats: cannot marshal event: %w", err)
	}

	subject := c.stream + "." + topic
	_, err = c.js.Publish(subject, data)
	if err != nil {
		return fmt.Errorf("nats: publish to %s failed: %w", subject, err)
	}

	log.Printf("NATS: published event to %s (%d bytes)", subject, len(data))
	return nil
}

// Subscribe subscribes to a topic and invokes the handler for each message.
func (c *Client) Subscribe(topic string, handler func(msg []byte) error) (*nats.Subscription, error) {
	if !c.cfg.Enabled || c.nc == nil {
		log.Printf("NATS [disabled]: would subscribe to %s", topic)
		return nil, nil
	}

	subject := c.stream + "." + topic
	sub, err := c.js.Subscribe(subject, func(m *nats.Msg) {
		if err := handler(m.Data); err != nil {
			log.Printf("NATS: handler error for %s: %v", subject, err)
		}
	})
	if err != nil {
		return nil, fmt.Errorf("nats: subscribe to %s failed: %w", subject, err)
	}

	log.Printf("NATS: subscribed to %s", subject)
	return sub, nil
}

// IsEnabled returns true if NATS messaging is enabled.
func (c *Client) IsEnabled() bool { return c.cfg.Enabled && c.nc != nil }

// Close closes the NATS connection.
func (c *Client) Close() {
	if c.nc != nil {
		c.nc.Close()
	}
}
