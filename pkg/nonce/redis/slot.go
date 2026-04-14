package redis

import (
	"context"
	"sync/atomic"
	"time"
)

type slot struct {
	manager  *NonceManager
	resolved atomic.Bool
}

func (s *slot) Commit() {
	if !s.resolved.CompareAndSwap(false, true) {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = s.manager.releaseLock(ctx)
}

func (s *slot) Reuse() {
	if !s.resolved.CompareAndSwap(false, true) {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = s.manager.rdb.Decr(ctx, s.manager.counterKey())
	_ = s.manager.releaseLock(ctx)
}

func (s *slot) Reclaim() {
	if !s.resolved.CompareAndSwap(false, true) {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	s.manager.dirty.Store(true)
	_ = s.manager.releaseLock(ctx)
}
