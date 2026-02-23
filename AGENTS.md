# AGENTS.md

## Project Overview

`slack-status` is a CLI tool for quickly switching Slack status for common work scenarios. It uses the Slack Web API with a user OAuth token (`xoxp-`).

Binary name: `slack-status`

## Tech Stack

- Go (1.25), zero external dependencies (stdlib only)
- Slack Web API (`users.profile.set`, `chat.postMessage`)
- XDG-compliant config/state paths
- Makefile build with ldflags for OAuth URL injection

## File Layout

| File | Purpose |
|------|---------|
| `main.go` | Entry point, command routing (`work`, `lunch`, `clear`, `login`), CLI UX |
| `slack.go` | Slack API calls: `SetStatus`, `ClearStatus`, `PostMessage` |
| `config.go` | XDG path resolution, token load/save (`config.json`) |
| `scheduler.go` | Background worker: spawn, kill, PID tracking for lunch auto-return |
| `Makefile` | Build/install targets, reads `APP_ID` from `.env` |

## Architecture

### Command Flow

All commands go through `main()` → switch on `os.Args[1]`:

- **`work`** — kills any pending worker, sets "Working remotely" status (expires 6pm), sends `:wave:` to `#remote_work`
- **`lunch`** — kills worker, sets "Lunch" (1h expiry), spawns detached `_return-worker` process that restores "work" status after 1 hour
- **`clear`** — kills worker, clears status
- **`login`** — interactive OAuth token setup, saves to `config.json`
- **`_return-worker`** — internal: sleeps 1h then restores "Working remotely" status

### Slack API Layer (`slack.go`)

All API calls go through a common pattern: marshal JSON → POST with Bearer token → parse response → check `ok` field. Two endpoints:
- `users.profile.set` — status changes
- `chat.postMessage` — channel messages

### Background Worker (`scheduler.go`)

The lunch command spawns a detached process (`Setsid: true`) that sleeps and then restores status. PID is tracked in a file. Any new command kills the existing worker first.

### Config (`config.go`)

Single JSON file with `token` field. Paths follow XDG spec:
- Config: `$XDG_CONFIG_HOME/slack-status/config.json` (default `~/.config/...`)
- State: `$XDG_STATE_HOME/slack-status/worker.pid` (default `~/.local/state/...`)

## Conventions

- No external dependencies — use Go stdlib only
- Error handling: `fmt.Errorf` with `%w` wrapping, print to stderr, `os.Exit(1)`
- File permissions: `0o700` for dirs, `0o600` for files containing tokens
- Status values (emoji, text, channel names) are hardcoded constants in `main.go`
- The `oauthURL` variable is injected at build time via ldflags

## Build & Run

```bash
make init       # creates .env from .env.example
# edit .env to set APP_ID
make build      # go build with ldflags
make install    # build + install to /usr/local/bin
```

## Required Slack OAuth Scopes

- `users.profile:write` — for setting/clearing status
- `chat:write` — for posting messages to channels
