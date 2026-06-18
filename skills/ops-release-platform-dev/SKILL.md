---
name: ops-release-platform-dev
description: Use for continuing development, review, testing, deployment notes, or Git submission work in the ops-release-platform repository. Includes project context, completed milestone state, repo organization, validation commands, security constraints, mock integration boundaries, and required Git workflow for this specific ops-release-platform project.
---

# Ops Release Platform Dev

Use this skill before making code, docs, deployment, or Git changes in this repository.

## First Steps

1. Check the current workspace before assuming state:
   - `git status --short --branch`
   - `git log -1 --oneline`
   - Read task-specific docs under `docs/`.
2. Read references only as needed:
   - Current milestone and known state: `references/project-state.md`
   - Repository layout and ownership boundaries: `references/repo-map.md`
   - Validation, deployment, and Git workflow: `references/workflows.md`
   - TODO/backlog work is split into `../ops-release-platform-todo/`
   - Architecture decisions are split into `../ops-release-platform-architecture/`
   - Deployment and local runtime rules are split into `../ops-release-platform-deployment/`
3. Prefer existing project patterns over new abstractions.
4. Keep third-party systems behind adapter interfaces. Do not implement real Jenkins, Harbor, Kubernetes, GitLab, ArgoCD, or Nacos integration unless explicitly requested.
5. Never commit server credentials, `.secrets/`, deployment passwords, SSH connection details, or real external-system credentials.

## Development Rules

- Frontend stack: Vue 3, Vite, TypeScript, Pinia, Vue Router, Element Plus.
- Backend stack: Go, Gin, GORM, PostgreSQL, Redis Stream.
- V1 mainline must use real data phase by phase. Once a phase enters mainline replacement, remove that phase's mock data and mock fallback before moving to the next phase.
- If a phase depends on external tools or runtime environments to replace mock, those prerequisites are mandatory gates. If they are not ready, do not claim that phase is complete and do not continue to the next phase.
- Local development runtime: frontend and backend run locally, but PostgreSQL and Redis must come from the remote services recorded in `.secrets/`. Do not switch back to local container mock services during V1 mainline development.
- K8s, Harbor, and Jenkins must be maintainable platform resources. Environments reference those resource IDs and add per-environment scope fields: Kubernetes `namespace`, Harbor `registryProject`, and Jenkins `jenkinsView`.
- Resource create forms must be user-oriented: K8s uses kubeconfig/context, Harbor uses URL + HTTP/HTTPS + username/password, Jenkins uses URL + username/password or API token. Users must not enter `credentialRef` directly.
- Resource status is system-owned. Keep connection test and refresh/probe actions; cache K8s namespaces, Harbor projects, and Jenkins views/jobs, and keep old cache when refresh fails.
- Local/direct environments are probed by the platform backend. Remote/project environments are probed by Agent tasks, then Agent reports status and cache back to the platform.
- `.secrets/` is only the development-stage private value source for local process startup and real integration checks. It is not the formal platform resource master data source.
- Platform resource records may store non-secret metadata such as API server or service URL and only internal credential references for credentials. Never copy real credentials, kubeconfig contents, token values, database DSNs, or Redis passwords into docs, code, tests, logs, commits, or chat output.
- Local development runtime: run frontend with npm and backend with `go run`; do not use docker-compose for frontend/backend during development. See `../ops-release-platform-deployment/`.
- For user requests like "继续开发", choose the next clear item from `docs/development-plan.md` and current repository state.
- For user requests like "提交", follow `docs/git-submit-workflow.md` and the extra checks in `references/workflows.md`.
- If you must inspect log files, only read a small tail of the newest lines, such as the latest 10 or 20 lines. Never load the full log file into context.

## V1 Mainline Gates

Follow this order exactly. Do not skip forward. Do not reintroduce mock for a completed phase.

