# Browser terminal for holt: builds the holt CLI and the holt-web PTY↔WebSocket
# server, then serves the full holt TUI in a browser via xterm.js.
#
# This image runs a persistent process with a real PTY, so it deploys to a
# container host (Railway, Fly.io, Render, or any Docker/VPS) — NOT to
# serverless platforms such as Vercel.
FROM golang:1.26-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/holt ./cmd/holt \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/holt-web ./cmd/holt-web

FROM debian:bookworm-slim
# The agent shells out, so the runtime needs git, a shell and search tooling.
RUN apt-get update \
 && apt-get install -y --no-install-recommends git ca-certificates ripgrep curl \
 && rm -rf /var/lib/apt/lists/* \
 && useradd -m -u 10001 holt \
 && mkdir -p /workspace && chown holt:holt /workspace
COPY --from=build /out/holt /usr/local/bin/holt
COPY --from=build /out/holt-web /usr/local/bin/holt-web
USER holt
WORKDIR /workspace
# Provider keys (e.g. ANTHROPIC_API_KEY / OPENROUTER_API_KEY) and HOLT_WEB_TOKEN
# are supplied at runtime by the host, not baked into the image.
ENV HOLT_WEB_WORKDIR=/workspace
EXPOSE 8080
ENTRYPOINT ["holt-web"]
