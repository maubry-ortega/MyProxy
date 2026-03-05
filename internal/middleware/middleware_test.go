package middleware

import (
	"net/http/httptest"
	"testing"
)

func TestSecurityHeaders(t *testing.T) {
	w := httptest.NewRecorder()
	ApplySecurityHeaders(w)

	expectedHeaders := []string{
		"Strict-Transport-Security",
		"X-Frame-Options",
		"X-Content-Type-Options",
		"X-XSS-Protection",
		"Referrer-Policy",
		"Content-Security-Policy",
	}

	for _, h := range expectedHeaders {
		if w.Header().Get(h) == "" {
			t.Errorf("Missing security header: %s", h)
		}
	}
}

func TestIPLimiter(t *testing.T) {
	l := NewIPLimiter()
	ip := "127.0.0.1"

	// Trigger burst limit (default 10 qps, 20 burst)
	for i := 0; i < 20; i++ {
		if !l.Allow(ip) {
			t.Errorf("Request %d should have been allowed", i)
		}
	}

	if l.Allow(ip) {
		t.Error("Request 21 should have been blocked (exceeded burst)")
	}

	// Different IP should be allowed
	if !l.Allow("1.1.1.1") {
		t.Error("Request from different IP should be allowed")
	}
}
