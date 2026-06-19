# Architecture Reference

Last updated: 2026-06-07

This is a compact architecture guide for Codex sessions. Use project docs as source of truth for detailed requirements.

## Runtime Topology

- Frontend runs locally during development.
- Backend runs locally during development.
- PostgreSQL and Redis are remote development services; connection settings live only in `.secrets/`.
- Production-like local compose still exists, but normal development should not start local PostgreSQL/Redis unless the user explicitly changes the rule.

## Backend Layers

- `cmd/server`: process entrypoint.
- `internal/app`: assembly layer. Owns config loading results, DB migration trigger, Redis queue setup, integration suite setup, and router startup.
- `internal/api`: HTTP boundary. Owns Gin routes, handlers, response envelope, request/response behavior, and API tests.
- `internal/domain`: DTOs and domain-shaped API structs.
- `internal/repository`: mock repository, embedded mock data, GORM models, migrations.
- `internal/agent`: Redis Stream queue and mock Agent worker.
- `internal/integration`: external-system adapter contracts and mock adapters.
- `internal/config`: environment variables.
- `internal/middleware`: cross-cutting HTTP middleware.

Introduce `internal/service` when handlers need to coordinate multiple repositories, queues, or adapters with meaningful business rules.

## Agent Module Skeleton

The real project-side Agent source tree is reserved under top-level `agent/`.

- `agent/cmd/agent`: future Agent process entrypoint.
- `agent/internal/config`: Agent config loading.
- `agent/internal/heartbeat`: heartbeat reporting.
- `agent/internal/kubernetes`: K8s workload scanning and rollout operations.
- `agent/internal/harbor`: Harbor/image query and sync helpers.
- `agent/internal/reporter`: RuntimeSnapshot, logs, and task result reporting.
- `agent/internal/runtime`: local runtime and environment discovery.
- `agent/internal/task`: task pull/execute loop.

Current status: directories only. Do not implement real Agent code until requested.

## Frontend Layers

- `src/api`: backend API clients and mock fallback access.
- `src/stores`: Pinia state containers.
- `src/router`: route definitions and auth guards.
- `src/pages`: route-level views.
- `src/components`: reusable UI components.
- `src/style.css`: global visual system.

Keep page components responsible for interaction composition; move reusable table/panel behavior into components when repeated.

## Data Flow

1. User acts in Vue page.
2. Pinia store or API module calls backend REST API.
3. Gin handler reads mock repository or coordinates queue/adapter.
4. For release/deploy creation, backend enqueues Redis Stream task when `REDIS_ADDR` is configured.
5. Mock Agent worker consumes stream, writes task status and logs to Redis keys.
6. Frontend can poll task status API when task IDs are available.

## Environment Management Rules

- Environment page exposes only local environment and remote environment.
- Local environment is internally `DIRECT`, binds platform-managed K8s namespace, Harbor project, and optional Jenkins view.
- Remote environment is internally `AGENT`, does not bind platform-managed K8s/Harbor/Jenkins resources, and depends on Agent heartbeat/reporting.
- `networkMode` remains an internal compatibility field and must not be presented as a user choice.
- Newly created environment status starts as `UNKNOWN`; local connection tests or Agent reports update it later.

## Release And Deployment Flow

- Service release targets services that already exist in the target environment. It must not be based on a source baseline.
- Service release has two sources:
  - `JENKINS_JOB`: choose a Jenkins Job associated by view or naming/label convention, build jar/dist, build image, and push to local Harbor.
  - `LOCAL_HARBOR_IMAGE`: scan local Harbor image versions for the service and choose a tag directly; do not choose or trigger Jenkins Job for this path.
- Both service release sources eventually require the project-environment Agent to sync the image to the project environment and update workload tag. Local environments currently keep using GitOps.
- Service deployment targets services missing in the target environment. It is based on source baseline/production environment comparison and creates a deploy task.
- `MISSING_IN_TARGET` diff items represent service deployment candidates, not normal service release candidates.

## External Integration Boundary

Current adapter contracts:

- `JenkinsAdapter`: trigger build, get build status.
- `RegistryAdapter`: check registry connection, get image, sync image.
- `KubernetesAdapter`: check cluster connection, list workloads, set image, get rollout status.

Rules:

- No direct SDK calls from handlers.
- No real Jenkins/Harbor/Kubernetes/Nacos/GitLab/ArgoCD calls unless explicitly requested.
- Real adapter implementations must preserve interface contracts and must not store credentials in Git.

## Task Queue Boundary

- Redis Stream is the platform-to-Agent task handoff.
- Mock worker simulates steps and appends logs.
- API must degrade cleanly when Redis is not configured.

## Database Boundary

- GORM models and migrations live under `backend/internal/repository`.
- MVP uses PostgreSQL for relational records.
- Large log bodies can remain mocked or cached for now; do not introduce object storage unless requested.

## Docker Boundary

- Root `docker-compose.yml` describes full stack.
- Development mode uses local frontend/backend and remote PostgreSQL/Redis from `.secrets/`.
- Do not place remote server connection details in compose files.
