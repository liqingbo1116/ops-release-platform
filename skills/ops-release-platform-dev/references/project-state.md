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

V1 mainline has been reset around resource-first delivery. Current phase is phase 1: Resource management. Do not advance to environment completion, release creation, or baseline work until K8s/Harbor/Jenkins resources use user-oriented forms, system-owned status, real probe cache, refresh actions, and Agent-based remote probing.

Existing foundation:

- Frontend environment API list no longer imports or falls back to mock data; it calls `/api/environments`.
- Backend runtime requires real `DATABASE_DSN` and `REDIS_ADDR` before startup.
- Backend runtime wires `DatabaseStore` directly for the main repository, so environment list/detail/create/update use PostgreSQL-backed data in normal runtime.
- Environment dependency check now rejects mock integrations instead of returning fake healthy Kubernetes/Registry checks.
- K8s clusters, Harbor registries, and Jenkins instances have an initial platform-maintained resource model.
- Environment records carry `clusterId`, `registryId`, and `jenkinsId` as references to those reusable resources.
- Environment records carry `deployTargetType` and resource bindings. V1 implements `KUBERNETES` and only reserves `DOCKER_COMPOSE`; default `namespace`, `registryProject`, and `jenkinsView` fields are compatibility fields derived from default bindings.
- One environment may bind multiple K8s namespaces, Harbor projects, and Jenkins views. Remote/project environments still bind platform-side local Harbor project and Jenkins view, while remote K8s/runtime operations are handled by Agent tasks and reports.
- Users only enter environment name and environment code. The backend generates the environment ID as `env-<code>` when the create request omits `id`.
- Environment page has separate tabs for maintaining K8s, Harbor, and Jenkins resources. Environments associate those resource rows and their per-environment scopes.
- `.secrets/` is development-only for private runtime values. Formal resource master data belongs in the platform database, and credentials must be hidden behind internal credential references.
- Resource create/edit forms use user-oriented fields and no longer expose `credentialRef`: K8s kubeconfig/context, Harbor URL + HTTP/HTTPS + username/password, Jenkins URL + username/password or API token.
- Resource status is system-owned. Users can test connection or refresh probe cache, but cannot manually edit status.
- Backend resource probe endpoints support local/direct checks and cache refresh:
  - K8s `/readyz` and namespace list from kubeconfig/API server.
  - Harbor `/api/v2.0/systeminfo` and project list, including HTTP registries.
  - Jenkins `/api/json` and view/job list.
- Probe responses update `status`, `lastCheckAt`, `probeMessage`, and successful cache lists: `namespaces`, `projects`, `views`, `jobs`.
- Refresh failure keeps the previous cache and records the failure reason in `probeMessage`.
- Frontend environment create/edit selects platform resource rows without exposing secrets, and environment scope options come from cached namespaces/projects/views while still allowing manual input when cache is empty.
- Skill and docs now record the `.secrets/` integration rule and real-data gate for environment management.
- Kubeconfig paths from `.secrets/` resolve from repo root, backend runtime, or package test working directories.
- Real environment checks have previously passed through `POST /api/environments/:id/check` for local and project sample environments, but this is not the final acceptance standard for remote/project environments. Remote resource checks must be converted to Agent tasks that report status and probe cache.

Phase 1 remaining work:

- Local/direct probes run in the platform backend; remote/project probes run through Agent tasks and report back to the platform.
- Remote/project Agent-based resource probing remains the phase 1 gate that is not complete.

Agent foundation already implemented but must be rechecked after phase 1 resource semantics:

- Agent page no longer imports or falls back to mock data; it calls real `/api/agents` and `/api/environments`.
- Agent registration token generation accepts an explicit `agentId`, validates the target environment, and returns an R&D direct-start config command using `./ops-release-agent -f ./agent.env`.
- Remote Agent heartbeat writes to PostgreSQL-backed agent records and binds the Agent to a real environment record.
- Agent online status is calculated from persisted heartbeat time instead of static mock status.
- Environment list/detail derive bound Agent status from persisted Agent heartbeat state.
- Agent task lease, step status, logs, result, and status query are persisted in PostgreSQL-backed tables.
- Backend runtime wires the persisted Agent task protocol store; the in-memory protocol store remains only for tests and isolated scaffolding.
- Real remote Agent has been observed online through `/api/agents`, and heartbeat/lease requests return HTTP 200 while the backend is running.

Agent foundation evidence:

- Remote Agent heartbeat and lease flow has been verified against a manually created remote environment.
- Agent heartbeat timestamp updates through the platform API.
- Agent lease path no longer requires static in-memory task state in normal backend runtime.
- Agent registration drawer uses real environment options and generates binary direct-start config text.

Next phase gate:

- Continue with phase 1 Resource management.
- Do not mark phase 1 complete until resource forms, system-owned status, probe cache, refresh actions, and Agent-based remote probing are implemented with real data and mock fallback removed.
- Do not start release creation until phases 1 through 4 in `docs/development-plan.md` are complete.

Validation for this local work:

- Backend tests:
  - `go test ./...` passed on 2026-06-18.
- Environment check API:
  - Previous direct checks returned healthy Kubernetes and Harbor checks on 2026-06-18.
  - These checks are useful evidence for local/direct connectivity only; remote/project resource probing still needs Agent-task semantics.
- Frontend tests:
  - `npm run test:unit -- --run` passed on 2026-06-18.
- Frontend build:
  - `npm run build` passed on 2026-06-18.
  - The build produced only third-party Rolldown pure-annotation warnings from `@vueuse/core`; no type or build failure.

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
