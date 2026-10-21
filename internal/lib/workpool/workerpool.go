// Package workpool provides a worker pool for the application to do
// background work.
package workpool

import (
	"fmt"
	"log"
	"time"

	"github.com/panjf2000/ants/v2"
)

type Pool struct {
	p *ants.Pool
}

// New starts a pool of size workers. shutdownFn releases it and waits up to
// releaseTimeout for running tasks to finish.
func New(size int, releaseTimeout time.Duration) (pool *Pool, shutdownFn func() error) {
	if size <= 0 {
		size = 10
	}

	p, err := ants.NewPool(size)
	if err != nil {
		log.Fatal(fmt.Errorf("initialize worker pool: %w", err))
	}

	pool = &Pool{p: p}
	return pool, func() error { return pool.Release(releaseTimeout) }
}

func (p *Pool) AntsPool() *ants.Pool {
	if p == nil {
		return nil
	}

	return p.p
}

// Release closes the pool and waits up to timeout for running tasks to finish.
func (p *Pool) Release(timeout time.Duration) error {
	if p == nil || p.p == nil {
		return nil
	}

	if err := p.p.ReleaseTimeout(timeout); err != nil {
		return fmt.Errorf("release worker pool: %w", err)
	}

	return nil
}
