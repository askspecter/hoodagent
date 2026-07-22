<p align="center">
  <img src="docs/assets/holt-logo.svg" alt="Holt" width="385">
</p>

<p align="center"><strong>A terminal coding agent you own.</strong></p>

<p align="center">
  <a href="LICENSE"><img alt="license" src="https://img.shields.io/badge/license-MIT-blue"></a>
  <img alt="Go 1.25+" src="https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white">
  <img alt="25+ providers" src="https://img.shields.io/badge/providers-25+-34E2EA">
  <br>
  <strong>English</strong> | <a href="README_ZH.md">中文</a>
</p>

Holt is an AI coding agent for your local terminal. It can inspect a repository,
edit files, run commands, use browser/terminal helpers, and keep durable local
sessions while you choose the model and the permission level.

```bash
holt
holt exec "fix the failing test in ./pkg"
holt exec --output-format stream-json < turns.jsonl
```

## Why Holt

- **Use the model you want.** Bring OpenAI, Anthropic, Gemini, Groq, OpenRouter,
  DeepSeek, Mistral, xAI, Qwen, Kimi, GitHub Models, Ollama, LM Studio, or any
  OpenAI-/Anthropic-compatible endpoint.
- **Stay in control.** File writes, shell commands, network access, and
  out-of-workspace writes go through Holt's permission and sandbox policy.
- **Works in the terminal.** The TUI has model/provider pickers, image input,
  slash commands, live plan/tool rendering, scrollback, themes, and resume/fork
  support.
- **Works without the TUI.** `holt exec` is scriptable, supports text/JSON/
  stream-JSON I/O, isolated worktrees, spec-first runs, and meaningful exit
  codes for CI.
- **Keeps context local.** Sessions are stored on disk, searchable, resumable,
  and never uploaded as telemetry by Holt.
- **Extensible when you need it.** Use MCP servers, skills, plugins, hooks, and
  specialist subagents from the same CLI.

## Install

### npm

```bash
npm install -g @askspecter/holt
holt
```

The npm package installs a small wrapper plus the matching Holt binary for your
platform from GitHub Releases. It supports Linux, macOS, and Windows on x64 and
arm64.

### Bun

Bun does not run dependency lifecycle scripts by default, so the `postinstall`
that fetches the Holt binary is skipped and the first run fails with
`No native binary found next to the npm wrapper`.

Install with Bun, then fetch the binary in one of two ways:

```bash
# Option A: run the installer manually
bun add @askspecter/holt
node node_modules/@askspecter/holt/scripts/postinstall.mjs

# Option B: allow the postinstall to run on install
# add to your package.json:  "trustedDependencies": ["@askspecter/holt"]
bun add @askspecter/holt
```

For global installs (`bun add -g @askspecter/holt`), use Option A against the
global install path, or prefer the install scripts below.

### Install scripts

Linux/macOS:

```bash
curl -fsSL https://raw.githubusercontent.com/askspecter/holt/main/scripts/install.sh | bash
```

Windows PowerShell:

```powershell
irm https://raw.githubusercontent.com/askspecter/holt/main/scripts/install.ps1 | iex
```

### From source

Source builds require Go 1.25+.

```bash
git clone https://github.com/askspecter/holt.git
cd holt
go run ./cmd/holt
```

Release installers and the npm wrapper require published GitHub Release assets.
If you are testing before the first public release, build from source:

```bash
go build -o holt ./cmd/holt
```

On Linux, build the sandbox helper too if you want native sandboxing:

```bash
go build -o holt-linux-sandbox ./cmd/holt-linux-sandbox
go build -o holt-seccomp ./cmd/holt-seccomp   # optional compatibility wrapper
```

Put `holt` and `holt-linux-sandbox` in the same directory on `PATH`
(`~/.local/bin` is a good default). macOS does not need an extra helper binary.
Windows source builds can use the main `holt.exe` as their sandbox helper; release
archives still ship standalone Windows helper executables.

