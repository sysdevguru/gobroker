package sources

import (
	"sync"
)

type Key string

const (
	IEX Key = "iex"
	SIP Key = "sip"
)

type Sources struct {
	sync.Mutex
	m map[Key]*Source
}

func (s *Sources) Slice() []*Source {
	srcs := make([]*Source, len(s.m))
	i := 0

	for _, src := range s.m {
		srcs[i] = src
		i++
	}

	return srcs
}

func (s *Sources) Register(key Key, src *Source) {
	if s.m == nil {
		s.m = make(map[Key]*Source)
	}

	s.m[key] = src
}

func (s *Sources) Get(key Key) *Source {
	if s.m == nil {
		return nil
	}

	return s.m[key]
}

type Source struct {
	http   func(string, interface{}) (interface{}, error)
	stream func()
}

func (s *Source) HTTP(name string, args interface{}) (interface{}, error) {
	return s.http(name, args)
}

func (s *Source) Stream() {
	if s.stream != nil {
		s.stream()
	}
}

type Streamer interface {
	Stream()
}

func NewSource(http func(string, interface{}) (interface{}, error), stream func()) *Source {
	return &Source{http: http, stream: stream}
}
