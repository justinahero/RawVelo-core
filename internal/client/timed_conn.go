package client

import (
	"context"
	"fmt"
	"rawvelo/internal/conf"
	"rawvelo/internal/flog"
	"rawvelo/internal/protocol"
	"rawvelo/internal/socket"
	"rawvelo/internal/tnet"
	"rawvelo/internal/tnet/kcp"
	"sync"
	"time"
)

const (
	reconnectBaseDelay = 2 * time.Second
	reconnectMaxDelay  = 60 * time.Second
)

type timedConn struct {
	cfg    *conf.Conf
	conn   tnet.Conn
	pConn  *socket.PacketConn // PacketConn متعلق به این timedConn — فقط اینجا بسته میشه
	expire time.Time
	ctx    context.Context
	mu     sync.Mutex
}

func newTimedConn(ctx context.Context, cfg *conf.Conf) (*timedConn, error) {
	tc := &timedConn{cfg: cfg, ctx: ctx}
	conn, err := tc.connectWithRetry()
	if err != nil {
		return nil, err
	}
	tc.conn = conn
	tc.expire = time.Now().Add(300 * time.Second)
	return tc, nil
}

func (tc *timedConn) connectWithRetry() (tnet.Conn, error) {
	delay := reconnectBaseDelay
	attempt := 0
	for {
		select {
		case <-tc.ctx.Done():
			return nil, fmt.Errorf("context cancelled during connect")
		default:
		}

		conn, err := tc.createConn()
		if err == nil {
			if attempt > 0 {
				flog.Infof("reconnected successfully after %d attempt(s)", attempt+1)
			}
			return conn, nil
		}

		attempt++
		flog.Warnf("connection attempt %d failed: %v — retrying in %s", attempt, err, delay)

		select {
		case <-tc.ctx.Done():
			return nil, fmt.Errorf("context cancelled while waiting to retry")
		case <-time.After(delay):
		}

		delay *= 2
		if delay > reconnectMaxDelay {
			delay = reconnectMaxDelay
		}
	}
}

func (tc *timedConn) createConn() (tnet.Conn, error) {
	return tc.createConnLocked()
}

// createConnLocked — مثل createConn ولی فرض میکنه tc.mu قبلاً گرفته شده
// از newConn (dial.go) صدا زده میشه که قفل رو داره
func (tc *timedConn) createConnLocked() (tnet.Conn, error) {
	netCfg := tc.cfg.Network
	pConn, err := socket.NewWithObfs(tc.ctx, &netCfg, &tc.cfg.Obfs)
	if err != nil {
		return nil, fmt.Errorf("could not create packet conn: %w", err)
	}
	conn, err := kcp.Dial(tc.cfg.Server.Addr, tc.cfg.Transport.KCP, pConn)
	if err != nil {
		pConn.Close()
		return nil, fmt.Errorf("kcp dial failed: %w", err)
	}
	if err := tc.sendTCPF(conn); err != nil {
		conn.Close()
		pConn.Close()
		return nil, fmt.Errorf("tcpf handshake failed: %w", err)
	}
	// pConn قدیمی رو ببند، جدید رو ذخیره کن
	if tc.pConn != nil {
		tc.pConn.Close()
	}
	tc.pConn = pConn
	return conn, nil
}

func (tc *timedConn) sendTCPF(conn tnet.Conn) error {
	strm, err := conn.OpenStrm()
	if err != nil {
		return err
	}
	defer strm.Close()
	p := protocol.Proto{Type: protocol.PTCPF, TCPF: tc.cfg.Network.TCP.RF}
	return p.Write(strm)
}

// reconnect — با lock امن، با exponential backoff
func (tc *timedConn) reconnect() {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	flog.Warnf("connection lost, attempting reconnect...")
	if tc.conn != nil {
		tc.conn.Close()
		tc.conn = nil
	}
	conn, err := tc.connectWithRetry()
	if err != nil {
		flog.Errorf("reconnect failed: %v", err)
		return
	}
	tc.conn = conn
	tc.expire = time.Now().Add(300 * time.Second)
}

func (tc *timedConn) isAlive() bool {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	return tc.conn != nil && !tc.conn.IsClosed()
}

func (tc *timedConn) close() {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	if tc.conn != nil {
		tc.conn.Close()
		tc.conn = nil
	}
	if tc.pConn != nil {
		tc.pConn.Close()
		tc.pConn = nil
	}
}

// Load — تعداد واقعی stream های فعال برای least-connections balancing
func (tc *timedConn) Load() int64 {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	if tc.conn == nil || tc.conn.IsClosed() {
		return 9999 // dead — هرگز انتخاب نشه
	}
	return int64(tc.conn.NumStreams())
}
