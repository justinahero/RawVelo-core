// Package fec — Adaptive FEC
// بر اساس packet loss واقعی، FEC رو تنظیم میکنه
package fec

import (
	"sync"
	"sync/atomic"
	"time"
)

const (
	updateInterval = 5 * time.Second
)

// AdaptiveFEC — loss رو اندازه میگیره و FEC رو تنظیم میکنه
type AdaptiveFEC struct {
	sent   atomic.Int64
	lost   atomic.Int64
	dshard atomic.Int32
	pshard atomic.Int32
	mu     sync.Mutex
	stopCh chan struct{}
	once   sync.Once // باگ ۷ fix: جلوگیری از double-close panic
}

func New(initialDshard, initialPshard int) *AdaptiveFEC {
	a := &AdaptiveFEC{
		stopCh: make(chan struct{}),
	}
	a.dshard.Store(int32(initialDshard))
	a.pshard.Store(int32(initialPshard))
	go a.updateLoop()
	return a
}

// RecordSent — پکت فرستاده شده رو ثبت میکنه
func (a *AdaptiveFEC) RecordSent() {
	a.sent.Add(1)
}

// RecordLost — پکت از دست رفته رو ثبت میکنه
func (a *AdaptiveFEC) RecordLost() {
	a.lost.Add(1)
}

// Dshard — مقدار فعلی data shards
func (a *AdaptiveFEC) Dshard() int {
	return int(a.dshard.Load())
}

// Pshard — مقدار فعلی parity shards
func (a *AdaptiveFEC) Pshard() int {
	return int(a.pshard.Load())
}

// LossRate — نرخ loss فعلی (0.0 - 1.0)
func (a *AdaptiveFEC) LossRate() float64 {
	sent := a.sent.Load()
	if sent == 0 {
		return 0
	}
	return float64(a.lost.Load()) / float64(sent)
}

func (a *AdaptiveFEC) updateLoop() {
	ticker := time.NewTicker(updateInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			a.adjust()
		case <-a.stopCh:
			return
		}
	}
}

func (a *AdaptiveFEC) adjust() {
	loss := a.LossRate()

	var dshard, pshard int32
	switch {
	case loss == 0:
		dshard, pshard = 10, 1
	case loss <= 0.02:
		dshard, pshard = 10, 2
	case loss <= 0.05:
		dshard, pshard = 10, 3
	case loss <= 0.10:
		dshard, pshard = 10, 4
	default:
		dshard, pshard = 10, 5
	}

	a.dshard.Store(dshard)
	a.pshard.Store(pshard)

	// reset counters برای window بعدی
	a.sent.Store(0)
	a.lost.Store(0)
}

// Stop — باگ ۷ fix: با sync.Once جلوگیری از double-close panic
func (a *AdaptiveFEC) Stop() {
	a.once.Do(func() {
		close(a.stopCh)
	})
}
