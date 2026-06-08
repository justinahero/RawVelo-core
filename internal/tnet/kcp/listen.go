package kcp

import (
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

type Listener struct {
	packetConn *socket.PacketConn
	cfg        *conf.KCP
	listener   *kcp.Listener
}

func Listen(cfg *conf.KCP, pConn *socket.PacketConn) (tnet.Listener, error) {
	l, err := kcp.ServeConn(cfg.Block, cfg.Dshard, cfg.Pshard, pConn)
	if err != nil {
		return nil, err
	}
	return &Listener{packetConn: pConn, cfg: cfg, listener: l}, nil
}

func (l *Listener) Accept() (tnet.Conn, error) {
	conn, err := l.listener.AcceptKCP()
	if err != nil {
		return nil, err
	}
	aplConf(conn, l.cfg)

	// باگ ۸ fix: اگه camouflage فعاله، conn رو wrap کن و handshake انجام بده
	var smuxConn io.ReadWriteCloser = conn
	if obfsCfg := l.packetConn.ObfsCfg(); obfsCfg != nil && obfsCfg.IsCamouflage() {
		tlsConn := camouflage.Wrap(conn, false)
		if err := tlsConn.Handshake(); err != nil {
			conn.Close()
			return nil, err
		}
		smuxConn = tlsConn
		flog.Debugf("camouflage TLS handshake completed (server)")
	}

	sess, err := smux.Server(smuxConn, smuxConf(l.cfg))
	if err != nil {
		conn.Close()
		return nil, err
	}
	// PacketConn رو به Conn نمیدیم — lifetime اون فقط با Listener مدیریت میشه
	return &Conn{UDPSession: conn, Session: sess}, nil
}

func (l *Listener) Close() error {
	if l.listener != nil {
		l.listener.Close()
	}
	if l.packetConn != nil {
		l.packetConn.Close()
	}
	return nil
}

func (l *Listener) Addr() net.Addr {
	return l.listener.Addr()
}
