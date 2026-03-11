# slack-status macOS app

This directory contains the native macOS menu bar frontend for the [`slack-status`](../README.md) CLI backend.

## Structure

- [`SlackStatusApp/`](SlackStatusApp/) — Swift/AppKit sources
- [`SlackStatusApp.xcodeproj/`](SlackStatusApp.xcodeproj/) — Xcode project metadata

## Integration model

The macOS app stays intentionally thin. It does not talk to Slack directly and it does not duplicate backend rules. Instead, it shells out to the Go CLI through [`CLIService`](SlackStatusApp/CLIService.swift:9), using JSON for stateful commands and foreground execution for [`login()`](SlackStatusApp/CLIService.swift:21).

Expected backend commands:

- `slack-status --json status`
- `slack-status --json start`
- `slack-status --json work`
- `slack-status --json lunch`
- `slack-status --json clear`
- `slack-status login`

Executable discovery is owned by [`resolveExecutablePath()`](SlackStatusApp/CLIService.swift:81). It checks, in order:

1. `/usr/local/bin/slack-status`
2. `/opt/homebrew/bin/slack-status`
3. the bundled helper inside the app at `Contents/Resources/slack-status`

That bundled path is the basis of the team distribution workflow documented below.

## Current scope

This app provides:

- a menu bar status item
- a transient popover with compact status UI
- current state rendering from CLI JSON
- action rows for Start, Work, Lunch, Clear, Refresh, Login, and Quit
- one process boundary for backend process execution

## Local development builds

Build from the repository root with [`Makefile`](../Makefile):

- `make macos-build` — builds the `SlackStatusApp` target with `xcodebuild`
- `make macos-open` — builds and launches the resulting `.app` bundle with `open`
- `make macos-run` — builds and executes the app binary directly

The build output is written under `macos/build/` by default.

You can also open [`SlackStatusApp.xcodeproj`](SlackStatusApp.xcodeproj/) in Xcode and run the `SlackStatusApp` target.

## Team distribution workflow

The repository now includes a Make-based distribution flow for producing a self-contained macOS app and DMG for teammates. The key project-specific detail is that the app bundles the Go CLI inside the application at `Contents/Resources/slack-status`, which matches the existing fallback path already used by [`resolveExecutablePath()`](SlackStatusApp/CLIService.swift:81).

This means recipients do not need a separate CLI install in `/usr/local/bin` or `/opt/homebrew/bin`. The app can run directly against its bundled backend.

### Distribution targets

The distribution targets live in [`Makefile`](../Makefile):

- `make build-universal` — builds the Go CLI for both `darwin/amd64` and `darwin/arm64`, then merges them into the universal helper binary `slack-status-universal`
- `make macos-build-release` — builds the macOS app in Release configuration
- `make macos-bundle` — copies the universal CLI into `SlackStatusApp.app/Contents/Resources/slack-status` and makes it executable
- `make macos-sign` — signs the bundled app with [`CODESIGN_IDENTITY`](../Makefile) (default is ad-hoc signing via `-`)
- `make macos-stage` — copies the signed `.app` into `dist/` as a clean staging area
- `make macos-dmg` — packages the staged app into `SlackStatusApp.dmg`
- `make macos-dist` — runs the full pipeline end to end
- `make macos-clean` — removes derived data and distribution artifacts

### Recommended command

From the repository root, run:

```bash
make macos-dist
```

That pipeline does the following in order:

1. Builds a universal Go CLI for Intel and Apple Silicon Macs.
2. Builds the AppKit frontend in Release mode.
3. Bundles the CLI at `Contents/Resources/slack-status`.
4. Signs the resulting `.app`.
5. Stages the signed app in `dist/`.
6. Creates `SlackStatusApp.dmg`.

## Signing model and Apple Developer account requirements

### Default workflow: no Apple Developer account required

The default distribution workflow does **not** require a paid Apple Developer account. [`Makefile`](../Makefile) sets `CODESIGN_IDENTITY ?= -`, so [`macos-sign`](../Makefile) uses ad-hoc signing by default.

Ad-hoc signing is appropriate for this repository's internal team distribution flow because it:

- produces a structurally signed `.app`
- signs the embedded CLI helper as part of the app bundle
- requires no certificate provisioning or Apple Developer Program membership
- keeps the workflow identical for all developers on the team

### Recipient first-launch behavior and tradeoffs

Ad-hoc signing is still **not** Developer ID signing and it is **not** notarization. For teammates receiving the generated DMG, the expected first-launch behavior is:

- after dragging the app to Applications, macOS will warn on first launch because the app is not notarized for public distribution
- the recipient must use **right-click → Open** once, then confirm the dialog
- after that one-time override, later launches behave normally

Tradeoffs of the default workflow:

- **Pros:** free, simple, reproducible, and sufficient for internal team sharing
- **Cons:** first launch is not seamless, and some locked-down corporate environments may apply stricter Gatekeeper or MDM policies

### Upgrading later to Developer ID signing

If a paid Apple Developer account and Developer ID certificate are available later, the same Make targets can continue to be used. Override [`CODESIGN_IDENTITY`](../Makefile) with the Developer ID identity instead of changing the workflow.

Example:

```bash
make macos-dist CODESIGN_IDENTITY="Developer ID Application: Example Team"
```

That keeps the same bundling and packaging process while swapping the signing identity. If notarization is added later, it can build on top of this same pipeline.

## Teammate installation steps from the DMG

Share the generated `SlackStatusApp.dmg` with teammates. Their installation flow is:

1. Open `SlackStatusApp.dmg`.
2. Drag `SlackStatusApp.app` into `/Applications`.
3. In Applications, right-click `SlackStatusApp.app` and choose **Open** for the first launch.
4. Confirm the macOS prompt.
5. Use the app's Login action to run `slack-status login` and save the user's Slack token.

No separate Go toolchain, Homebrew install, or standalone CLI installation is required for recipients because the app already contains the backend helper at `Contents/Resources/slack-status`.
