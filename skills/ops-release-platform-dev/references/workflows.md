# Workflows

## Before Development

Run:

```powershell
git status --short --branch
git log -1 --oneline
```

Then read only the docs needed for the task. For broad "continue development" requests, start from:

```powershell
Get-Content docs\development-plan.md
Get-Content docs\codex-implementation-tasks.md
```

## Validation

Backend change:

```powershell
cd backend
go test ./...
```

Frontend change:

```powershell
cd frontend
npm run test:unit
npm run build
```

Compose change:

```powershell
docker compose config
```

If Docker is unavailable locally, report that explicitly.

Clean generated frontend output before status checks or commits:

```powershell
Remove-Item -Recurse -Force frontend\dist -ErrorAction SilentlyContinue
```

## Local Development Runtime

During development, run frontend and backend locally. PostgreSQL and Redis must be the remote services recorded in the local `.secrets/` files, not local containers.

Strict rule:

- Frontend must be started with npm commands, normally `npm run dev`.
- Backend must be started with Go commands, normally `go run ./cmd/server`.
- Do not use docker-compose to start frontend or backend during development.
- Use `ops-release-platform-deployment` for detailed runtime and deployment rules.

Before starting the backend in PowerShell, load the local secret environment file:

```powershell
. .\.secrets\local-dev-env.ps1
```

Required backend environment after loading that file:

- `APP_PORT`
- `DATABASE_DSN`
- `REDIS_ADDR`
- `INTEGRATION_MODE=mock`

Do not copy the actual remote host, database connection string, Redis address, or credentials from `.secrets/` into tracked files or final summaries.

### Frontend Commands

Install dependencies:

```powershell
cd frontend
npm install
```

Start local dev server:

```powershell
cd frontend
npm run dev
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

### Backend Commands

Install/update Go modules:

```powershell
cd backend
go mod tidy
```

Start local backend with remote PostgreSQL and Redis:

```powershell
. ..\.secrets\local-dev-env.ps1
go run ./cmd/server
```

Tests:

```powershell
cd backend
go test ./...
```

## Git Submission

When the user asks to commit, push, submit, or "提交":

1. Follow `docs/git-submit-workflow.md`.
2. Re-run relevant validation for changed areas.
3. Stage only files related to the current task.
4. Scan staged diff:

```powershell
git diff --cached | Select-String -Pattern 'DEPLOY_PASSWORD|DATABASE_DSN|REDIS_ADDR|\.secrets|ssh|password|passwd|secret|credential' -CaseSensitive
git diff --cached --check
```

5. Commit with a concise Chinese message.
6. Push to the current branch, normally `origin main`.
7. Finish with commit hash, pushed branch, validation results, warnings, and final `git status`.

## Security

- Never commit `.secrets/`.
- Never commit server credentials, SSH credentials, database passwords for real environments, or private deployment notes.
- Do not include the server password or SSH details in final summaries.
- Mock passwords such as `password:"mock"` in tests are allowed.

## External Integrations

- Use mock adapters by default.
- Keep business/API code dependent on interfaces from `backend/internal/integration`.
- Do not call Jenkins, Harbor, Kubernetes, GitLab, ArgoCD, or Nacos SDKs directly from handlers.
- Real adapters require an explicit user request and must avoid committing credentials.
