package kcp

import (
	"fmt"
	"io"
	"net"
	"rawvelo/internal/camouflage"
	"rawvelo/internal/conf"
	"rawvelo/internal/flog"
	"rawvelo/internal/socket"
	"rawvelo/internal/tnet"

	"github.com/xtaci/kcp-go/v5"
	"github.com/xtaci/smux"
)

func Dial(addr *net.UDPAddr, cfg *conf.KCP, pConn *socket.PacketConn) (tnet.Conn, error) {
	conn, err := kcp.NewConn(addr.String(), cfg.Block, cfg.Dshard, cfg.Pshard, pConn)
	if err != nil {
		return nil, fmt.Errorf("connection attempt failed: %v", err)
	}
	aplConf(conn, cfg)
	flog.Debugf("KCP connection created, creating smux session")

	// باگ ۸ fix: اگه camouflage فعاله، conn رو wrap کن و handshake انجام بده
	var smuxConn io.ReadWriteCloser = conn
	if obfsCfg := pConn.ObfsCfg(); obfsCfg != nil && obfsCfg.IsCamouflage() {
		tlsConn := camouflage.Wrap(conn, true)
		if err := tlsConn.Handshake(); err != nil {
			conn.Close()
			return nil, fmt.Errorf("camouflage handshake failed: %w", err)
		}
		smuxConn = tlsConn
		flog.Debugf("camouflage TLS handshake completed (client)")
	}

	sess, err := smux.Client(smuxConn, smuxConf(cfg))
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create smux session: %w", err)
	}

	flog.Debugf("smux session created successfully")
	// PacketConn رو به Conn نمیدیم — timedConn مسئول بستنشه
	return &Conn{UDPSession: conn, Session: sess}, nil
}
