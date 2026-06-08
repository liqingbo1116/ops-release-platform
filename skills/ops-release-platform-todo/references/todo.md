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

## Current Step

- V1 mainline is currently at step 1:
  - release/deploy detail closure
- Local completion status:
  - mostly done in code
  - frontend tests passed for release/deploy detail pages
  - frontend build passed
- Next default step:
  - step 2, Agent protocol completion

## V1 Mainline Goal

V1 must prioritize functional closure over optimization work. The minimum acceptable V1 outcome is:

- the platform can manage remote project environments
- the platform can create and track remote deployment/release tasks
- Agent-driven execution and status reporting are visible end to end

Until this mainline is complete, performance tuning, warning cleanup, and refactor-only work stay behind feature work unless they block delivery.

## Agent Deployment Assumption

- V1 Agent deployment model:
  - Linux host
  - `docker compose`
  - Agent is outside Kubernetes
  - Agent accesses remote Kubernetes, Jenkins, Harbor/Registry, and platform API
- Do not treat “deploy Agent into Kubernetes” as a V1 prerequisite unless the docs are deliberately changed later.

## Recommended Development Path

1. Complete release/deploy detail closure.
   - retain `agentTaskId`
   - poll and display task state/logs
   - show failure reasons and retry-related states
   - user-visible outcomes:
     - release detail shows status, steps, logs, action records, report
     - deploy detail shows status, steps, logs, action records
     - retry / skip / manual confirm / rollback entries are visible where applicable
   - external readiness:
     - mock is enough by default
     - for real verification, prepare Agent Linux host with `docker compose`
2. Complete Agent task protocol.
   - heartbeat
   - task pull
   - step status report
   - log report
   - final result report
   - user-visible outcomes:
     - Agent page shows online status and recent heartbeat
     - release/deploy tasks are actually pulled and executed by Agent
     - detail-page logs and status are sourced from real Agent callbacks
   - external readiness:
     - Agent Linux host
     - `docker compose`
     - platform connectivity
     - repeatable test service
3. Complete real release integration closure.
   - Jenkins-triggered release path
   - Harbor image selection and sync path
   - Agent execution for image sync and tag update
   - user-visible outcomes:
     - users can choose Jenkins Job or Harbor image tag at release creation
     - users no longer need manual Harbor lookup or manual tag change
   - external readiness:
     - Jenkins test job or job namespace
     - Harbor/Registry test project and test images
     - Agent connectivity to Jenkins and Harbor
     - one test service repository
     - `Dockerfile`
     - Jenkinsfile or build script
     - pushed test image tags for verification
4. Complete real deployment integration closure.
   - runtime snapshot collection
   - deploy missing services
   - workload update and health check
   - user-visible outcomes:
     - compare page identifies target-missing services
     - users can submit deploy tasks and inspect health-check results
   - external readiness:
     - Kubernetes test cluster
     - namespace
     - workload
     - kube access path for Agent
     - deployable K8s manifests
     - one sample for existing-service tag update
     - one sample for missing-service first deployment
5. Complete remote project-environment deploy/manage V1 bar.
   - environment management visibility
   - Agent status visibility
   - remote release/deploy from platform with end-to-end tracking
   - user-visible outcomes:
     - users can manage remote project environments and drive remote release/deploy from the platform
   - external readiness:
     - full integration environment
     - environment owner
     - access path and test window
6. Complete audit, permission, and persistence requirements.
   - operator
   - target environment
   - affected services
   - source/target tag changes
   - success/failure and failed step
7. Only after the above, continue non-functional work.
   - bundle optimization
   - build warning cleanup
   - UI polish
   - refactor-only cleanup

## External Environment Readiness Rule

- Before work moves from mock flow to real integration, the required external environment must be called out explicitly.
- The default components to request early are:
  - Jenkins for build and release job execution
  - Harbor or compatible registry for image query, sync, and push/pull verification
  - Kubernetes for runtime snapshot, workload deploy/update, and health verification
  - remote Agent Linux host with `docker compose` for task pull, log report, and result report
- The default integration samples to request early are:
  - at least one deployable test image
  - Jenkinsfile or equivalent build script
  - Dockerfile
  - deployable K8s manifests
- Do not wait until final integration to raise these dependencies.
- Do not record credentials, cluster addresses, or secret material here. Record only the fact that the environment must be prepared and which capability is needed.

## Next Suggested Tasks

1. Commit and stabilize the local release/deploy detail closure work.
2. Start Agent task protocol completion for heartbeat, task pull, step status, log report, and final result report.
3. Before real verification of step 2, prepare:
   - one Linux host for Agent
   - `docker` and `docker compose`
   - platform API connectivity from Agent host
   - one repeatable test project or service
4. Before steps 3 and 4, also prepare:
   - Harbor test image and tag set
   - Jenkins pipeline and build script
   - deployable K8s manifests

## User-View Test Order

1. Login page.
2. Environment management page and Agent management page.
3. Baseline list page and baseline detail page.
4. Compare page.
5. Create release page.
6. Release list page and release detail page.
7. Deploy task list page and deploy detail page.
8. User, role, and permission pages.

## User-View Acceptance Questions

1. Can the user enter the page?
2. Can the user understand the current state?
3. Can the user complete the core action?
4. Does the state update after the action?
5. Is there a clear error message and fallback display when something fails?

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