1. Resource management
   - Goal: K8s, Harbor, and Jenkins resources are maintained as real platform data with user-oriented create forms, system-owned status, connection tests, probe cache, and refresh actions.
   - Required tools/environment:
     - frontend local runtime
     - backend local runtime
     - remote PostgreSQL from `.secrets/`
     - remote Redis from `.secrets/`
     - `.secrets/` loaded with the real DSN and Redis address
     - real K8s kubeconfig for direct/local resource checks
     - real Harbor URL and credential input; HTTP registries must be supported
     - real Jenkins URL and credential input when Jenkins is part of the current acceptance path
     - remote Agent runtime before accepting remote/project resource probing
     - `.secrets/integration-connections.*` loaded only for development-stage private values when `INTEGRATION_MODE=real`
   - Gate: if PostgreSQL or Redis is not ready, resource management cannot replace mock and next phase cannot start.
   - Gate: if resource forms still expose `credentialRef`, statuses are user-editable, probe cache is mock/missing, or remote probing bypasses Agent, this phase is not complete.
2. Environment management
   - Goal: environment list, detail, create, update, status, and dependency visibility all come from real backend data; environments reference resources and only add `namespace`, `registryProject`, and `jenkinsView`.
   - Required tools/environment:
     - phase 1 complete
     - real K8s namespace cache
     - real Harbor project cache
     - real Jenkins view/job cache when Jenkins is in scope
   - Gate: if environments still embed credentials, use mock scope options, or force users to maintain both environment ID and environment code, this phase is not complete.
3. Agent management and remote probing
   - Goal: agent registration, heartbeat, environment binding, online status, and task lease data all come from real backend data.
   - Required tools/environment:
     - phase 1 and phase 2 complete
     - created environment records in platform
     - reachable platform API
     - built agent binary for R&D direct startup
     - agent config file support via `-f`
     - remote Linux host that can reach platform API outbound
     - Agent-side network and credentials for remote K8s/Harbor/Jenkins probing
   - Gate: if environment records, remote agent runtime, or Agent-reported remote probe status/cache are not ready, this phase is not complete.
4. Service and version sources
   - Goal: service lists, Jenkins jobs, Harbor image tags, and runtime versions come from real sources.
   - Required tools/environment:
     - phase 1 through phase 3 complete
     - real service/version source such as Jenkins metadata, registry tags, Kubernetes runtime data, or persisted runtime snapshots
   - Gate: if service/version data is still mock, this phase is not complete.
5. Release creation
   - Goal: release form reads real environments, real agents, and real service/version sources.
   - Required tools/environment:
     - phase 1 through phase 4 complete
     - real environment-agent association
   - Gate: if release source data is still mock, release creation is not complete and next phase cannot start.
6. Baseline management
   - Goal: baseline list, baseline detail, and baseline source metadata come from real runtime snapshot or persistent business data.
   - Required tools/environment:
     - phase 1 through phase 4 complete
     - clear real baseline source, such as Kubernetes runtime data, registry data, or platform persisted snapshot tables
   - Gate: if baseline source is still mock, baseline management is not complete and next phase cannot start.
7. Deployment execution
   - Goal: deployment tasks are leased by the real agent and executed against real target infrastructure.
   - Required tools/environment:
     - phase 1 through phase 6 complete
     - agent host with required executors, including `kubectl`
     - target cluster credentials
     - registry access credentials
     - network path from agent to cluster, registry, and platform
   - Gate: if agent cannot execute against the real target infrastructure, deployment execution is not complete and next phase cannot start.
8. Release/deployment detail
   - Goal: detail pages display real task status, logs, steps, and final result from persisted execution data.
   - Required tools/environment:
     - phase 7 complete
     - backend persistence for task status, step logs, and final report
   - Gate: if detail pages still depend on mock task data, this phase is not complete and next phase cannot start.
9. Auth and permissions
   - Goal: login, user, role, permission, and environment-level authorization all use real backend data.
   - Required tools/environment:
     - user table, role table, permission table
     - real login mechanism
     - environment-level permission model
   - Gate: if login or permission data is still mock, V1 mainline is not complete.
10. Final mock cleanup
   - Goal: remove remaining runtime mock handlers, mock repositories, and mock page fallbacks that were only used during early scaffolding.
   - Required tools/environment:
     - phase 1 through phase 9 complete
   - Gate: after this phase, V1 mainline must not depend on runtime mock data.

## Update This Skill

After a meaningful milestone or commit, update `references/project-state.md` with:

- completed task or phase
- important files changed
- validation results
- known warnings or blockers
- whether the work was committed and pushed
