# Robinhood trading (MCP)

Holt ships a Robinhood trading MCP server at
[`mcp/robinhood/`](../mcp/robinhood). It exposes account and trading tools to the
agent over MCP (stdio, zero external dependencies — just Node ≥ 18).

## Tools

| Tool | What it does |
|------|--------------|
| `rh_quote` | Latest last/ask/bid price for one or more symbols. |
| `rh_portfolio` | Equity, market value, buying power. |
| `rh_positions` | Current non-zero holdings. |
| `rh_orders` | Recent orders. |
| `rh_place_order` | Place a buy/sell order — **gated** (see safety). |
| `rh_cancel_order` | Cancel an open order by id. |

## ⚠️ Safety

This uses Robinhood's unofficial API with **real money**.

- `rh_place_order` places a live order **only** when both are true:
  `HOLT_RH_ALLOW_TRADING=1` in the server env **and** the call passes
  `confirm: true`. Otherwise it returns a dry-run preview and places nothing.
- The read-only tools only need `ROBINHOOD_TOKEN`.
- Treat `ROBINHOOD_TOKEN` like a password. Prefer running with trading disabled
  until you've watched the dry-run previews.

## Setup

1. Get a Robinhood OAuth bearer token and export it where you'll configure Holt.
2. Add the server to your Holt config — user (`~/.config/holt/config.json`) or
   project (`.holt/config.json`). Use the absolute path to `server.js`:

```json
{
  "mcp": {
    "servers": {
      "robinhood": {
        "type": "stdio",
        "command": "node",
        "args": ["/absolute/path/to/mcp/robinhood/server.js"],
        "env": {
          "ROBINHOOD_TOKEN": "your-oauth-bearer-token",
          "HOLT_RH_ALLOW_TRADING": "0"
        }
      }
    }
  }
}
```

Set `HOLT_RH_ALLOW_TRADING` to `"1"` only when you actually want live orders to
go through (and still pass `confirm: true` per call).

3. Start Holt; the `rh_*` tools appear alongside the built-ins. Disable anytime
   with `holt mcp disable robinhood`.

## Quick check

```bash
printf '%s\n' \
 '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"holt","version":"dev"}}}' \
 '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' \
 | node mcp/robinhood/server.js
```

You should see the `initialize` result and the six `rh_*` tools listed.
