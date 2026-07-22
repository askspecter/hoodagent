# Holt in the browser (`holt-web`)

`holt-web` serves the full holt TUI in a web browser. It runs `holt` inside a
real pseudo-terminal (PTY) and streams it to the page over a WebSocket, rendered
with [xterm.js](https://xtermjs.org). The page ships a mobile key bar (ESC, CTRL,
ALT, TAB, arrows, copy/paste, mic) so a phone can drive the TUI too.

```
browser (xterm.js)  ⇄  WebSocket  ⇄  holt-web  ⇄  PTY  ⇄  holt
```

## Two parts: static frontend + a backend

- **Frontend** — the static page in [`cmd/holt-web/web/`](../cmd/holt-web/web).
  Just HTML/JS, so it deploys anywhere static, **including Vercel**.
- **Backend** — the `holt-web` server. It holds the persistent process, the PTY,
  and the WebSocket, so it must run on a **container host** (Railway, Fly.io,
  Render, Cloud Run, any Docker/VPS) — **not** on Vercel/serverless, which can't
  keep a long-lived process or PTY.

You can run them together (the `holt-web` binary serves its own page — simplest)
or split them (static frontend on Vercel, backend on a container). Split is how a
frontend on a serverless host reaches the agent: the page just points its
WebSocket at the backend.

## ⚠️ Security

`holt-web` hands every visitor a live agent that can run shell commands and edit
files using your provider API keys. Before exposing it:

- **Always set an auth token** — `HOLT_WEB_TOKEN` (or `--token`). Without it the
  server prints a warning and lets anyone in.
- **Run it isolated** — inside the container and a disposable `/workspace`, never
  pointed at a host directory you care about. The `Dockerfile` runs as a non-root
  user in `/workspace`.
- Treat the URL (and token) like a credential.

## Run locally (all-in-one)

```bash
go build -o holt ./cmd/holt
go build -o holt-web ./cmd/holt-web
HOLT_WEB_TOKEN=dev ./holt-web --holt ./holt
# open http://localhost:8080/?token=dev
```

Flags (each also reads an env var):

| Flag | Env | Default | Purpose |
|------|-----|---------|---------|
| `--addr` | `PORT` | `:8080` | Listen address (a bare `PORT` becomes `:PORT`). |
| `--token` | `HOLT_WEB_TOKEN` | _(none)_ | Shared token required to open a session. |
| `--holt` | `HOLT_WEB_BIN` | `holt` | Path to the holt binary. |
| `--workdir` | `HOLT_WEB_WORKDIR` | cwd | Directory holt runs in. |

## Deploy the backend (Docker)

```bash
docker build -t holt-web .
docker run --rm -p 8080:8080 \
  -e HOLT_WEB_TOKEN=change-me \
  -e OPENROUTER_API_KEY=sk-... \
  holt-web
```

The image listens on `$PORT` (default `8080`), which Railway, Render, and Cloud
Run set automatically. On Fly.io use `fly launch` (it detects the `Dockerfile`)
and `fly secrets set HOLT_WEB_TOKEN=... OPENROUTER_API_KEY=...`.

## Deploy the frontend to Vercel (optional split)

Point a Vercel project at [`cmd/holt-web/web/`](../cmd/holt-web/web) (set it as the
**Root Directory**; it's a static site with a `vercel.json`). The page finds its
backend in this order:

1. `?backend=wss://your-backend.fly.dev` — per-visit override.
2. `window.HOLT_BACKEND` — bake it in by adding a one-line `config.js`
   (`window.HOLT_BACKEND = "wss://your-backend.fly.dev"`) and a
   `<script src="config.js"></script>` before the main script.
3. Same origin — used when `holt-web` serves the page itself (all-in-one).

So a Vercel-hosted page opens as:

```
https://your-frontend.vercel.app/?backend=wss://your-backend.fly.dev&token=change-me
```

The backend's WebSocket accepts cross-origin connections, so the token is the
access control — keep it secret.
