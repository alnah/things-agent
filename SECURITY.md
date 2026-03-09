# Security

`things-agent` controls Things 3 through AppleScript and the Things URL Scheme.
Normal operations must not access the Things SQLite database directly.
The only narrow exception is the internal restore path, which uses a scoped SQLite-backed package-swap workflow for restore only.

## Security warning read before use

Use this project at your own risk.

### Agent risk model

- To be useful, AI agents often need broad system permissions.
- Agents can bypass expectations or instructions if they are sufficiently capable.
- This repository includes safety rails such as backups, restore checks, and scoped auth-token usage, but not a full safety guarantee.
- You remain fully responsible for what the agent executes on your machine.

### Automation and permissions

- macOS with Things installed
- Apple Events automation allowed for the terminal or agent app invoking the CLI
- filesystem access sufficient for Things data and backups when required
- a valid auth token (`THINGS_AUTH_TOKEN` or `--auth-token`) for URL update operations

If these permissions are missing, commands can fail even when syntax is correct.
If they are granted broadly, the risk surface increases accordingly.

### Backups and destructive actions

- `things-agent session-start` creates a session backup.
- `things-agent backup` creates a manual backup.
- restore creates a pre-restore safety backup for rollback.
- Backups are stored under the Things data directory in `Backups/`.
- Timestamp format is `YYYY-MM-DD:HH-MM-SS`.
- Retention is capped at 50 snapshots.
- Emptying Things trash is intentionally not exposed by the CLI.
- Item deletion remains available (`delete-task`, `delete-project`, `delete-area`, `delete-child-task`) with backup beforehand.

### Auth token handling

Do not expose your Things auth token to your AI provider unless strictly necessary.
Prefer resolving it locally from a secret store at runtime instead of hardcoding it in shell history, scripts, or repo files.

The token is required only for the URL update surfaces that need authorization, such as checklist-related updates and other `update`-style flows.

Get the token on macOS:

1. Open `Things 3`.
2. Go to `Things > Settings > General`.
3. In the `Things URLs` section, open token management and copy the auth token.
4. Export it in your shell if you need a direct local setup:

```bash
export THINGS_AUTH_TOKEN="<your-token>"
```

A better approach is to keep the token in a local secret manager and resolve it only at runtime on your Mac.

Example with `pass`:

```bash
export THINGS_AUTH_TOKEN="$(pass show things/auth-token)"
```

If you use `zsh`, you can add that command to `~/.zshrc` so new terminal sessions resolve the token locally without storing it in the repository.

Other macOS-local secret managers such as Keychain, 1Password CLI, or Bitwarden CLI work too, as long as the token is resolved locally on the machine at runtime.

This reduces accidental exposure, but is not a perfect guarantee. A sufficiently capable or over-permissioned agent may still leak the token if it is allowed to access it.

## Reporting a security issue

For validated non-sensitive findings, use the repository's `Bug or security report` issue form.

For sensitive findings, use GitHub private vulnerability reporting for this repository.

Include:

- macOS version
- Things version
- `things-agent` version
- command used
- expected vs actual behavior
- reproduction steps
- impact
- remediation advice
