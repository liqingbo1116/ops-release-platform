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

V1 mainline development is closing phase 1: Environment management. Do not advance to release creation until the environment page uses platform-maintained K8s/Harbor/Jenkins resource data and environment creation/editing follows the rules below.

Completed in phase 1:

- Frontend environment API list no longer imports or falls back to mock data; it calls `/api/environments`.
- Backend runtime requires real `DATABASE_DSN` and `REDIS_ADDR` before startup.
- Backend runtime wires `DatabaseStore` directly for the main repository, so environment list/detail/create/update use PostgreSQL-backed data in normal runtime.
- Environment dependency check now rejects mock integrations instead of returning fake healthy Kubernetes/Registry checks.
- K8s clusters, Harbor registries, and Jenkins instances are now platform-maintained resource master data.
- Environment records carry `clusterId`, `registryId`, and `jenkinsId` as references to those reusable resources.
- Environment records also carry environment-level scope fields: `namespace` for Kubernetes, `registryProject` for Harbor, and `jenkinsView` for Jenkins.
- Users only enter environment name and environment code. The backend generates the environment ID as `env-<code>` when the create request omits `id`.
- Environment page has separate tabs for maintaining K8s, Harbor, and Jenkins resources. Environments associate those resource rows and their per-environment scopes.
- `.secrets/` is development-only for private runtime values. Formal resource master data belongs in the platform database, and credential fields store only `credentialRef`.
- Backend `INTEGRATION_MODE=real` now supports Harbor systeminfo and Kubernetes readyz connectivity checks through platform resource records plus development-only credentials loaded from `.secrets/integration-connections.*`.
- Frontend environment create/edit selects platform resource rows without exposing secrets.
- Skill and docs now record the `.secrets/` integration rule and real-data gate for environment management.
- Kubeconfig paths from `.secrets/` resolve from repo root, backend runtime, or package test working directories.
- Real environment checks passed through `POST /api/environments/:id/check` for both `env-local-prod` and `env-project-xjzt-test`.

Phase 1 completion evidence:

- `env-local-prod`: platform-maintained K8s and Harbor resources passed connectivity checks.
- `env-project-xjzt-test`: platform-maintained K8s and Harbor resources passed connectivity checks.
- Backend startup used real PostgreSQL, real Redis, and `INTEGRATION_MODE=real`.
- The remote agent heartbeat and lease endpoints returned HTTP 200 while the backend was running.

Phase 2 work already implemented but should be rechecked after phase 1 UI/data cleanup:

- Agent page no longer imports or falls back to mock data; it calls real `/api/agents` and `/api/environments`.
- Agent registration token generation accepts an explicit `agentId`, validates the target environment, and returns an R&D direct-start config command using `./ops-release-agent -f ./agent.env`.
- Remote Agent heartbeat writes to PostgreSQL-backed agent records and binds the Agent to a real environment record.
- Agent online status is calculated from persisted heartbeat time instead of static mock status.
- Environment list/detail derive bound Agent status from persisted Agent heartbeat state.
- Agent task lease, step status, logs, result, and status query are persisted in PostgreSQL-backed tables.
- Backend runtime wires the persisted Agent task protocol store; the in-memory protocol store remains only for tests and isolated scaffolding.
- Real remote Agent has been observed online through `/api/agents`, and heartbeat/lease requests return HTTP 200 while the backend is running.

Phase 2 completion evidence:

- Remote Agent `agent-project-xjzt-test` is bound to `env-project-xjzt-test` and reports `ONLINE`.
- Agent heartbeat timestamp updates through the platform API.
- Agent lease path no longer requires static in-memory task state in normal backend runtime.
- Agent registration drawer uses real environment options and generates binary direct-start config text.

Next phase gate:

- Continue with phase 3 Release creation.
- Do not start baseline management until release creation reads real environments, real agents, and real service/version source data.
- Phase 3 requires a real release source before completion, such as Jenkins job metadata or Harbor image tags from platform-maintained resource records plus credentials resolved through `credentialRef`.

Validation for this local work:

- Backend tests:
  - `go test ./...` passed on 2026-06-18.
- Environment check API:
  - `POST /api/environments/env-local-prod/check` returned healthy Kubernetes and Harbor checks on 2026-06-18.
  - `POST /api/environments/env-project-xjzt-test/check` returned healthy Kubernetes and Harbor checks on 2026-06-18.
- Frontend tests:
  - `npm run test:unit -- --run` passed on 2026-06-18.
- Frontend build:
  - `npm run build` passed on 2026-06-18.

Current runtime:

- Frontend dev server is running locally with Vite.
- Backend is running locally with `go run ./cmd/server`.
- Backend runtime loaded remote PostgreSQL, remote Redis, and real integration config from `.secrets/`.
- Harbor integration supports the URL scheme configured in `.secrets/`, including HTTP registries used during local V1 development.

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
