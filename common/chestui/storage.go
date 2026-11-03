package chestui

// HasViewers reports whether a player has the storage's window open. Callers
// that only keep the window's display items up to date (progress, gauges) can
// skip that work while it is false.
func (s *Storage) HasViewers() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.viewers) > 0
}
