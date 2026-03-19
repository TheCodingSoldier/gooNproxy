# gooNproxy

`gooNproxy` is a lightweight privacy-focused web search proxy.

It provides:

- A simple web UI for privacy-oriented search
- A proxy route that fetches allowed search engine pages server-side
- A simulated multi-hop IP/MAC randomizer chain for each search request

## Run

```bash
go run .
```

Then open [http://localhost:8080](http://localhost:8080).

### Optional Tor proxy

To route outbound search requests through Tor, set `GOONPROXY_TOR_PROXY` to a Tor SOCKS endpoint before starting the app:

```bash
GOONPROXY_TOR_PROXY=socks5h://127.0.0.1:9050 go run .
```

## Test

```bash
go test ./...
```
