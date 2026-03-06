# things-go

CLI Go pour piloter Things (macOS) via AppleScript uniquement avec `cobra`.

## Prérequis

- macOS
- Application Things installée
- macOS `osascript`

Le CLI n'accède jamais directement à la base SQLite de Things.

## Installation

```bash
cd /Users/alexis/workspace/things-go
go mod tidy
go build -o /usr/local/bin/things .
```

Vous pouvez aussi définir le nom final du binaire lors du `go build`.

## Utilisation

```bash
things backup
things lists
things projects
things tasks --list "À classer"
things search --query "Wagner"
things add-task --name "Dizer ola" --notes "Mensagem" --list "À classer"
things add-task --name "Préparer semaine" --subtasks "Relire emails, Faire backup, Envoyer rapport"
things add-project --name "Mon projet" --list "À classer"
things complete-task --name "Dizer ola"
things list-subtasks --task "Dizer ola"
things add-subtask --task "Dizer ola" --name "Vérifier le message"
```

### Commandes utiles

- `backup`
- `restore [--file <chemin ou timestamp>]`
- `lists`
- `projects`
- `tasks [--list <nom>] [--query <texte>]`
- `search --query <texte> [--list <nom>]`
- `add-task`
- `add-task --subtasks "Nom 1, Nom 2, ..."`
- `add-project`
- `add-list`
- `edit-task`
- `edit-project`
- `edit-list`
- `delete-task`
- `delete-project`
- `delete-list`
- `complete-task`
- `uncomplete-task`
- `list-subtasks --task`
- `add-subtask --task --name [--notes]`
- `edit-subtask --task (--index|--name) (--new-name|--notes)`
- `delete-subtask --task (--index|--name)`
- `complete-subtask --task (--index|--name)`
- `uncomplete-subtask --task (--index|--name)`
- `set-tags`
- `version`

## Sauvegardes

Format des noms générés:

`main.sqlite.YYYY-MM-DD:HH-MM-SS.bak`

Le même format s'applique à `main.sqlite-shm` et `main.sqlite-wal`.
Une sauvegarde est créée automatiquement avant les commandes mutantes.
