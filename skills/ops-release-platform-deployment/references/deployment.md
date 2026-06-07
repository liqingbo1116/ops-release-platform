# Deployment And Local Runtime

Last updated: 2026-06-07

## Development Runtime

Development must use this topology:

- Frontend: local process started by npm.
- Backend: local process started by Go.
- PostgreSQL: remote development service configured in `.secrets/local-dev-env.ps1`.
- Redis: remote development service configured in `.secrets/local-dev-env.ps1`.

Do not use docker-compose to start frontend or backend during development.

## Frontend Commands

Install dependencies:

```powershell
cd frontend
npm install
```

Start development server:

```powershell
cd frontend
npm run dev
```

Build:

```powershell
cd frontend
npm run build
```

Unit tests:

```powershell
cd frontend
npm run test:unit
```

## Backend Commands

Load local development environment variables before starting the backend:

```powershell
. .\.secrets\local-dev-env.ps1
```

Start backend from the backend directory:

```powershell
cd backend
go run ./cmd/server
```

If already inside `backend`, load the env file with:

```powershell
. ..\.secrets\local-dev-env.ps1
go run ./cmd/server
```

Tests:

```powershell
cd backend
go test ./...
```

## Docker Compose Rule

Do not run frontend/backend through docker-compose in normal development.

Allowed docker-compose usage:

- Validating compose syntax when Docker is available.
- Explicit infrastructure/deployment tasks requested by the user.
- Remote or server-side PostgreSQL/Redis-only deployment tasks.

Disallowed during development:

- `docker compose up frontend`
- `docker compose up backend`
- `docker compose up` for the purpose of running local frontend/backend development servers

## Configuration

Tracked files may mention `.secrets/local-dev-env.ps1` as the secret source, but must not include its values.

The env file should provide:

- `APP_PORT`
- `DATABASE_DSN`
- `REDIS_ADDR`
- `INTEGRATION_MODE=mock`

## Reporting

When reporting local startup or deployment status:

- Say whether frontend/backend are local processes.
- Say whether PostgreSQL/Redis are remote services.
- Do not print the actual remote host, ports, passwords, DSNs, SSH details, or tokens.
