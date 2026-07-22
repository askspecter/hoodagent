#!/usr/bin/env node
// Robinhood trading MCP server for Holt.
//
// A zero-dependency Model Context Protocol server (stdio, newline-delimited
// JSON-RPC 2.0) that exposes Robinhood account and trading tools. Holt spawns it
// via `node server.js` and calls its tools.
//
// Auth: set ROBINHOOD_TOKEN to a Robinhood OAuth bearer token in the server's
// env (Holt passes the MCP server's `env` through). Read-only tools work with
// just the token.
//
// SAFETY: order placement is gated. rh_place_order only sends a live order when
// BOTH HOLT_RH_ALLOW_TRADING=1 is set in the env AND the call passes
// confirm:true. Otherwise it returns a dry-run preview and places nothing. This
// uses Robinhood's unofficial API with real money — understand the risk.

'use strict';

const readline = require('node:readline');

const API = 'https://api.robinhood.com';
const TOKEN = process.env.ROBINHOOD_TOKEN || '';
const TRADING_ENABLED = process.env.HOLT_RH_ALLOW_TRADING === '1';
const SERVER = { name: 'robinhood', version: '0.1.0' };
const PROTOCOL_VERSION = '2024-11-05';

// ---- Robinhood REST helper -------------------------------------------------

async function rh(path, { method = 'GET', body } = {}) {
  if (!TOKEN) {
    throw new Error('ROBINHOOD_TOKEN is not set. Add it to the robinhood MCP server env in your Holt config.');
  }
  const url = path.startsWith('http') ? path : API + path;
  const res = await fetch(url, {
    method,
    headers: {
      Authorization: 'Bearer ' + TOKEN,
      Accept: 'application/json',
      'Content-Type': 'application/json',
      'User-Agent': 'holt-robinhood-mcp/0.1.0',
    },
    body: body ? JSON.stringify(body) : undefined,
  });
  const text = await res.text();
  let json;
  try { json = text ? JSON.parse(text) : {}; } catch (_) { json = { raw: text }; }
  if (!res.ok) {
    const detail = json && (json.detail || json.error || JSON.stringify(json));
    throw new Error(`Robinhood API ${res.status}: ${detail}`);
  }
  return json;
}

async function accountURL() {
  const data = await rh('/accounts/');
  const acct = data.results && data.results[0];
  if (!acct) throw new Error('No Robinhood account found for this token.');
  return acct;
}

async function instrumentForSymbol(symbol) {
  const data = await rh('/instruments/?symbol=' + encodeURIComponent(symbol.toUpperCase()));
  const inst = data.results && data.results[0];
  if (!inst) throw new Error('Unknown symbol: ' + symbol);
  return inst;
}

// ---- Tools -----------------------------------------------------------------

