# Ops Release Platform TODO

Last updated: 2026-06-24

Always verify this file against `git status --short --branch`, `git log -1 --oneline`, and implementation docs before acting.

## Done And Pushed

- Task 1: frontend/backend/docker-compose initialization.
- Task 2: frontend static page prototype.
- Task 3: backend REST API prototype.
- Task 4: login, route guard, user/role/permission page and API prototype.
- Task 5: changelog page and API prototype.
- Task 6: frontend/backend API integration.
- Task 7: PostgreSQL/GORM model and migration foundation.
- Task 8: Redis Stream Agent task model prototype.
- Task 9: backend API tests and frontend Vitest tests.
- Phase 4 integration adapter preparation:
  - Jenkins, Registry/Harbor, Kubernetes adapter interfaces.
  - Real adapter suite.
  - Environment connection check through real Kubernetes and Registry adapters.
  - Runtime integration has no mock/real mode switch; adapters are real-only.
  - Adapter tests.
- Project skills:
  - `ops-release-platform-dev`
  - `ops-release-platform-todo`
  - `ops-release-platform-architecture`
  - `ops-release-platform-deployment`

Latest pushed milestone:

- `cda9605 完善服务版本来源展示`

## Current Local Work

- Local uncommitted documentation update:
  - V1 service release flow, product service list priority, Jenkins Pipeline binding, local Harbor confirmation, and local/remote release execution rules are being fixed in docs and this TODO.
  - V1 real-environment source-of-truth rule is being fixed in docs and skill: if a managed service disappears from the real product environment, platform must auto-unmanage it, hide it from the main service list, and record the change instead of keeping stale active service data.
  - Remote product service-change reconciliation must happen when Agent reports arrive; it must not wait for a user refresh click after the platform has already received newer Agent data.

## Current Step

- V1 mainline is currently in 服务与版本来源 / 发布单创建前置收口.
- Foundation status:
  - 基础资源管理: functionally complete for V1; keep only bug fixes and integration follow-up.
  - Agent 管理与远程资源上报: functionally complete enough for the current V1 release-flow work; keep only bug fixes and integration follow-up unless remote release execution exposes gaps.
  - 项目管理 / 产品管理: functionally complete for the current V1 transition; the backend still reuses environment records to carry product deployment scope and resource bindings.
  - 服务与版本来源: initial service discovery, managed service list, image source classification, private registry confirmation, and release-source readiness display are implemented.
- Next default step:
  - Strengthen the product service list as the main release entry, then implement service-to-Jenkins Pipeline binding, then create release orders from service rows with V1 flow nodes.
- Completed and pushed early prototype status:
  - release/deploy detail closure
  - Agent protocol closure
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
  - Product service list closure, service Pipeline binding, and service-row release order creation. Do not use mock service, Pipeline, image, or version data as completion evidence.

## V1 Implementation Baseline

This is the authoritative order for subsequent development. Each phase must use real data before it is considered complete. If a required external tool or runtime environment is not ready, stop at that phase and do not move on.

1. 基础资源管理.
2. 环境管理 / 产品管理过渡.
3. Agent 管理与远程资源上报.
4. 项目管理.
5. 产品管理.
6. 服务与版本来源.
7. 发布单创建.
8. 基线管理.
9. 部署执行.
10. 发布详情 / 部署详情.
11. 登录与权限.
12. 清理剩余 mock.

Current step is step 6/7 boundary: 服务与版本来源收口 and 发布单创建前置. The next implementation must keep the user entry as 项目 -> 产品 -> 服务. Users should find a service in the product service list, bind or confirm the Jenkins Pipeline if needed, then click release from that service row. Release orders are execution records and flow detail carriers, not the primary place to find services.

Agent registration design for V1:

- The Agent registration secret/token is generated from the Agent management page.
- The page must show the platform URL, one-time registration token, expiration, and copyable Agent config text. Do not show shell commands on the registration page.
- The project-side operator copies the generated config text into the Agent config file, fills the remote K8s/Harbor connection values, then starts the Agent binary with `-f <config-file>`.
- Agent should connect to the platform with platform URL and one-time registration token first.
- After first registration, the platform issues a long-lived Agent token/secret. Heartbeat, task lease, resource reporting, and callback APIs must use and validate this Agent token; the one-time registration token must not be reused for runtime calls.
- The platform must support unbound or pending-claim Agents so an Agent can appear before it is associated with a project/product.
- Environment/product ID may be accepted as an optional startup convenience, but product binding and namespace/project-to-product mapping must be manageable from the platform.
- Agent token validation is part of the Agent phase gate; generating tokens without validating them on heartbeat/task APIs is not acceptable for V1 closure.
- If an Agent is online and can already report service versions or remote resource summaries but is still unbound, show it as `在线 / 待认领`. Its reported data may be displayed as unowned probe data for claim decisions, but it must not enter the official product/service view or execute release/deploy tasks until it is bound to a project/product.
- Before requiring the user to deploy or change project-side Agent config, tell the user exactly which file/value/command must be changed and what success signal should appear on the platform page.
- Do not use mock Agent data, mock probe data, or mock execution as V1 completion evidence.

