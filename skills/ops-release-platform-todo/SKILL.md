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
  - `docker compose`
  - Agent operates remote Kubernetes, but is not itself required to run in Kubernetes
  - Agent must be independently deployable before V1 remote release/deploy can be accepted
  - Agent leases/pulls release/deploy task payloads from the platform and reports heartbeat, service list, image versions, status, logs, and final result back to the platform
- Treat the following path as the mainline unless the user explicitly reprioritizes:
  - remote Agent deployment package
  - Agent outbound task lease/pull flow
  - remote Agent mock executor
  - release/deploy detail closure against Agent callbacks
  - real release integration through Jenkins and Harbor/Registry
  - real deployment integration through Kubernetes
  - remote project environment deploy/manage V1 acceptance
  - audit, permission, and persistence completion
- Performance tuning, bundle optimization, warning cleanup, refactor-only cleanup, and UI polish should be scheduled after the mainline unless they block build, test, or feature delivery.
- When the user says `继续` or `继续开发`, select the next unfinished item on this mainline before taking optimization work.

## V1 Ordered Path

1. Remote Agent deployment package for Linux + `docker compose`.
2. Agent outbound task lease/pull protocol:
   - Agent registration and environment binding
   - Agent leases/pulls task payload and execution data from the platform API
   - Agent reports heartbeat, step status, logs, and final result
3. Remote Agent mock executor that runs outside the platform process and reports mock steps/logs/results.
4. Release/deploy detail closure against remote Agent callbacks:
   - `agentTaskId`
   - lease state
   - steps, logs, failure reasons, retry/skip/manual-confirm/rollback visibility
5. Real release integration:
   - Jenkins-triggered release
   - Harbor/Registry image selection/sync
   - workload tag update through Agent
6. Real deployment integration:
   - runtime snapshot collection
   - target-missing service deployment
   - workload update and health check
7. Remote project-environment deploy/manage V1 acceptance.
8. Audit, permission, and persistence completion for the above flows.
9. Non-functional optimization work after V1 functional closure.

## User-View Acceptance Path

When reconciling TODO with implementation, prefer checking whether the user can complete the V1 path in the UI:

1. log in and enter the platform
2. view remote environments and Agent status
3. verify the remote Agent can be deployed by `docker compose` and connects outbound to the platform
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
  - remote Agent Linux host with `docker compose`
- Do not store credentials, server addresses, SSH details, or connection strings in TODO files.
- For architectural questions, use `ops-release-platform-architecture`.
- For commit/push workflow, use `ops-release-platform-dev` and `docs/git-submit-workflow.md`.
- If log inspection is needed while checking progress, only read the newest 10 or 20 lines from a log file. Do not load full logs into context.
