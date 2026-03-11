# AGENTS.md

## Project Overview

[`slack-status`](README.md) is a two-surface Slack status tool:

- a Go CLI named [`slack-status`](main.go)
- a native macOS menu bar frontend under [`macos/`](macos)

The CLI remains the backend source of truth. The macOS app does not talk to Slack directly; it shells out to the CLI and consumes its JSON/state output. Both surfaces rely on a user OAuth token starting with `xoxp-` and the same local config/state files.

## Tech Stack

- Go 1.25, stdlib only, for the CLI and Slack API integration
- Swift plus AppKit for the macOS menu bar app
- Slack Web API endpoints implemented in [`slack.go`](slack.go)
- XDG-style config and state storage implemented in [`config.go`](config.go)
- `make` orchestration in [`Makefile`](Makefile)
- `xcodebuild` plus Xcode project metadata in [`macos/SlackStatusApp.xcodeproj/project.pbxproj`](macos/SlackStatusApp.xcodeproj/project.pbxproj)

## Current File Layout

| Path | Purpose |
|------|---------|
| [`main.go`](main.go) | CLI entry point, option parsing, command routing, JSON response emission, login flow |
| [`slack.go`](slack.go) | Slack Web API calls for status updates, clears, and channel posting |
| [`config.go`](config.go) | Config/state path resolution and JSON persistence for token and local state |
| [`scheduler.go`](scheduler.go) | Detached lunch return worker lifecycle and PID file management |
| [`Makefile`](Makefile) | CLI build/install plus macOS build/open/run targets |
| [`README.md`](README.md) | Top-level product usage and setup documentation |
| [`macos/README.md`](macos/README.md) | macOS app specific overview and integration notes |
| [`macos/SlackStatusApp/SlackStatusApp.swift`](macos/SlackStatusApp/SlackStatusApp.swift) | macOS app entry point and top-level app bootstrap |
| [`macos/SlackStatusApp/AppModels.swift`](macos/SlackStatusApp/AppModels.swift) | Swift command enums, backend response models, error types, and view state |
| [`macos/SlackStatusApp/CLIService.swift`](macos/SlackStatusApp/CLIService.swift) | Single process boundary between the app and the CLI backend |
| [`macos/SlackStatusApp/StatusViewModel.swift`](macos/SlackStatusApp/StatusViewModel.swift) | UI-facing state orchestration for refresh and command execution |
| [`macos/SlackStatusApp/StatusItemController.swift`](macos/SlackStatusApp/StatusItemController.swift) | Menu bar item and popover presentation wiring |
| [`macos/SlackStatusApp/PopoverViewController.swift`](macos/SlackStatusApp/PopoverViewController.swift) | AppKit popover UI and action buttons |
| [`macos/SlackStatusApp/EventMonitor.swift`](macos/SlackStatusApp/EventMonitor.swift) | Event monitoring for dismissing the transient popover |
| [`macos/SlackStatusApp.xcodeproj/project.pbxproj`](macos/SlackStatusApp.xcodeproj/project.pbxproj) | Native macOS project configuration |

## Architecture

### CLI Architecture

All user-facing CLI commands are routed through [`main()`](main.go:16) after option parsing in [`parseOptions()`](main.go:118). The CLI supports plain text output for humans and JSON output for the macOS app through the `--json` flag. Timed status-setting commands also accept optional `--until` input through [`parseOptions()`](main.go:118).

Current commands handled by the switch in [`main()`](main.go:47):

- `login` — interactive OAuth token setup and config persistence
- `start` — sets the working status, supports optional `--until`, is only available once per local calendar day, and posts `:wave:` to `#remote_work` only on the first successful daily use
- `work` — sets the working status without posting a channel message and supports optional `--until`
- `lunch` — sets lunch status, supports optional `--until`, records local state, and spawns a detached return worker that returns at the explicit requested time
- `clear` — clears Slack status and local state
- `status` — returns the locally persisted backend state as JSON
- `_return-worker` — internal process used to restore the work state after lunch

Important implementation detail: the menu bar app currently uses `start`, `work`, `lunch`, `clear`, `status`, and `login` through [`CLIService`](macos/SlackStatusApp/CLIService.swift). The app can pass explicit target times for `start`, `work`, and `lunch`, but backend command rules and first-start-of-day gating remain owned by the CLI.

### Slack API Layer

[`slack.go`](slack.go) is the only Slack API integration layer. Keep network behavior centralized there. The current responsibilities are:

- set status
- clear status
- post a message to `#remote_work`

Future agents should avoid duplicating Slack request code in other files or in the macOS app.

