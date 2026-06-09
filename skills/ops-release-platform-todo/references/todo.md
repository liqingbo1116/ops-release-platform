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

- `3d15562 完成V1 mock-first发布部署闭环`

## Current Local Work

- No uncommitted V1 implementation work is tracked here.
- Current gap for V1 remote release/deploy:
  - the repository has only an empty Agent module skeleton under `agent/`
  - there is no standalone Agent process yet
  - there is no Agent `Dockerfile`
  - there is no remote Agent `docker-compose.yml`
  - there is no real Agent outbound task lease/pull implementation
  - without those items, real remote project-environment release/deploy testing cannot start

## Current Step

- V1 mainline must now continue at step 1:
  - build an independently deployable remote Agent package
  - switch task delivery from platform-side mock assumptions to real Agent outbound task lease/pull delivery
- Completed and pushed mock-first status:
  - release/deploy detail closure
  - Agent protocol mock closure
  - service release/deployment boundary closure
  - release list/detail source metadata
  - deploy list missing-service first-deployment view
  - release/deploy detail audit and impact visibility
  - environment/Agent user-view readiness
  - failure action to Agent status consistency
  - backend full tests passed on 2026-06-09: `go test ./...`
  - frontend unit tests passed on 2026-06-09: 10 files, 39 tests
  - frontend build passed on 2026-06-09 with existing dependency annotation warnings only
- Next default step:
  - implement remote Agent docker-compose deployment package and Agent outbound task lease/pull protocol before real Jenkins/Harbor/K8s integration

## V1 Implementation Baseline

This is the authoritative order for subsequent development. V1 only targets project-environment iterative release and target-missing service deployment. Do not move optimization, broad refactors, or UI polish ahead of this path unless they block build, tests, or the V1 flow.

1. Standalone remote Agent package.
2. Agent outbound task lease/pull protocol.
3. Remote Agent mock executor.
4. Release/deploy detail closure against remote Agent callbacks.
5. Real release integration through Jenkins and Harbor/Registry.
6. Real deployment integration through Kubernetes.
7. Remote project-environment deploy/manage V1 acceptance.
8. Audit, permission, and persistence completion.

Current step is step 1: standalone remote Agent package.

## V1 Mainline Goal

V1 must prioritize functional closure over optimization work. The minimum acceptable V1 outcome is:

- the platform can manage project environments
- the platform can create and track project-environment deployment/release tasks
- Agent can be deployed independently to a project environment by `docker compose`
- remote Agent can lease/pull release/deploy task payloads and required execution data from the platform API
- Agent-driven execution and status reporting are visible end to end

Until this mainline is complete, performance tuning, warning cleanup, and refactor-only work stay behind feature work unless they block delivery.

## Agent Deployment Assumption

- Environment access rule:
  - local environments are platform-direct by default and do not require Agent
  - project environments are not assumed reachable from the platform and require Agent
  - Agent only communicates outbound to the platform API; the platform must not call Agent endpoints or push tasks to Agent
- V1 Agent deployment model:
  - Linux host
  - `docker compose`
  - Agent is outside Kubernetes
  - Agent does not need to expose an endpoint reachable by the platform
  - Agent connects outbound to the platform API to lease/pull tasks and report heartbeat, service list, image versions, step status, logs, and final result
  - Agent accesses remote Kubernetes, Jenkins, Harbor/Registry, and platform API
- Do not treat “deploy Agent into Kubernetes” as a V1 prerequisite unless the docs are deliberately changed later.

## Recommended Development Path

1. Build remote Agent deployment package.
   - implement standalone Agent process under `agent/`
   - add Agent config loading and validation
   - add `agent/Dockerfile`
   - add remote Agent `docker-compose.yml`
   - add `.env.example` without secrets
   - add health endpoint and concise logs
   - user-visible outcomes:
     - environment owner can deploy Agent on a Linux host with `docker compose up -d`
     - platform can show the Agent as registered or reachable
   - external readiness:
     - Linux host
     - `docker`
     - `docker compose`
     - network path from Agent to platform API
