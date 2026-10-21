package workpool

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/panjf2000/ants/v2"
)

func TestNewUsesDefaultSizeAndShutdown(t *testing.T) {
	pool, shutdown := New(0, time.Second)
	if pool == nil {
		t.Fatal("New(0) expected non-nil pool")
	}
	if pool.AntsPool() == nil {
		t.Fatal("New(0) expected wrapped ants pool")
	}

	if err := shutdown(); err != nil {
		t.Fatalf("shutdown() error = %v", err)
	}
}

func TestAntsPoolNilReceiver(t *testing.T) {
	var pool *Pool
	if got := pool.AntsPool(); got != nil {
		t.Fatalf("AntsPool() on nil receiver = %v, want nil", got)
	}
}

func TestReleaseNilSafe(t *testing.T) {
	var nilPool *Pool
	if err := nilPool.Release(time.Second); err != nil {
		t.Fatalf("Release() on nil receiver error = %v", err)
	}

	zero := &Pool{}
	if err := zero.Release(time.Second); err != nil {
		t.Fatalf("Release() on zero pool error = %v", err)
	}
}

func TestShutdownWaitsForRunningTasks(t *testing.T) {
	pool, shutdown := New(1, 2*time.Second)

	started := make(chan struct{})
	var finished atomic.Bool
	if err := pool.AntsPool().Submit(func() {
		close(started)
		time.Sleep(100 * time.Millisecond)
		finished.Store(true)
	}); err != nil {
		t.Fatalf("Submit() error = %v", err)
	}
	<-started

	if err := shutdown(); err != nil {
		t.Fatalf("shutdown() error = %v", err)
	}
	if !finished.Load() {
		t.Fatal("shutdown() returned before the running task finished")
	}
}

func TestShutdownTimesOut(t *testing.T) {
	pool, shutdown := New(1, 50*time.Millisecond)

	started := make(chan struct{})
	release := make(chan struct{})
	defer close(release)
	if err := pool.AntsPool().Submit(func() {
		close(started)
		<-release
	}); err != nil {
		t.Fatalf("Submit() error = %v", err)
	}
	<-started

	if err := shutdown(); !errors.Is(err, ants.ErrTimeout) {
		t.Fatalf("shutdown() error = %v, want %v", err, ants.ErrTimeout)
	}
}
