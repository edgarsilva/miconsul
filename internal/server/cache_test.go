package server

import (
	"errors"
	"testing"
	"time"
)

type cacheStub struct {
	readErr  error
	writeErr error
	wroteKey string
	readKey  string
}

func (c *cacheStub) Read(key string, _ *[]byte) error {
	c.readKey = key
	return c.readErr
}

func (c *cacheStub) Write(key string, _ *[]byte, _ time.Duration) error {
	c.wroteKey = key
	return c.writeErr
}

func TestCacheReadWrite(t *testing.T) {
	s := &Server{}
	payload := []byte("v")

	if err := s.CacheWrite("k", &payload, time.Second); err != nil {
		t.Fatalf("expected nil cache write to be no-op: %v", err)
	}
	if err := s.CacheRead("k", &payload); err != nil {
		t.Fatalf("expected nil cache read to be no-op: %v", err)
	}

	stub := &cacheStub{}
	s.Cache = stub
	if err := s.CacheWrite("write-key", &payload, time.Second); err != nil {
		t.Fatalf("unexpected cache write error: %v", err)
	}
	if stub.wroteKey != "write-key" {
		t.Fatalf("expected write key to be captured, got %q", stub.wroteKey)
	}

	if err := s.CacheRead("read-key", &payload); err != nil {
		t.Fatalf("unexpected cache read error: %v", err)
	}
	if stub.readKey != "read-key" {
		t.Fatalf("expected read key to be captured, got %q", stub.readKey)
	}

	stub.writeErr = errors.New("write failed")
	if err := s.CacheWrite("k2", &payload, time.Second); err == nil {
		t.Fatalf("expected cache write error")
	}
}
