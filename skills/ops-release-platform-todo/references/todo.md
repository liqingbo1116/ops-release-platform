# Ops Release Platform TODO

Last updated: 2026-06-07

Always verify this file against `git status --short --branch`, `git log -1 --oneline`, and implementation docs before acting.

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

Latest pushed milestone:

- `f0b9f69 更新skills，规范部署规则`

## Current Local Work

- Release/deploy detail Agent task polling is implemented locally and not yet committed:
  - frontend Agent task status API
  - release creation API call and redirect with `agentTaskId`
  - release detail polling for `GET /api/agent-tasks/:id/status`
  - deploy detail polling for `GET /api/agent-tasks/:id/status`
  - backend release/deploy detail fallback for newly created mock IDs
  - API contract update for Agent task status and integration checks
- Release creation flow adjustment is in progress:
  - service release must not be based on a source baseline
  - service release source supports Jenkins Job and local Harbor image tag
  - both service release sources must eventually use the project Agent to sync image and update tag
  - service deployment should create deploy tasks for target-missing services
  - diff `MISSING_IN_TARGET` should be shown as service deployment
  - real Agent module directories should be created without implementation code

## Recommended Development Path

1. Complete the mock release/deploy runtime loop:
   - after creating release/deploy tasks, keep or expose the Agent task ID clearly
   - make release/deploy detail pages poll `GET /api/agent-tasks/:id/status`
   - render mock Agent status and logs in the frontend
2. Add backend `internal/service` when handlers coordinate multiple dependencies:
   - release service
   - deploy task service
   - environment check service
   - agent task status service
3. Update API contract docs:
   - `GET /api/agent-tasks/{id}/status`
   - `POST /api/environments/{id}/check` response `checks`
   - integration health response DTOs
4. Expand frontend tests:
   - changelog filtering
   - diff table filtering and non-publishable selection behavior
   - login/logout flow
   - Agent log polling display
5. Keep integration mock-first:
   - refine Jenkins, Registry/Harbor, Kubernetes adapter contracts
   - do not implement real adapters until explicitly requested
6. Replace mock data with persistence gradually:
   - release orders
   - deploy tasks
   - operation logs
   - changelog

## Next Suggested Tasks

1. Add backend service layer if API handlers start accumulating business orchestration.
2. Implement real project-side Agent scanner and reporter after the directory skeleton is agreed.
3. Add focused frontend tests for changelog filtering and diff table selection behavior.
4. Persist release orders, deploy tasks, operation logs, and changelog gradually.

## Validation Checklist

- `go test ./...`
- `npm run test:unit`
- `npm run build`
- Skill validation for all project skills:
  - `ops-release-platform-dev`
  - `ops-release-platform-todo`
  - `ops-release-platform-architecture`
  - `ops-release-platform-deployment`
- Docker compose config only if Docker is available locally.

## Do Not Track Here

- Server IPs or SSH ports.
- Database or Redis connection strings.
- Passwords or tokens.
- Contents of `.secrets/`.
