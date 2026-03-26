# Implementation Plan: MariaDB DAL API

## Overview

Implement a generic Go HTTP data access layer over MariaDB using httprouter, go-sql-driver/mysql, and API key authentication. Tasks follow the project layout: `cmd/server/main.go`, `internal/config`, `internal/auth`, `internal/handler`, `internal/dal`, `internal/middleware`, `internal/model`.

## Tasks

- [x] 1. Initialize Go module and project structure
  - Run `go mod init` and create all package directories under `cmd/` and `internal/`
  - Add dependencies: `github.com/julienschmidt/httprouter`, `github.com/go-sql-driver/mysql`, `pgregory.net/rapid`
  - Create stub `main.go` in `cmd/server/`
  - _Requirements: 1.1_

- [x] 2. Implement configuration parsing
  - [x] 2.1 Create `internal/config/config.go` with `Config` struct and `Load() (*Config, error)` that reads `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD`, `API_KEYS` (comma-separated), `LISTEN_ADDR` (default `:8080`) from environment variables
    - Return an error if any required DB variable is missing or `API_KEYS` is empty
    - _Requirements: 1.1, 2.1_
  - [x] 2.2 Write property test for config parsing round-trip
    - **Property 1: Config parsing round-trip**
    - **Validates: Requirements 1.1, 2.1**

- [x] 3. Implement shared response model
  - [x] 3.1 Create `internal/model/response.go` with `ErrorResponse` struct and `WriteJSON` / `WriteError` helpers that set `Content-Type: application/json` and encode the response
    - _Requirements: 8.1_

- [x] 4. Implement request logging middleware
  - [x] 4.1 Create `internal/middleware/logging.go` with `NewLoggingMiddleware()` that wraps an `http.Handler`, captures method, path, status code, and latency, and emits a structured log line
    - Use a `responseWriter` wrapper to capture the status code
    - _Requirements: 8.3_

- [x] 5. Implement API key authentication middleware
  - [x] 5.1 Create `internal/auth/middleware.go` with `NewAuthMiddleware(keys []string) func(http.Handler) http.Handler`
    - Skip auth for `GET /health`
    - Return 401 JSON error for missing or non-matching `X-API-Key`
    - _Requirements: 2.2, 2.3, 2.4, 2.5_
  - [x] 5.2 Write property test for auth middleware — missing or invalid key yields 401
    - **Property 2: Auth rejects missing or invalid API keys**
    - **Validates: Requirements 2.2, 2.3, 2.4**
  - [x] 5.3 Write property test for health endpoint auth exemption
    - **Property 3: Health endpoint is exempt from authentication**
    - **Validates: Requirements 2.5**

- [x] 6. Implement resource name and parameter validation
  - [x] 6.1 Create `internal/dal/validate.go` with `ValidateResourceName(name string) error` (regex `^[a-zA-Z0-9_]+$`) and `ParseLimitOffset(limitStr, offsetStr string) (int, int, error)` that reject negative or non-integer values
    - _Requirements: 9.2, 9.3_
  - [x] 6.2 Write property test for resource name validation
    - **Property 15: Resource name validation rejects invalid characters**
    - **Validates: Requirements 9.2**
  - [x] 6.3 Write property test for limit/offset validation
    - **Property 16: Limit and offset validation rejects invalid values**
    - **Validates: Requirements 9.3**

- [x] 7. Implement dynamic query builder
  - [x] 7.1 Create `internal/dal/query.go` with functions that build parameterized SQL strings for INSERT, SELECT (with filters, limit, offset), UPDATE (full replace), UPDATE (partial/PATCH), and DELETE
    - Backtick-quote table and column identifiers; all values passed as `?` placeholders
    - _Requirements: 9.1_
  - [x] 7.2 Write unit tests for query builder output
    - Test each operation (INSERT, SELECT with/without filters, UPDATE, PATCH, DELETE) for correct SQL shape and placeholder count
    - _Requirements: 9.1_

- [x] 8. Implement DAL layer
  - [x] 8.1 Create `internal/dal/dal.go` implementing the `DAL` interface: `Insert`, `GetByID`, `List`, `Update`, `Patch`, `Delete`, `Ping`
    - Use `database/sql` with `go-sql-driver/mysql`; scan rows into `map[string]any` using `sql.RawBytes` / `ColumnTypes`
    - Map MySQL error numbers 1146 → 404, 1062/1451/1452 → 409, `sql.ErrNoRows` → 404
    - _Requirements: 3.1, 3.3, 3.4, 4.1, 4.2, 4.3, 5.1, 5.2, 5.4, 5.5, 5.6, 6.1, 6.2, 6.3, 6.5, 7.1, 7.2, 7.3, 9.1_
  - [x] 8.2 Write unit tests for DB error code mapping
    - Test that each MySQL error number maps to the correct HTTP-equivalent sentinel error
    - _Requirements: 3.3, 6.5, 7.3_

