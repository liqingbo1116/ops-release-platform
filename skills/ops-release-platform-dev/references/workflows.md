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

V1 mainline rule:

- `DATABASE_DSN` and `REDIS_ADDR` must point to the remote services prepared for the project, currently the remote host `100.120.3.230` recorded in `.secrets/`.
- Do not switch PostgreSQL or Redis back to local mock containers for V1 mainline development.
- If remote PostgreSQL or Redis is unavailable, environment management and all downstream real-data phases are blocked.
- Frontend/backend local startup is only the process mode. Data dependencies must still be real.

## V1 Phase Prerequisites

Use this gate list before starting the next feature area:

1. Environment management
   - Required: frontend, backend, remote PostgreSQL, remote Redis, `.secrets/` loaded.
   - Blocker: if remote PostgreSQL/Redis is not ready, do not replace mock and do not move on.
2. Agent management
   - Required: environment data already real, remote Linux host, built agent binary, `-f` config file support, outbound connectivity to platform.
   - Blocker: if a real environment cannot be created first, agent registration and binding cannot be validated.
3. Release creation
   - Required: real environments, real agents, real release source data.
   - Blocker: if source service/version data is mock, stop here.
4. Baseline management
   - Required: real baseline source such as Kubernetes runtime snapshot or persisted business snapshot data.
   - Blocker: if baseline is mock, stop here.
5. Deployment execution
   - Required: agent executor tools, cluster credentials, registry credentials, network connectivity.
   - Blocker: if the agent cannot reach the real target infrastructure, stop here.
6. Detail pages
   - Required: persisted task/step/log/result data from real execution.
   - Blocker: if detail rendering still depends on mock task records, stop here.
7. Auth and permissions
   - Required: real login, user, role, permission, and environment-level authorization data.
   - Blocker: if auth is mock, V1 is not complete.
8. Final mock cleanup
   - Required: all previous phases complete.
   - Blocker: if any runtime mock fallback is still required for the mainline path, V1 is not complete.

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

- For the V1 mainline, do not use mock adapters as the completion standard for any phase.
- Mock adapters are only allowed for isolated scaffolding or when the user explicitly asks for a mock-only experiment.
- If Jenkins, Harbor/Registry, Kubernetes, Agent runtime, PostgreSQL, Redis, or another real dependency is required to replace mock data and is not ready, record the blocker and stop at that phase.
- Keep business/API code dependent on interfaces from `backend/internal/integration`.
- Do not call Jenkins, Harbor, Kubernetes, GitLab, ArgoCD, or Nacos SDKs directly from handlers.
- Real adapters require an explicit user request and must avoid committing credentials.
