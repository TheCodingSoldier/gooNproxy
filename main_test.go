package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

func TestBuildChain(t *testing.T) {
	r := newRandomizer(42)
	chain := r.buildChain(3)
	if len(chain.Hops) != 3 {
		t.Fatalf("expected 3 hops, got %d", len(chain.Hops))
	}

	ipPattern := regexp.MustCompile(`^\d{1,3}(\.\d{1,3}){3}$`)
	macPattern := regexp.MustCompile(`^[0-9a-f]{2}(:[0-9a-f]{2}){5}$`)
	for _, h := range chain.Hops {
		if !ipPattern.MatchString(h.IP) {
			t.Fatalf("unexpected IP format: %s", h.IP)
		}
		if !macPattern.MatchString(h.MAC) {
			t.Fatalf("unexpected MAC format: %s", h.MAC)
		}
	}
}

func TestSearchMissingQuery(t *testing.T) {
	a := &app{client: http.DefaultClient, randomSource: newRandomizer(1)}
	req := httptest.NewRequest(http.MethodGet, "/search", nil)
	rec := httptest.NewRecorder()

	a.search(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestAPIChainHopCounts(t *testing.T) {
	a := &app{client: http.DefaultClient, randomSource: newRandomizer(1)}

	t.Run("default", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/random-chain", nil)
		rec := httptest.NewRecorder()

		a.apiChain(rec, req)

		var chain hopChain
		if err := json.Unmarshal(rec.Body.Bytes(), &chain); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if len(chain.Hops) != 3 {
			t.Fatalf("expected default 3 hops, got %d", len(chain.Hops))
		}
	})

	t.Run("five", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/random-chain?hops=5", nil)
		rec := httptest.NewRecorder()

		a.apiChain(rec, req)

		var chain hopChain
		if err := json.Unmarshal(rec.Body.Bytes(), &chain); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if len(chain.Hops) != 5 {
			t.Fatalf("expected 5 hops, got %d", len(chain.Hops))
		}
	})
}
