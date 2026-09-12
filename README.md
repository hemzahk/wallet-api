# Wallet API

> A digital wallet backend built with Go, PostgreSQL, and REST APIs.

## Project Snapshot

- **Purpose:** Manage customer wallets, merchant checkout sessions, payments, transfers, withdrawals, and refunds.
- **Status:** Portfolio project
- **API docs:** Swagger at `/api/v1/swagger/index.html`

## Highlights

- Customer and merchant registration with JWT authentication and role-based authorization.
- Wallet balances, transaction history, top-ups, transfers, and withdrawals.
- Merchant checkout sessions, payment flows, and customer refund requests.
- Idempotent payment operations and webhook event tracking.
- Background refund worker with graceful shutdown.
- Request rate limiting, structured logging, health checks, and graceful HTTP shutdown.
- Database migrations and focused store/service tests.

## Tech Stack

- Go 1.25+
- PostgreSQL 16
- Chi HTTP router
- JWT, bcrypt, Swagger/OpenAPI
- Docker Compose
- `go test` and SQL mocks

## Run Locally

```bash
docker compose up -d
copy .env.example .env
migrate -path ./cmd/migrate/migrations -database "postgres://LKGVqjLa:QEtxBn5NKnqQ4NvGKsWu@localhost:5432/wallet?sslmode=disable" up
go test ./...
go run ./cmd/api
go run ./cmd/worker
```

Create `.env.example` (copy it to `.env`) with the following variables:

```dotenv
ADDR=:8080
ENV=development
EXTERNAL_URL=localhost:8080
FRONTEND_URL=http://localhost:3000

DB_ADDR=postgres://LKGVqjLa:QEtxBn5NKnqQ4NvGKsWu@localhost:5432/wallet?sslmode=disable
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=25
DB_MAX_IDLE_TIME=15m

FROM_EMAIL=no-reply@example.com
MAILTRAP_USERNAME=your_mailtrap_username
MAILTRAP_PASSWORD=your_mailtrap_password

AUTH_TOKEN_SECRET=replace_with_a_long_random_secret
WEBHOOK_SECRET=replace_with_a_webhook_secret
GATEWAY_BASE_URL=https://example-payment-gateway.test

RATELIMITER_REQUESTS_COUNT=100
RATELIMITER_ENABLED=true
```

The migration command uses the PostgreSQL container from Docker Compose. To roll back the latest migration, run:

```bash
migrate -path ./cmd/migrate/migrations -database "postgres://LKGVqjLa:QEtxBn5NKnqQ4NvGKsWu@localhost:5432/wallet?sslmode=disable" down 1
```

## Contact

- **Author:** Hemza Kareche
- **LinkedIn:** `https://www.linkedin.com/in/hemza-kareche-56933a302/`
- **Email:** hemza.hk18@gmail.com
