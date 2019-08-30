package set

import "sync"

var okay = struct{}{}

type set struct {
	mu sync.RWMutex
	m  map[string]struct{}
}

func New() *set {
	s := &set{}
	s.m = make(map[string]struct{})
	return s
}

func (s *set) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.m)
}

func (s *set) Add(value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[value] = okay
}

func (s *set) Remove(value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, value)
}

func (s *set) Contains(value string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, c := s.m[value]
	return c
}

func (s *set) List() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	li := make([]string, len(s.m))
	i := 0
	for key := range s.m {
		li[i] = key
		i++
	}
	return li
}
