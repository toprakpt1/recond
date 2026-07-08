# AGENTS.md

## Project Overview

`recond` is a Go-based recon job orchestrator with a daemon + CLI architecture. The daemon (`recond`) runs as a background process, and the CLI (`recon`) communicates with it via Unix socket.

## Quick Commands

```bash
# Build both binaries
make build

# Run tests
make test

# Lint (go vet)
make lint

# Build and run daemon
make run-daemon

# Build and run CLI
make run-cli

# Clean build artifacts
make clean
```

## Architecture

- **Two binaries**: `cmd/recond/main.go` (daemon) and `cmd/recon/main.go` (CLI)
- **Daemon**: Long-running process, listens on Unix socket at `~/.recond/recond.sock`
- **CLI**: Cobra-based, sends JSON requests to daemon via socket
- **Storage**: SQLite at `~/.recond/recond.db` (WAL mode, single connection)
- **Pipeline**: Sequential execution of 5 recon tools with data flow between steps

## Key Directories

```
cmd/
├── recon/       # CLI entry point
└── recond/      # Daemon entry point
internal/
├── cli/         # Cobra CLI commands
├── config/      # Viper config + profile management
├── daemon/      # Daemon server + Unix socket handler
├── models/      # Data models (Job, Step, Log, Output)
├── pipeline/    # Pipeline controller (sequential step execution)
├── runner/      # Tool adapters + executor with retry
├── storage/     # SQLite CRUD operations
└── template/    # Pipeline template engine
configs/         # Default config templates
bin/             # Compiled binaries (gitignored)
```

## Important Notes

- **Go module path**: `github.com/recond`
- **Go version**: 1.26.4 (required)
- **No tests exist**: `make test` will run but find no test files
- **Empty packages**: `pkg/utils`, `internal/tui`, `internal/orchestrator`, `internal/scheduler` are empty
- **Config paths**: Searches `./`, `$HOME/.recond`, `/etc/recond` for `config.yaml`
- **Environment variables**: Prefixed with `RECOND_` (e.g., `RECOND_DATA_DIR`)

## Tool Execution

Tools are executed via `internal/runner/executor.go`:
- Uses `os/exec` with process groups (`Setpgid: true`)
- Implements retry with exponential backoff (3 retries, 30s max backoff)
- Output files stored at `~/.recond/jobs/<job-id>/`
- Each tool writes to its own output file (e.g., `subfinder.txt`, `httpx.txt`)

### Resume Behavior

When a job is resumed (`recon resume`), tools continue from where they left off instead of restarting:

| Tool | Resume Method | Details |
|------|--------------|---------|
| httpx | `-resume` flag | Uses `resume.cfg` state file |
| katana | `-resume` flag | Uses `resume.cfg` state file |
| subfinder | No resume support | Restarts from beginning (fast tool) |
| ffuf | Checkpoint-based | Iterates alive.txt domains, skips completed ones via checkpoint |
| gau | No resume support | Restarts from beginning (planned: fork with `-resume`) |

- Executor uses append mode (`O_APPEND`) instead of truncate when `IsResume=true`
- Pipeline preserves checkpoint data on resume instead of clearing it
- ffuf checkpoint tracks `completed_domains` list to skip already-fuzzed subdomains

### Wordlist (ffuf)

ffuf requires a wordlist file. Configured via `wordlist` in config or per-profile:
- Default: `~/.recond/wordlists/common.txt`
- Auto-download: If missing, downloads from SecLists via curl/wget
- Config key: `wordlist` (global or `profiles.<name>.wordlist`)

## Pipeline Flow

```
subfinder → subfinder.txt → httpx → httpx.txt → katana/gau → katana.txt → ffuf → directories.json
```

Each step reads from the previous step's output file. Pipeline skips completed steps on resume.

### ffuf Multi-Domain Execution

ffuf iterates over each subdomain in `alive.txt` individually:
1. Reads all alive subdomains from `alive.txt`
2. Checks checkpoint for already-completed domains
3. Runs ffuf for each remaining domain sequentially
4. Updates checkpoint after each domain completes
5. Merges all results into `directories.json` at the end

## Resource Profiles

| Profile | Concurrency | Rate Limit | Timeout | Retries |
|---------|-------------|------------|---------|---------|
| safe | 3 | 10 req/s | 30s | 5 |
| balanced | 10 | 50 req/s | 15s | 3 |
| aggressive | 25 | 100 req/s | 10s | 2 |

## Common Pitfalls

1. **Missing tools**: Only tools installed on the system will work; missing tools cause step failures
2. **Daemon must be running**: CLI commands require the daemon to be running first
3. **Socket cleanup**: Daemon removes stale socket file on startup if it exists
4. **Process groups**: Executor uses `SysProcAttr{Setpgid: true}` for clean process termination
5. **SQLite concurrency**: Single connection with WAL mode; avoid concurrent writes
