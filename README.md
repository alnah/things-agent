# agent-things

CLI Go pour piloter Things (macOS) via AppleScript uniquement avec `cobra`.

Voir [AGENTS.md](/workspace/things-agent/AGENTS.md) pour les règles opérationnelles (backup initial de session, retention, sécurité, conventions).

## Prérequis

- macOS
- Application Things installée
- `osascript`

Le CLI n'accède jamais directement à la base SQLite de Things.
Certaines opérations de checklist native (URL scheme `update`) nécessitent un jeton d'authentification Things (`THINGS_AUTH_TOKEN` ou `--auth-token`).

## Installation

```bash
cd /workspace/things-agent
go mod tidy
go build -o /usr/local/bin/agent-things .
```

Vous pouvez aussi définir un nom de binaire différent lors du build.

## Utilisation

```bash
agent-things session-start
agent-things backup
agent-things tasks --list "À classer"
agent-things search --query "Wagner"
agent-things add-task --name "Dizer ola" --notes "Mensagem" --list "À classer"
agent-things add-task --name "Checklist native" --subtasks "Point 1, Point 2" --auth-token "<token>"
agent-things complete-task --name "Dizer ola"
agent-things list-subtasks --task "Dizer ola"
agent-things add-subtask --task "Dizer ola" --name "Vérifier le message"
```

### Commandes utiles

- `session-start`
- `backup`, `restore [--file <chemin ou timestamp>]`
- `lists`, `projects`
- `tasks [--list <nom>] [--query <texte>]`
- `search --query <texte> [--list <nom>]`
- `add-task`, `edit-task`, `delete-task`, `complete-task`, `uncomplete-task`
- `add-task --subtasks "a, b"` crée une checklist native (nécessite `--auth-token` ou `THINGS_AUTH_TOKEN`)
- `set-task-notes`, `append-task-notes`, `set-task-date`
- `add-project`, `edit-project`, `delete-project`
- `add-list`, `edit-list`, `delete-list`
- `add-subtask` ajoute un item de checklist native (nécessite token), `edit-subtask`, `delete-subtask`, `complete-subtask`, `uncomplete-subtask`, `list-subtasks`
- `set-tags`
- `set-task-tags`, `add-task-tags`, `remove-task-tags`
- `version`
