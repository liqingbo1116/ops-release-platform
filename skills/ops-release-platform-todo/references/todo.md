# Ops Release Platform TODO

Last updated: 2026-06-21

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

- `d84d7f3 feat: add v1 remote agent lease flow`

## Current Local Work

- Local uncommitted environment page refinement:
  - environment list separates resource readiness from remote Agent execution readiness
  - remote Agent `UNBOUND` is displayed as Chinese `未绑定`
  - resource scope problems stay in the environment/status summary
  - Agent readiness is shown independently and documented in environment detail
- Validation for current local work:
  - `npm run test:unit -- EnvironmentPage` passed in `frontend`

## Current Step

- V1 mainline is currently ready to move from environment management to Agent management and remote probing.
- Foundation status:
  - 基础资源管理: functionally complete for V1; keep only bug fixes and integration follow-up.
  - 环境管理: functionally complete for V1; environments can bind multiple K8s namespaces, Harbor projects, and Jenkins views for later service association.
- Next default step:
  - Agent 管理与远程探测: verify real Agent registration, heartbeat, environment binding, online status, task leasing, and remote resource probing through the platform API.
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
  - run the Agent from `agent/docker-compose.yml` on a remote Linux host or local remote-like host and verify heartbeat, lease, duplicate lease protection, expired lease retry, mock execution logs, and final result through platform APIs

## V1 Implementation Baseline

This is the authoritative order for subsequent development. Each phase must use real data before it is considered complete. If a required external tool or runtime environment is not ready, stop at that phase and do not move on.

1. 基础资源管理.
2. 环境管理.
3. Agent 管理与远程探测.
4. 服务与版本来源.
5. 发布单创建.
6. 基线管理.
7. 部署执行.
8. 发布详情 / 部署详情.
9. 登录与权限.
10. 清理剩余 mock.

Current step is step 3: Agent 管理与远程探测. Do not move to 服务与版本来源 until Agent registration, heartbeat, environment binding, status visibility, task leasing, and remote probing use real backend data and the mock/fallback boundary is removed or explicitly recorded as blocked by missing real runtime or infrastructure.

Environment management is considered ready for the next phase only because it supports multi-scope bindings:

- An environment must not be modeled as only one Harbor project, one K8s namespace, and one Jenkins view/job.
- Add or complete the environment resource binding model so one environment can bind multiple K8s namespaces, Harbor projects, and Jenkins views/jobs.
- Multi-scope bindings are for later service-to-environment association: each service must be able to use the correct namespace, Harbor project, and Jenkins view/job inside an environment. Do not implement multi-binding as an isolated configuration feature with no service-level consumer.
- The page may keep a minimal default-binding UI in V1, but backend models, APIs, and persistence must support the full binding list.
- Release/deploy task creation must snapshot the actual namespace/project/view/job used by that task, so historical tasks are not affected when environment bindings change later.

## V1 Ordered TODO

1. 基础资源管理. Done for V1.
   - User-visible outcome: users can maintain K8s, Harbor, and Jenkins resources, test connectivity, refresh probes, and distinguish K8s resources by API Server.
   - Remaining scope: bug fixes only unless later phases expose integration gaps.
2. 环境管理. Done for V1.
   - User-visible outcome: users can create local and remote environments, bind multiple K8s namespaces, Harbor projects, and Jenkins views, and see resource readiness separately from Agent execution readiness.
   - Remaining scope: follow-up adjustments only when service association consumes these bindings.
3. Agent 管理与远程探测. Next.
   - User-visible outcome: users can see real Agent registration, heartbeat, environment binding, online/offline/unbound status, and remote resource probe results.
   - Required before next phase: real Agent binary or docker-compose runtime, outbound connectivity to platform API, and remote resource access from Agent.
4. 服务与版本来源.
   - User-visible outcome: users can create services and select actual namespace, Harbor project, Jenkins view/job, repository, branch, image name, and version source inside the target environment.
   - Dependency: environment bindings from step 2 must be consumed here, not duplicated.
