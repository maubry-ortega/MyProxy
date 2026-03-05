package health

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"MyProxy/internal/telemetry"
)

func TestMonitor(t *testing.T) {
	telemetry.InitLogger()

	// 1. Setup a mock healthy server
	healthyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer healthyServer.Close()

	// 2. Setup a mock unhealthy server
	unhealthyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer unhealthyServer.Close()

	targets := map[string][]*Target{
		"test.my.os": {
			{URL: healthyServer.URL, Healthy: false},
			{URL: unhealthyServer.URL, Healthy: true},
		},
	}

	// Run monitor for a brief moment
	// We use a small interval but manually trigger to keep test fast
	go Monitor(func() map[string][]*Target { return targets }, 100*time.Millisecond)

	// Wait for monitor to run at least once
	time.Sleep(300 * time.Millisecond)

	if !targets["test.my.os"][0].Healthy {
		t.Error("Expected healthy server to be marked Healthy")
	}
	if targets["test.my.os"][1].Healthy {
		t.Error("Expected unhealthy server to be marked Not Healthy")
	}
}
