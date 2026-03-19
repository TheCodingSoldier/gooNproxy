package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type hop struct {
	IP  string `json:"ip"`
	MAC string `json:"mac"`
}

type hopChain struct {
	Hops []hop `json:"hops"`
}

type randomizer struct {
	r *rand.Rand
}

func newRandomizer(seed int64) *randomizer {
	return &randomizer{r: rand.New(rand.NewSource(seed))}
}

func (r *randomizer) buildChain(count int) hopChain {
	if count < 1 {
		count = 1
	}
	hops := make([]hop, 0, count)
	for i := 0; i < count; i++ {
		hops = append(hops, hop{
			IP:  fmt.Sprintf("%d.%d.%d.%d", r.r.Intn(223)+1, r.r.Intn(256), r.r.Intn(256), r.r.Intn(256)),
			MAC: fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", r.r.Intn(256), r.r.Intn(256), r.r.Intn(256), r.r.Intn(256), r.r.Intn(256), r.r.Intn(256)),
		})
	}
	return hopChain{Hops: hops}
}

type app struct {
	client       *http.Client
	randomSource *randomizer
}

const page = `<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>gooNproxy</title>
  <style>
    body { font-family: Arial, sans-serif; margin: 2rem; max-width: 900px; }
    .card { border: 1px solid #ddd; border-radius: 8px; padding: 1rem; margin-top: 1rem; }
    input[type=text] { width: 70%; padding: .5rem; }
    button { padding: .5rem .8rem; }
    pre { white-space: pre-wrap; word-break: break-word; background:#fafafa; padding: 1rem; border-radius: 8px; }
  </style>
</head>
<body>
  <h1>gooNproxy</h1>
  <p>Privacy-focused search proxy with simulated multi-hop IP/MAC randomization.</p>
  <form method="get" action="/search">
    <label for="q">Search</label><br>
    <input id="q" name="q" type="text" value="{{ .Query }}" placeholder="type your query">
    <button type="submit">Search via proxy</button>
  </form>
  {{ if .ShowResult }}
  <div class="card">
    <h2>Search Result (DuckDuckGo via proxy)</h2>
    <p><strong>Target:</strong> {{ .TargetURL }}</p>
    <h3>Simulated Multi-hop Identity Chain</h3>
    <pre>{{ .ChainJSON }}</pre>
    <h3>Response Preview</h3>
    <pre>{{ .ResultPreview }}</pre>
  </div>
  {{ end }}
</body>
</html>`

var tmpl = template.Must(template.New("page").Parse(page))

type pageData struct {
	Query         string
	ShowResult    bool
	TargetURL     string
	ChainJSON     string
	ResultPreview string
}

func (a *app) index(w http.ResponseWriter, r *http.Request) {
	if err := tmpl.Execute(w, pageData{}); err != nil {
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
}

func (a *app) search(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		http.Error(w, "missing query parameter q", http.StatusBadRequest)
		return
	}

	target := "https://duckduckgo.com/html/?q=" + url.QueryEscape(query)
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, target, nil)
	if err != nil {
		http.Error(w, "failed to create upstream request", http.StatusInternalServerError)
		return
	}
	req.Header.Set("User-Agent", "gooNproxy/1.0 (+privacy-search-proxy)")

	chain := a.randomSource.buildChain(3)
	chainBytes, err := json.MarshalIndent(chain, "", "  ")
	if err != nil {
		http.Error(w, "failed to build randomized chain", http.StatusInternalServerError)
		return
	}
	resultPreview := ""

	res, err := a.client.Do(req)
	if err != nil {
		resultPreview = "upstream request failed: " + err.Error()
	} else {
		defer res.Body.Close()
		body, readErr := io.ReadAll(io.LimitReader(res.Body, 3000))
		if readErr != nil {
			resultPreview = "upstream response read failed: " + readErr.Error()
		} else {
			resultPreview = string(body)
		}
	}

	data := pageData{
		Query:         query,
		ShowResult:    true,
		TargetURL:     target,
		ChainJSON:     string(chainBytes),
		ResultPreview: resultPreview,
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "failed to render search result", http.StatusInternalServerError)
	}
}

func (a *app) apiChain(w http.ResponseWriter, r *http.Request) {
	hops := 3
	if val := strings.TrimSpace(r.URL.Query().Get("hops")); val != "" {
		parsed, err := strconv.Atoi(val)
		if err != nil || parsed < 1 || parsed > 10 {
			http.Error(w, "hops must be an integer between 1 and 10", http.StatusBadRequest)
			return
		}
		hops = parsed
	}
	chain := a.randomSource.buildChain(hops)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(chain); err != nil {
		http.Error(w, "failed to encode randomized chain", http.StatusInternalServerError)
	}
}

func routes(now time.Time) http.Handler {
	a := &app{
		client:       &http.Client{Timeout: 15 * time.Second},
		randomSource: newRandomizer(now.UnixNano()),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", a.index)
	mux.HandleFunc("/search", a.search)
	mux.HandleFunc("/api/random-chain", a.apiChain)
	return mux
}

func main() {
	server := &http.Server{
		Addr:              ":8080",
		Handler:           routes(time.Now()),
		ReadHeaderTimeout: 5 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "server failed: %v\n", err)
	}
}
