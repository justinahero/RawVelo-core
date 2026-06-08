package kcp

import (
	"fmt"
	"net"
	"rawvelo/internal/protocol"
	"rawvelo/internal/tnet"
	"time"

	"github.com/xtaci/kcp-go/v5"
	"github.com/xtaci/smux"
)

type Conn struct {
	UDPSession *kcp.UDPSession
	Session    *smux.Session
}

func (c *Conn) OpenStrm() (tnet.Strm, error) {
	strm, err := c.Session.OpenStream()
	if err != nil {
		return nil, err
	}
	return &Strm{strm}, nil
}

func (c *Conn) AcceptStrm() (tnet.Strm, error) {
	strm, err := c.Session.AcceptStream()
	if err != nil {
		return nil, err
	}
	return &Strm{strm}, nil
}

func (c *Conn) Ping(wait bool) error {
	strm, err := c.Session.OpenStream()
	if err != nil {
		return fmt.Errorf("ping failed: %v", err)
	}
	defer strm.Close()
	if wait {
		p := protocol.Proto{Type: protocol.PPING}
		err = p.Write(strm)
		if err != nil {
			return fmt.Errorf("strm ping write failed: %v", err)
		}
		err = p.Read(strm)
		if err != nil {
			return fmt.Errorf("strm ping read failed: %v", err)
		}
		if p.Type != protocol.PPONG {
			return fmt.Errorf("expected PONG, got type %d", p.Type)
		}
	}
	return nil
}

// NumStreams — تعداد stream های فعال رو برمیگردونه (برای load balancing)
func (c *Conn) NumStreams() int {
	if c.Session == nil {
		return 0
	}
	return c.Session.NumStreams()
}

func (c *Conn) Close() error {
	// PacketConn اینجا بسته نمیشه — مسئولیتش با Listener یا Dial کننده‌ست
	// بستن PacketConn اینجا باعث میشه قطع شدن یه کلاینت، همه رو بکشه
	if c.Session != nil {
		c.Session.Close()
	}
	if c.UDPSession != nil {
		c.UDPSession.Close()
	}
	return nil
}

func (c *Conn) LocalAddr() net.Addr                { return c.Session.LocalAddr() }
func (c *Conn) RemoteAddr() net.Addr               { return c.Session.RemoteAddr() }
func (c *Conn) SetDeadline(t time.Time) error      { return c.Session.SetDeadline(t) }
func (c *Conn) SetReadDeadline(t time.Time) error  { return c.UDPSession.SetReadDeadline(t) }
func (c *Conn) SetWriteDeadline(t time.Time) error { return c.UDPSession.SetWriteDeadline(t) }

func (c *Conn) IsClosed() bool {
	if c.Session == nil {
		return true
	}
	return c.Session.IsClosed()
}
