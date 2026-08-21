package history

import "sync"

type Ring struct {
	mu     sync.RWMutex
	values []float64
	size   int
	next   int
	filled bool
}

func NewRing(size int) *Ring {
	if size < 1 {
		size = 1
	}
	return &Ring{values: make([]float64, size), size: size}
}

func (r *Ring) Add(value float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.values[r.next] = value
	r.next = (r.next + 1) % r.size
	if r.next == 0 {
		r.filled = true
	}
}

func (r *Ring) Values() []float64 {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if !r.filled {
		out := make([]float64, r.next)
		copy(out, r.values[:r.next])
		return out
	}

	out := make([]float64, 0, r.size)
	out = append(out, r.values[r.next:]...)
	out = append(out, r.values[:r.next]...)
	return out
}

func (r *Ring) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.filled {
		return r.size
	}
	return r.next
}
