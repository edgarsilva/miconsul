package workerpool

import (
	"fmt"

	"github.com/panjf2000/ants/v2"
)

func New(size int) (*ants.Pool, error) {
	p, err := ants.NewPool(10)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize worker pool: %w", err)
	}

	return p, nil
}