More install details: [docs/INSTALL.md](docs/INSTALL.md).

## First Run

Start the TUI:

```bash
holt
```

The setup wizard helps you pick a provider and model. You can also configure
providers from the command line:

```bash
holt setup
holt providers list
holt models list
holt doctor
```

For API providers, set the matching environment variable before setup or enter
the key in the wizard:

```bash
export OPENAI_API_KEY=sk-...
export ANTHROPIC_API_KEY=...
export GEMINI_API_KEY=...
export LONGCAT_API_KEY=...
```

To configure Meituan LongCat (LongCat-2.0) directly, run:

```bash
holt providers setup longcat --set-active
```

For local models, run Ollama or LM Studio and then use `holt setup` or
`holt providers detect`.

## Daily Use

### Interactive TUI

```bash
holt
```

Useful controls:

| Control | Action |
|---|---|
| `Enter` | send the prompt |
| `/` | open slash-command suggestions |
| `Shift+Tab` | cycle permission mode |
| `Ctrl+B` | show/hide the sidebar |
| `Ctrl+C` | cancel or exit |

Common slash commands:

| Command | Purpose |
|---|---|
| `/model`, `/provider` | switch the active model/provider |
| `/spec`, `/plan` | draft and review a plan before building |
| `/image` | attach an image for vision-capable models |
| `/resume`, `/rewind` | continue or roll back local sessions |
| `/compact`, `/context` | manage context usage |
| `/permissions`, `/tools` | inspect available tools and policy |
| `/add-dir` | allow an extra write directory for this session |
| `/theme`, `/doctor`, `/config` | adjust appearance and inspect setup |

### Headless `exec`

```bash
holt exec "explain internal/agent/loop.go"
holt exec --model claude-sonnet-4.5 "refactor the config loader"
holt exec --use-spec "add rate limiting to the API client"
holt exec --worktree "try the migration in an isolated worktree"
holt exec --resume
holt exec --fork <session-id> "try the other approach"
```

Programmatic use:

```bash
holt exec --input-format stream-json --output-format stream-json < turns.jsonl
```

The stream-JSON contract is documented in
[docs/STREAM_JSON_PROTOCOL.md](docs/STREAM_JSON_PROTOCOL.md).

## Safety Model

Holt is designed to make side effects visible.

- Workspace reads are allowed by default.
- File writes are limited to the workspace unless you grant another directory.
- Shell commands, network access, destructive commands, and elevated actions are
  permission-gated.
- `--add-dir <path>` and `/add-dir <path>` grant additional write roots without
  giving the agent the whole filesystem.
- Unsafe/autonomous modes are explicit opt-ins.
- Secrets are redacted from tool output and logs where Holt controls the surface.

Example:

```bash
holt --add-dir ../docs-site
holt exec --add-dir ../shared "update both repos"
```

Sandbox behavior can be inspected with:

```bash
holt sandbox policy
holt sandbox grants list
```

## Web And Local Control

Holt includes local file/search/edit/shell tools, `web_fetch` for public URLs,
and MCP support for additional tools.

For local dev servers, use shell commands such as `curl` through `exec_command`
so the normal sandbox and permission policy applies. Long-running commands stay
attached to a background terminal session and can be listed or stopped from the
TUI.

The npm package also includes browser and terminal helper packages used by local
browser/terminal tools. Source builds can use the same helpers when they are on
`PATH` or configured in Holt's local-control settings.

## Run In A Browser

