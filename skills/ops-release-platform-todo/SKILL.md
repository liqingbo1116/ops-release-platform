---
name: ops-release-platform-todo
description: Use for maintaining, reviewing, prioritizing, or selecting TODO/backlog work in the ops-release-platform repository. Applies when the user asks what to do next, says continue development, asks to update task progress, split tasks, track completed work, or reconcile docs with implementation state.
---

# Ops Release Platform Todo

Use this skill to keep project progress explicit and avoid losing the current development thread.

## Workflow

1. Check live repository state first:
   - `git status --short --branch`
   - `git log -1 --oneline`
2. Read `references/todo.md`.
3. Compare TODO state with:
   - `docs/development-plan.md`
   - `docs/codex-implementation-tasks.md`
   - recent commits and current uncommitted files
4. When starting work, choose one clear task boundary.
5. When finishing work, update `references/todo.md` with:
   - completed item
   - validation result
   - commit hash if pushed
   - remaining next task

## Git Submit Rule

- When the user asks to commit, submit, or save changes to Git, commit the scoped changes and push the current branch to the configured remote in the same workflow.
- Only skip pushing when the user explicitly asks for a local-only commit or when no remote is configured; report the reason clearly.
- After pushing, report the commit hash, target branch, validation performed, and whether the worktree is clean.
- Follow `docs/git-submit-workflow.md` for the detailed repository workflow.

## Priority Strategy

- The default priority is V1 feature closure, not optimization work.
- The V1 delivery bar is: the platform must at least support project-environment deployment and management.
- If the next task is about real Jenkins, Harbor/Registry, Kubernetes, or Agent integration, call out the required environment prerequisites before implementation continues.
- Environment access is a hard rule:
  - local environments are platform-direct by default and must not require Agent
  - project environments are not assumed reachable from the platform and must use Agent
  - Agent only communicates outbound to the platform API; the platform must not call Agent endpoints or push tasks to Agent
- Assume the V1 Agent deployment model unless docs are explicitly changed:
  - Linux host
  - direct binary startup during development and integration debugging
  - `docker compose` only for later formal production deployment verification
  - Agent operates remote Kubernetes, but is not itself required to run in Kubernetes
  - Agent must be independently deployable before V1 remote release/deploy can be accepted
  - Agent leases/pulls release/deploy task payloads from the platform and reports heartbeat, service list, image versions, status, logs, and final result back to the platform
- Agent work must not be closed with mock data. If real Agent runtime, registration, heartbeat, token validation, remote probing, or required project-side configuration is missing, stop and tell the user exactly what must be deployed or changed in the project environment.
- Treat the following path as the mainline unless the user explicitly reprioritizes:
  - 基础资源管理
  - 环境管理
  - Agent 管理与远程探测
  - 项目管理
  - 产品管理
  - 服务与版本来源
  - 发布单创建
  - 基线管理
  - 部署执行
  - 发布详情 / 部署详情
  - 登录与权限
  - 清理剩余 mock
- Performance tuning, bundle optimization, warning cleanup, refactor-only cleanup, and UI polish should be scheduled after the mainline unless they block build, test, or feature delivery.
- When the user says `继续` or `继续开发`, select the next unfinished item on this mainline before taking optimization work.
- Each phase must use real data before it is considered complete. If Jenkins, Harbor/Registry, Kubernetes, PostgreSQL, Redis, Agent runtime, or another required tool is needed to replace mock and is not ready, stop at that phase and record the blocker.

## V1 Ordered Path

1. 基础资源管理: real K8s, Harbor, and Jenkins resource data, connectivity checks, probe refresh, and cached namespaces/projects/views.
2. 环境管理: real local/remote environments, multi-scope resource bindings, status visibility, and Agent readiness separation.
3. Agent 管理与远程探测: real registration key generation, first registration, long-lived Agent token issuance and validation, heartbeat, online status, unbound/pending-claim visibility, project/product binding, task lease data, and remote probing.
4. 项目管理: real project records such as 项目A and 项目B, used as the top-level business ownership boundary.
5. 产品管理: real product records such as 数据中台 and 物联中台 under projects; current environment records are the V1 transition implementation for product deployment scope.
6. 服务与版本来源: real services under products, consuming product deployment-scope namespace/project/view/job ranges and real version source configuration.
7. 发布单创建: real projects, products, agents, services, version sources, and readiness checks.
8. 基线管理: real baseline list, detail, source metadata, and service snapshot source.
9. 部署执行: real platform-direct or Agent execution against the target infrastructure.
10. 发布详情 / 部署详情: persisted real task status, steps, logs, and results.
11. 登录与权限: real login, users, roles, permissions, and project/product/service-level authorization.
12. 清理剩余 mock: remove remaining runtime mock handlers, mock repositories, page fallbacks, and mock-only mainline dependencies.
13. Non-functional optimization work after V1 functional closure.

## User-View Acceptance Path

When reconciling TODO with implementation, prefer checking whether the user can complete the V1 path in the UI:

1. log in and enter the platform
2. view remote environments and Agent status
3. verify the remote Agent can start by direct binary in development with outbound connectivity to the platform; defer `docker compose` to formal production deployment verification
4. create a release for target-existing services
5. create a deploy task for target-missing services
6. track execution state, logs, failure reasons, and action history in detail pages
7. verify real Jenkins/Harbor release only after those environments are ready
8. verify real Kubernetes deployment only after cluster/manifests are ready
9. verify audit and environment-level permission boundaries after the above flow works

If the user asks “现在到哪一步了” or “下一步做什么”, answer against this user-view path and the ordered V1 path together.

## Rules

- Do not duplicate full PRD or API contract content here; link to `docs/` instead.
- Keep TODOs short and actionable.
- Separate committed work from local uncommitted work.
- Prefer backend/domain completion over UI-only expansion when the core flow is incomplete.
- Do not move optimization-only work ahead of the remote deploy/manage mainline unless the optimization is blocking delivery.
- For any step that requires external systems, record the prerequisite environment request in TODO first:
  - Jenkins
  - Harbor or compatible registry
  - Kubernetes
  - remote Agent Linux host with direct binary startup for development and `docker compose` for formal deployment
- Do not store credentials, server addresses, SSH details, or connection strings in TODO files.
- For architectural questions, use `ops-release-platform-architecture`.
- For commit/push workflow, use `ops-release-platform-dev` and `docs/git-submit-workflow.md`.
- If log inspection is needed while checking progress, only read the newest 10 or 20 lines from a log file. Do not load full logs into context.
