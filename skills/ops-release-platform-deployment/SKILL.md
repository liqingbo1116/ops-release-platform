---
name: ops-release-platform-deployment
description: Use for local startup, development runtime, docker-compose usage, deployment topology, service start/stop instructions, or infrastructure runtime decisions in the ops-release-platform repository. Applies when deciding whether frontend/backend should run via npm/go run or docker-compose, and when documenting PostgreSQL/Redis runtime requirements.
---

# Ops Release Platform Deployment

Use this skill before starting services, changing deployment files, or documenting runtime topology.

## Required Development Rule

During development:

- Start frontend locally with npm commands.
- Start backend locally with `go run`.
- Start the standalone Agent by building the binary with Go and running it directly with `-f <config-file>` when Agent-side development or remote-like verification is needed.
- Do not start frontend or backend through docker-compose.
- Do not treat `agent/docker-compose.yml` as the default development entrypoint for the Agent during development.
- PostgreSQL and Redis must use the fixed remote development services recorded in `.secrets/local-dev-env.ps1`.
- Do not silently fall back to local PostgreSQL, local Redis, docker-compose PostgreSQL/Redis, or empty backend env when `.secrets/local-dev-env.ps1` is expected for development.

For formal remote deployment, the Agent still uses the packaged `docker compose` path on the project-environment Linux host.

## Workflow

1. Read `references/deployment.md`.
2. Check `.secrets/local-dev-env.ps1` exists before starting backend.
3. Treat missing `.secrets/local-dev-env.ps1` as a blocker for normal backend development startup and ask the user to restore it instead of inventing replacement connection settings.
4. Start backend from `backend/` and frontend from `frontend/`; never launch from the repository root by accident.
5. For Linux/Bash startup, convert `.secrets/local-dev-env.ps1` into shell exports in-process and do not print the resulting values.
6. When services need to remain running after the current command finishes, start them with the documented `setsid -f bash -lc ...` commands and write logs to `/tmp/ops-release-platform-backend.log` and `/tmp/ops-release-platform-frontend.log`.
7. Validate backend with `/api/environments` and frontend with `/`; do not use `/api/health` unless that endpoint is added later.
8. Use npm/go commands for local frontend/backend runtime, and use `go build` output for Agent runtime.
9. Use docker-compose only for explicit infrastructure/deployment tasks during development; keep it as the formal Agent deployment path.

## Security

- Do not commit `.secrets/`.
- Do not write real host addresses, passwords, SSH ports, database DSNs, Redis addresses, or tokens into tracked files.
- Final summaries may mention that remote PostgreSQL/Redis are used, but must not print their connection details.
- If you inspect service logs, only read a small tail of the newest 10 or 20 lines. Do not load full log files into context.