Project/product model decision for V1:

- The user-facing hierarchy should be 项目 -> 产品 -> 服务 -> 发布 / 部署.
- Examples: 项目A、项目B are projects; 数据中台、物联中台 are products; 服务A、服务B are services under a product.
- The current environment concept corresponds to the product concept for V1. Treat environment records and resource bindings as the transition implementation of product deployment scope, not as an extra user-facing level under product.
- Do not rebuild completed 基础资源管理 or current 产品管理 solely for this hierarchy change.
- Add projects in step 4 and product ownership in step 5, then make service, release, deployment, baseline, and permission flows consume that ownership in later steps.

Product management is considered ready for the next phase only because it supports multi-scope bindings:

- A product must not be modeled as only one Harbor project, one K8s namespace, and one Jenkins view/job.
- Local products bind local K8s namespaces, local Harbor projects, and local Jenkins views/jobs.
- Remote products bind local Harbor projects and local Jenkins views/jobs as build/version sources. Remote runtime K8s namespaces and remote Harbor projects are reported by Agent and mapped to products on the platform side.
- Agent config must not carry namespace/project-to-product mapping. One product may map multiple remote namespaces and multiple remote Harbor projects after Agent reports them.
- Multi-scope bindings are for later service-to-product association: each service must be able to use the correct namespace and Harbor project inside a product; build/version-source flows may also use Jenkins view/job. Do not implement multi-binding as an isolated configuration feature with no service-level consumer.
- The page may keep a minimal default-binding UI in V1, but backend models, APIs, and persistence must support the full binding list.
- Release/deploy task creation must snapshot the actual namespace/project/view/job used by that task, so historical tasks are not affected when product bindings change later.

## V1 Ordered TODO

1. 基础资源管理. Done for V1.
   - User-visible outcome: users can maintain K8s, Harbor, and Jenkins resources, test connectivity, refresh probes, and distinguish K8s resources by API Server.
   - Remaining scope: bug fixes only unless later phases expose integration gaps.
2. 产品管理. Done for the current V1 transition.
   - User-visible outcome: users can create local and remote products. Local products bind local K8s namespaces, local Harbor projects, and local Jenkins views. Remote products bind local Harbor projects and local Jenkins views as build/version sources, while their remote runtime K8s/Harbor scopes wait for Agent-reported data and platform-side mapping.
   - Remaining scope: follow-up adjustments only when service association consumes these bindings.
3. Agent 管理与远程资源上报. Done for current V1 release-flow prerequisites.
   - User-visible outcome: users can generate a registration token on the Agent page, copy the generated config text into a project-side Agent config file, start the Agent binary, see `在线 / 待认领`, inspect unowned reported resource summaries, bind the Agent to a product, and then map Agent-reported remote namespaces/Harbor projects to that product.
   - Remaining scope: bug fixes and follow-up gaps exposed by remote release execution. `docker compose` is not required for development closure and is deferred to formal production deployment verification.
4. 项目管理. Done for current V1 transition.
   - User-visible outcome: users can create and select top-level projects such as 项目A and 项目B as the business ownership boundary.
   - Dependency: completed environments/products must be attachable to projects without rewriting resource management.
5. 产品管理. Done for current V1 transition.
   - User-visible outcome: users can create products such as 数据中台 and 物联中台 under a project. The current environment model is reused as the V1 transition implementation for product deployment scope and resource bindings.
   - Dependency: completed environment bindings from step 2 must be consumed here as product resource scope, not duplicated.
