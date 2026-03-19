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

## Test

```bash
go test ./...
```
