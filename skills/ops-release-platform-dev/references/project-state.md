# Project State

Last updated: 2026-06-18

Always verify with `git status --short --branch` and `git log -1 --oneline`; this file is an onboarding aid, not a substitute for checking the working tree.

## Completed And Pushed

- Task 1: initialized frontend, backend, and docker-compose.
- Task 2: implemented frontend static pages from the HTML prototype with mock data.
- Task 3: implemented backend mock REST API.
- Task 4: added mock login, route guard, user/role/permission pages and APIs.
- Task 5: added changelog page and mock API.
- Task 6: connected frontend API layer to backend with mock-mode fallback.
- Task 7: added PostgreSQL/GORM model and migration foundation.
- Task 8: added Redis Stream mock Agent worker.
- Task 9: added backend API tests and frontend Vitest tests.

Latest pushed commit at time of this note:

- `acf8d9f 补充 Redis 任务模拟与测试`

## Current Local Work

V1 mainline development has completed phase 1: Environment management. The next phase is phase 2: Agent management.

Completed in phase 1:

- Frontend environment API list no longer imports or falls back to mock data; it calls `/api/environments`.
- Backend runtime requires real `DATABASE_DSN` and `REDIS_ADDR` before startup.
- Backend runtime wires `DatabaseStore` directly for the main repository, so environment list/detail/create/update use PostgreSQL-backed data in normal runtime.
- Environment dependency check now rejects mock integrations instead of returning fake healthy Kubernetes/Registry checks.
- Environment records now carry `clusterId` and `registryId`, using logical IDs `local` and `remote` for real integration selection.
- Backend `INTEGRATION_MODE=real` now supports Harbor systeminfo and Kubernetes readyz connectivity checks through config loaded from `.secrets/integration-connections.*`.
- Frontend environment create/edit and connection drawer now expose logical cluster/registry IDs without exposing secrets.
- Skill and docs now record the `.secrets/` integration rule and real-data gate for environment management.
- Kubeconfig paths from `.secrets/` resolve from repo root, backend runtime, or package test working directories.
- Real environment checks passed through `POST /api/environments/:id/check` for both `env-local-prod` and `env-project-xjzt-test`.

Phase 1 completion evidence:

- `env-local-prod`: Kubernetes logical ID `local` healthy, Harbor logical ID `local` healthy.
- `env-project-xjzt-test`: Kubernetes logical ID `remote` healthy, Harbor logical ID `remote` healthy.
- Backend startup used real PostgreSQL, real Redis, and `INTEGRATION_MODE=real`.
- The remote agent heartbeat and lease endpoints returned HTTP 200 while the backend was running.

Next phase gate:

- Continue with phase 2 Agent management.
- Do not start release creation until Agent registration, heartbeat, environment binding, online status, and task lease data use real backend data with the real environment records from phase 1.

Validation for this local work:

- Backend tests:
  - `go test ./...`
- Environment check API:
  - `POST /api/environments/env-local-prod/check`
  - `POST /api/environments/env-project-xjzt-test/check`
- Frontend tests:
  - `npm run test:unit`
- Frontend build:
  - `npm run build`

## Known Warnings

- Frontend build can emit Rolldown warnings for third-party `#__PURE__` annotations from `@vueuse/core`.
- Frontend build can warn about a large JS chunk.
- These warnings have not blocked successful builds.

## Security State

- Server and database credentials are stored outside Git under `.secrets/` or conversation-local context.
- `.secrets/` must never be staged or committed.
- Do not print deployment passwords or SSH credentials in summaries.

## Local Development Runtime

- Frontend runs locally with `npm run dev`.
- Backend runs locally with `go run ./cmd/server`.
- PostgreSQL and Redis for development are remote services; load their connection settings from `.secrets/local-dev-env.ps1`.
- Do not use local PostgreSQL/Redis containers for normal development unless the user explicitly changes this rule.
- Do not commit or quote the remote host, connection string, SSH details, or credentials.
