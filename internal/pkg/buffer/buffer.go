package buffer

import "fmt"

const defaultBufSize = 32 * 1024 // 32 KB — مقدار پیش‌فرض امن

var (
	TPool int
	UPool int
)

func Initialize(tPool, uPool int) {
	if tPool <= 0 {
		panic(fmt.Sprintf("buffer.Initialize: tPool must be > 0, got %d", tPool))
	}
	if uPool <= 0 {
		panic(fmt.Sprintf("buffer.Initialize: uPool must be > 0, got %d", uPool))
	}
	TPool = tPool
	UPool = uPool
}

// tcpBufSize — باگ ۵ fix: اگه Initialize صدا زده نشده، از مقدار پیش‌فرض امن استفاده میکنه
func tcpBufSize() int {
	if TPool <= 0 {
		return defaultBufSize
	}
	return TPool
}

// udpBufSize — باگ ۵ fix: مثل tcpBufSize
func udpBufSize() int {
	if UPool <= 0 {
		return defaultBufSize
	}
	return UPool
}
