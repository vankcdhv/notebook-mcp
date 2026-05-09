# notebooklm-mcp-go

> Unofficial MCP stdio server for [Google NotebookLM](https://notebooklm.google.com/), written in Go. Lets AI agents (Claude Code, Cursor, Continue, etc.) read your notebooks, manage sources, ask questions, and — most importantly — pull **citation-backed snippets** straight from your sources to keep the agent honest.

[![Go](https://img.shields.io/badge/go-1.26-00ADD8?logo=go)](go.mod)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![MCP](https://img.shields.io/badge/protocol-MCP%20stdio-brightgreen)](https://modelcontextprotocol.io)

> **About this project** — `notebooklm-mcp-go` is a Go re-implementation inspired by the original Python project [`notebooklm-py`](https://github.com/teng-lin/notebooklm-py). The Python parent is the reference for NotebookLM's undocumented `batchexecute` RPC payload shapes and method IDs; this Go project re-implements the protocol from scratch — **no Python dependency, no shelling out, no FFI** — and adds a citation-backed `knowledge_search` tool tailored for AI agent workflows. Credit to the upstream Python project for paving the way.

---

## Why

LLMs hallucinate when summarising your knowledge base. NotebookLM has the answer in its index — but a generic chat call only gives you prose, not **what was actually quoted from where**. This server exposes a `knowledge_search` tool that:

1. asks NotebookLM the question
2. parses the citation references it returns
3. fetches the **real source fulltext** for each citation
4. returns the exact quote + a configurable context window + character offsets

So your agent gets *evidence*, not a paraphrase.

## Features

- **Independent** — no Python parent dependency, no network shims, just Go.
- **MCP stdio** — Content-Length framing, drops straight into Claude Code / Cursor / Continue / any MCP-compatible host.
- **Auth that works** — Playwright-based persistent Chrome profile with the anti-detection flags Google's login flow actually requires. Cookie-import fallback for headless boxes.
- **Knowledge search with real citations** — not a wrapper around chat.
- **Local-only** — cookies + tokens stay in `~/.notebooklm-mcp/` at `0600`. Nothing is logged.

## Tools exposed

| Tool | Purpose |
|---|---|
| `notebook_list` | List your notebooks (`include_shared` to include shared ones) |
| `notebook_get` | Notebook + summary + sources in one call |
| `notebook_create` | Create a new notebook |
| `source_list` | List sources in a notebook |
| `source_add` | Add a URL or text source (`type: "url"` or `"text"`) |
| `source_delete` | Remove a source |
| `ask` | Blocking chat against a notebook (optionally scoped to `source_ids`, with `conversation_id` for follow-ups) |
| `knowledge_search` | Citation-backed snippets — query → references → fulltext → exact quote + context window |

---

## Quick start

```bash
# 1. Build
git clone <this-repo> && cd notebooklm-mcp-go
go build ./cmd/notebooklm-mcp

# 2. First-time setup (installs Playwright browser if missing, opens Google login)
./notebooklm-mcp setup

# 3. Register with Claude Code, scope = user (global)
claude mcp add -s user notebooklm-mcp $(pwd)/notebooklm-mcp

# 4. Restart Claude Code → tools show up under `notebooklm-mcp`
```

That's it. Skip to [Usage](#usage-from-an-ai-agent) or read on for details.

---

## Requirements

- Go ≥ 1.26
- A Google account with NotebookLM access
- macOS / Linux (Windows untested)
- ~150 MB free for Playwright's Chromium download (one-off)

---

## Install

### Option A — build from source

```bash
git clone <this-repo>
cd notebooklm-mcp-go
go build ./cmd/notebooklm-mcp
```

You'll get a `notebooklm-mcp` binary in the project root.

### Option B — install to `$PATH`

```bash
go install github.com/vanlt/notebooklm-mcp-go/cmd/notebooklm-mcp@latest
# or, after building locally:
sudo install -m 755 ./notebooklm-mcp /usr/local/bin/notebooklm-mcp
```

---

## First-time setup

```bash
notebooklm-mcp setup
```

What it does:

1. Checks whether Playwright Chromium is installed; offers to install if missing (~150 MB).
2. Launches a **visible** Chrome window with a persistent profile at `~/.notebooklm-mcp/browser-profile/`.
3. You log in to Google → wait for the NotebookLM home page to load.
4. Back in the terminal, press **ENTER**. Cookies + tokens are written to `~/.notebooklm-mcp/profile.json` (`0600`).

Non-interactive (CI / scripted): `notebooklm-mcp setup --yes` auto-confirms the browser-install prompt.

### Cookie-import fallback

If Google's login flow refuses to cooperate (headless boxes, exotic 2FA, corporate SSO), grab the cookies from a logged-in browser and import them:

```bash
notebooklm-mcp login --cookie "SID=...; HSID=...; SSID=...; APISID=...; SAPISID=..."
```

The required cookies are: `SID`, `HSID`, `SSID`, `APISID`, `SAPISID` (and any `__Secure-*` variants your account uses). Easiest source: DevTools → Application → Cookies → `https://notebooklm.google.com`.

### Standalone subcommands

```bash
notebooklm-mcp install-browsers   # just install Playwright Chromium
notebooklm-mcp login              # interactive browser login only
notebooklm-mcp version
```

---

## Wiring it into an AI agent

The server speaks MCP over stdio. Default invocation (no args) starts the server.

### Claude Code

Global / user scope (recommended — available in every project):

```bash
claude mcp add -s user notebooklm-mcp /usr/local/bin/notebooklm-mcp
```

Per-project scope:

```bash
cd your-project
claude mcp add notebooklm-mcp /usr/local/bin/notebooklm-mcp
```

Verify:

```bash
claude mcp list
claude mcp get notebooklm-mcp
```

Restart your Claude Code session and the tools appear under the `notebooklm-mcp` namespace.

Override the profile location for a per-project profile:

```bash
claude mcp add -s user notebooklm-work \
  -e NOTEBOOKLM_MCP_PROFILE=$HOME/.notebooklm-mcp/work.json \
  /usr/local/bin/notebooklm-mcp
```

### Claude Desktop

Edit `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) or the equivalent on your platform:

```json
{
  "mcpServers": {
    "notebooklm-mcp": {
      "command": "/usr/local/bin/notebooklm-mcp"
    }
  }
}
```

Restart Claude Desktop.

### Cursor / Continue / generic MCP host

Same config shape — point `command` at the binary, no args needed for the stdio server.

---

## Usage from an AI agent

Once registered, you can prompt your agent naturally and it will pick the right tool. Examples:

- *"List my NotebookLM notebooks."* → `notebook_list`
- *"Add https://example.com/article to notebook AIClass."* → `source_add`
- *"What does my AIClass notebook say about clustering? I need exact quotes."* → `knowledge_search`
- *"Ask AIClass: how does k-means initialise centroids?"* → `ask`

`knowledge_search` is the one to lean on when you don't want the agent inventing details. Sample shape:

```json
{
  "answer": "K-means starts with k initial centroids ...",
  "conversation_id": "abc-123",
  "count": 3,
  "snippets": [
    {
      "source_id": "src-...",
      "source_title": "Lecture 04 — Clustering",
      "exact_quote": "centroids are sampled uniformly at random from the dataset",
      "snippet": "...In the basic Lloyd algorithm, centroids are sampled uniformly at random from the dataset, then iteratively...",
      "start_char": 1042,
      "end_char": 1106
    }
  ]
}
```

---

## Configuration

| Variable | Purpose | Default |
|---|---|---|
| `NOTEBOOKLM_MCP_PROFILE` | Path to the profile JSON file | `~/.notebooklm-mcp/profile.json` |

Profile directory layout:

```
~/.notebooklm-mcp/
├── profile.json           # cookies + CSRF/session tokens (0600)
└── browser-profile/       # Playwright persistent Chrome profile (0700)
```

---

## Troubleshooting

**`This browser or app may not be secure` during login**
Use the bundled `setup` / `login` command — it sets the anti-detection flags Google's login form requires (`--disable-blink-features=AutomationControlled`, ignoring `--enable-automation`, etc.). If it still blocks you, fall back to `login --cookie`.

**Auth works once, then 401 / 403 a few hours later**
Tokens (`SNlM0e`, `FdrFJe`) refresh automatically on each RPC call from the homepage. If the cookie itself is dead, re-run `notebooklm-mcp login`.

**`knowledge_search` returns `count: 0`**
The chat call returned a textual answer but no citations. Try a more specific query, or ensure the notebook has indexed sources (sources show up in `source_list`).

**Tool not visible in Claude Code**
`claude mcp list` should show it. If it does, restart the Claude Code session — MCP servers are loaded once on session start.

---

## Architecture

```
cmd/notebooklm-mcp        binary entrypoint (setup, login, serve)
internal/mcp              MCP stdio framing + tool registry
internal/notebooklm       domain client (notebooks, sources, chat, knowledge_search)
internal/rpc              batchexecute encoder/decoder + chat citation parser
internal/auth             Playwright login, cookie store, token refresh
internal/store            profile.json read/write with 0600 perms
```

The RPC layer talks to NotebookLM's `batchexecute` endpoint with obfuscated method IDs — the same surface Google's web UI uses. These IDs can change at any time; if something breaks, that's the first place to look.

## Security

- All credentials live under `~/.notebooklm-mcp/` with `0700` directory / `0600` file perms.
- Cookies, CSRF tokens, and session IDs are **never** logged.
- The HTTP client only follows redirects to NotebookLM/Google domains.
- The server speaks stdio only — no network listener, no inbound surface.

## Development

```bash
go test ./...
go vet ./...
go build ./cmd/notebooklm-mcp
```

The project lives in a nested git repo inside [notebooklm-py](https://github.com/teng-lin/notebooklm-py)'s `tools/` folder. See the *About this project* note at the top of this README for the relationship to the Python parent.

## Roadmap

- [ ] Streaming `ask` (currently blocking only)
- [ ] Studio artefacts (audio / video / mind-map) generation tools
- [ ] Note CRUD tools
- [ ] Windows support
- [ ] Pre-built release binaries

PRs welcome.

## Disclaimer

This project uses NotebookLM's **undocumented internal RPC API**. It is not affiliated with, endorsed by, or supported by Google. Method IDs and payload shapes can break without warning. Use at your own risk and within Google's Terms of Service.

## License

MIT
