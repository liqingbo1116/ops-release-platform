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
- Do not start frontend or backend through docker-compose.
- PostgreSQL and Redis use the remote development services recorded in `.secrets/`.

## Workflow

1. Read `references/deployment.md`.
2. Check `.secrets/local-dev-env.ps1` exists before starting backend.
3. Use npm/go commands for local frontend/backend runtime.
4. Use docker-compose only for explicit infrastructure/deployment tasks, not for normal frontend/backend development.

## Security

- Do not commit `.secrets/`.
- Do not write real host addresses, passwords, SSH ports, database DSNs, Redis addresses, or tokens into tracked files.
- Final summaries may mention that remote PostgreSQL/Redis are used, but must not print their connection details.