- [x] 9. Checkpoint — Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [x] 10. Implement health handler
  - [x] 10.1 Create `internal/handler/health.go` with `HealthHandler` that calls `dal.Ping(ctx)`
    - Return 200 `{"status":"ok"}` on success; 503 JSON error on failure
    - _Requirements: 1.4, 1.5_
  - [x] 10.2 Write unit tests for health handler with mocked DAL
    - Test both DB-up (200) and DB-down (503) states
    - _Requirements: 1.4, 1.5_

- [x] 11. Implement resource CRUD handlers
  - [x] 11.1 Create `internal/handler/resource.go` with `ResourceHandler` struct and methods: `Create`, `GetByID`, `List`, `Update`, `Patch`, `Delete`
    - Each method: validate resource name, parse/validate body or params, call DAL, map DAL errors to HTTP status codes, write JSON response
    - `List` reads `limit`, `offset`, and remaining query params as filters; default limit 100
    - _Requirements: 3.1, 3.2, 3.3, 3.4, 4.1, 4.2, 4.3, 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 6.1, 6.2, 6.3, 6.4, 6.5, 7.1, 7.2, 7.3_
  - [x] 11.2 Write property test for invalid JSON body yields 400
    - **Property 4: Invalid JSON body yields 400**
    - **Validates: Requirements 3.2, 6.4**
  - [x] 11.3 Write property test for unknown resource yields 404
    - **Property 5: Unknown resource yields 404**
    - **Validates: Requirements 3.4, 4.3, 5.6**
  - [x] 11.4 Write property test for missing record yields 404
    - **Property 6: Missing record yields 404**
    - **Validates: Requirements 4.2, 6.3, 7.2**
  - [x] 11.5 Write property test for error responses are well-formed JSON
    - **Property 14: Error responses are well-formed JSON**
    - **Validates: Requirements 8.1**

- [x] 12. Checkpoint — Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [x] 13. Wire server in main.go
  - [x] 13.1 Update `cmd/server/main.go` to: load config, open DB connection pool (`SetMaxOpenConns(25)`, `SetMaxIdleConns(5)`, `SetConnMaxLifetime(5*time.Minute)`), ping DB (exit 1 on failure), construct DAL, handlers, and middlewares, register all routes on httprouter, start HTTP server
    - Routes: `GET /health`, `POST /:resource`, `GET /:resource`, `GET /:resource/:id`, `PUT /:resource/:id`, `PATCH /:resource/:id`, `DELETE /:resource/:id`
    - Wrap router with logging then auth middleware
    - _Requirements: 1.1, 1.2, 1.3, 1.4_

- [x] 14. Integration tests — full CRUD lifecycle
  - [x] 14.1 Create `internal/dal/integration_test.go` using testcontainers-go (or a real MariaDB via env) to test the full CRUD lifecycle: create table, Insert, GetByID, List with filters, Update, Patch, Delete
    - _Requirements: 3.1, 4.1, 5.1, 5.2, 6.1, 6.2, 7.1_
  - [x] 14.2 Write property test for insert round-trip
    - **Property 7: Insert round-trip**
    - **Validates: Requirements 3.1, 4.1**
  - [x] 14.3 Write property test for list filter correctness
    - **Property 8: List filter correctness**
    - **Validates: Requirements 5.1, 5.2, 5.3**
  - [x] 14.4 Write property test for limit invariant
    - **Property 9: Limit invariant**
    - **Validates: Requirements 5.4**
  - [x] 14.5 Write property test for offset pagination consistency
    - **Property 10: Offset pagination consistency**
    - **Validates: Requirements 5.5**
  - [x] 14.6 Write property test for PUT round-trip
    - **Property 11: PUT round-trip**
    - **Validates: Requirements 6.1**
  - [x] 14.7 Write property test for PATCH partial update preserves unmodified fields
    - **Property 12: PATCH partial update preserves unmodified fields**
    - **Validates: Requirements 6.2**
  - [x] 14.8 Write property test for delete round-trip
    - **Property 13: Delete round-trip**
    - **Validates: Requirements 7.1, 7.2**

- [x] 15. Final checkpoint — Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for a faster MVP
- Each task references specific requirements for traceability
- Property tests use `pgregory.net/rapid` with a minimum of 100 iterations each
- Each property test must include a comment: `// Feature: mariadb-dal-api, Property N: <property text>`
- All DB queries must use parameterized placeholders — never string interpolation
