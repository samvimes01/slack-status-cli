# slack-status

CLI tool to quickly switch your Slack status for common work scenarios.

## Commands

| Command              | Status         | Emoji | Expires                               |
| -------------------- | -------------- | ----- | ------------------------------------- |
| `slack-status work`  | Working remotely | 💻    | Today 6pm by default, or `--until`    |
| `slack-status start` | Working remotely | 💻    | Today 6pm by default, or `--until`; posts `:wave:` once per local calendar day |
| `slack-status lunch` | Lunch          | 🍔    | +1 hour by default, or `--until`, then auto-restores to "work" |
| `slack-status clear` | _(cleared)_    | —     | —                                     |

## Setup

### 1. Get a User Token

If you're in the Bidscube Slack workspace, the app is already set up:

1. Go to https://api.slack.com/apps and select `status service` app.
2. Click **Install to Workspace** and authorize
3. Copy the **User OAuth Token** — it starts with `xoxp-`

That's it — skip to step 2 below.

#### (Optional) Create your own Slack App

If you're in a different workspace:

1. Go to https://api.slack.com/apps
2. Click **Create New App** → **From scratch**
3. Name it (e.g. "Status CLI"), select your workspace, click **Create App**
4. In the sidebar go to **OAuth & Permissions**
5. Under **User Token Scopes** (not Bot Token Scopes), add: `users.profile:write`
6. Click **Install to Workspace** and authorize
7. Copy the **User OAuth Token**

### 2. Create Config File

Open a terminal and run:

```bash
slack-status login
```

It will prompt you to paste your token.

Or manually:

```bash
mkdir -p ~/.config/slack-status
cat > ~/.config/slack-status/config.json <<'EOF'
{ "token": "xoxp-YOUR-TOKEN-HERE" }
EOF
chmod 600 ~/.config/slack-status/config.json
```

### 3. Build and Install

App requires .env with APP_ID set.

Go to https://api.slack.com/apps and select status service app.

Copy the app id from the URL <https://api.slack.com/apps/APP_ID/> and paste it into .env

```bash
# Using make:
make init # Only once, creates .env
make install

# Build the macOS menu bar app scaffold:
make macos-build

# Launch the built macOS menu bar app:
make macos-open

# Build and run the macOS menu bar app directly:
make macos-run

# Or build and move manually:
go build -o slack-status
# Move to somewhere on your PATH, e.g.:
mv slack-status /usr/local/bin/
```

## Usage

```bash
slack-status login   # Authenticate with Slack
slack-status work    # Working remotely until 6pm
slack-status work --until 17:30
slack-status start   # Working remotely until 6pm and posts :wave: once per day
slack-status start --until "2026-03-11 18:00"
slack-status lunch   # Lunch for 1h, auto-returns to "work" status after
slack-status lunch --until 14:30
slack-status clear   # Clear status entirely
```

`--until` accepts a local-time `HH:MM`, `YYYY-MM-DD HH:MM`, `YYYY-MM-DDTHH:MM`, or full RFC3339 timestamp. For `HH:MM`, the CLI uses the next occurrence of that time.

`slack-status start` is enforced by the backend as a once-per-local-calendar-day action. The shared state file records whether Start is still available today, plus the timestamp/day of the last successful Start, so the macOS menu bar app can reflect the backend authority without duplicating the rule.

## How the Lunch Auto-Return Works

`slack-status lunch` spawns a detached background process (`_return-worker`) that sleeps until the chosen return time, then restores your "Working remotely" status. Without `--until`, it defaults to 1 hour from now. The worker's PID is stored at `~/.local/state/slack-status/worker.pid`.

Running `slack-status work`, `slack-status start`, or `slack-status clear` cancels any pending worker.

## XDG Path Overrides

| Variable          | Default          | Used For             |
| ----------------- | ---------------- | -------------------- |
| `XDG_CONFIG_HOME` | `~/.config`      | Config file location |
| `XDG_STATE_HOME`  | `~/.local/state` | PID file location    |
