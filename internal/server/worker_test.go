package server

import (
	"testing"
	"time"

	"github.com/panjf2000/ants/v2"
)

func TestSendToWorker(t *testing.T) {
	s := &Server{}

	ran := false
	if err := s.SendToWorker(func() { ran = true }); err != nil {
		t.Fatalf("expected nil workpool fallback to succeed: %v", err)
	}
	if !ran {
		t.Fatalf("expected fallback worker function to run synchronously")
	}

	wp, err := ants.NewPool(1)
	if err != nil {
		t.Fatalf("create ants pool: %v", err)
	}
	defer wp.Release()

	s.wp = wp
	done := make(chan struct{})
	if err := s.SendToWorker(func() { close(done) }); err != nil {
		t.Fatalf("expected workpool submit success: %v", err)
	}

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("expected submitted worker function to execute")
	}

	wp.Release()
	if err := s.SendToWorker(func() {}); err == nil {
		t.Fatalf("expected worker submit error after pool release")
	}
}