6. 服务与版本来源. Current.
   - User-visible outcome: users manage services under products, see current workload images, classify private/external image sources, confirm product private registry, and use the product service list as the main release entry.
   - Next required work: strengthen the product service list with search/filter, release entry, Pipeline status, latest release status, and batch manage/unmanage actions; keep service management based on real platform or Agent discovery.
   - Data rule: real local/remote product runtime is the source of truth. Managed-service records are only platform relationships. If an already managed service disappears from the real product environment, the platform must auto-unmanage it when latest probe/Agent report data arrives, remove it from the active service list, prevent release, and optionally write a resource-change event. Remote Agent reports must be reconciled on ingestion, not only after a user clicks refresh.
   - Dependency: project ownership from step 4, product deployment scope from step 5, and environment bindings from step 2 as the V1 transition implementation must be consumed here, not duplicated.
7. 发布单创建. Next.
   - User-visible outcome: users click release from a service row; the platform creates a release order automatically with project, product, service, bound Jenkins Pipeline, target image/tag, and V1 flow nodes.
   - Rule: release orders are execution records and flow detail carriers, not the primary service selection UI.
8. 基线管理.
   - User-visible outcome: users can view environment baselines, record service version snapshots, and compare target release versions with the current baseline.
9. 部署执行.
   - User-visible outcome: users can execute deployments; local products run through platform-direct K8s access, remote products run through Agent-reported runtime resources and Agent task execution.
10. 发布详情 / 部署详情.
   - User-visible outcome: users can inspect execution progress, step status, logs, failure reason, retry/rollback entry, and Agent-reported results.
11. 登录与权限.
   - User-visible outcome: users log in with real identity, and key operations are controlled by role/project/product/service permissions.
12. 清理剩余 mock.
   - User-visible outcome: the V1 mainline no longer depends on page fallback data, mock repositories, or mock-only API behavior.

## V1 Mainline Goal

V1 must prioritize functional closure over optimization work. The minimum acceptable V1 outcome is:

- the platform can manage products and their resource scopes
- the platform can group products under projects before services, releases, and deployments are created
- the platform can create and track product deployment/release tasks
- Agent can be started directly by binary during development and later deployed by docker-compose in formal production use
- remote Agent can lease/pull release/deploy task payloads and required execution data from the platform API
- Agent-driven execution and status reporting are visible end to end

Until this mainline is complete, performance tuning, warning cleanup, and refactor-only work stay behind feature work unless they block delivery.

## V1 Service Release Flow TODO

Use this section as the current implementation guide after service/version-source closure.

1. Product service list closure.
   - Make `项目 -> 产品 -> 服务` the main release entry.
   - Show service name, workload type, namespace, current image, current tag, image source, private registry confirmation, Jenkins Pipeline binding status, and latest release status.
   - Add search and filters for service name, namespace, workload type, image source, Pipeline binding status, and release readiness.
   - Keep `纳管服务`, `移除纳管`, and `刷新服务` as service-list maintenance actions.
   - Managed/unmanaged service actions must use real platform-direct or Agent-reported workload data. Do not add manual service entry.
   - Removing managed services only removes the platform management relationship; it must not delete Kubernetes workloads, Harbor images, Jenkins Pipelines, or historical release records.
   - If a managed workload is deleted or disappears from the real product environment, the platform must auto-remove the management relationship when latest probe/report data arrives, hide it from the active service list, and block release actions. Keep the fact in event/change history instead of keeping a stale active row.
   - Agent-reported changes for remote products must be processed by backend ingestion immediately after report validation; do not defer reconciliation until the user manually refreshes the page.
   - If an unmanaged workload is deleted or disappears from the real product environment, it disappears from discovery when latest probe/report data arrives.
2. Service-to-Jenkins Pipeline binding.
   - Pipeline choices must come from the product's bound Jenkins view/job data or live Jenkins query.
   - The user selects a Pipeline from a list. Do not allow free-text Pipeline names.
   - The first release of an unbound service must ask the user to bind a Pipeline first.
   - The platform may recommend a Pipeline by matching service name, workload name, or image name, but the user must confirm.
   - Store the binding on the service and reuse it for later releases; allow changing the binding from service detail.
3. Service-row release order creation.
   - The user clicks release on a service row.
   - The platform opens a confirmation view with project, product, service, current version, target version, bound Jenkins Pipeline, and planned flow nodes.
   - Confirming creates a release order automatically.
   - The release order records execution state and flow details. It is not the primary place where users find services.
4. Jenkins execution and logs.
   - The platform triggers the bound Jenkins Pipeline, captures build number, polls build state, and reads console logs.
   - Jenkins runs shell/Pipeline logic that already exists outside the platform.
   - During Jenkins execution the platform observes status and logs only; it does not pause or insert intermediate control inside the Jenkins script.
