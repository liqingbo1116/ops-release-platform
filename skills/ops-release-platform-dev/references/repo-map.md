# Repository Map

## Important Docs

- `README.md`: startup commands and high-level project notes.
- `docs/PRD.md`: product requirements and business direction.
- `docs/development-plan.md`: phase plan; use it to choose the next development target.
- `docs/codex-implementation-tasks.md`: initial Codex task split for MVP tasks 1-9.
- `docs/api-contract.md`: REST API contract.
- `docs/domain-model.md`: domain and database model guidance.
- `docs/state-machine.md`: status enum and state transition source of truth.
- `docs/integration-boundary.md`: adapter boundary for external systems.
- `docs/git-submit-workflow.md`: required workflow when the user asks to commit or push.

## Backend

- `backend/cmd/server`: Gin server entrypoint.
- `backend/internal/api`: routes, handlers, response envelope, API tests.
- `backend/internal/app`: server assembly, DB migration, Redis queue, integration suite wiring.
- `backend/internal/agent`: Redis Stream queue and Agent task lease/callback handling.
- `backend/internal/config`: environment variable loading.
- `backend/internal/domain`: API/domain DTOs.
- `backend/internal/repository`: GORM models, migrations, and persistent store implementations.
- `backend/internal/integration`: third-party adapter interfaces and real adapters.
- `backend/internal/middleware`: CORS and request middleware.

## Frontend

- `frontend/src/api`: backend API clients. Runtime mock fallback is forbidden.
- `frontend/src/components`: shared UI components.
- `frontend/src/pages`: route pages.
- `frontend/src/router`: Vue Router setup and auth guard tests.
- `frontend/src/stores`: Pinia stores.
- `frontend/src/style.css`: global visual system.

## Deployment Files

- `docker-compose.yml`: local compose stack. Root compose includes frontend, backend, postgres, redis.
- Remote server deployment details are intentionally not stored in Git.
- Remote deployment dir used previously: `/data/ops-release-platform`.

## Skills

- `skills/ops-release-platform-dev`: development runtime, validation, security, and Git workflow entrypoint.
- `skills/ops-release-platform-todo`: completed work, current local work, backlog, and next-task selection.
- `skills/ops-release-platform-architecture`: architecture map, module boundaries, data flow, and integration boundary.
- `skills/ops-release-platform-deployment`: local startup, docker-compose usage, and deployment/runtime topology.
- Update the relevant skill after major milestones so future sessions can rebuild context quickly.
