package socket

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"os"
	"rawvelo/internal/conf"
	"rawvelo/internal/obfs"
	"sync"
	"sync/atomic"
	"time"
)

type PacketConn struct {
	cfg           *conf.Network
	sendHandle    *SendHandle
	recvHandle    *RecvHandle
	readDeadline  atomic.Value
	writeDeadline atomic.Value
	obfuscator    *obfs.Obfuscator
	obfsCfg       *conf.Obfs

	ctx    context.Context
	cancel context.CancelFunc

	// باگ ۲ fix: WaitGroup برای ردیابی writeهای در جریان
	// Close صبر میکنه تا همه writeها تموم بشن قبل از بستن handle
	writeWg sync.WaitGroup
}

// New — بدون obfs (backward compatible)
func New(ctx context.Context, cfg *conf.Network) (*PacketConn, error) {
	return NewWithObfs(ctx, cfg, nil)
}

// NewWithObfs — با obfs config؛ اگه nil بود یا disabled بود obfs فعال نمیشه
func NewWithObfs(ctx context.Context, cfg *conf.Network, obfsCfg *conf.Obfs) (*PacketConn, error) {
	if cfg.Port == 0 {
		cfg.Port = 32768 + rand.Intn(32768)
	}

	sendHandle, err := NewSendHandle(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create send handle on %s: %v", cfg.Interface.Name, err)
	}

	recvHandle, err := NewRecvHandle(cfg)
	if err != nil {
		sendHandle.Close()
		return nil, fmt.Errorf("failed to create receive handle on %s: %v", cfg.Interface.Name, err)
	}

	ctx, cancel := context.WithCancel(ctx)
	conn := &PacketConn{
		cfg:        cfg,
		sendHandle: sendHandle,
		recvHandle: recvHandle,
		ctx:        ctx,
		cancel:     cancel,
		obfsCfg:    obfsCfg,
	}

	if obfsCfg != nil && obfsCfg.Enabled && obfsCfg.Key != "" {
		conn.obfuscator = obfs.New(obfsCfg.Key)
	}

	return conn, nil
}

// ObfsCfg — برای دسترسی از kcp layer
func (c *PacketConn) ObfsCfg() *conf.Obfs {
	return c.obfsCfg
}

// ReadFrom — blocking read با deadline و ctx support
// باگ ۱ fix: دیگه goroutine جداگانه نمیسازیم.
// recvHandle.Read() روی pcap.ReadPacketData بلاک میشه.
// وقتی Close() صدا زده میشه، recvHandle.Close() pcap handle رو میبنده
// و ReadPacketData رو آزاد میکنه — بدون نشت گوروتین.
func (c *PacketConn) ReadFrom(data []byte) (n int, addr net.Addr, err error) {
	// چک کن ctx قبلاً کنسل شده یا نه
	select {
	case <-c.ctx.Done():
		return 0, nil, c.ctx.Err()
	default:
	}

	for {
		payload, rAddr, readErr := c.recvHandle.Read()
		if readErr != nil {
			// اگه ctx کنسل شده، error اصلی رو برگردون نه pcap error
			select {
			case <-c.ctx.Done():
				return 0, nil, c.ctx.Err()
			default:
			}
			if errors.Is(readErr, ErrNoPayload) {
				// پکت بدون payload — skip کن و دوباره بخون
				select {
				case <-c.ctx.Done():
					return 0, nil, c.ctx.Err()
				default:
					continue
				}
			}
			return 0, nil, readErr
		}

		// deadline چک بعد از read
		if d, ok := c.readDeadline.Load().(time.Time); ok && !d.IsZero() {
			if time.Now().After(d) {
				return 0, nil, os.ErrDeadlineExceeded
			}
		}

		if c.obfuscator != nil {
			payload = c.obfuscator.Unscramble(payload)
			if c.obfsCfg.IsPadding() {
				payload = c.obfuscator.RemovePadding(payload)
			}
		}
		n = copy(data, payload)
		return n, rAddr, nil
	}
}

// WriteTo — با obfs encode
// باگ ۲ fix: writeWg.Add/Done برای جلوگیری از use-after-close
func (c *PacketConn) WriteTo(data []byte, addr net.Addr) (n int, err error) {
	select {
	case <-c.ctx.Done():
		return 0, c.ctx.Err()
	default:
	}

	if d, ok := c.writeDeadline.Load().(time.Time); ok && !d.IsZero() {
		if time.Now().After(d) {
			return 0, os.ErrDeadlineExceeded
		}
	}

	daddr, ok := addr.(*net.UDPAddr)
	if !ok {
		return 0, net.InvalidAddrError("invalid address")
	}

	// ثبت write در جریان — Close صبر میکنه تا این تموم بشه
	c.writeWg.Add(1)
	defer c.writeWg.Done()

	// چک مجدد بعد از Add — ممکنه Close بین دو چک اول و اینجا صدا زده شده باشه
	select {
	case <-c.ctx.Done():
		return 0, c.ctx.Err()
	default:
	}

	payload := data
	if c.obfuscator != nil {
		if c.obfsCfg.IsPadding() {
			padded, err := c.obfuscator.AddPadding(payload)
			if err == nil {
				payload = padded
			}
		}
		payload = c.obfuscator.Scramble(payload)
		if c.obfsCfg.IsJitter() {
			c.obfuscator.Jitter()
		}
	}

	err = c.sendHandle.Write(payload, daddr)
	if err != nil {
		return 0, err
	}

	return len(data), nil
}

// Close — باگ ۲ fix: صبر میکنه تا writeهای در جریان تموم بشن، بعد handleها رو synchronous میبنده
// باگ ۱ fix: recvHandle.Close() pcap handle رو میبنده و ReadPacketData بلاکینگ رو آزاد میکنه
func (c *PacketConn) Close() error {
	c.cancel()
	// صبر کن تا writeهای در جریان تموم بشن
	c.writeWg.Wait()
	if c.sendHandle != nil {
		c.sendHandle.Close()
	}
	if c.recvHandle != nil {
		c.recvHandle.Close()
	}
	return nil
}

func (c *PacketConn) LocalAddr() net.Addr { return nil }

func (c *PacketConn) SetDeadline(t time.Time) error {
	c.readDeadline.Store(t)
	c.writeDeadline.Store(t)
	return nil
}

func (c *PacketConn) SetReadDeadline(t time.Time) error {
	c.readDeadline.Store(t)
	return nil
}

func (c *PacketConn) SetWriteDeadline(t time.Time) error {
	c.writeDeadline.Store(t)
	return nil
}

func (c *PacketConn) SetDSCP(dscp int) error { return nil }

func (c *PacketConn) SetClientTCPF(addr net.Addr, f []conf.TCPF) {
	c.sendHandle.setClientTCPF(addr, f)
}