5. Local Harbor image confirmation.
   - After Jenkins ends, the platform must query the local Harbor project and confirm the target image tag exists.
   - This check is required for both local and remote products.
   - If Jenkins succeeds but the image tag is missing, the release fails.
   - If Jenkins fails but the image tag exists, show that explicitly and do not automatically continue.
   - If Harbor cannot be queried, mark the release abnormal and show the connection or permission reason.
6. Local product release result.
   - Jenkins is expected to build, push to local Harbor, and update GitLab YAML.
   - Argo CD performs the CD side outside the platform.
   - The platform must still connect to local Kubernetes and confirm the workload image tag or rollout result after local Harbor confirmation.
   - The final result must show Jenkins, local Harbor, and local Kubernetes status.
7. Remote product release result.
   - Jenkins is expected to build and push to local Harbor only.
   - The platform may dispatch Agent remote release work only after local Harbor confirms the target image tag exists.
   - If local Harbor image confirmation fails, do not dispatch Agent.
   - Agent must pull/copy the image into remote Harbor using remote Harbor's pull/replication ability, confirm the remote image tag exists, update remote Kubernetes workload image, and report status/logs/final result.
   - The final result must show Jenkins, local Harbor, remote Harbor, remote Kubernetes, and Agent status.
8. Flow extension boundary.
   - Future approval, manual confirmation, risk check, canary, and rollback controls should be inserted before Jenkins or after Jenkins as platform-controlled nodes.
   - V1 does not try to pause or control steps inside the existing Jenkins script.

## Agent Deployment Assumption

- Environment access rule:
  - local environments are platform-direct by default and do not require Agent
  - remote product runtime resources are not assumed reachable from the platform and require Agent reporting/execution
  - Agent only communicates outbound to the platform API; the platform must not call Agent endpoints or push tasks to Agent
  - V1 Agent deployment model:
  - Linux host
  - direct binary startup is the required development-time path for debugging and integration
  - `docker compose` is the later formal production deployment path, not the current development gate
  - Agent is outside Kubernetes
  - Agent does not need to expose an endpoint reachable by the platform
  - Agent connects outbound to the platform API to lease/pull tasks and report heartbeat, service list, image versions, step status, logs, and final result
  - Agent accesses remote runtime Kubernetes, remote Harbor/Registry, and platform API. Agent does not access Jenkins; Jenkins is platform-side local infrastructure for build/version-source flows.
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
     - environment owner can later deploy Agent on a Linux host with `docker compose` in formal production use
     - platform can show the Agent as registered or reachable
   - external readiness:
     - Linux host
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
     - outbound connectivity from Agent to platform API
     - repeatable test service
3. Complete real executor in remote Agent. Historical simulation executor is not a current V1 completion gate and must not be restored.
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
   - Agent execution for remote image pull/sync verification and runtime tag update
   - user-visible outcomes:
     - users can choose local Jenkins job or local Harbor image tag at release creation
     - users no longer need manual Harbor lookup or manual tag change
   - external readiness:
     - local Jenkins test job or view
     - local Harbor/Registry test project and test images
     - platform backend connectivity to local Jenkins and local Harbor
     - remote Agent connectivity to remote Harbor, remote Kubernetes, and platform API
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
7. Complete remote product deploy/manage V1 bar.
   - product management visibility
   - Agent status visibility
   - remote release/deploy from platform with end-to-end tracking
   - user-visible outcomes:
     - users can manage remote products and drive remote release/deploy from the platform
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

1. 收敛 Agent 页面交互:
   - Agent 列表只保留“刷新”作为状态更新入口，不做平台主动探测按钮。
   - 页面展示 Agent 上报的心跳、在线状态、绑定状态、最近上报摘要和最近任务状态。
   - 注册弹窗只展示可复制配置文本，不展示 shell 命令。配置里填写平台地址、注册 token、远程 K8s/Harbor 连接信息，不填写产品映射。
   - 保留 Agent 绑定/解绑产品入口；解绑是否开放要以安全规则为准：有运行中任务、已绑定正式产品且仍在线执行的 Agent 不允许直接解绑。
   - 环境/产品页面也可以提供绑定/解绑 Agent 入口，但必须与 Agent 页面使用同一套后端规则，避免两个页面状态不一致。
   - 待认领 Agent 可以显示未归属上报摘要，但不能进入正式产品/服务视图，也不能执行发布/部署任务。
   - 页面不解释 Agent 内部工作细节，只呈现用户需要判断的结果：是否在线、是否待认领、是否已绑定、是否有上报数据、是否可执行任务。
