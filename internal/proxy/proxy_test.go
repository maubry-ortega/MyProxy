package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"MyProxy/internal/telemetry"
)

func TestMain(m *testing.M) {
	telemetry.InitLogger()
	m.Run()
}

func TestCoreFunctionality(t *testing.T) {
	// Create a local backend server to avoid timeouts
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	r := NewRouter()
	domain := "app.my.os"
	
	t.Run("AddRoute", func(t *testing.T) {
		r.AddRoute(domain, backend.URL)
		if _, ok := r.Routes[domain]; !ok {
			t.Errorf("Expected route for %s to exist", domain)
		}
	})

	t.Run("RoundRobin", func(t *testing.T) {
		r.AddRoute(domain, "http://127.0.0.1:9999") // Unhealthy one
		t1 := r.Routes[domain].NextTarget()
		t2 := r.Routes[domain].NextTarget()
		if t1 == t2 {
			t.Errorf("Round Robin failed: got same target twice %s", t1)
		}
	})

	t.Run("HealthCheckExclusion", func(t *testing.T) {
		// Target 0 is backend.URL (Healthy)
		// Target 1 is 127.0.0.1:9999 (Unhealthy by default in this manual check)
		r.Routes[domain].Targets[0].Healthy = true
		r.Routes[domain].Targets[1].Healthy = false
		
		for i := 0; i < 5; i++ {
			target := r.Routes[domain].NextTarget()
			if target != backend.URL {
				t.Errorf("Expected only healthy target %s, got %s", backend.URL, target)
			}
		}
	})

	t.Run("SecurityHeadersAndRouting", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "http://app.my.os", nil)
		r.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d", w.Code)
		}

		headers := []string{"Strict-Transport-Security", "X-Frame-Options", "X-Content-Type-Options"}
		for _, h := range headers {
			if w.Header().Get(h) == "" {
				t.Errorf("Missing security header: %s", h)
			}
		}
	})

	t.Run("RateLimitingIP", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://app.my.os", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		
		// Fill burst (20)
		for i := 0; i < 20; i++ {
			r.ServeHTTP(httptest.NewRecorder(), req)
		}
		
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusTooManyRequests {
			// Trigger more if needed (depending on fill rate during test)
			for i := 0; i < 10; i++ {
				r.ServeHTTP(httptest.NewRecorder(), req)
			}
			w = httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != http.StatusTooManyRequests {
				t.Errorf("Expected 429 after burst, got %d", w.Code)
			}
		}
	})
}

func TestRemoveTarget(t *testing.T) {
	r := NewRouter()
	domain := "remove.my.os"
	target := "http://1.1.1.1:3000"
	
	r.AddRoute(domain, target)
	r.RemoveTarget(domain, target)
	
	if _, ok := r.Routes[domain]; ok {
		t.Error("Expected route to be deleted when last target removed")
	}
}


