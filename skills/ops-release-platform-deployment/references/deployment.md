# Deployment And Local Runtime

Last updated: 2026-06-07

## Development Runtime

Development must use this topology:

- Frontend: local process started by npm.
- Backend: local process started by Go.
- Agent: binary built from `agent/cmd/agent`, started directly with `-f <config-file>` when Agent-side development or remote-like verification is needed.
- PostgreSQL: remote development service configured in `.secrets/local-dev-env.ps1`.
- Redis: remote development service configured in `.secrets/local-dev-env.ps1`.

These remote PostgreSQL and Redis settings are the fixed development default for this repository. If `.secrets/local-dev-env.ps1` is missing, do not substitute local services, compose services, or ad hoc connection strings. Restore the secret file first.

Do not use docker-compose to start frontend or backend during development. Agent development should also prefer direct binary startup over docker-compose.

For formal project-environment deployment, the Agent still runs through `agent/docker-compose.yml` on the remote Linux host.

## Frontend Commands

Install dependencies:

```powershell
cd frontend
npm install
```

Start development server:

```powershell
cd frontend
npm run dev -- --host 0.0.0.0
```

Linux/Bash:

```bash
cd /home/kuma/桌面/projects/ops-release-platform/frontend
npm run dev -- --host 0.0.0.0
```

Build:

```powershell
cd frontend
npm run build
```

Unit tests:

```powershell
cd frontend
npm run test:unit
```

## Backend Commands

Load local development environment variables before starting the backend:

```powershell
. .\.secrets\local-dev-env.ps1
```

If this file is missing, stop and restore it before starting the backend. Do not continue with empty `DATABASE_DSN` / `REDIS_ADDR` in normal development.

Start backend from the backend directory:

```powershell
cd backend
. ..\.secrets\local-dev-env.ps1
go run ./cmd/server
```

If already inside `backend`, load the env file with:

```powershell
. ..\.secrets\local-dev-env.ps1
go run ./cmd/server
```

Tests:

```powershell
cd backend
go test ./...
```

Linux/Bash startup from this repository must load the same `.secrets/local-dev-env.ps1` file by converting the PowerShell assignments in the current shell. Do not print the converted values:

```bash
cd /home/kuma/桌面/projects/ops-release-platform/backend
source <(sed -E 's/^\$env:([A-Za-z_][A-Za-z0-9_]*)[[:space:]]*=[[:space:]]*(.*)$/export \1=\2/' ../.secrets/local-dev-env.ps1)
go run ./cmd/server
```

## Long Running Local Startup

When the user asks to start the platform frontend and backend for later testing, this is the fixed startup method. Start them as detached local processes so they survive the current Codex command session. Do not use plain background `go run ... &` from a tool command because the process may be cleaned up with the session.

Before starting or restarting, check the real listener processes:

```bash
ss -ltnp | rg ':8080|:5173' || true
ps -ef | rg 'go run ./cmd/server|/tmp/go-build|npm run dev|vite' | rg -v rg || true
```

If port `8080` is occupied by a compiled `server` child process from a previous `go run`, stop that actual listener before restarting. Stopping only the `go run` parent is not enough if the compiled child keeps listening.

Backend:

```bash
setsid -f bash -lc 'cd /home/kuma/桌面/projects/ops-release-platform/backend && source <(sed -E '\''s/^\$env:([A-Za-z_][A-Za-z0-9_]*)[[:space:]]*=[[:space:]]*(.*)$/export \1=\2/'\'' ../.secrets/local-dev-env.ps1) && exec go run ./cmd/server > /tmp/ops-release-platform-backend.log 2>&1'
```

Frontend:

```bash
setsid -f bash -lc 'cd /home/kuma/桌面/projects/ops-release-platform/frontend && exec npm run dev -- --host 0.0.0.0 > /tmp/ops-release-platform-frontend.log 2>&1'
```

Validation:

```bash
ss -ltnp | rg ':8080|:5173'
curl -fsS http://127.0.0.1:8080/api/environments
curl -I -fsS http://127.0.0.1:5173/
tail -n 20 /tmp/ops-release-platform-backend.log
tail -n 20 /tmp/ops-release-platform-frontend.log
```

Do not use `GET /api/health` as the backend startup check unless that endpoint is added later; the current verified backend check is `GET /api/environments`.

If a frontend startup log includes runtime mock fallback warnings from older release or baseline sample endpoints, do not treat that alone as a backend startup failure. Verify the real API being worked on, such as environment management or K8s resource management, against the backend endpoint directly. For K8s resource field changes, verify `GET /api/kubernetes-clusters` and confirm the changed fields are present in the response.

## Agent Commands

Build the Agent binary:

```powershell
cd agent
go build -o bin/ops-release-agent ./cmd/agent
```

Start the Agent directly with an explicit config file:

```powershell
cd agent
.\bin\ops-release-agent.exe -f .\.env.example
```

On Linux or a remote Linux host:

```bash
cd agent
go build -o bin/ops-release-agent ./cmd/agent
./bin/ops-release-agent -f ./.env.example
```

Use a copied env file with real non-secret identifiers for remote verification. The config file format is the same as `agent/.env.example`.

## Docker Compose Rule

Do not run frontend/backend through docker-compose in normal development.

Allowed docker-compose usage:

- Validating compose syntax when Docker is available.
- Explicit infrastructure/deployment tasks requested by the user.
- Remote or server-side PostgreSQL/Redis-only deployment tasks.
- Packaging or validating the optional Agent container deployment path.
- Formal Agent deployment on the project-environment Linux host.

Disallowed during development:

- `docker compose up frontend`
- `docker compose up backend`
- `docker compose up` as the default way to run the standalone Agent during development
- `docker compose up` for the purpose of running local frontend/backend development servers

## Configuration

Tracked files may mention `.secrets/local-dev-env.ps1` as the secret source, but must not include its values.

The env file should provide:

- `APP_PORT`
- `DATABASE_DSN`
- `REDIS_ADDR`
- `INTEGRATION_MODE=mock`

`DATABASE_DSN` and `REDIS_ADDR` in this file are the repository's fixed remote development endpoints. Keep them in `.secrets/local-dev-env.ps1` only and do not copy them into tracked files.

## Reporting

When reporting local startup or deployment status:

- Say whether frontend/backend are local processes.
- Say whether the Agent is running from a directly built binary or from docker-compose.
- Say whether PostgreSQL/Redis are remote services.
- Do not print the actual remote host, ports, passwords, DSNs, SSH details, or tokens.
