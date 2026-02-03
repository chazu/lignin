# Agent Instructions

This project uses **bd** (beads) for issue tracking. Run `bd onboard` to get started.

## Build Prerequisites

- **Go 1.21+** (project uses go 1.24.5; any 1.21+ should work)
- **Node.js 18+** (for Vite-based frontend build)
- **Wails CLI v2**: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- **npm dependencies**: `cd frontend && npm install`
- **Optional**: Manifold C library for CGo kernel (not required for MVP)

### Platform-specific notes

**macOS:**
- `wails build` produces a `.app` bundle under `build/bin/`
- `make build-macos` builds a universal binary (arm64 + amd64) via `-platform darwin/universal`

**Linux:**
- Requires `webkit2gtk-4.0` and related dev headers
- On Debian/Ubuntu: `sudo apt install libgtk-3-dev libwebkit2gtk-4.0-dev`
- On Fedora: `sudo dnf install gtk3-devel webkit2gtk4.0-devel`
- `make build-linux` targets `linux/amd64`

### Makefile targets

| Target | Description |
|---|---|
| `make build` | Build for current platform (`wails build`) |
| `make build-macos` | Build macOS universal binary (arm64 + amd64) |
| `make build-linux` | Build Linux amd64 binary |
| `make dev` | Development mode with hot reload |
| `make test` | Run all Go tests |
| `make test-v` | Run Go tests with verbose output |
| `make lint` | Run `go vet` |
| `make clean` | Clean build artifacts |

## Quick Reference

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --status in_progress  # Claim work
bd close <id>         # Complete work
bd sync               # Sync with git
```

## Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd sync
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds

