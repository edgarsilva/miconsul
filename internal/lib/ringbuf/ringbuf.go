// Package ringbuf provides a fixed-size ring buffer that keeps the most recent
// values and is safe for concurrent use.
package ringbuf

import "sync"

// Buffer keeps the last size values added to it, dropping the oldest first.
type Buffer[T any] struct {
	mu     sync.RWMutex
	values []T
	next   int
	full   bool
}

// New returns a buffer that keeps the last size values. A size below 1 keeps 1.
func New[T any](size int) *Buffer[T] {
	return &Buffer[T]{values: make([]T, max(size, 1))}
}

// Add appends v and drops the oldest value when the buffer is full.
func (b *Buffer[T]) Add(v T) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.values[b.next] = v
	b.next++
	if b.next == len(b.values) {
		b.next = 0
		b.full = true
	}
}

// Snapshot returns a copy of the values added so far, oldest first, e.g. adding
// 1, 2, 3 to a buffer of size 2 -> [2 3]. The copy is the caller's: later calls
// to Add do not change it.
func (b *Buffer[T]) Snapshot() []T {
	b.mu.RLock()
	defer b.mu.RUnlock()

	out := make([]T, 0, b.count())

	if !b.full {
		return append(out, b.values[:b.next]...)
	}

	older := b.values[b.next:]
	newer := b.values[:b.next]
	out = append(out, older...)
	out = append(out, newer...)

	return out
}

// Len returns how many values the buffer holds, up to its size.
func (b *Buffer[T]) Len() int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.count()
}

// count returns how many values the buffer holds. The caller holds the lock.
func (b *Buffer[T]) count() int {
	if b.full {
		return len(b.values)
	}

	return b.next
}
