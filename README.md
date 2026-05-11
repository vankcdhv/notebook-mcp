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
| `source_add` | Add a source (`type: "url"`, `"youtube"`, `"text"`, `"drive"`, or `"file"`) |
| `source_delete` | Remove a source |
| `note_list` / `note_get` | Read notebook notes |
| `note_create` / `note_update` / `note_delete` | Manage notebook notes |
| `research_start` / `research_poll` / `research_import` | Run web/Drive research and import selected results |
| `ask` | Blocking chat against a notebook (optionally scoped to `source_ids`, with `conversation_id` for follow-ups) |
| `ask_stream` | Chat response with structured chunks plus final answer; live MCP progress notifications are not emitted |
| `knowledge_search` | Citation-backed snippets — query → references → fulltext → exact quote + context window |

---

## Quick start

One-liner — installs the latest release into `~/.notebooklm-mcp/bin/notebooklm-mcp` and prints the next steps:

```bash
curl -sSL https://raw.githubusercontent.com/vankcdhv/notebook-mcp/main/install.sh | sh
```

Then follow the printed instructions:

```bash
~/.notebooklm-mcp/bin/notebooklm-mcp setup
claude mcp add -s user notebooklm-mcp ~/.notebooklm-mcp/bin/notebooklm-mcp
```

Restart Claude Code → tools show up under `notebooklm-mcp`. **To update later, re-run the same `curl ... | sh` command** — it always pulls the latest release.

Skip to [Usage](#usage-from-an-ai-agent) or read on for details.

---

## Requirements

- Go ≥ 1.26
- A Google account with NotebookLM access
- macOS / Linux (Windows untested)
- ~150 MB free for Playwright's Chromium download (one-off)

---

## Install

### Option A — install script (recommended)

Linux/macOS:

```bash
curl -sSL https://raw.githubusercontent.com/vankcdhv/notebook-mcp/main/install.sh | sh
```

Windows PowerShell:

```powershell
iwr -useb https://raw.githubusercontent.com/vankcdhv/notebook-mcp/main/install.ps1 | iex
```

What it does:
- detects your OS + arch
- resolves the latest release tag from GitHub
- downloads the matching archive
- verifies its SHA-256 against `checksums.txt`
- installs the binary to `~/.notebooklm-mcp/bin/notebooklm-mcp` (mode `0755`)
- on macOS, strips the quarantine xattr so Gatekeeper won't block it
- prints the next-step `setup` and `claude mcp add` commands

Pin a specific version:

```bash
curl -sSL https://raw.githubusercontent.com/vankcdhv/notebook-mcp/main/install.sh | sh -s -- v0.2.3
```

```powershell
$env:NOTEBOOKLM_MCP_VERSION="v0.2.3"; iwr -useb https://raw.githubusercontent.com/vankcdhv/notebook-mcp/main/install.ps1 | iex
```

Override install directory:

```bash
curl -sSL https://raw.githubusercontent.com/vankcdhv/notebook-mcp/main/install.sh \
  | NOTEBOOKLM_MCP_INSTALL_DIR=/usr/local/bin sh
```

On Linux, installing to system directories such as `/usr/local/bin` may require privileges:

```bash
curl -sSL https://raw.githubusercontent.com/vankcdhv/notebook-mcp/main/install.sh \
  | sudo NOTEBOOKLM_MCP_INSTALL_DIR=/usr/local/bin sh
```

On Windows:

```powershell
$env:NOTEBOOKLM_MCP_INSTALL_DIR="$env:USERPROFILE\bin"; iwr -useb https://raw.githubusercontent.com/vankcdhv/notebook-mcp/main/install.ps1 | iex
```

**Updating** — re-run the same command. The script always replaces the existing binary with the latest release.

### Option B — `go install`

```bash
go install github.com/vankcdhv/notebook-mcp/cmd/notebooklm-mcp@latest
```

### Option C — manual download

Grab the archive matching your platform from [Releases](https://github.com/vankcdhv/notebook-mcp/releases), extract, drop the binary somewhere on your `PATH`. Verify against `checksums.txt`.

### Option D — build from source

```bash
git clone https://github.com/vankcdhv/notebook-mcp
cd notebook-mcp
go build -o notebooklm-mcp ./cmd/notebooklm-mcp
```

### macOS Gatekeeper

Release binaries are unsigned. If macOS shows:

> Apple could not verify “notebooklm-mcp” is free of malware that may harm your Mac or compromise your privacy.

remove the quarantine attribute after extracting the archive:

```bash
xattr -dr com.apple.quarantine ./notebooklm-mcp
chmod +x ./notebooklm-mcp
./notebooklm-mcp version
```

Only do this for a binary you downloaded from this repository's official GitHub Releases page. You can also verify the archive against `checksums.txt` from the same release.

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
- *"Add https://example.com/article to notebook AIClass."* → `source_add` with `type: "url"`
- *"Upload ./paper.pdf into AIClass."* → `source_add` with `type: "file"`
- *"Create a note in AIClass from this summary."* → `note_create`
- *"Research recent clustering papers and import the best links."* → `research_start` → `research_poll` → `research_import`
- *"What does my AIClass notebook say about clustering? I need exact quotes."* → `knowledge_search`
- *"Ask AIClass: how does k-means initialise centroids?"* → `ask` or `ask_stream`

`source_add` accepts these type-specific arguments:

| Type | Required args |
|---|---|
| `url` | `notebook_id`, `url` |
| `youtube` | `notebook_id`, `url` |
| `text` | `notebook_id`, `content` (`title` optional) |
| `drive` | `notebook_id`, `file_id`, `title` (`mime_type` optional) |
| `file` | `notebook_id`, `file_path` (`mime_type` optional) |

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
