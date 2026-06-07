# Ops Release Platform TODO

Last updated: 2026-06-07

Always verify this file against `git status`, `git log`, and implementation docs before acting.

## Done And Pushed

- Task 1: frontend/backend/docker-compose initialization.
- Task 2: frontend static pages and mock data.
- Task 3: backend mock REST API.
- Task 4: mock login, route guard, user/role/permission pages and APIs.
- Task 5: changelog page and mock API.
- Task 6: frontend/backend API integration with mock fallback.
- Task 7: PostgreSQL/GORM model and migration foundation.
- Task 8: Redis Stream mock Agent worker.
- Task 9: backend API tests and frontend Vitest tests.

Latest pushed milestone:

- `acf8d9f 补充 Redis 任务模拟与测试`

## Current Local Work

These items are implemented locally but not committed at the time of this update:

- Phase 4 mock integration adapter preparation:
  - Jenkins, Registry/Harbor, Kubernetes adapter interfaces.
  - Mock adapter suite.
  - Environment connection check through mock Kubernetes and Registry adapters.
  - `INTEGRATION_MODE=mock` configuration.
  - Adapter tests.
- Project skills:
  - `ops-release-platform-dev`
  - `ops-release-platform-todo`
  - `ops-release-platform-architecture`
  - `ops-release-platform-deployment`

## Next Suggested Tasks

1. Commit the local Phase 4 adapter and skills work after validation.
2. Add backend service layer if API handlers start accumulating business orchestration.
3. Add API contract entries for integration health checks if the response shape should be public.
4. Extend frontend release/deploy details to poll `GET /api/agent-tasks/:id/status` when a created task ID is available.
5. Add focused tests for changelog filtering and diff table selection behavior if frontend test coverage is expanded.

## Validation Checklist For Current Local Work

- `go test ./...`
- `npm run test:unit`
- `npm run build`
- Skill validation for all project skills:
  - `ops-release-platform-dev`
  - `ops-release-platform-todo`
  - `ops-release-platform-architecture`
- Docker compose config if Docker is available locally.

## Do Not Track Here

- Server IPs or SSH ports.
- Database or Redis connection strings.
- Passwords or tokens.
- Contents of `.secrets/`.