5. 发布单创建.
   - User-visible outcome: users can select services, target environment, and version source to create a release order with environment, resource, and Agent readiness checks.
6. 基线管理.
   - User-visible outcome: users can view environment baselines, record service version snapshots, and compare target release versions with the current baseline.
7. 部署执行.
   - User-visible outcome: users can execute deployments; local environments run through platform-direct K8s access, remote environments run through Agent.
8. 发布详情 / 部署详情.
   - User-visible outcome: users can inspect execution progress, step status, logs, failure reason, retry/rollback entry, and Agent-reported results.
9. 登录与权限.
   - User-visible outcome: users log in with real identity, and key operations are controlled by role/environment/service permissions.
10. 清理剩余 mock.
   - User-visible outcome: the V1 mainline no longer depends on page fallback data, mock repositories, or mock-only API behavior.

## V1 Mainline Goal

V1 must prioritize functional closure over optimization work. The minimum acceptable V1 outcome is:

- the platform can manage project environments
- the platform can create and track project-environment deployment/release tasks
- Agent can be started directly by binary during development and deployed by docker-compose in formal project environments
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
  - direct binary startup is the preferred development-time path
  - `docker compose` is the formal packaged deployment path
  - Agent is outside Kubernetes
  - Agent does not need to expose an endpoint reachable by the platform
  - Agent connects outbound to the platform API to lease/pull tasks and report heartbeat, service list, image versions, step status, logs, and final result
  - Agent accesses remote Kubernetes, Jenkins, Harbor/Registry, and platform API
- Do not treat “deploy Agent into Kubernetes” as a V1 prerequisite unless the docs are deliberately changed later.

## Recommended Development Path

1. Build remote Agent deployment package. Locally implemented.
   - implement standalone Agent process under `agent/`
   - add Agent config loading and validation
   - support direct binary startup with `-f <config-file>`
   - add `agent/Dockerfile`
   - add remote Agent `docker-compose.yml`
   - add `.env.example` without secrets
   - add health endpoint and concise logs
   - user-visible outcomes:
     - developers can start Agent on a Linux host directly from the built binary during development
     - environment owner can deploy Agent on a Linux host with `docker compose` in formal use
     - platform can show the Agent as registered or reachable
   - external readiness:
     - Linux host
     - `docker`
     - `docker compose`
     - network path from Agent to platform API
2. Complete Agent outbound task lease/pull protocol. Locally implemented.
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
3. Complete mock executor in remote Agent. Locally implemented.
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
  - remote Agent Linux host for development-time direct binary startup and formal docker-compose deployment, leasing/pulling task payloads, and reporting logs/results
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

1. Verify the standalone Agent deployable package:
   - build `agent/cmd/agent` into a binary
   - copy `agent/.env.example` to an Agent config file on the host and fill non-secret identifiers
   - run the Agent with `-f <config-file>` for development-time verification
   - then verify the formal `docker compose` deployment path
   - verify `/healthz`
   - verify heartbeat reaches `/api/agents/{id}/heartbeat`
2. Verify Agent outbound task lease/pull dispatch:
   - create a project-environment release/deploy task
   - verify `/api/agent-tasks/lease` returns the bound task only to the matching Agent/environment
   - verify task status changes to leased/running and logs appear through callback APIs
3. Close release/deploy detail against remote Agent callbacks:
   - show lease state and callback-driven logs
   - show lease/execution failure reasons
   - keep retry/skip/manual-confirm/rollback state consistent with Agent task status
4. Keep the current scope narrow:
   - existing service release/update
   - target-missing service first deployment
   - remote project environment task tracking
5. Before remote Agent verification, prepare:
  - one Linux host for Agent
  - Go toolchain or a prebuilt Agent binary for development-time direct startup
  - `docker` and `docker compose` for formal deployment verification
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
