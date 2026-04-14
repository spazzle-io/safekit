package local

import "sync/atomic"

type slot struct {
	manager  *NonceManager
	resolved atomic.Bool
}

func (s *slot) Commit() {
	if !s.resolved.CompareAndSwap(false, true) {
		return
	}

	s.manager.inflightCh <- struct{}{}
}

func (s *slot) Reuse() {
	if !s.resolved.CompareAndSwap(false, true) {
		return
	}

	s.manager.mu.Lock()
	if s.manager.hasNonce {
		s.manager.nonce--
	}
	s.manager.mu.Unlock()

	s.manager.inflightCh <- struct{}{}
}

func (s *slot) Reclaim() {
	if !s.resolved.CompareAndSwap(false, true) {
		return
	}

	s.manager.mu.Lock()
	s.manager.dirty = true
	s.manager.mu.Unlock()

	s.manager.inflightCh <- struct{}{}
}
