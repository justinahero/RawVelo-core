package client

import (
	"fmt"
	"rawvelo/internal/flog"
	"rawvelo/internal/tnet"
	"time"
)

const (
	autoExpire   = 300 * time.Second
	maxStrmRetry = 3
)

func (c *Client) newConn() (tnet.Conn, error) {
	tc := c.iter.Next()

	// باگ ۳ fix: کل بلوک "چک و reconnect" زیر قفل — بدون TOCTOU
	tc.mu.Lock()
	if tc.conn == nil || tc.conn.IsClosed() {
		flog.Warnf("selected connection is dead, reconnecting...")
		// createConn داخلش sendTCPF رو هم صدا میزنه
		conn, err := tc.createConnLocked()
		if err != nil {
			tc.mu.Unlock()
			return nil, fmt.Errorf("reconnect failed: %w", err)
		}
		tc.conn = conn
		tc.expire = time.Now().Add(autoExpire)
	}
	conn := tc.conn
	tc.mu.Unlock()
	return conn, nil
}

func (c *Client) newStrm() (tnet.Strm, error) {
	for attempt := 0; attempt < maxStrmRetry; attempt++ {
		conn, err := c.newConn()
		if err != nil {
			flog.Debugf("newConn attempt %d failed: %v", attempt+1, err)
			continue
		}
		strm, err := conn.OpenStrm()
		if err != nil {
			flog.Debugf("OpenStrm attempt %d failed: %v", attempt+1, err)
			continue
		}
		return strm, nil
	}
	return nil, fmt.Errorf("failed to open stream after %d attempts", maxStrmRetry)
}