`holt-web` serves the full holt TUI in a browser tab by bridging a real PTY to
[xterm.js](https://xtermjs.org) over a WebSocket. It ships with a `Dockerfile`.

```bash
docker build -t holt-web .
docker run --rm -p 8080:8080 -e HOLT_WEB_TOKEN=change-me -e OPENROUTER_API_KEY=sk-... holt-web
# open http://localhost:8080/?token=change-me
```

The static frontend in `cmd/holt-web/web/` deploys anywhere static (including
Vercel); the `holt-web` **backend** holds the PTY and WebSocket and runs on a
container host (Railway, Fly.io, Render, any Docker/VPS). Run them together or
split them — a Vercel-hosted page just points its WebSocket at the backend with
`?backend=wss://…`. It hands visitors a live agent, so always set `HOLT_WEB_TOKEN`
and keep it isolated. See [docs/BROWSER_TERMINAL.md](docs/BROWSER_TERMINAL.md).

## Robinhood Trading (MCP)

Holt ships a Robinhood trading MCP server ([`mcp/robinhood/`](mcp/robinhood)) that
gives the agent `rh_quote`, `rh_portfolio`, `rh_positions`, `rh_orders`,
`rh_place_order`, and `rh_cancel_order` tools. It's zero-dependency (Node ≥ 18)
and off until you configure it with a `ROBINHOOD_TOKEN`. Order placement is gated
behind `HOLT_RH_ALLOW_TRADING=1` and a per-call `confirm: true`, so nothing trades
by accident. See [docs/ROBINHOOD_MCP.md](docs/ROBINHOOD_MCP.md).

## Common Commands

```text
holt                  interactive TUI
holt exec             one-shot or scripted agent run
holt setup            first-run provider setup
holt auth             OAuth/login helpers for supported providers
holt models           model registry and capabilities
holt providers        provider profiles and detection
holt doctor           setup, key, and connectivity checks
holt context          context-budget report
holt repo-map         deterministic repository map
holt repo-info        local repository summary
holt search | find    search local session history
holt sessions         inspect, resume, fork, and rewind sessions
holt spec             manage spec-mode drafts
holt specialist       manage specialist subagents
holt skills           manage markdown instruction skills
holt plugins          manage plugins
holt hooks            manage lifecycle hooks
holt mcp              manage MCP servers and tools
holt serve --mcp      expose Holt tools over MCP stdio
holt sandbox          inspect sandbox policy and grants
holt worktrees        prepare isolated git worktrees
holt verify           detect and run local verification checks
holt changes          inspect and commit local git changes
holt usage            token usage and estimated cost
holt cron             scheduled agent jobs
holt update           check for newer releases
```

## Appearance And Accessibility

| Control | Effect |
|---|---|
| `NO_COLOR=<anything>` | disables color output |
| `HOLT_THEME=<name>` | selects the startup theme (`auto`, `dark`, `light`, or a color theme like `dracula`, `nord`, `gruvbox`, `tokyo-night`, `catppuccin`, `one-dark`, `solarized-dark`, `rose-pine`, `everforest`, `solarized-light`) |
| `--theme <name>` | selects the TUI theme from the CLI (same names) |
| `/theme` | opens the theme picker inside the TUI (live preview; `/theme <name>` switches directly) |
| `HOLT_NO_FADE=1` | disables streaming fade animation |

Meaning does not rely on color alone; diffs, permissions, and statuses also use
text or glyph markers.

## Development

```bash
go test ./...
go run ./cmd/holt-release build
go run ./cmd/holt-release smoke
go run ./cmd/holt-perf-bench
```

Cross-compile examples:

```bash
go run ./cmd/holt-release build --goos linux --goarch amd64
go run ./cmd/holt-release build --goos windows --goarch amd64 --output dist/holt.exe
```

## Documentation

- [Install](docs/INSTALL.md)
- [Update flow](docs/UPDATE.md)
- [Stream-JSON protocol](docs/STREAM_JSON_PROTOCOL.md)
- [Specialists](docs/SPECIALISTS.md)
- [GitHub Action](docs/GITHUB_ACTION.md)
- [Benchmarks](docs/BENCHMARK.md)
- [Performance](docs/PERFORMANCE.md)
- [Agent evals](docs/AGENT_EVALS.md)

## Contributing

Contributions are welcome. Read [CONTRIBUTING.md](CONTRIBUTING.md), run the
relevant tests, and open a focused pull request.

Security reports should follow [SECURITY.md](SECURITY.md).

## License

Holt is released under the [MIT License](LICENSE).
