package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

const testTorProxyURL = "socks5h://127.0.0.1:9050"

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

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
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("ok")),
				Header:     make(http.Header),
			}, nil
		}),
	}
	a := &app{client: client, randomSource: newRandomizer(1)}
	req := httptest.NewRequest(http.MethodGet, "/search", nil)
	rec := httptest.NewRecorder()

	a.search(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Please enter a search query.") {
		t.Fatalf("expected friendly website error, got body: %s", rec.Body.String())
	}
}

func TestIndexIncludesWebsiteRandomChainTool(t *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("ok")),
				Header:     make(http.Header),
			}, nil
		}),
	}
	a := &app{client: client, randomSource: newRandomizer(1)}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	a.index(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Random Chain (Website Tool)") {
		t.Fatalf("expected random chain tool in page, got body: %s", body)
	}
	if !strings.Contains(body, "id=\"generateChain\"") {
		t.Fatalf("expected generate chain button in page, got body: %s", body)
	}
}

func TestAPIChainHopCounts(t *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("ok")),
				Header:     make(http.Header),
			}, nil
		}),
	}
	a := &app{client: client, randomSource: newRandomizer(1)}

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

	t.Run("five_hops", func(t *testing.T) {
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

	t.Run("custom", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/random-chain?hops=7", nil)
		rec := httptest.NewRecorder()

		a.apiChain(rec, req)

		var chain hopChain
		if err := json.Unmarshal(rec.Body.Bytes(), &chain); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if len(chain.Hops) != 7 {
			t.Fatalf("expected 7 hops, got %d", len(chain.Hops))
		}
	})

	t.Run("invalid", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/random-chain?hops=0", nil)
		rec := httptest.NewRecorder()

		a.apiChain(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})
}

func TestNewHTTPClientTorProxyConfig(t *testing.T) {
	t.Run("uses_tor_proxy_when_valid", func(t *testing.T) {
		t.Setenv(torProxyEnv, testTorProxyURL)

		client := newHTTPClient()
		transport, ok := client.Transport.(*http.Transport)
		if !ok {
			t.Fatalf("expected *http.Transport, got %T", client.Transport)
		}

		req := httptest.NewRequest(http.MethodGet, "https://duckduckgo.com", nil)
		proxyURL, err := transport.Proxy(req)
		if err != nil {
			t.Fatalf("unexpected proxy error: %v", err)
		}
		if proxyURL == nil {
			t.Fatal("expected tor proxy URL, got nil")
		}
		if proxyURL.String() != testTorProxyURL {
			t.Fatalf("expected tor proxy URL, got %q", proxyURL.String())
		}
	})

	t.Run("falls_back_to_env_proxy_when_invalid", func(t *testing.T) {
		t.Setenv(torProxyEnv, "")
		baselineClient := newHTTPClient()
		baselineTransport, ok := baselineClient.Transport.(*http.Transport)
		if !ok {
			t.Fatalf("expected *http.Transport, got %T", baselineClient.Transport)
		}
		req := httptest.NewRequest(http.MethodGet, "https://duckduckgo.com", nil)
		baselineProxyURL, baselineErr := baselineTransport.Proxy(req)
		if baselineErr != nil {
			t.Fatalf("unexpected baseline proxy error: %v", baselineErr)
		}

		t.Setenv(torProxyEnv, "http://127.0.0.1:9050")

		client := newHTTPClient()
		transport, ok := client.Transport.(*http.Transport)
		if !ok {
			t.Fatalf("expected *http.Transport, got %T", client.Transport)
		}

		proxyURL, err := transport.Proxy(req)
		if err != nil {
			t.Fatalf("unexpected proxy error: %v", err)
		}
		if (baselineProxyURL == nil) != (proxyURL == nil) {
			t.Fatalf("expected fallback proxy nil=%v, got nil=%v", baselineProxyURL == nil, proxyURL == nil)
		}
		if baselineProxyURL != nil && baselineProxyURL.String() != proxyURL.String() {
			t.Fatalf("expected fallback proxy URL %q, got %q", baselineProxyURL.String(), proxyURL.String())
		}
	})

	t.Run("falls_back_to_env_proxy_when_unparseable", func(t *testing.T) {
		t.Setenv(torProxyEnv, "")
		baselineClient := newHTTPClient()
		baselineTransport, ok := baselineClient.Transport.(*http.Transport)
		if !ok {
			t.Fatalf("expected *http.Transport, got %T", baselineClient.Transport)
		}
		req := httptest.NewRequest(http.MethodGet, "https://duckduckgo.com", nil)
		baselineProxyURL, baselineErr := baselineTransport.Proxy(req)
		if baselineErr != nil {
			t.Fatalf("unexpected baseline proxy error: %v", baselineErr)
		}

		t.Setenv(torProxyEnv, "://bad")

		client := newHTTPClient()
		transport, ok := client.Transport.(*http.Transport)
		if !ok {
			t.Fatalf("expected *http.Transport, got %T", client.Transport)
		}

		proxyURL, err := transport.Proxy(req)
		if err != nil {
			t.Fatalf("unexpected proxy error: %v", err)
		}
		if (baselineProxyURL == nil) != (proxyURL == nil) {
			t.Fatalf("expected fallback proxy nil=%v, got nil=%v", baselineProxyURL == nil, proxyURL == nil)
		}
		if baselineProxyURL != nil && baselineProxyURL.String() != proxyURL.String() {
			t.Fatalf("expected fallback proxy URL %q, got %q", baselineProxyURL.String(), proxyURL.String())
		}
	})
}
