# Security

`things-agent` controls Things 3 through AppleScript and the Things URL Scheme.
It does not access the Things SQLite database directly.

## Security warning read before use

Use this project at your own risk.

### Agent risk model

- To be useful, AI agents often need broad system permissions.
- Agents can bypass expectations or instructions if they are sufficiently capable.
- This repository includes safety rails, but not a full safety harness.
- You remain fully responsible for what the agent executes on your machine.

### Backups and destructive actions

- `things-agent session-start` creates a session backup.
- `things-agent backup` creates a manual backup.
- Backups are stored under the Things data directory in `backups/`.
- Timestamp format is `YYYY-MM-DD:HH-MM-SS`.
- Retention is capped at 50 backups.
- Emptying Things trash is intentionally not exposed by the CLI.
- Item deletion remains available (`delete-task`, `delete-project`, `delete-list`) with backup beforehand.

### Required permissions

- macOS with Things installed
- Apple Events automation allowed for the terminal/agent app invoking the CLI
- A valid auth token (`THINGS_AUTH_TOKEN` or `--auth-token`) for URL update operations

If these permissions are missing, commands can fail even when syntax is correct.

### Auth token handling

Do not expose your Things auth token to your AI provider unless strictly necessary.

```bash
export THINGS_AUTH_TOKEN="$(pass show things/auth-token)"
```

This reduces accidental exposure, but is not a perfect guarantee.

## Agent instructions sync

This project is designed for Codex and Claude Code.
Keep `AGENTS.md` as the source and symlink:

```bash
ln -sf AGENTS.md CLAUDE.md
```

## AppleScript unavailable fallback

If AppleScript support is unavailable on your machine or CI:

- explain fallback options clearly
- do not modify the Things database manually
- ask the user how to proceed

## Reporting a security issue

Open a private report with reproduction steps and impact details.
Include:

- macOS version
- Things version
- command used
- expected vs actual behavior