### Local Config and State Layer

[`ResolvePaths()`](config.go:20) defines the shared on-disk contract used by both the CLI and the macOS app:

- config file: `XDG_CONFIG_HOME/slack-status/config.json` or default `~/.config/slack-status/config.json`
- PID file: `XDG_STATE_HOME/slack-status/worker.pid` or default `~/.local/state/slack-status/worker.pid`
- state file: `XDG_STATE_HOME/slack-status/status.json` or default `~/.local/state/slack-status/status.json`

[`SaveConfig()`](config.go:85) stores the OAuth token JSON. [`SaveLocalState()`](config.go:74) and [`LoadLocalState()`](config.go:60) maintain the CLI-owned local status snapshot consumed by the app. [`withDerivedState()`](config.go:96) computes additive availability metadata used by the frontend without changing the underlying status command semantics.

### Background Worker

[`SpawnReturnWorker()`](scheduler.go:14) starts a detached `_return-worker` process using `Setsid` and passes the explicit return timestamp through `--until`. [`KillWorker()`](scheduler.go:36) is called before any mutating foreground command so only one lunch-return worker is active. [`RunReturnWorker()`](scheduler.go:52) resolves the target time from its arguments, sleeps until that exact instant, restores the working status, updates local state, and removes the PID file.

This worker model is part of the current contract. Future changes must preserve the single-worker invariant unless both the CLI and the app contract are updated together.

### macOS App Architecture

The app under [`macos/SlackStatusApp/`](macos/SlackStatusApp) is a native AppKit menu bar application layered on top of the CLI backend.

High-level flow:

1. [`SlackStatusApp`](macos/SlackStatusApp/SlackStatusApp.swift) boots the app.
2. [`StatusItemController`](macos/SlackStatusApp/StatusItemController.swift) owns the menu bar item and popover presentation.
3. [`PopoverViewController`](macos/SlackStatusApp/PopoverViewController.swift:3) renders the compact UI with vertical action rows for Start, Work, Lunch, Clear, Refresh, Login, and Quit, plus native time pickers for Start, Work, and Lunch.
4. [`StatusViewModel`](macos/SlackStatusApp/StatusViewModel.swift) translates UI actions into backend requests and exposes `idle`, `loading`, `loaded`, and `failure` states.
5. [`CLIService`](macos/SlackStatusApp/CLIService.swift:9) resolves the CLI path, executes the process, and decodes JSON into the Swift models from [`AppModels.swift`](macos/SlackStatusApp/AppModels.swift).

The app is intentionally thin. Backend rules, Slack behavior, timing semantics, first-start-of-day availability, and persistent state semantics belong to the Go CLI.

## Backend JSON and State Contract

The macOS app expects the CLI JSON shape emitted from [`commandResult`](main.go:115) and the state schema defined by [`LocalState`](config.go:40) and [`StateStatus`](config.go:47).

Expected command form for app-driven stateful actions:

- `slack-status --json status`
- `slack-status --json start --until <timestamp-or-local-time>`
- `slack-status --json work`
- `slack-status --json lunch`
- `slack-status --json clear`

For timed commands, the app may also pass `--until` to `work` and `lunch`. The CLI remains responsible for parsing local clock input and absolute timestamps.

Expected top-level JSON shape:

- `command` string
- `ok` boolean
- `state` object

Expected `state` fields:

- `current_status.command`
- `current_status.text`
- `current_status.emoji`
- `current_status.status_expires_at`
- `current_status.will_return_to`
- `current_status.source`
- `worker_scheduled`
- `worker_pid` optional
- `start_available_today`
- `last_start_at` optional
- `last_start_day` optional
- `updated_at`

`start_available_today` is additive frontend-facing metadata derived from persisted state. It exists so the macOS app can disable Start after the first successful local-day use while preserving the CLI as the source of truth for enforcement.

Important rough edges to preserve or update carefully:

- The app decodes JSON for `status`, `start`, `work`, `lunch`, and `clear` through [`runJSONCommand()`](macos/SlackStatusApp/CLIService.swift:48).
- `login` is intentionally non-JSON and is executed as an interactive foreground CLI process in [`login()`](macos/SlackStatusApp/CLIService.swift:21).
- `status` reports the locally saved state file, not a live Slack readback.
- The app treats `ok: false`, non-zero exit status, empty stdout, or schema drift as backend failures.
- If the JSON schema changes, [`CommandResponse`](macos/SlackStatusApp/AppModels.swift:11), [`CurrentStatus`](macos/SlackStatusApp/AppModels.swift:19), and [`StatusState`](macos/SlackStatusApp/AppModels.swift:40) must be updated in lockstep.
- Future changes must preserve additive state compatibility so older saved status meaning is not reinterpreted when exposing new app affordances such as Start availability.

