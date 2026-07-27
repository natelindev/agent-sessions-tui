# agent-sessions-tui

`agent-sessions-tui` is a focused terminal browser for local coding-agent sessions. It discovers histories from supported tools, keeps filtering immediate, and resumes the selected CLI session in the current terminal.

<p align="center">
  <img src="docs/assets/agent-sessions-tui.png" alt="agent-sessions-tui browsing local coding-agent sessions" width="100%">
</p>

<p align="center"><em>Demo data is used in the screenshot.</em></p>

It is intentionally limited to three jobs:

- Discover and normalize local agent sessions.
- Search and navigate them quickly with a keyboard or mouse.
- Resume supported sessions without copying IDs or commands by hand.

Session data stays local and provider stores are opened read-only.

## Supported tools

| Tool | Default local store | Resume |
| --- | --- | --- |
| Codex | `~/.codex/sessions`, `~/.codex/archived_sessions` | `codex resume <id>` |
| Claude | `~/.claude*/projects`, `~/.claude*/transcripts`, Claude desktop local sessions | `claude --resume <id>` |
| Antigravity | `~/.gemini/antigravity*/brain` | `agy --conversation <id>` |
| OpenCode | `~/.local/share/opencode/opencode.db` and legacy JSON storage | `opencode --session <id>` |
| Hermes | `~/.hermes/state.db` and `~/.hermes/sessions` | `hermes --resume <id>` |
| GitHub Copilot CLI | `~/.copilot/session-state` | `copilot --resume=<id>` |
| Droid | `~/.factory/sessions`, `~/.factory/projects` | Not exposed because the source app has no stable ID resume workflow |
| OpenClaw | `~/.openclaw/agents`, legacy `~/.clawdbot/agents` | Not exposed because the source app has no stable ID resume workflow |
| Cursor Agent | `~/.cursor/projects/*/agent-transcripts` | `cursor agent --resume <id>` |
| Pi | `~/.pi/agent/sessions` | `pi --session <path-or-id>` |

The app respects `OPENCLAW_STATE_DIR` when it is set.

## Appearance

Each provider has a distinct color, while timestamps, projects, titles, search state, and the active row use separate visual weights. The adaptive palette is designed for light and dark terminals. Standard `NO_COLOR` behavior is respected.

## Install

Go 1.24 or newer is required.

Install the latest version with one command:

```sh
curl -fsSL https://raw.githubusercontent.com/natelindev/agent-sessions-tui/main/scripts/install.sh | sh
```

The script downloads the Go module, builds it, and installs the binary atomically into `$XDG_BIN_HOME` or `~/.local/bin`.

Alternatively, install directly with Go:

```sh
go install github.com/natelindev/agent-sessions-tui/cmd/agent-sessions-tui@latest
```

For a local checkout:

```sh
./scripts/install.sh
agent-sessions-tui
```

When run from a local checkout, the same installer builds the checked-out source. Override the destination when needed:

```sh
./scripts/install.sh --bin-dir /usr/local/bin
```

## Controls

| Action | Keyboard | Mouse |
| --- | --- | --- |
| Navigate | Arrow keys, `j` / `k`, Page Up / Page Down | Scroll wheel |
| Search | `/`, then type | Click the search field, then type |
| Leave search | Enter or Escape | Click a row |
| Clear search | Escape outside the search field | - |
| Resume | Enter | Double-click a row |
| Refresh | `r` | - |
| Quit | `q` or Ctrl+C | - |

Search is case-insensitive and updates on every keystroke. Space-separated terms use AND matching across provider, session ID, title, project, working directory, model, path, and indexed transcript text. File-backed sessions scan the complete transcript and deduplicate normalized search terms, so later messages remain searchable without retaining repeated text. OpenCode and Hermes database rows use their session metadata.

## Design

The implementation has four small layers:

- `internal/discovery` locates provider stores and extracts lightweight session records concurrently.
- `internal/session` owns the normalized model, sorting, and in-memory search.
- `internal/resume` maps supported providers to argument-safe `exec.Cmd` values.
- `internal/ui` owns the responsive Bubble Tea interface and input handling.

Resume commands are built as executable paths plus argument arrays. They are not passed through a shell. The selected working directory is used when it still exists, and Bubble Tea temporarily hands the terminal to the resumed process.

## Inspiration and scope

This is an independent Go TUI inspired by [Agent Sessions](https://github.com/jazzyalex/agent-sessions), a local-first desktop app for browsing work across coding agents. It carries forward the original project's most useful core workflow: discover sessions across tools, search them instantly, and resume work with minimal friction.

The scope is deliberately smaller. It does not include transcript and image viewers, usage quotas, analytics, saved-session management, settings screens, menu-bar features, or an application updater. The result is a fast terminal-native tool centered on session discovery, search, and resume.

## Development

```sh
go test ./...
go vet ./...
go build -trimpath -o ./bin/agent-sessions-tui ./cmd/agent-sessions-tui
```

The project uses Bubble Tea and Lip Gloss for terminal rendering and the pure-Go `modernc.org/sqlite` driver for read-only OpenCode and Hermes access.

## License

Released under the [MIT License](LICENSE).
