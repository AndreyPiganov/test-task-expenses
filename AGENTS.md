# Repository guide for coding agents

## Project purpose

Family expense tracker with a Go REST API, PostgreSQL, a Telegram bot that uses
LLM tool calling, and a React/Vite web panel served by nginx.

## Repository map

- `backend/cmd/server`: dependency wiring and process lifecycle.
- `backend/internal/domain`: transport-independent entities and validation.
- `backend/internal/service`: application use cases.
- `backend/internal/repository`: persistence ports.
- `backend/internal/repository/postgres`: PostgreSQL adapters.
- `backend/internal/httpapi`: REST transport, request/response DTOs, middleware.
- `backend/internal/telegram`: Telegram long-polling transport.
- `backend/internal/agent`: LLM tool definitions, argument decoding, and expense
  command orchestration.
- `backend/internal/currency`: foreign-currency conversion through the Bank of
  Russia API.
- `backend/migrations`: embedded SQL migrations.
- `web`: React and TypeScript frontend.

Dependencies point inward: transports and adapters may depend on services and
domain types, while domain and services must not depend on HTTP, Telegram, LLM,
or PostgreSQL implementations.

## Commands

Run from the repository root:

```shell
make fmt
make test
make vet
docker compose up --build
```

Frontend checks:

```shell
cd web
npm ci
npm run build
```

The database integration test runs only when `TEST_DATABASE_URL` points to a
disposable PostgreSQL database.

## Change rules

- Keep secrets in `.env`; never commit `.env`, Telegram tokens, API keys, or
  production database URLs.
- Use `shopspring/decimal` for money. Do not convert monetary values through
  `float64`.
- Keep domain structs independent from JSON and database representation.
- Define HTTP JSON contracts in `internal/httpapi` and map them explicitly.
- Put business validation in `domain` or `service`, not only in handlers.
- Keep interfaces close to the consumer that owns the dependency.
- Add or update focused tests with every behavior change.
- Do not change migrations that may already have been applied; add a new
  migration instead.

## LLM and Telegram safety

- Treat tool names and tool arguments as untrusted input even with strict tool
  schemas; decode and validate them before performing an operation.
- Never expose `user_id` as an LLM tool argument. The Telegram user ID must come
  from the authenticated Telegram update and be injected by backend code.
- Restrict Telegram reads and mutations by that user ID in repository queries.
- Require explicit delete intent and never mutate an ambiguous match.
- Currency conversion belongs to backend code. The model only extracts the
  original amount and ISO 4217 currency code.
- Keep one tool call per Telegram message unless the orchestration design and
  safety checks are deliberately revised.

## Definition of done

A change is complete when affected behavior is tested, Go code is formatted,
`go test ./...` and `go vet ./...` pass, the frontend builds when touched, and
README/API documentation reflects user-visible configuration changes.
