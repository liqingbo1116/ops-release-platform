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
- The V1 delivery bar is: the platform must at least support remote project-environment deployment and management.
- If the next task is about real Jenkins, Harbor/Registry, Kubernetes, or Agent integration, call out the required environment prerequisites before implementation continues.
- Assume the V1 Agent deployment model unless docs are explicitly changed:
  - Linux host
  - `docker compose`
  - Agent operates remote Kubernetes, but is not itself required to run in Kubernetes
- Treat the following path as the mainline unless the user explicitly reprioritizes:
  - project environment management
  - Agent registration and status
  - runtime snapshot collection or mock-equivalent collection flow
  - baseline generation and lock
  - baseline-to-target diff
  - release creation for target-existing services
  - deploy task creation for target-missing services
  - remote execution through Agent task flow
  - release/deploy logs, retry states, and audit visibility
- Performance tuning, bundle optimization, warning cleanup, refactor-only cleanup, and UI polish should be scheduled after the mainline unless they block build, test, or feature delivery.
- When the user says `继续` or `继续开发`, select the next unfinished item on this mainline before taking optimization work.

## V1 Ordered Path

1. Release/deploy detail closure.
2. Agent protocol completion.
3. Runtime snapshot and baseline generation flow.
4. Baseline lock and baseline detail persistence/display.
5. Backend-owned diff classification and action generation.
6. Release management closure:
   - single-service release
   - multi-service release
   - baseline diff release
   - remote Agent execution state updates
7. Deployment management closure for remote project environments:
   - deploy task creation for missing services
   - step orchestration
   - retry / skip / manual confirm support
   - detail logs and result display
8. Audit and permission completion for the above flows.
9. Product/service management and image management features that are required to operate the above flows.
10. Non-functional optimization work after V1 functional closure.

## User-View Acceptance Path

When reconciling TODO with implementation, prefer checking whether the user can complete the V1 path in the UI:

1. log in and enter the platform
2. view remote environments and Agent status
3. generate or inspect a baseline from a source environment
4. compare source baseline with target environment
5. create a release for target-existing services
6. create a deploy task for target-missing services
7. track execution state, logs, failure reasons, and action history in detail pages
8. observe Agent online status, recent heartbeat, and recent task result
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
