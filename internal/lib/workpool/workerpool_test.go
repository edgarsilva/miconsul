package workpool

import "testing"

func TestNewUsesDefaultSizeAndShutdown(t *testing.T) {
	pool, shutdown := New(0)
	if pool == nil {
		t.Fatal("New(0) expected non-nil pool")
	}
	if pool.AntsPool() == nil {
		t.Fatal("New(0) expected wrapped ants pool")
	}

	shutdown()
}

func TestAntsPoolNilReceiver(t *testing.T) {
	var pool *Pool
	if got := pool.AntsPool(); got != nil {
		t.Fatalf("AntsPool() on nil receiver = %v, want nil", got)
	}
}

func TestReleaseNilSafe(t *testing.T) {
	var nilPool *Pool
	nilPool.Release()

	zero := &Pool{}
	zero.Release()
}