const tools = {
  rh_quote: {
    description: 'Get the latest quote (last/ask/bid price) for one or more symbols.',
    inputSchema: {
      type: 'object',
      properties: { symbols: { type: 'string', description: 'Comma-separated tickers, e.g. "AAPL,MSFT".' } },
      required: ['symbols'],
    },
    async run({ symbols }) {
      const data = await rh('/quotes/?symbols=' + encodeURIComponent(String(symbols).toUpperCase()));
      const rows = (data.results || []).filter(Boolean).map((q) => ({
        symbol: q.symbol,
        last: q.last_trade_price,
        ask: q.ask_price,
        bid: q.bid_price,
        previous_close: q.previous_close,
      }));
      return rows;
    },
  },

  rh_portfolio: {
    description: 'Get account portfolio value: equity, market value, buying power.',
    inputSchema: { type: 'object', properties: {} },
    async run() {
      const acct = await accountURL();
      const pf = await rh('/portfolios/');
      const p = (pf.results && pf.results[0]) || {};
      return {
        account_number: acct.account_number,
        buying_power: acct.buying_power,
        equity: p.equity,
        market_value: p.market_value,
        extended_hours_equity: p.extended_hours_equity,
      };
    },
  },

  rh_positions: {
    description: 'List current non-zero stock positions (holdings).',
    inputSchema: { type: 'object', properties: {} },
    async run() {
      const data = await rh('/positions/?nonzero=true');
      const rows = [];
      for (const pos of data.results || []) {
        if (!pos || Number(pos.quantity) === 0) continue;
        let symbol = pos.symbol;
        if (!symbol && pos.instrument) {
          try { symbol = (await rh(pos.instrument)).symbol; } catch (_) {}
        }
        rows.push({
          symbol,
          quantity: pos.quantity,
          average_buy_price: pos.average_buy_price,
        });
      }
      return rows;
    },
  },

  rh_orders: {
    description: 'List recent orders (most recent first).',
    inputSchema: {
      type: 'object',
      properties: { limit: { type: 'number', description: 'Max orders to return (default 10).' } },
    },
    async run({ limit = 10 }) {
      const data = await rh('/orders/');
      return (data.results || []).slice(0, limit).map((o) => ({
        id: o.id,
        state: o.state,
        side: o.side,
        type: o.type,
        quantity: o.quantity,
        price: o.price,
        average_price: o.average_price,
        created_at: o.created_at,
      }));
    },
  },

  rh_place_order: {
    description:
      'Place a stock buy/sell order. GATED: only executes when HOLT_RH_ALLOW_TRADING=1 and confirm:true; otherwise returns a dry-run preview.',
    inputSchema: {
      type: 'object',
      properties: {
        symbol: { type: 'string' },
        side: { type: 'string', enum: ['buy', 'sell'] },
        quantity: { type: 'number' },
        type: { type: 'string', enum: ['market', 'limit'], description: 'Default "market".' },
        price: { type: 'number', description: 'Limit price (required for type "limit").' },
        time_in_force: { type: 'string', enum: ['gfd', 'gtc'], description: 'Default "gfd".' },
        confirm: { type: 'boolean', description: 'Must be true to place a live order.' },
      },
      required: ['symbol', 'side', 'quantity'],
    },
    async run(args) {
      const { symbol, side, quantity } = args;
      const type = args.type || 'market';
      const timeInForce = args.time_in_force || 'gfd';
      if (type === 'limit' && !(Number(args.price) > 0)) {
        throw new Error('A positive "price" is required for a limit order.');
      }
      const inst = await instrumentForSymbol(symbol);
      const acct = await accountURL();

      let price = args.price;
      if (type === 'market') {
        const q = await rh('/quotes/?symbols=' + encodeURIComponent(symbol.toUpperCase()));
        price = q.results && q.results[0] && q.results[0].last_trade_price;
      }

      const order = {
        account: acct.url,
        instrument: inst.url,
        symbol: symbol.toUpperCase(),
        type,
        time_in_force: timeInForce,
        trigger: 'immediate',
        side,
        quantity: String(quantity),
        price: String(price),
      };

      if (!TRADING_ENABLED || args.confirm !== true) {
        return {
          dry_run: true,
          reason: !TRADING_ENABLED
            ? 'HOLT_RH_ALLOW_TRADING is not "1" — trading disabled.'
            : 'confirm was not true.',
          would_place: order,
        };
      }
      const placed = await rh('/orders/', { method: 'POST', body: order });
      return { placed: true, id: placed.id, state: placed.state, order };
    },
  },

  rh_cancel_order: {
    description: 'Cancel an open order by its id.',
    inputSchema: {
      type: 'object',
      properties: { id: { type: 'string' } },
      required: ['id'],
    },
    async run({ id }) {
      await rh('/orders/' + encodeURIComponent(id) + '/cancel/', { method: 'POST' });
      return { cancelled: true, id };
    },
  },
};

// ---- MCP JSON-RPC plumbing (newline-delimited) -----------------------------

function send(msg) {
  process.stdout.write(JSON.stringify(msg) + '\n');
}

function result(id, value) {
  send({ jsonrpc: '2.0', id, result: value });
}

function rpcError(id, code, message) {
  send({ jsonrpc: '2.0', id, error: { code, message } });
}

function toolList() {
  return Object.entries(tools).map(([name, t]) => ({
    name,
    description: t.description,
    inputSchema: t.inputSchema,
  }));
}

async function handle(msg) {
  const { id, method, params } = msg;
  const isNotification = id === undefined || id === null;

  switch (method) {
    case 'initialize':
      return result(id, {
        protocolVersion: PROTOCOL_VERSION,
        capabilities: { tools: {} },
        serverInfo: SERVER,
      });
    case 'notifications/initialized':
      return; // notification, no response
    case 'ping':
      return result(id, {});
    case 'tools/list':
      return result(id, { tools: toolList() });
    case 'tools/call': {
      const name = params && params.name;
      const tool = tools[name];
      if (!tool) return rpcError(id, -32602, 'Unknown tool: ' + name);
      try {
        const value = await tool.run((params && params.arguments) || {});
        return result(id, { content: [{ type: 'text', text: JSON.stringify(value, null, 2) }] });
      } catch (err) {
        return result(id, { content: [{ type: 'text', text: 'Error: ' + err.message }], isError: true });
      }
    }
    default:
      if (!isNotification) rpcError(id, -32601, 'Method not found: ' + method);
  }
}

const rl = readline.createInterface({ input: process.stdin });
rl.on('line', (line) => {
  const trimmed = line.trim();
  if (!trimmed) return;
  let msg;
  try { msg = JSON.parse(trimmed); } catch (_) { return; }
  Promise.resolve(handle(msg)).catch((err) => {
    if (msg && msg.id != null) rpcError(msg.id, -32603, String(err && err.message));
  });
});
