# Ops Release Platform TODO

Last updated: 2026-06-09

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
- Agent protocol mock closure is implemented locally and not yet committed:
  - backend in-memory Agent protocol store
  - Agent heartbeat endpoint
  - Agent task pull endpoint
  - step status report endpoint
  - log report endpoint
  - final result report endpoint
  - created release/deploy tasks are enqueued into mock protocol store
  - Agent page loads Agent status from API instead of static page data
- Diff-to-task end-to-end mock verification is implemented and locally verified, not yet committed:
  - frontend pure mock release creation returns `agentTaskId`
  - frontend pure mock deploy creation returns `agentTaskId`
  - frontend pure mock Agent task status returns task type, current step, status, and logs
  - create release page tests verify detail routing keeps `agentTaskId`
  - backend tests passed
  - frontend unit tests passed
  - frontend build passed with existing dependency annotation warnings only
- Release creation flow adjustment is in progress:
  - service release must not be based on a source baseline
  - service release source supports Jenkins Job and local Harbor image tag
  - both service release sources must eventually use the project Agent to sync image and update tag
  - service deployment should create deploy tasks for target-missing services
  - diff `MISSING_IN_TARGET` should be shown as service deployment
  - release list/detail should show release source, build/sync task, and image metadata instead of implying baseline-driven release
  - real Agent module directories should be created without implementation code
- V1 mock-first service release/deployment boundary closure is implemented locally and not yet committed:
  - `SERVICE_RELEASE` rejects `sourceBaselineId`
  - `SERVICE_RELEASE` supports `JENKINS_JOB` and `LOCAL_HARBOR_IMAGE`
  - `SERVICE_DEPLOYMENT` requires `sourceBaselineId`
  - `SERVICE_DEPLOYMENT` only accepts `MISSING_IN_TARGET` diff services
  - release list/detail show source, build/sync task, and image metadata
  - release list user-view tests cover Jenkins release, Harbor image release, and missing-service deployment display/search
- V1 mock-first deployment list user-view closure is implemented locally and not yet committed:
  - deploy task list uses `SERVICE_DEPLOYMENT` / missing-service first-deployment wording
  - list rows show source baseline, missing services, Agent, Agent task, current step, next action
  - list search covers missing service names and Agent metadata
  - API contract and domain model document the deploy list fields needed by the page
- V1 mock-first audit and impact visibility is implemented locally and not yet committed:
  - release detail shows operator, target environment, affected services, result, failed step, last action
  - deploy detail shows operator, target environment, affected services, result, failed step, last action
  - release/deploy detail API examples document `auditSummary`
  - frontend user-view tests assert audit and affected-service visibility
- V1 mock-first environment and Agent user-view readiness is implemented locally and not yet committed:
  - environment page loads environment status from API/mock fallback instead of direct static data
  - environment page shows real integration prerequisites before switching away from mock flow
  - environment page warns when a project environment uses Agent mode but Agent is not online
  - Agent page shows V1 deployment assumption: Linux host plus `docker compose`
  - Agent page warns that offline Agents block remote release/deploy for bound project environments
  - frontend user-view tests cover environment readiness, environment filtering, Agent readiness, heartbeat, capabilities, recent task, and offline blocker
- V1 mock-first failure action to Agent status consistency is implemented locally and not yet committed:
  - release retry updates mock Agent task status to `retry` / `RUNNING`
  - release rollback updates mock Agent task status to `rollback` / `ROLLED_BACK`
  - deploy step retry / skip / confirm update mock Agent task status to the selected step and result state
  - backend tests cover release failure actions and deploy step actions against `GET /api/agent-tasks/{id}/status`

## Current Step

- V1 mainline is currently at step 3:
  - mock-first release/deployment user-view validation before real integration
- Local completion status:
  - release/deploy detail closure is implemented locally
  - Agent protocol mock closure is implemented locally
  - service release/deployment boundary closure is implemented locally
  - release list/detail source metadata is implemented locally
  - deploy list missing-service first-deployment view is implemented locally
  - release/deploy detail audit and impact visibility is implemented locally
  - environment/Agent user-view readiness is implemented locally
  - failure action to Agent status consistency is implemented locally
  - backend full tests passed on 2026-06-09: `go test ./...`
  - backend focused tests passed on 2026-06-09: `go test ./internal/api ./internal/service`
  - frontend focused tests passed on 2026-06-09: release detail, deploy detail, create release submit
  - frontend unit tests passed on 2026-06-09: 10 files, 39 tests
  - frontend build passed on 2026-06-09 with existing dependency annotation warnings only
- Next default step:
  - review the UI manually in the documented user-view order, then switch to real Agent/Jenkins/Harbor/K8s integration only after the required environment is ready

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

1. Stabilize and commit the local release/deploy detail, Agent protocol mock closure, runtime snapshot baseline metadata work, diff-to-task mock verification, and release source list/detail validation.
2. Continue V1 mock-first functional refinement that does not require Jenkins/Harbor/K8s:
   - confirm whether any user-view page still misses the existing-service release/update or target-missing first-deployment path
   - environment-level permission failures have clear user messages before task creation
   - environment and Agent pages show whether remote project environments are ready for release/deploy
   - offline Agent status must be visible before users submit remote release/deploy tasks
   - API contract examples stay aligned with mock response fields for release list/detail and deploy detail
   - page tests follow the user-view order and cover the existing service release/update plus target-missing service first deployment paths
3. Keep the current scope narrow:
   - existing service release/update
   - target-missing service first deployment
   - remote project environment task tracking
4. Before real Agent verification, prepare:
   - one Linux host for Agent
   - `docker` and `docker compose`
   - platform API connectivity from Agent host
   - one repeatable test project or service
5. Before real Jenkins/Harbor/K8s integration, also prepare:
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
