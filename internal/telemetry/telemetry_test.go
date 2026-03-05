package telemetry

import (
	"testing"
)

func TestInitLogger(t *testing.T) {
	InitLogger()
	if Logger == nil {
		t.Fatal("Expected Logger to be initialized")
	}
}

func TestMetricsInitialization(t *testing.T) {
	// Simple sanity check to ensure metrics objects are created
	if HttpRequestsTotal == nil {
		t.Error("Expected HttpRequestsTotal to be initialized")
	}
	if ActiveBackends == nil {
		t.Error("Expected ActiveBackends to be initialized")
	}
}
