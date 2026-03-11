# slack-status macOS app scaffold

This directory contains a minimal native macOS menu bar app scaffold for the [`slack-status`](../README.md) CLI backend.

## Structure

- `SlackStatusApp/` — Swift/AppKit sources
- `SlackStatusApp.xcodeproj/` — lightweight Xcode project metadata

## Integration model

The app shell does not reimplement Slack behavior. It invokes the existing Go CLI through a single boundary in [`CLIService.swift`](SlackStatusApp/CLIService.swift), using JSON output for stateful commands.

Expected backend commands:

- `slack-status --json status`
- `slack-status --json work`
- `slack-status --json lunch`
- `slack-status --json clear`
- `slack-status login`

By default the app looks for the backend binary at `/usr/local/bin/slack-status`, then falls back to `/opt/homebrew/bin/slack-status`, then a bundled helper path if one is added later.

## Current scope

This scaffold provides:

- menu bar status item
- transient popover with compact status UI
- current state rendering from CLI JSON
- buttons for Work, Lunch, Clear, Refresh, Login, and Quit
- a single service boundary for backend process execution

## Building

Build from the repository root with [`Makefile`](../Makefile):

- `make macos-build` — builds the `SlackStatusApp` target with `xcodebuild`
- `make macos-open` — builds and launches the resulting `.app` bundle with `open`
- `make macos-run` — builds and executes the app binary directly

The build output is written under `macos/build/` by default.

You can also open [`SlackStatusApp.xcodeproj`](SlackStatusApp.xcodeproj) in Xcode and run the `SlackStatusApp` target.
