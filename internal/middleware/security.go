package middleware

import (
	"net/http"
	"sync"

	"golang.org/x/time/rate"
)

type IPLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.Mutex
}

func NewIPLimiter() *IPLimiter {
	return &IPLimiter{
		limiters: make(map[string]*rate.Limiter),
	}
}

func (l *IPLimiter) Allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	limiter, ok := l.limiters[ip]
	if !ok {
		limiter = rate.NewLimiter(10, 20)
		l.limiters[ip] = limiter
	}

	return limiter.Allow()
}

func ApplySecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
	w.Header().Set("Content-Security-Policy", "default-src 'self';")
}
