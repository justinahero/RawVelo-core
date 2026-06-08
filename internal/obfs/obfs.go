// Package obfs — لایه obfuscation
// ترافیک tunnel رو مخفی میکنه تا DPI نتونه تشخیص بده
package obfs

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"io"
	"math/big"
	mathrand "math/rand"
	"sync"
	"time"
)

const (
	MinPadding  = 16
	MaxPadding  = 256
	JitterMinMs = 2
	JitterMaxMs = 15
)

type Obfuscator struct {
	key    []byte
	keyLen int
	mu     sync.Mutex       // محافظت از rng که thread-safe نیست
	rng    *mathrand.Rand
}

func New(key string) *Obfuscator {
	h := sha256.Sum256([]byte(key))
	seed := int64(binary.LittleEndian.Uint64(h[:8]))
	return &Obfuscator{
		key:    h[:],
		keyLen: 32,
		rng:    mathrand.New(mathrand.NewSource(seed)),
	}
}

// Scramble — XOR با rotating key (بدون نیاز به lock چون key ثابته)
func (o *Obfuscator) Scramble(data []byte) []byte {
	out := make([]byte, len(data))
	for i, b := range data {
		out[i] = b ^ o.key[i%o.keyLen]
	}
	return out
}

// Unscramble — XOR قرینه‌ست
func (o *Obfuscator) Unscramble(data []byte) []byte {
	return o.Scramble(data)
}

// AddPadding — [1 byte padLen][data][random padding]
func (o *Obfuscator) AddPadding(data []byte) ([]byte, error) {
	padLen, err := randomInt(MinPadding, MaxPadding)
	if err != nil {
		padLen = MinPadding
	}
	padding := make([]byte, padLen)
	if _, err := io.ReadFull(rand.Reader, padding); err != nil {
		return nil, err
	}
	out := make([]byte, 1+len(data)+padLen)
	out[0] = byte(padLen)
	copy(out[1:], data)
	copy(out[1+len(data):], padding)
	return out, nil
}

// RemovePadding — padding حذف میکنه
func (o *Obfuscator) RemovePadding(data []byte) []byte {
	if len(data) < 1 {
		return data
	}
	padLen := int(data[0])
	end := len(data) - padLen
	if end < 1 || end > len(data) {
		return data[1:]
	}
	return data[1:end]
}

// Jitter — تاخیر رندم برای شکستن timing fingerprint
// از mu برای محافظت از rng استفاده میکنه
func (o *Obfuscator) Jitter() {
	o.mu.Lock()
	ms := JitterMinMs + int(o.rng.Int63n(int64(JitterMaxMs-JitterMinMs)))
	o.mu.Unlock()
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

func randomInt(min, max int) (int, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max-min)))
	if err != nil {
		return min, err
	}
	return min + int(n.Int64()), nil
}
