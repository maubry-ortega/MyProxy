package middleware

import (
	"net"
	"net/http"
	"strings"
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

func ApplySecurityHeaders(w http.ResponseWriter, r *http.Request) {
	host := r.Host
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}

	// Only apply HSTS if NOT an IP address, NOT localhost, and NOT internal .my.os domains
	if net.ParseIP(host) == nil && host != "localhost" && !strings.HasSuffix(host, ".local") && !strings.HasSuffix(host, ".my.os") {
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
	}

	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
	w.Header().Set("Content-Security-Policy", "default-src 'self';")
}
