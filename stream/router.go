package stream

import (
	"context"
	"strings"
	"sync"
)

var router *Router

// Router manages the relationship betweens streams and listeners
type Router struct {
	sync.RWMutex
	cancel             context.CancelFunc
	streamsToListeners *routeMap
	listenersToStreams *routeMap
}

// NewRouter allocates the required maps to build a new
// stream router
func NewRouter() *Router {
	return &Router{
		streamsToListeners: newRouteMap(),
		listenersToStreams: newRouteMap(),
	}
}

type routeMap struct {
	m map[interface{}]interface{}
}

func newRouteMap() *routeMap {
	return &routeMap{m: map[interface{}]interface{}{}}
}

func (rm *routeMap) Empty() bool {
	return len(rm.m) == 0
}

func (rm *routeMap) Store(k, v interface{}) {
	rm.m[k] = v
}

func (rm *routeMap) Delete(k interface{}) {
	delete(rm.m, k)
}

func (rm *routeMap) LoadOrStore(k, v interface{}) (val interface{}, loaded bool) {
	val, loaded = rm.m[k]
	if !loaded {
		rm.m[k] = v
		return v, loaded
	}
	return val, loaded
}

func (rm *routeMap) Load(k interface{}) (v interface{}, loaded bool) {
	v, loaded = rm.m[k]
	return v, loaded
}

func (rm *routeMap) Range(f func(key, value interface{}) bool) {
	for k, v := range rm.m {
		if !f(k, v) {
			break
		}
	}
}

// Update updates the stream router by updating its internal maps to maintain
// the connection <-> stream state of the streaming interface. This is a
// thread-safe locking call to the router.
func (r *Router) Update(l *Listener, streams []string) {
	r.Lock()
	defer r.Unlock()
	// clean out streams the listener no longer cares about
	for _, stream := range r.getStreams(l) {
		remove := true
		for _, s := range streams {
			if strings.EqualFold(stream, s) {
				remove = false
				break
			}
		}
		if remove {
			r.removeStream(stream, l)
		}
	}
	// add the streams the listener does care about
	sM := newRouteMap()
	for _, stream := range streams {
		lM := newRouteMap()
		lM.Store(l, struct{}{})
		if v, loaded := r.streamsToListeners.LoadOrStore(stream, lM); loaded {
			lM = v.(*routeMap)
			lM.Store(l, struct{}{})
			r.streamsToListeners.Store(stream, lM)
		}
		sM.Store(stream, struct{}{})
	}
	if v, loaded := r.listenersToStreams.LoadOrStore(l, sM); loaded {
		sM = v.(*routeMap)
		for _, stream := range streams {
			sM.Store(stream, struct{}{})
			r.listenersToStreams.Store(l, sM)
		}
		r.listenersToStreams.Store(l, sM)
	}
}

// GetListeners is a thread-safe call to retrives the listeners
// for a given stream
func (r *Router) GetListeners(stream string) []*Listener {
	r.RLock()
	defer r.RUnlock()
	return r.getListeners(stream)
}

// getListeners returns all listeners listening to the specified stream.
// this call is not in itself thread-safe.
func (r *Router) getListeners(stream string) (listeners []*Listener) {
	if m, ok := r.streamsToListeners.Load(stream); ok {
		if m != nil {
			listeners = make([]*Listener, 0)
			m.(*routeMap).Range(func(key, value interface{}) bool {
				listeners = append(listeners, key.(*Listener))
				return true
			})
		}
	}
	return listeners
}

// GetStreams is a thread-safe call to retrives the streams
// for a given listener
func (r *Router) GetStreams(l *Listener) []string {
	r.RLock()
	defer r.RUnlock()
	return r.getStreams(l)
}

// getStreams returns all streams that the specified listener is listening to
func (r *Router) getStreams(l *Listener) (streams []string) {
	if m, ok := r.listenersToStreams.Load(l); ok {
		if m != nil {
			streams = make([]string, 0)
			m.(*routeMap).Range(func(key, value interface{}) bool {
				streams = append(streams, key.(string))
				return true
			})
		}
	}
	return streams
}

func (r *Router) removeStream(stream string, l *Listener) {
	if v, ok := r.streamsToListeners.Load(stream); ok {
		lM := v.(*routeMap)
		lM.Delete(l)
		if lM.Empty() {
			r.streamsToListeners.Delete(stream)
		} else {
			r.streamsToListeners.Store(stream, lM)
		}
	}
	if v, ok := r.listenersToStreams.Load(l); ok {
		sM := v.(*routeMap)
		sM.Delete(stream)
		if sM.Empty() {
			r.listenersToStreams.Delete(l)
		} else {
			r.listenersToStreams.Store(l, sM)
		}
	}
}
