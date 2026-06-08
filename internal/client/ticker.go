package client

import (
	"context"
	"rawvelo/internal/flog"
	"time"
)

const healthCheckInterval = 10 * time.Second

func (c *Client) ticker(ctx context.Context) {
	ticker := time.NewTicker(healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.checkAndReconnect()
		case <-ctx.Done():
			return
		}
	}
}

func (c *Client) checkAndReconnect() {
	for i, tc := range c.iter.Items {
		if !tc.isAlive() {
			flog.Warnf("connection %d is dead, reconnecting...", i+1)
			go tc.reconnect()
		}
	}
}
