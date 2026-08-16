package auth

import (
	"context"
	"sync"
	"time"
)

// LoginLimiter is a fixed-window failure counter keyed by client IP. Only
// failed attempts count, so a working login is never throttled.
type LoginLimiter struct {
	mu      sync.Mutex
	max     int
	window  time.Duration
	clients map[string]*failureWindow
}

type failureWindow struct {
	failures int
	resetAt  time.Time
}

func NewLoginLimiter(max int, window time.Duration) *LoginLimiter {
	return &LoginLimiter{
		max:     max,
		window:  window,
		clients: make(map[string]*failureWindow),
	}
}

func (l *LoginLimiter) Allowed(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	w, ok := l.clients[ip]
	if !ok || time.Now().After(w.resetAt) {
		return true
	}
	return w.failures < l.max
}

func (l *LoginLimiter) RecordFailure(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	w, ok := l.clients[ip]
	if !ok || time.Now().After(w.resetAt) {
		l.clients[ip] = &failureWindow{failures: 1, resetAt: time.Now().Add(l.window)}
		return
	}
	w.failures++
}

func (l *LoginLimiter) Reset(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.clients, ip)
}

func (l *LoginLimiter) Sweep(ctx context.Context, every time.Duration) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			l.mu.Lock()
			for ip, w := range l.clients {
				if now.After(w.resetAt) {
					delete(l.clients, ip)
				}
			}
			l.mu.Unlock()
		}
	}
}
