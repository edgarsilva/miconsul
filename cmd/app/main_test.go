package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
)

func TestIsExpectedServerCloseError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil error", err: nil, want: true},
		{name: "context canceled", err: context.Canceled, want: true},
		{name: "http server closed", err: http.ErrServerClosed, want: true},
		{name: "wrapped http server closed", err: fmt.Errorf("wrapped: %w", http.ErrServerClosed), want: true},
		{name: "network closed", err: net.ErrClosed, want: true},
		{name: "message contains server closed", err: errors.New("server closed unexpectedly"), want: true},
		{name: "message contains listener closed", err: errors.New("listener closed"), want: true},
		{name: "message contains closed network connection", err: errors.New("use of closed network connection"), want: true},
		{name: "unexpected error", err: errors.New("boom"), want: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := isExpectedServerCloseError(tc.err)
			if got != tc.want {
				t.Fatalf("isExpectedServerCloseError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestShouldLogServerError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil error", err: nil, want: false},
		{name: "context canceled", err: context.Canceled, want: false},
		{name: "http server closed", err: http.ErrServerClosed, want: false},
		{name: "unexpected error", err: errors.New("boom"), want: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := shouldLogServerError(tc.err)
			if got != tc.want {
				t.Fatalf("shouldLogServerError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestServerLifecycleSmoke(t *testing.T) {
	app := fiber.New()

	addr, err := reserveAddress()
	if err != nil {
		t.Fatalf("reserve address: %v", err)
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- app.Listen(addr)
	}()

	if err := waitForServerReady(addr, 2*time.Second); err != nil {
		t.Fatalf("server did not become ready: %v", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = app.ShutdownWithContext(shutdownCtx)
	if shouldLogServerError(err) {
		t.Fatalf("shutdown returned unexpected error: %v", err)
	}

	select {
	case err = <-serveErr:
		if shouldLogServerError(err) {
			t.Fatalf("listen exited with unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for listen to exit")
	}
}

func reserveAddress() (string, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	defer ln.Close()

	return ln.Addr().String(), nil
}

func waitForServerReady(addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 50*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		time.Sleep(20 * time.Millisecond)
	}

	return fmt.Errorf("server at %s not reachable within %s", addr, timeout)
}