## Executable Discovery Contract

[`resolveExecutablePath()`](macos/SlackStatusApp/CLIService.swift:64) currently probes these paths in order:

1. `/usr/local/bin/slack-status`
2. `/opt/homebrew/bin/slack-status`
3. bundled helper path inside the app at `Contents/Resources/slack-status`

Do not change installation paths or bundle expectations casually. The macOS app depends on this fallback order.

## Makefile Targets

[`Makefile`](Makefile) is the canonical automation entry point.

- `make init` — create `.env` from [`.env.example`](.env.example)
- `make env` — print resolved `APP_ID`, OAuth URL, and ldflags
- `make build` — build the CLI binary with OAuth URL injected via ldflags
- `make install` — install the CLI to `/usr/local/bin/slack-status`
- `make uninstall` — remove the installed CLI
- `make macos-build` — build the `SlackStatusApp` Xcode target with `xcodebuild`
- `make macos-open` — build then launch the `.app` bundle with `open`
- `make macos-run` — build then execute the app binary directly

Relevant Make variables:

- `BINARY`
- `DESTDIR`
- `MACOS_PROJECT`
- `MACOS_SCHEME`
- `MACOS_CONFIGURATION`
- `MACOS_DERIVED_DATA`
- `MACOS_APP`

The OAuth install URL used by `login` is injected into the CLI by the `LDFLAGS` definition in [`Makefile`](Makefile).

## Build and Run Workflow

Typical setup for the CLI:

1. `make init`
2. edit `.env` and set `APP_ID`
3. `make build`
4. `make install`
5. run `slack-status login`

Typical setup for the macOS app:

1. ensure the CLI has been built and installed where [`CLIService`](macos/SlackStatusApp/CLIService.swift:9) can find it
2. run `make macos-build`, `make macos-open`, or `make macos-run`
3. use the menu bar popover actions to trigger backend commands

The macOS app is not a replacement backend. It requires the CLI to exist and remain compatible.

## Frontend UX Conventions

- Keep the popup action layout as vertical rows, with timed rows for Start, Work, and Lunch and single-action rows for Clear, Refresh, Login, and Quit, as implemented in [`PopoverViewController`](macos/SlackStatusApp/PopoverViewController.swift:3).
- Keep Start-specific availability UX aligned with backend state: when `start_available_today` is false, disable both the Start button and its paired time picker rather than attempting a speculative frontend-only override.
- Preserve native macOS time-entry controls for timed actions instead of replacing them with freeform text inputs.
- Preserve dynamic status icon behavior in both the popup and menu bar so status presentation tracks backend state rather than a fixed symbol set.
- Preserve human-friendly date and time formatting for status details so explicit expiration and return targets remain understandable to users at a glance.

## Conventions for Future Agents

- Treat the Go CLI as the backend authority and the macOS app as a native frontend.
- Keep the backend JSON contract stable unless the Swift decoding models are updated in the same change.
- Preserve XDG config and state paths and file permissions: `0o700` for created directories and `0o600` for token or state files.
- Keep Go dependency footprint at stdlib only unless there is an explicit project-level decision to change that.
- Prefer adding behavior to the CLI first, then exposing it through the macOS app through [`CLIService`](macos/SlackStatusApp/CLIService.swift:9).
- Do not duplicate Slack Web API logic in Swift.
- Keep status labels, emojis, channel names, time semantics, and command semantics aligned across CLI text output, saved state, and app UI.
- If adding new CLI commands for app consumption, define the JSON shape explicitly and update both [`main.go`](main.go) and [`macos/SlackStatusApp/AppModels.swift`](macos/SlackStatusApp/AppModels.swift).
- If changing timed status behavior, update CLI option parsing in [`parseOptions()`](main.go:118), persisted state in [`LocalState`](config.go:41), and Swift decoding plus UI assumptions together.
- If changing lunch scheduling behavior, update both the worker logic in [`scheduler.go`](scheduler.go) and the UI assumptions around `worker_scheduled`, `worker_pid`, and `will_return_to`.
- Preserve the once-per-local-day Start contract and the rule that the hello post is tied to the first successful daily Start, not to later Work or Lunch transitions.
- When documenting or extending the macOS app, prefer describing it as a menu bar frontend over the existing CLI backend, because that is the current architecture.

## Required Slack OAuth Scopes

- `users.profile:write`
- `chat:write`
