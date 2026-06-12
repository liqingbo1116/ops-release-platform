# Project State

Last updated: 2026-06-12

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

V1 mainline development is in phase 1: Environment management.

Completed in phase 1 local work:

- Frontend environment API list no longer imports or falls back to mock data; it calls `/api/environments`.
- Backend runtime requires real `DATABASE_DSN` and `REDIS_ADDR` before startup.
- Backend runtime wires `DatabaseStore` directly for the main repository, so environment list/detail/create/update use PostgreSQL-backed data in normal runtime.
- Environment dependency check now rejects mock integrations instead of returning fake healthy Kubernetes/Registry checks.

Still blocking phase 1 completion:

- Real Kubernetes and Registry integration adapters/configuration are not implemented yet.
- Environment dependency visibility cannot be marked complete until those real adapters are available and configured.
- Do not move to phase 2 Agent management until environment dependency checks use real integrations or the phase-1 scope is explicitly reduced.

Validation for this local work:

- Targeted frontend environment tests passed before the latest environment-check hardening:
  - `npm run test:unit -- src/api/environments.test.ts src/pages/EnvironmentPage.test.ts`
- Targeted backend app/API tests passed before the latest environment-check hardening:
  - `go test ./internal/app ./internal/api`
- Rerun targeted backend and frontend validation after any follow-up edits.

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
