# Requirements Document

## Introduction

A data access layer (DAL) API built in Go that sits in front of a MariaDB database. The API exposes HTTP endpoints for generic CRUD operations, abstracting direct database access from consumers. It handles connection management, query execution, error handling, and response serialization.

## Glossary

- **DAL_API**: The Go HTTP server that exposes CRUD endpoints over MariaDB.
- **Client**: Any application or service that sends HTTP requests to the DAL_API.
- **Record**: A single row in a MariaDB table, represented as a JSON object.
- **Resource**: A named MariaDB table exposed through the API (e.g., `/users`, `/orders`).
- **DB_Connection**: The managed connection pool between the DAL_API and MariaDB.
- **Request_Body**: The JSON payload sent by the Client in a POST or PUT request.
- **Response_Body**: The JSON payload returned by the DAL_API to the Client.
- **Primary_Key**: The unique identifier column used to address individual Records. The Primary_Key column is always named `id`.
- **API_Key**: A secret token issued to a Client, used to authenticate requests to the DAL_API.
- **Filter**: A set of query parameters used to narrow down a list query.

---

## Requirements

### Requirement 1: Server Initialization

**User Story:** As an operator, I want the DAL_API to start up and connect to MariaDB, so that it is ready to serve requests.

#### Acceptance Criteria

1. THE DAL_API SHALL read database connection parameters (host, port, database name, username, password) from environment variables at startup.
2. WHEN the DAL_API starts, THE DAL_API SHALL establish a DB_Connection pool to MariaDB before accepting any HTTP requests.
3. IF the DB_Connection cannot be established at startup, THEN THE DAL_API SHALL log a descriptive error message and exit with a non-zero status code.
4. THE DAL_API SHALL expose a health check endpoint at `GET /health` that returns HTTP 200 when the DB_Connection is active.
5. IF the DB_Connection is unavailable during a health check, THEN THE DAL_API SHALL return HTTP 503 with a descriptive error message.

---

### Requirement 2: API Key Authentication

**User Story:** As an operator, I want all API requests to require a valid API key, so that only authorized clients can access the DAL_API.

#### Acceptance Criteria

1. THE DAL_API SHALL read one or more valid API_Keys from an environment variable at startup.
2. WHEN a Client sends any request, THE DAL_API SHALL require the API_Key to be present in the `X-API-Key` HTTP request header.
3. IF the `X-API-Key` header is missing, THEN THE DAL_API SHALL return HTTP 401 with a descriptive error message without processing the request.
4. IF the `X-API-Key` header contains a value that does not match any configured API_Key, THEN THE DAL_API SHALL return HTTP 401 with a descriptive error message without processing the request.
5. THE DAL_API SHALL exempt the `GET /health` endpoint from API_Key authentication.

---

### Requirement 3: Create Record

**User Story:** As a Client, I want to create a new record in a table, so that I can persist new data.

#### Acceptance Criteria

1. WHEN a Client sends `POST /{resource}` with a valid Request_Body, THE DAL_API SHALL insert a new Record into the corresponding table and return HTTP 201 with the created Record in the Response_Body.
2. IF the Request_Body is not valid JSON, THEN THE DAL_API SHALL return HTTP 400 with a descriptive error message.
3. IF the insert operation violates a database constraint (e.g., duplicate key, foreign key), THEN THE DAL_API SHALL return HTTP 409 with a descriptive error message.
4. IF the specified Resource does not correspond to an existing table, THEN THE DAL_API SHALL return HTTP 404 with a descriptive error message.

---

### Requirement 4: Read Single Record

**User Story:** As a Client, I want to retrieve a single record by its primary key, so that I can view specific data.

#### Acceptance Criteria

1. WHEN a Client sends `GET /{resource}/{id}`, THE DAL_API SHALL query the table for the Record where the `id` column matches the provided value and return HTTP 200 with the Record in the Response_Body.
2. IF no Record with the given Primary_Key exists, THEN THE DAL_API SHALL return HTTP 404 with a descriptive error message.
3. IF the specified Resource does not correspond to an existing table, THEN THE DAL_API SHALL return HTTP 404 with a descriptive error message.

---

### Requirement 5: Read Multiple Records

**User Story:** As a Client, I want to list records from a table with optional filtering, so that I can query and browse data.

#### Acceptance Criteria

1. WHEN a Client sends `GET /{resource}`, THE DAL_API SHALL return HTTP 200 with a JSON array of all Records in the table.
2. WHEN a Client sends `GET /{resource}` with query parameters, THE DAL_API SHALL apply those parameters as equality Filters on the query and return only matching Records.
3. WHEN no Records match the applied Filters, THE DAL_API SHALL return HTTP 200 with an empty JSON array.
4. THE DAL_API SHALL support a `limit` query parameter to restrict the number of Records returned, with a default maximum of 100 Records per request.
5. THE DAL_API SHALL support an `offset` query parameter to enable pagination of results.
6. IF the specified Resource does not correspond to an existing table, THEN THE DAL_API SHALL return HTTP 404 with a descriptive error message.

---

### Requirement 6: Update Record

**User Story:** As a Client, I want to update an existing record by its primary key, so that I can modify stored data.

#### Acceptance Criteria

1. WHEN a Client sends `PUT /{resource}/{id}` with a valid Request_Body, THE DAL_API SHALL update the Record where the `id` column matches the provided value and return HTTP 200 with the updated Record in the Response_Body.
2. WHEN a Client sends `PATCH /{resource}/{id}` with a valid Request_Body, THE DAL_API SHALL apply a partial update to the Record where the `id` column matches the provided value and return HTTP 200 with the updated Record in the Response_Body.
3. IF no Record with the given Primary_Key exists, THEN THE DAL_API SHALL return HTTP 404 with a descriptive error message.
4. IF the Request_Body is not valid JSON, THEN THE DAL_API SHALL return HTTP 400 with a descriptive error message.
5. IF the update operation violates a database constraint, THEN THE DAL_API SHALL return HTTP 409 with a descriptive error message.

---

### Requirement 7: Delete Record

**User Story:** As a Client, I want to delete a record by its primary key, so that I can remove data that is no longer needed.

#### Acceptance Criteria

1. WHEN a Client sends `DELETE /{resource}/{id}`, THE DAL_API SHALL delete the Record where the `id` column matches the provided value and return HTTP 204 with no Response_Body.
2. IF no Record with the given Primary_Key exists, THEN THE DAL_API SHALL return HTTP 404 with a descriptive error message.
3. IF the delete operation violates a database constraint (e.g., foreign key reference), THEN THE DAL_API SHALL return HTTP 409 with a descriptive error message.

---

### Requirement 8: Error Handling and Logging

**User Story:** As an operator, I want all errors to be logged and returned in a consistent format, so that I can diagnose issues quickly.

#### Acceptance Criteria

1. THE DAL_API SHALL return all error responses as JSON objects with at least an `error` field containing a human-readable message.
2. WHEN an unhandled internal error occurs, THE DAL_API SHALL return HTTP 500 with a generic error message and log the full error details server-side.
3. THE DAL_API SHALL log each incoming request with its HTTP method, path, response status code, and latency.

---

### Requirement 9: Security

**User Story:** As an operator, I want the API to protect against common injection attacks, so that the database is not compromised.

#### Acceptance Criteria

1. THE DAL_API SHALL use parameterized queries or prepared statements for all database operations.
2. THE DAL_API SHALL validate that Resource names in URL path segments contain only alphanumeric characters and underscores before executing any query.
3. THE DAL_API SHALL validate that the `limit` and `offset` query parameters are non-negative integers before executing any query.
