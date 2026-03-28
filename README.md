# MariaDB DAL API

A generic HTTP data access layer written in Go that exposes RESTful CRUD endpoints over any MariaDB table. Resource names come from URL path segments — no per-table configuration required.

## Requirements

- Go 1.24+
- MariaDB (or MySQL-compatible) instance

## Quick start

```bash
# 1. Generate an API key
go run ./cmd/keygen

# 2. Export environment variables
export DB_HOST=localhost
export DB_PORT=3306
export DB_NAME=mydb
export DB_USER=root
export DB_PASSWORD=secret
export API_KEYS=<paste key from step 1>

# 3. Run the server
go run ./cmd/server
```

## Docker

### Build the image

```bash
docker build -t mariadb-dal-api .
```

### Run the container

```bash
docker run -d \
  -p 8080:8080 \
  -e DB_HOST=your-db-host \
  -e DB_PORT=3306 \
  -e DB_NAME=mydb \
  -e DB_USER=root \
  -e DB_PASSWORD=secret \
  -e API_KEYS=your-api-key \
  --name mariadb-dal-api \
  mariadb-dal-api
```

Override the listen address or expose a different port:

```bash
docker run -d \
  -p 9090:9090 \
  -e LISTEN_ADDR=:9090 \
  ... \
  mariadb-dal-api
```

### Docker Compose example

```yaml
services:
  db:
    image: mariadb:11
    environment:
      MARIADB_ROOT_PASSWORD: secret
      MARIADB_DATABASE: mydb
    healthcheck:
      test: ["CMD", "healthcheck.sh", "--connect"]
      interval: 5s
      retries: 10

  api:
    image: mariadb-dal-api
    build: .
    ports:
      - "8080:8080"
    environment:
      DB_HOST: db
      DB_PORT: 3306
      DB_NAME: mydb
      DB_USER: root
      DB_PASSWORD: secret
      API_KEYS: your-api-key
    depends_on:
      db:
        condition: service_healthy
```

```bash
docker compose up -d
```

### Show help

```bash
docker run --rm mariadb-dal-api --help
```

---

## Generating API keys

The `keygen` helper creates cryptographically secure 32-byte hex keys.

```bash
# One key
go run ./cmd/keygen

# Multiple keys (also prints comma-separated value for API_KEYS)
go run ./cmd/keygen -n 3
```

Output for `-n 3`:

```
# Comma-separated (paste into API_KEYS):
a1b2c3...,d4e5f6...,g7h8i9...

# Individual keys:
  key 1: a1b2c3...
  key 2: d4e5f6...
  key 3: g7h8i9...
```

## Configuration

All configuration is via environment variables. The server exits with a non-zero status if any required variable is missing.

| Variable      | Required | Default | Description                                      |
|---------------|----------|---------|--------------------------------------------------|
| `DB_HOST`     | yes      | —       | MariaDB hostname or IP                           |
| `DB_PORT`     | yes      | —       | MariaDB port (typically `3306`)                  |
| `DB_NAME`     | yes      | —       | Database name                                    |
| `DB_USER`     | yes      | —       | Database username                                |
| `DB_PASSWORD` | yes      | —       | Database password                                |
| `API_KEYS`    | yes      | —       | Comma-separated list of valid API keys           |
| `LISTEN_ADDR` | no       | `:8080` | Address and port the HTTP server listens on      |

Multiple API keys example:

```bash
export API_KEYS="key-one,key-two,key-three"
```

## Authentication

Every request (except `GET /health`) must include the `X-API-Key` header.

```bash
curl -H "X-API-Key: <your-key>" http://localhost:8080/users
```

Missing or invalid key returns `401 Unauthorized`:

```json
{ "error": "missing API key" }
{ "error": "invalid API key" }
```

## API reference

### Health check

```
GET /health
```

No authentication required. Returns `200` when the database is reachable, `503` otherwise.

```bash
curl http://localhost:8080/health
# 200 {"status":"ok"}
```

---

### Create a record

```
POST /:resource
```

Inserts a new row into the named table. Returns `201` with the created record (including the auto-generated `id`).

```bash
curl -X POST http://localhost:8080/users \
  -H "X-API-Key: <key>" \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice","email":"alice@example.com"}'
# 201 {"id":"1","name":"Alice","email":"alice@example.com"}
```

| Status | Meaning                                      |
|--------|----------------------------------------------|
| 201    | Record created                               |
| 400    | Invalid JSON body or invalid resource name   |
| 404    | Table does not exist                         |
| 409    | Constraint violation (duplicate key, FK)     |

---

### Get a record by ID

```
GET /:resource/:id
```

Returns the row where `id = :id`.

