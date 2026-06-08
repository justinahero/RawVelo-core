package iterator

import (
	"sync"
	"sync/atomic"
)

// SimpleIterator — round-robin برای type های ساده
type SimpleIterator[T any] struct {
	Items []T
	index atomic.Uint64
}

func (it *SimpleIterator[T]) Next() T {
	n := uint64(len(it.Items))
	if n == 0 {
		var zero T
		return zero
	}
	i := it.index.Add(1) - 1
	if n&(n-1) == 0 {
		return it.Items[i&(n-1)]
	}
	return it.Items[i%n]
}

// Iterator — least-connections load balancing
type Iterator[T interface{ Load() int64 }] struct {
	Items []T
	index atomic.Uint64
	mu    sync.Mutex
}

func (it *Iterator[T]) Next() T {
	it.mu.Lock()
	defer it.mu.Unlock()

	n := len(it.Items)
	if n == 0 {
		var zero T
		return zero
	}
	if n == 1 {
		return it.Items[0]
	}

	// اگه همه مرده بودن — round-robin fallback
	best := it.Items[0]
	bestLoad := best.Load()
	allDead := bestLoad >= 9999

	for _, item := range it.Items[1:] {
		load := item.Load()
		if load < 9999 {
			allDead = false
		}
		if load < bestLoad {
			best = item
			bestLoad = load
		}
	}

	// اگه همه مرده — round-robin تا reconnect بشن
	if allDead {
		i := it.index.Add(1) - 1
		return it.Items[i%uint64(n)]
	}

	return best
}
