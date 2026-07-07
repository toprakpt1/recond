<p align="center">
  <img src="assets/logo.svg" alt="recond" width="300">
</p>

<h4 align="center">A daemon-based recon job orchestrator for penetration testers and bug bounty hunters.</h4>

<p align="center">
  <a href="#features">Features</a> •
  <a href="#quick-start">Quick Start</a> •
  <a href="#installation">Installation</a> •
  <a href="#usage">Usage</a> •
  <a href="#configuration">Configuration</a> •
  <a href="#architecture">Architecture</a> •
  <a href="#contributing">Contributing</a> •
  <a href="#license">License</a>
</p>

---

`recond` is a Go-based recon job orchestrator that manages reconnaissance workflows through a daemon + CLI architecture. It chains together popular recon tools ([subfinder](https://github.com/projectdiscovery/subfinder), [httpx](https://github.com/projectdiscovery/httpx), [katana](https://github.com/projectdiscovery/katana), [gau](https://github.com/lc/gau), [ffuf](https://github.com/ffuf/ffuf)) into automated pipelines with pause/resume, checkpointing, resource profiles, and export capabilities.

## Features

- **Daemon + CLI Architecture** — Long-running daemon manages jobs via Unix socket; lightweight CLI sends JSON commands
- **5 Tool Support** — subfinder, httpx, katana, gau, ffuf with automatic data flow between steps
- **Pipeline System** — Tools are chained into sequential steps; each step reads from the previous step's output
- **Pause / Resume / Stop** — Full lifecycle control over running jobs with context cancellation
- **Checkpoint-based Recovery** — Resume crashed jobs from last completed step
- **Resource Profiles** — `safe`, `balanced`, `aggressive` presets with concurrency, rate limits, CPU/RAM limits, and timeouts
- **Custom Profiles** — Define your own profiles with `recon config profiles create`
- **Template Engine** — 5 built-in YAML pipeline templates + create your own
- **Export System** — Export subdomains, alive hosts, URLs, or directories in text/JSON/CSV
- **Structured Logging** — Per-step logs with `--follow`, `--step`, and `--search` filters
- **SQLite Storage** — Zero-config WAL-mode database at `~/.recond/recond.db`
- **Retry with Exponential Backoff** — Automatic retry (3 attempts, 2s base, 30s max) on tool failures
- **Process Management** — Clean SIGTERM/SIGKILL with process groups (`Setpgid: true`)

## Quick Start

```sh
# Clone and build
git clone https://github.com/toprakpt1/recond.git
cd recond
make build

# Start the daemon
setsid ./bin/recond &

# Initialize config
./bin/recon config init

# Run a recon job
./bin/recon start example.com --profile safe

# Check status
./bin/recon status <job-id>

# Export results
./bin/recon export <job-id> --type subdomains -o subdomains.txt
```

## Installation

### Build from Source

`recond` requires **Go >= 1.21**.

```sh
git clone https://github.com/toprakpt1/recond.git
cd recond
make build
```

This builds two binaries into `bin/`:

| Binary | Description |
|--------|-------------|
| `bin/recond` | Daemon — long-running process that manages jobs |
| `bin/recon` | CLI — sends commands to daemon via Unix socket |

### Makefile Commands

```sh
make build          # Build both recond and recon binaries
make build-daemon   # Build only the daemon
make build-cli      # Build only the CLI
make run-daemon     # Build and run daemon in foreground
make run-cli        # Build and run CLI
make test           # Run all tests (go test ./...)
make lint           # Run linter (go vet ./...)
make clean          # Remove bin/ directory
```

### Install via `go install`

```sh
go install github.com/toprakpt1/recond/cmd/recond@latest
go install github.com/toprakpt1/recond/cmd/recon@latest
```

### Required Tools

The following tools must be installed and available in your `$PATH`:

| Tool | Purpose | Install |
|------|---------|---------|
| [subfinder](https://github.com/projectdiscovery/subfinder) | Subdomain discovery | `go install -v github.com/projectdiscovery/subfinder/v2/cmd/subfinder@latest` |
| [httpx](https://github.com/projectdiscovery/httpx) | HTTP probing | `go install -v github.com/projectdiscovery/httpx/cmd/httpx@latest` |
| [katana](https://github.com/projectdiscovery/katana) | Web crawling | `go install -v github.com/projectdiscovery/katana/cmd/katana@latest` |
| [gau](https://github.com/lc/gau) | URL discovery | `go install -v github.com/lc/gau/v2/cmd/gau@latest` |
| [ffuf](https://github.com/ffuf/ffuf) | Directory fuzzing | `go install -v github.com/ffuf/ffuf/v2@latest` |

> **Note:** Only the tools used in your pipeline need to be installed. Missing tools will cause their steps to fail gracefully with an error message.

## Usage

### Daemon

```sh
# Start in background (recommended)
setsid ./bin/recond &

# Start in foreground (useful for debugging)
./bin/recond
```

The daemon listens on a Unix socket at `~/.recond/recond.sock` and stores all data in `~/.recond/recond.db`.

### CLI Commands

#### Job Management

```sh
# Start a recon job
./bin/recon start example.com
./bin/recon start example.com --profile safe
./bin/recon start example.com --profile aggressive

# Check job status (shows per-step progress)
./bin/recon status <job-id>

# List all jobs
./bin/recon list
./bin/recon list --status running
./bin/recon list --status completed

# Pause / Resume / Stop
./bin/recon pause <job-id>
./bin/recon resume <job-id>
./bin/recon stop <job-id>

# Delete jobs
./bin/recon delete <job-id>
./bin/recon delete --completed          # Delete all completed jobs

# Retry a job (creates a new job)
./bin/recon retry <job-id>
./bin/recon retry <job-id> --from-step alive-check   # Skip completed steps

# Duplicate a job
./bin/recon duplicate <job-id>
```

#### Logs

```sh
# View all logs for a job
./bin/recon logs <job-id>

# Follow logs in real-time
./bin/recon logs <job-id> --follow

# Filter by step
./bin/recon logs <job-id> --step subdomain-discovery

# Search logs
./bin/recon logs <job-id> --search "error"

# Export logs to file
./bin/recon logs <job-id> --export logs.json
```

#### Export

```sh
# Export subdomains (text, default)
./bin/recon export <job-id> --type subdomains

# Export alive hosts as JSON
./bin/recon export <job-id> --type alive --format json

# Export URLs as CSV to file
./bin/recon export <job-id> --type urls --format csv -o urls.csv

# List available outputs
./bin/recon outputs <job-id>
```

| Export Type | Source Tool | Output File |
|-------------|-------------|-------------|
| `subdomains` | subfinder | `subfinder.txt` |
| `alive` | httpx | `httpx.txt` |
| `urls` | katana / gau | `katana.txt` / `gau.txt` |
| `directories` | ffuf | `directories.json` |

#### Templates

```sh
# List built-in templates
./bin/recon templates list

# Show template details
./bin/recon templates show full-recon

# Create custom template from YAML file
./bin/recon templates create my-template --file template.yaml

# Delete a custom template
./bin/recon templates delete my-template
```

#### Profiles

```sh
# List all profiles
./bin/recon config profiles list

# Show profile details
./bin/recon config profiles show safe

# Create a custom profile
./bin/recon config profiles create custom --concurrency 10 --rate-limit 50

# Delete a custom profile
./bin/recon config profiles delete custom
```

#### Config & Daemon

```sh
# Initialize default config
./bin/recon config init

# Show current config
./bin/recon config show

# Set a config value
./bin/recon config set default_profile safe

# Daemon management
./bin/recon daemon start
./bin/recon daemon stop
./bin/recon daemon status
./bin/recon daemon health
```

## Configuration

Configuration is stored at `~/.recond/config.yaml`:

```yaml
data_dir: ~/.recond
socket_path: ~/.recond/recond.sock
default_profile: balanced
max_retries: 3
retry_backoff: 2s
wordlist: ~/.recond/wordlists/common.txt

profiles:
  safe:
    concurrency: 3
    rate_limit: 10
    cpu_max: 20
    ram_max: 1GB
    timeout: 30s
    wordlist: ~/.recond/wordlists/common.txt
  balanced:
    concurrency: 10
    rate_limit: 50
    cpu_max: 50
    ram_max: 2GB
    timeout: 15s
    wordlist: ~/.recond/wordlists/common.txt
  aggressive:
    concurrency: 25
    rate_limit: 100
    cpu_max: 80
    ram_max: 4GB
    timeout: 10s
    wordlist: ~/.recond/wordlists/common.txt
```

Environment variables are supported with `RECOND_` prefix (e.g., `RECOND_DATA_DIR`).

### Wordlist

ffuf directory fuzzing requires a wordlist. By default, `~/.recond/wordlists/common.txt` is used.

- **Auto-download**: If the wordlist doesn't exist, it's automatically downloaded from [SecLists](https://github.com/danielmiessler/SecLists)
- **Custom wordlist**: Set globally in config or per-profile:

```yaml
# Global
wordlist: /path/to/wordlist.txt

# Per-profile
profiles:
  aggressive:
    wordlist: /path/to/big-wordlist.txt
```

### Resource Profiles

| Profile | Concurrency | Rate Limit | CPU Max | RAM Max | Timeout |
|---------|-------------|------------|---------|---------|---------|
| `safe` | 3 | 10 req/s | 20% | 1GB | 30s |
| `balanced` | 10 | 50 req/s | 50% | 2GB | 15s |
| `aggressive` | 25 | 100 req/s | 80% | 4GB | 10s |

### Pipeline Templates

| Template | Steps | Description |
|----------|-------|-------------|
| `full-recon` | 5 | subfinder → httpx → katana + gau → ffuf |
| `subdomain-only` | 1 | subfinder only |
| `alive-check` | 2 | subfinder → httpx |
| `url-collection` | 4 | subfinder → httpx → katana + gau |
| `directory-fuzz` | 3 | subfinder → httpx → ffuf |

Custom template example:

```yaml
name: my-pipeline
description: Custom recon pipeline
steps:
  - name: subdomain-discovery
    tool: subfinder
    order: 1
  - name: alive-check
    tool: httpx
    order: 2
  - name: crawling
    tool: katana
    order: 3
    parallel: true
  - name: url-collection
    tool: gau
    order: 4
    parallel: true
  - name: directory-fuzzing
    tool: ffuf
    order: 5
```

## Architecture

```
┌──────────────────────────────────────────────────────┐
│                    CLI (recon)                        │
│   start · status · pause · resume · export · logs    │
│   templates · profiles · delete · retry · duplicate  │
└────────────────────────┬─────────────────────────────┘
                         │  JSON over Unix Socket
                         │  (~/.recond/recond.sock)
┌────────────────────────▼─────────────────────────────┐
│                  Daemon (recond)                      │
│                                                       │
│  ┌──────────────┐  ┌─────────────┐  ┌─────────────┐ │
│  │   Storage     │  │  Pipeline   │  │   Runner    │ │
│  │  (SQLite)     │  │ Controller  │  │  Registry   │ │
│  │  jobs         │  │ sequential  │  │  subfinder  │ │
│  │  steps        │  │ step exec   │  │  httpx      │ │
│  │  outputs      │  │ checkpoint  │  │  katana     │ │
│  │  logs         │  │ progress    │  │  gau        │ │
│  └──────────────┘  └─────────────┘  │  ffuf       │ │
│                                      └─────────────┘ │
└──────────────────────────────────────────────────────┘
                         │
         ┌───────────────┼───────────────┐
         │               │               │
    ┌────▼────┐    ┌─────▼─────┐   ┌────▼────┐
    │subfinder│    │   httpx   │   │  katana │
    │  (OSINT) │    │  (probe)  │   │ (crawl) │
    └────┬────┘    └─────┬─────┘   └────┬────┘
         │               │               │
         ▼               ▼               ▼
    subfinder.txt   httpx.txt      katana.txt
                                      │
                              ┌───────┴───────┐
                              │               │
                         ┌────▼────┐    ┌────▼────┐
                         │   gau   │   │  ffuf   │
                         │ (URLs)  │   │  (fuzz) │
                         └────┬────┘   └────┬────┘
                              │              │
                         gau.txt     directories.json
```

### Data Flow

Each step reads from the previous step's output file:

```
subfinder (target)           → subfinder.txt (subdomains)
  └─ httpx (subfinder.txt)   → httpx.txt (alive hosts)
       ├─ katana (httpx.txt) → katana.txt (crawled URLs)
       ├─ gau (httpx.txt)    → gau.txt (discovered URLs)
       └─ ffuf (httpx.txt)   → directories.json (fuzzed dirs)
```

### SQLite Schema

| Table | Purpose |
|-------|---------|
| `jobs` | Job metadata: id, name, target, status, profile, timestamps |
| `steps` | Step state: status, progress, checkpoint, error message |
| `outputs` | Output files: path, kind, size |
| `logs` | Structured logs: level, message, step association |
| `targets` | Target tracking: value, type, status |
| `settings` | Key-value settings store |

### Process Management

- Each tool runs as a child process with `SysProcAttr{Setpgid: true}`
- On pause/stop: SIGTERM sent to process group, SIGKILL after 5s timeout
- Retry: exponential backoff (2s → 4s → 8s, max 30s), 3 attempts total
- Progress: polled every 3s during execution

## Project Structure

```
recond/
├── Makefile                        # Build, test, lint commands
├── cmd/
│   ├── recon/main.go               # CLI entry point
│   └── recond/main.go              # Daemon entry point
├── internal/
│   ├── cli/                        # Cobra CLI commands
│   │   ├── root.go                 # Root command + subcommand registration
│   │   ├── start.go                # recon start
│   │   ├── status.go               # recon status
│   │   ├── list.go                 # recon list
│   │   ├── pause.go                # recon pause
│   │   ├── resume.go               # recon resume
│   │   ├── stop.go                 # recon stop
│   │   ├── logs.go                 # recon logs (--follow, --step, --search, --export)
│   │   ├── export.go               # recon export + recon outputs
│   │   ├── templates.go            # recon templates list/show/create/delete
│   │   ├── config.go               # recon config show/set/init + profiles
│   │   ├── daemon.go               # recon daemon start/stop/status/health
│   │   ├── delete.go               # recon delete + delete --completed
│   │   └── retry.go                # recon retry + duplicate
│   ├── config/
│   │   ├── config.go               # Viper config, env vars, defaults
│   │   └── profiles.go             # Profile CRUD, LoadProfiles from YAML
│   ├── daemon/
│   │   ├── server.go               # Daemon struct, socket listener, all handlers
│   │   ├── client.go               # SendCommand() — CLI → daemon communication
│   │   └── socket.go               # Request/Response types
│   ├── export/
│   │   └── export.go               # ExportJobResults, readOutputFile, format conversion
│   ├── models/
│   │   ├── job.go                  # Job struct + JobStatus enum
│   │   ├── step.go                 # Step struct + StepStatus enum
│   │   ├── output.go               # Output struct + OutputKind enum
│   │   ├── log.go                  # Log struct + LogLevel enum
│   │   └── target.go               # Target struct
│   ├── pipeline/
│   │   └── pipeline.go             # DefaultSteps, CreateSteps, Execute, executeStep
│   ├── runner/
│   │   ├── runner.go               # Runner interface, RunOptions, StepResult
│   │   ├── executor.go             # Executor: RunWithRetry, runOnce, process mgmt
│   │   ├── registry.go             # Tool registry + CheckTools()
│   │   ├── subfinder.go            # subfinder adapter
│   │   ├── httpx.go                # httpx adapter
│   │   ├── katana.go               # katana adapter
│   │   ├── gau.go                  # gau adapter
│   │   └── ffuf.go                 # ffuf adapter
│   ├── storage/
│   │   ├── db.go                   # SQLite init, WAL mode, migrations
│   │   ├── jobs.go                 # Job CRUD
│   │   ├── steps.go                # Step CRUD + progress + checkpoint
│   │   ├── outputs.go              # Output CRUD
│   │   └── logs.go                 # Log insert + filtered list
│   └── template/
│       └── template.go             # Template struct, YAML loader, 5 built-ins
├── configs/                        # Default config templates
└── bin/                            # Compiled binaries (gitignored)
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the **GNU General Public License v3.0** — see the [LICENSE](LICENSE) file for details.

```
This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
```

---

<p align="center">
  <code>recond</code> is made with ❤️ by <a href="https://github.com/toprakpt1">Toprak Talha Karcılar</a>.
</p>
