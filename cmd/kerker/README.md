# Kerker

Lock up your coding agent for good.

## Features

- Restrict file system access to whitelisted directories
- Restrict network access to whitelisted domains and ports
- Restricted tool usage
- Add capabilities per project
- Introspect and manage multitple agents

## Usage

| Command         | description                                         |
|-----------------|-----------------------------------------------------|
| `spawn <input>` | Spawns a new coding agent in the current directory. |
| `list`          | Lists the currently running agents                  |
| `kill <id>`     | Kills a running agent                               |

## Config

```json
{
	"packages": ["curl", "grep", "go"],
	"hosts": ["localhost:5432", "go.dev"],
	"directories": ["~/.claude"]
}
```
