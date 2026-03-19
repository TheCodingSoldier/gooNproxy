# gooNproxy

`gooNproxy` is a lightweight privacy-focused web search proxy you can run as a website.

It provides:

- A simple web UI for privacy-oriented search
- A built-in search field on the homepage (similar to CroxyProxy-style usage as a website)
- An in-browser HTML5 random chain generator backed by `/api/random-chain`
- A proxy route that fetches allowed search engine pages server-side
- A simulated multi-hop IP/MAC randomizer chain for each search request

## Run locally

```bash
go run .
```

Then open [http://localhost:8080](http://localhost:8080).

## Publish as a website

Deploy this app to any host that can run Go services (for example a VM, container platform, or reverse-proxy setup), then set:

```bash
GOONPROXY_PUBLIC_URL=https://your-domain.example go run .
```

When `GOONPROXY_PUBLIC_URL` is set, the homepage shows a clickable **Public website** link so users can open and use the hosted site directly in a browser.

### Optional Tor proxy

To route outbound search requests through Tor, set `GOONPROXY_TOR_PROXY` to a Tor SOCKS endpoint before starting the app:

```bash
GOONPROXY_TOR_PROXY=socks5h://127.0.0.1:9050 go run .
```

## Test

```bash
go test ./...
```