```bash
curl http://localhost:8080/users/1 \
  -H "X-API-Key: <key>"
# 200 {"id":"1","name":"Alice","email":"alice@example.com"}
```

| Status | Meaning                  |
|--------|--------------------------|
| 200    | Record found             |
| 404    | Record or table not found|

---

### List records

```
GET /:resource[?limit=N&offset=N&col=val...]
```

Returns a JSON array of rows. Any query parameter other than `limit` and `offset` is treated as an equality filter on that column.

```bash
# All users (up to 100)
curl http://localhost:8080/users -H "X-API-Key: <key>"

# Filter by column value
curl "http://localhost:8080/users?status=active" -H "X-API-Key: <key>"

# Pagination
curl "http://localhost:8080/users?limit=10&offset=20" -H "X-API-Key: <key>"

# Combined
curl "http://localhost:8080/orders?status=pending&limit=5&offset=0" -H "X-API-Key: <key>"
```

| Parameter | Type    | Default | Description                              |
|-----------|---------|---------|------------------------------------------|
| `limit`   | integer | `100`   | Maximum number of records to return      |
| `offset`  | integer | `0`     | Number of records to skip (pagination)   |
| any other | string  | —       | Equality filter on that column           |

| Status | Meaning                          |
|--------|----------------------------------|
| 200    | Success (empty array if no match)|
| 400    | Invalid limit/offset or resource |
| 404    | Table does not exist             |

---

### Replace a record (full update)

```
PUT /:resource/:id
```

Fully replaces the row where `id = :id` with the request body. Returns `200` with the updated record.

```bash
curl -X PUT http://localhost:8080/users/1 \
  -H "X-API-Key: <key>" \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice Smith","email":"alice.smith@example.com"}'
# 200 {"id":"1","name":"Alice Smith","email":"alice.smith@example.com"}
```

| Status | Meaning                                  |
|--------|------------------------------------------|
| 200    | Record updated                           |
| 400    | Invalid JSON body or resource name       |
| 404    | Record or table not found                |
| 409    | Constraint violation                     |

---

### Partial update

```
PATCH /:resource/:id
```

Updates only the fields present in the request body. Unspecified fields are left unchanged. Returns `200` with the full updated record.

```bash
curl -X PATCH http://localhost:8080/users/1 \
  -H "X-API-Key: <key>" \
  -H "Content-Type: application/json" \
  -d '{"email":"newemail@example.com"}'
# 200 {"id":"1","name":"Alice Smith","email":"newemail@example.com"}
```

| Status | Meaning                                  |
|--------|------------------------------------------|
| 200    | Record patched                           |
| 400    | Invalid JSON body or resource name       |
| 404    | Record or table not found                |
| 409    | Constraint violation                     |

---

### Delete a record

```
DELETE /:resource/:id
```

Removes the row where `id = :id`. Returns `204` with no body.

```bash
curl -X DELETE http://localhost:8080/users/1 \
  -H "X-API-Key: <key>"
# 204 (no body)
```

| Status | Meaning                              |
|--------|--------------------------------------|
| 204    | Record deleted                       |
| 404    | Record or table not found            |
| 409    | Constraint violation (FK reference)  |

---

## Error responses

All errors use the same JSON envelope:

```json
{ "error": "human-readable message" }
```

| Status | Cause                                                  |
|--------|--------------------------------------------------------|
| 400    | Invalid JSON, invalid resource name, bad limit/offset  |
| 401    | Missing or invalid `X-API-Key`                         |
| 404    | Table or record not found                              |
| 409    | Database constraint violation                          |
| 500    | Unexpected server error (details logged server-side)   |
| 503    | Database unreachable (health check only)               |

## Resource name rules

Resource names (table names) must match `^[a-zA-Z0-9_]+$`. Any other characters return `400` before a query is executed.

## Running tests

```bash
# Unit and property tests
go test ./...

# Integration tests require a running MariaDB instance
DB_HOST=localhost DB_PORT=3306 DB_NAME=testdb DB_USER=root DB_PASSWORD=secret \
  go test ./internal/dal/... -run TestIntegration -v
```

## Connection pool defaults

| Setting              | Value      |
|----------------------|------------|
| `MaxOpenConns`       | 25         |
| `MaxIdleConns`       | 5          |
| `ConnMaxLifetime`    | 5 minutes  |

## Project layout

```
cmd/
  keygen/   — API key generator utility
  server/   — HTTP server entry point
internal/
  auth/     — X-API-Key middleware
  config/   — environment variable parsing
  dal/      — database access layer + query builder + validation
  handler/  — HTTP handlers (health, CRUD)
  middleware/— request logging
  model/    — shared response types
```
