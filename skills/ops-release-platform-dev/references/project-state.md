# Project State

Last updated: 2026-06-07

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

Phase 4 adapter preparation is currently implemented locally and not yet committed at the time this file was created:

- Added backend integration interfaces for Jenkins, Registry/Harbor, and Kubernetes.
- Added mock integration suite and adapter tests.
- Wired environment connection check through mock Kubernetes and Registry adapters.
- Added `INTEGRATION_MODE=mock` config.
- Updated README and docker-compose with integration mode notes.

Validation already run for this local work:

- `go test ./...` passed.
- `npm run test:unit` passed.
- `npm run build` passed.
- `docker compose config` could not run because local Docker command was unavailable.

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