2. Complete Agent outbound task lease/pull protocol.
   - Agent registration and environment binding
   - Agent leases/pulls release/deploy task payload and execution data from the platform API
   - idempotency key for repeated execution
   - lease acknowledgement, timeout, and failure state
   - step status report
   - log report
   - final result report
   - user-visible outcomes:
     - release/deploy task changes from created to leased/running
     - detail page shows dispatch result, execution steps, logs, and final result from Agent callbacks
   - external readiness:
     - Agent Linux host
     - `docker compose`
     - outbound connectivity from Agent to platform API
     - repeatable test service
3. Complete mock executor in remote Agent.
   - no Jenkins/Harbor/K8s dependency yet
   - Agent leases/pulls release/deploy payloads
   - Agent simulates execution steps and callbacks
   - user-visible outcomes:
     - remote Agent deployment can be verified before external systems are ready
     - platform details page displays logs generated by a remote process, not in-process mock data
   - external readiness:
     - only Agent host and network are required
4. Complete release/deploy detail closure against remote Agent callbacks.
   - retain `agentTaskId`
   - display dispatch state, task state, logs, failure reasons, retry states
   - user-visible outcomes:
     - release detail shows status, steps, logs, action records, report
     - deploy detail shows status, steps, logs, action records
     - retry / skip / manual confirm / rollback entries are visible where applicable
   - external readiness:
     - remote Agent package from step 1
     - Agent outbound task lease/pull protocol from step 2
5. Complete real release integration closure.
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
6. Complete real deployment integration closure.
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
7. Complete remote project-environment deploy/manage V1 bar.
   - environment management visibility
   - Agent status visibility
   - remote release/deploy from platform with end-to-end tracking
   - user-visible outcomes:
     - users can manage remote project environments and drive remote release/deploy from the platform
   - external readiness:
     - full integration environment
     - environment owner
     - access path and test window
8. Complete audit, permission, and persistence requirements.
   - operator
   - target environment
   - affected services
   - source/target tag changes
   - success/failure and failed step
9. Only after the above, continue non-functional work.
   - bundle optimization
   - build warning cleanup
   - UI polish
   - refactor-only cleanup

## External Environment Readiness Rule

- Before work moves from mock flow to real integration, the required external environment must be called out explicitly.
- The default components to request early are:
  - remote Agent Linux host with `docker compose` for leasing/pulling task payloads and reporting logs/results
  - Jenkins for build and release job execution
  - Harbor or compatible registry for image query, sync, and push/pull verification
  - Kubernetes for runtime snapshot, workload deploy/update, and health verification
- The default integration samples to request early are:
  - at least one deployable test image
  - Jenkinsfile or equivalent build script
  - Dockerfile
  - deployable K8s manifests
- Do not wait until final integration to raise these dependencies.
- Do not record credentials, cluster addresses, or secret material here. Record only the fact that the environment must be prepared and which capability is needed.

## Next Suggested Tasks

1. Implement the standalone Agent deployable package:
   - `agent/cmd/agent`
   - config module
   - health endpoint
   - task lease/pull client
   - callback reporter
   - `agent/Dockerfile`
   - remote Agent compose template
2. Implement Agent outbound task lease/pull dispatch:
   - bind environment to Agent identity
   - create pending task payload after release/deploy creation
   - track lease/running status
   - show lease or execution failure in release/deploy detail
3. Implement remote mock executor:
   - simulate Jenkins build/release steps
   - simulate Harbor sync steps
   - simulate K8s deploy/update/health-check steps
   - report logs and final result back to platform
4. Keep the current scope narrow:
   - existing service release/update
   - target-missing service first deployment
   - remote project environment task tracking
5. Before remote Agent verification, prepare:
   - one Linux host for Agent
   - `docker` and `docker compose`
   - platform API connectivity from Agent host
   - one repeatable test project or service
6. Before real Jenkins/Harbor/K8s integration, also prepare:
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
