# Holt in the browser (`holt-web`)

`holt-web` serves the full holt TUI in a web browser. It runs `holt` inside a
real pseudo-terminal (PTY) and streams it to the page over a WebSocket, rendered
with [xterm.js](https://xtermjs.org). Anything you can do in the terminal app you
can do in the browser tab.

```
browser (xterm.js)  ⇄  WebSocket  ⇄  holt-web  ⇄  PTY  ⇄  holt
```

## ⚠️ It cannot run on Vercel (or any serverless host)

`holt-web` needs a **persistent process**, a **real PTY**, a **long-lived
WebSocket**, and the ability to **spawn the `holt` binary**. Serverless
platforms such as Vercel, Netlify Functions, or Cloudflare Workers provide none
of these — a deploy there returns `404: NOT_FOUND` because there is no static
site to serve. Deploy to a **container host** instead: Railway, Fly.io, Render,
Google Cloud Run, or any Docker/VPS.

## ⚠️ Security

`holt-web` hands every visitor a live agent that can run shell commands and edit
files using your provider API keys. Before exposing it:

- **Always set an auth token** — `HOLT_WEB_TOKEN` (or `--token`). Without it the
  server prints a warning and lets anyone in.
- **Run it isolated** — inside the container and a disposable `/workspace`, never
  pointed at a host directory you care about. The provided `Dockerfile` runs as a
  non-root user in `/workspace`.
- Treat the URL like a credential and prefer putting it behind your own auth /
  network policy for anything beyond a personal demo.

## Run locally

```bash
go build -o holt ./cmd/holt
go build -o holt-web ./cmd/holt-web
HOLT_WEB_TOKEN=dev ./holt-web --holt ./holt
# open http://localhost:8080/?token=dev
```

Flags (all also read an env var):

| Flag | Env | Default | Purpose |
|------|-----|---------|---------|
| `--addr` | `PORT` | `:8080` | Listen address (a bare `PORT` becomes `:PORT`). |
| `--token` | `HOLT_WEB_TOKEN` | _(none)_ | Shared token required to open a session. |
| `--holt` | `HOLT_WEB_BIN` | `holt` | Path to the holt binary. |
| `--workdir` | `HOLT_WEB_WORKDIR` | cwd | Directory holt runs in. |

Any trailing arguments are passed through to `holt` (e.g. `holt-web -- --provider openrouter`).

## Run with Docker

```bash
docker build -t holt-web .
docker run --rm -p 8080:8080 \
  -e HOLT_WEB_TOKEN=change-me \
  -e OPENROUTER_API_KEY=sk-... \
  holt-web
# open http://localhost:8080/?token=change-me
```

## Deploy to a container host

The image listens on `$PORT` (default `8080`), which Railway, Render, and Cloud
Run set automatically.

- **Railway / Render**: create a service from this repo; both auto-detect the
  `Dockerfile`. Set `HOLT_WEB_TOKEN` and a provider key (e.g. `OPENROUTER_API_KEY`)
  in the service variables.
- **Fly.io**: `fly launch` detects the `Dockerfile`; set secrets with
  `fly secrets set HOLT_WEB_TOKEN=... OPENROUTER_API_KEY=...`.

Then open `https://<your-app>/?token=<HOLT_WEB_TOKEN>`.
