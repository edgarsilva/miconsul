// Package workpool provides a worker pool for the application to do
// background work.
package workpool

import (
	"fmt"
	"log"

	"github.com/panjf2000/ants/v2"
)

type Pool struct {
	p *ants.Pool
}

func New(size int) (pool *Pool, shutdownFn func()) {
	if size <= 0 {
		size = 10
	}

	p, err := ants.NewPool(size)
	if err != nil {
		log.Fatal(fmt.Errorf("failed to initialize worker pool: %w", err))
	}

	pool = &Pool{p: p}
	return pool, func() { pool.Release() }
}

func (p *Pool) AntsPool() *ants.Pool {
	if p == nil {
		return nil
	}

	return p.p
}

func (p *Pool) Release() {
	if p == nil || p.p == nil {
		return
	}

	p.p.Release()
}
