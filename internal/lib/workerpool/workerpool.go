package workerpool

import (
	"fmt"
	"log"

	"github.com/panjf2000/ants/v2"
)

func New(size int) (p *ants.Pool, shutdownFn func()) {
	p, err := ants.NewPool(10)
	if err != nil {
		log.Fatal(fmt.Errorf("failed to initialize worker pool: %w", err))
	}

	return p, func() { p.Release() }
}
