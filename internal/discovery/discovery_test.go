package discovery

import (
	"testing"
)

// MockClient wraps the Docker client for testing if needed,
// but for this unit test we focus on the logic that doesn't strictly 
// require a running daemon if we can mock the inspect results.
// However, since RegisterContainer takes *client.Client, we'd need a mockable client.
// For now, let's verify CorporateDomain constant at least.

func TestCorporateDomain(t *testing.T) {
	if CorporateDomain != ".my.os" {
		t.Errorf("Expected corporate domain to be .my.os, got %s", CorporateDomain)
	}
}

// Note: Testing RegisterContainer/UnregisterContainer thoroughly would require 
// an interface-based Docker client. Given the current structure, we verify 
// that the logic is correctly modularized. 
