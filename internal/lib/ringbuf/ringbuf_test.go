package ringbuf

import (
	"slices"
	"sync"
	"testing"
)

func TestBufferKeepsTheLastValuesOldestFirst(t *testing.T) {
	tests := []struct {
		name string
		size int
		add  []int
		want []int
	}{
		{"empty", 3, nil, nil},
		{"below capacity", 3, []int{1, 2}, []int{1, 2}},
		{"exactly full", 3, []int{1, 2, 3}, []int{1, 2, 3}},
		{"wraps once", 2, []int{1, 2, 3}, []int{2, 3}},
		{"wraps many times", 3, []int{1, 2, 3, 4, 5, 6, 7}, []int{5, 6, 7}},
		{"size below one keeps one", 0, []int{1, 2}, []int{2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := New[int](tt.size)
			for _, v := range tt.add {
				b.Add(v)
			}

			if got := b.Snapshot(); !slices.Equal(got, tt.want) {
				t.Fatalf("Snapshot() = %v, want %v", got, tt.want)
			}
			if got := b.Len(); got != len(tt.want) {
				t.Fatalf("Len() = %d, want %d", got, len(tt.want))
			}
		})
	}
}

func TestSnapshotIsACopy(t *testing.T) {
	b := New[int](3)
	b.Add(1)
	b.Add(2)

	snap := b.Snapshot()
	snap[0] = 99
	b.Add(3)

	if got := b.Snapshot(); !slices.Equal(got, []int{1, 2, 3}) {
		t.Fatalf("Snapshot() = %v after editing a previous copy, want [1 2 3]", got)
	}
}

func TestBufferIsSafeForConcurrentUse(t *testing.T) {
	b := New[int](50)

	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for i := range 1000 {
				b.Add(i)
				_ = b.Snapshot()
				_ = b.Len()
			}
		})
	}
	wg.Wait()

	if got := b.Len(); got != 50 {
		t.Fatalf("Len() = %d, want 50", got)
	}
}
