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

For real environment integration checks during development, also load the integration secret file for the current shell:

- `.secrets/integration-connections.env`
- `.secrets/integration-connections.ps1`

Development integration variables are:

- `INTEGRATION_MODE`
- `LOCAL_HARBOR_URL`
- `LOCAL_HARBOR_USERNAME`
- `LOCAL_HARBOR_PASSWORD`
- `LOCAL_K8S_KUBECONFIG`
- `REMOTE_HARBOR_URL`
- `REMOTE_HARBOR_USERNAME`
- `REMOTE_HARBOR_PASSWORD`
- `REMOTE_K8S_KUBECONFIG`
- `INTEGRATION_HTTP_TIMEOUT_MS`

V1 mainline rule:

- `DATABASE_DSN` and `REDIS_ADDR` must point to the remote services prepared for the project, currently the remote host `100.120.3.230` recorded in `.secrets/`.
- K8s, Harbor, and Jenkins are platform-maintained resource master data. Environment records reference resource IDs through binding records. One environment may bind multiple K8s namespaces, Harbor projects, and Jenkins views; default legacy fields are derived from the default binding.
- Environment `type` is local vs remote execution (`LOCAL` / `PROJECT`). Environment `deployTargetType` is the runtime target; V1 implements `KUBERNETES` and only reserves `DOCKER_COMPOSE`.
- Remote/project environments bind platform-side local Jenkins view and Harbor project for build/image source selection. Remote K8s/runtime operations are never platform-direct; they are Agent tasks and Agent-reported state.
- Resource create forms must match the tool's natural inputs: K8s kubeconfig/context, Harbor URL + HTTP/HTTPS + username/password, Jenkins URL + username/password or API token. Users must not type `credentialRef`.
- Resource status is system-owned. Keep connection test and refresh/probe actions. Cache K8s namespaces, Harbor projects, and Jenkins views/jobs; on refresh failure, keep the old cache and show the failure reason.
- Probe execution depends on network mode: local/direct resources are checked by the platform backend; remote/project resources must be checked by Agent tasks and reported back to the platform.
- `.secrets/` is only for development-stage private values and process startup. It is not the formal platform resource master data source.
- Platform resource records may store non-secret metadata such as API server or service URL. Credentials must be represented internally; do not store real passwords, tokens, or kubeconfig contents in environment rows.
- Do not switch PostgreSQL or Redis back to local mock containers for V1 mainline development.
- If remote PostgreSQL/Redis is unavailable, or real K8s/Harbor/Jenkins resources cannot be created, tested, probed, and cached, resource management and all downstream real-data phases are blocked.
- Frontend/backend local startup is only the process mode. Data dependencies must still be real.

## V1 Phase Prerequisites

Use this gate list before starting the next feature area:

1. Resource management
   - Required: frontend, backend, remote PostgreSQL, remote Redis, real K8s kubeconfig, real Harbor connection input, real Jenkins connection input when Jenkins is in scope, and remote Agent runtime before accepting remote/project resource probes.
   - Blocker: if resource forms expose `credentialRef`, status is user-editable, probe lists are mock, refresh is missing, or remote probes bypass Agent, stop here.
2. Environment management
   - Required: phase 1 complete, real resource records, cached namespace/project/view options, and real backend environment persistence.
   - Blocker: if environments embed credentials, use mock scope options, require users to maintain both environment ID and environment code, or directly connect from platform backend to remote K8s, stop here.
3. Agent management and remote probing
   - Required: real environments, remote Linux host, built agent binary, `-f` config file support, outbound connectivity to platform, and Agent-side access to remote K8s/Harbor/Jenkins.
   - Blocker: if a real environment cannot be created first, or remote resource probe status/cache cannot be reported by Agent, stop here.
4. Service and version sources
   - Required: real environments, real agents, real service/version source data from Jenkins, Harbor, Kubernetes runtime, or persisted runtime snapshots.
   - Blocker: if source service/version data is mock, stop here.
5. Release creation
   - Required: phase 1 through phase 4 complete.
   - Blocker: if release form data is mock, stop here.
6. Baseline management
   - Required: real baseline source such as Kubernetes runtime snapshot or persisted business snapshot data.
   - Blocker: if baseline is mock, stop here.
7. Deployment execution
   - Required: agent executor tools, cluster credentials, registry credentials, network connectivity.
   - Blocker: if the agent cannot reach the real target infrastructure, stop here.
8. Detail pages
   - Required: persisted task/step/log/result data from real execution.
   - Blocker: if detail rendering still depends on mock task records, stop here.
9. Auth and permissions
   - Required: real login, user, role, permission, and environment-level authorization data.
   - Blocker: if auth is mock, V1 is not complete.
10. Final mock cleanup
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