2. Verify the standalone Agent deployable package:
   - build `agent/cmd/agent` into a binary
   - copy `agent/.env.example` to an Agent config file on the host and fill non-secret identifiers
   - run the Agent with `-f <config-file>` for development-time verification
   - verify the formal `docker compose` deployment path only after development-time binary verification is complete and production deployment packaging is in scope
   - verify `/healthz`
   - verify registration token exchange, long-lived Agent token validation, heartbeat, and remote K8s/Harbor resource reports reach the platform
3. Add platform-side mapping for Agent-reported runtime scopes:
   - display Agent-reported remote K8s namespaces and remote Harbor projects as unowned resource data before product binding
   - after Agent is bound to a product, allow users to map one or more remote namespaces and one or more remote Harbor projects to that product
   - keep local Jenkins/local Harbor build-source bindings separate from remote runtime mappings
   - user-visible outcome: users can see which remote runtime scopes are ready for a product before service release/deploy uses them
4. Verify Agent outbound task lease/pull dispatch:
   - create a project-environment release/deploy task
   - verify `/api/agent-tasks/lease` returns the bound task only to the matching Agent/environment
   - verify task status changes to leased/running and logs appear through callback APIs
5. Close release/deploy detail against remote Agent callbacks:
   - show lease state and callback-driven logs
   - show lease/execution failure reasons
   - keep retry/skip/manual-confirm/rollback state consistent with Agent task status
6. Keep the current scope narrow:
   - existing service release/update
   - target-missing service first deployment
   - remote project environment task tracking
7. Before remote Agent verification, prepare:
  - one Linux host for Agent
  - Go toolchain or a prebuilt Agent binary for development-time direct startup
   - platform API connectivity from Agent host
   - one repeatable test project or service
8. Before real Jenkins/Harbor/K8s integration, also prepare:
   - Harbor test image and tag set
   - Jenkins pipeline and build script
   - deployable K8s manifests

## Project Management TODO

This section is the handoff checklist for the upcoming 项目管理 / 产品管理 work. It must preserve the V1 user hierarchy: 项目 -> 产品 -> 服务 -> 发布 / 部署.

1. Rename the user-facing 环境管理 entry to 产品管理 for the main business view.
   - User-visible outcome: users no longer need to understand a separate “environment” level; they see products such as 数据中台 and 物联中台.
   - Implementation rule: reuse the completed environment/resource-binding capability as the product deployment scope instead of rebuilding K8s/Harbor/Jenkins binding logic.
2. Add real project records and project list/detail pages.
   - User-visible outcome: users can create and maintain projects such as 项目A and 项目B.
   - Project fields should start small: name, code, description, status, created/updated metadata.
3. Keep foundational resources as a global resource pool.
   - K8s clusters, Harbor registries/projects, and Jenkins instances/views are maintained in 基础资源管理.
   - Projects do not directly own or bind K8s/Harbor/Jenkins resources.
   - Products reference the resource scopes they use; projects see resource usage indirectly through their bound products.
4. Add product ownership binding to projects.
   - User-visible outcome: one project can bind one or more products.
   - Product can be unbound/pending assignment before it is attached to a project, so existing products can be migrated without blocking page entry.
5. Add product binding status to support attach, detach, and move.
   - Suggested statuses:
     - `UNBOUND`: product exists but is not attached to any project.
     - `BOUND`: product is attached to one project and can be used by service/release flows.
     - `MOVING`: product is being moved to another project; block release/deploy creation during the move.
     - `DISABLED`: product is retained but hidden from new release/deploy selection.
   - User-visible outcome: users can tell whether a product is usable, waiting for project binding, being moved, or disabled.
6. Define project-product binding rules.
   - A product can belong to at most one project at a time in V1.
   - A project can bind multiple products.
   - Detach is allowed only when no running release/deploy task is using the product.
   - Move to another project should be recorded as a binding change and should not modify historical release/deploy records.
7. Make service creation consume product scope.
   - User-visible outcome: when users create a service under a product, they choose from that product's namespace, Harbor project, and optional Jenkins view/job scope.
   - This is the point where existing environment multi-bindings become meaningful to users.
8. Make release/deploy/baseline pages consume project and product.
   - User-visible outcome: users select project first, then product, then services.
   - Historical records must snapshot project/product/service identifiers and names so later product moves do not rewrite history.

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
