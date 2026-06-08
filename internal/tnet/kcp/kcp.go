package kcp

import (
	"rawvelo/internal/conf"

	kcp "github.com/xtaci/kcp-go/v5"
	"github.com/xtaci/smux"
)

func aplConf(conn *kcp.UDPSession, cfg *conf.KCP) {
	var noDelay, interval, resend, noCongestion int
	var wDelay, ackNoDelay bool

	switch cfg.Mode {
	case "normal":
		noDelay, interval, resend, noCongestion = 0, 20, 2, 1
		wDelay, ackNoDelay = true, false
	case "fast":
		noDelay, interval, resend, noCongestion = 0, 10, 2, 1
		wDelay, ackNoDelay = false, true
	case "fast2":
		noDelay, interval, resend, noCongestion = 1, 8, 2, 1
		wDelay, ackNoDelay = false, true
	case "fast3":
		noDelay, interval, resend, noCongestion = 1, 5, 2, 1
		wDelay, ackNoDelay = false, true
	case "extreme":
		// کمترین latency ممکن — فقط لینک‌های باکیفیت
		noDelay, interval, resend, noCongestion = 1, 2, 2, 1
		wDelay, ackNoDelay = false, true
	case "manual":
		noDelay = cfg.NoDelay
		interval = cfg.Interval
		resend = cfg.Resend
		noCongestion = cfg.NoCongestion
		wDelay = cfg.WDelay
		ackNoDelay = cfg.AckNoDelay
	}

	conn.SetNoDelay(noDelay, interval, resend, noCongestion)
	conn.SetWindowSize(cfg.Sndwnd, cfg.Rcvwnd)
	conn.SetMtu(cfg.MTU)
	conn.SetWriteDelay(wDelay)
	conn.SetACKNoDelay(ackNoDelay)

	// DSCP 46 = Expedited Forwarding — بالاترین اولویت
	conn.SetDSCP(46)

	// buffer روی session برای zero-copy
	conn.SetReadBuffer(cfg.Smuxbuf)
	conn.SetWriteBuffer(cfg.Smuxbuf)
}

func smuxConf(cfg *conf.KCP) *smux.Config {
	sconf := smux.DefaultConfig()
	sconf.Version = 2
	sconf.KeepAliveInterval = cfg.Smuxkalive
	sconf.KeepAliveTimeout = cfg.Smuxktimeout
	sconf.KeepAliveDisabled = false
	sconf.MaxFrameSize = 65535
	sconf.MaxReceiveBuffer = cfg.Smuxbuf
	sconf.MaxStreamBuffer = cfg.Streambuf
	return sconf
}
