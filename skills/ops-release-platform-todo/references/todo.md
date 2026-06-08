# Ops Release Platform TODO

Last updated: 2026-06-08

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

## V1 Mainline Goal

V1 must prioritize functional closure over optimization work. The minimum acceptable V1 outcome is:

- the platform can manage remote project environments
- the platform can create and track remote deployment/release tasks
- Agent-driven execution and status reporting are visible end to end

Until this mainline is complete, performance tuning, warning cleanup, and refactor-only work stay behind feature work unless they block delivery.

## Recommended Development Path

1. Complete runtime snapshot and baseline generation flow.
   - define backend-owned runtime snapshot acquisition path
   - support baseline creation from environment runtime data
   - support baseline lock state transition
   - make baseline detail pages consume the generated data path
2. Complete backend-owned diff and action classification.
   - classify `NEED_UPDATE` as release candidates
   - classify `MISSING_IN_TARGET` as deploy candidates
   - keep action rules out of frontend-only logic
3. Complete release management closure.
   - single-service release
   - multi-service release
   - baseline diff release
   - retain `agentTaskId` and surface task state/logs in details
   - support failure reason display and retry state modeling
4. Complete deployment management closure for remote project environments.
   - create deploy tasks for target-missing services
   - model deploy steps and state transitions
   - support retry / skip / manual confirm
   - expose deploy detail logs and final result clearly
5. Complete Agent task protocol.
   - heartbeat
   - task pull
   - step status report
   - log report
   - final result report
6. Complete audit and permission requirements for release/deploy flows.
   - operator
   - target environment
   - affected services
   - source/target tag changes
   - success/failure and failed step
7. Gradually persist runtime data instead of relying only on mock payloads.
   - baselines
   - release orders
   - deploy tasks
   - operation logs
8. Only after the above, continue non-functional work.
   - bundle optimization
   - build warning cleanup
   - UI polish
   - refactor-only cleanup

## Next Suggested Tasks

1. Finish backend baseline generation and lock flow so the current baseline pages are backed by real service logic.
2. Move diff classification rules fully into backend service/domain logic and align release/deploy creation with that split.
3. Continue release/deploy detail closure around `agentTaskId`, structured logs, retry states, and audit fields.

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
