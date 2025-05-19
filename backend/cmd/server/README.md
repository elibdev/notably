# Notably Server

A versioned database API with user authentication and API key management.

## 1. API Design

### Base URL

    http://<host>:<port>

All endpoints are rooted at /.  (E.g. default :8080.)

-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------

### Authentication / Configuration

The server requires a DynamoDB table name which is provided via the `DYNAMODB_TABLE_NAME` environment variable. Authentication is done via API keys that you receive upon registration or login.

    go run cmd/server/main.go [--addr :8080]

You may also point to a local DynamoDB emulator by setting DYNAMODB_ENDPOINT_URL.

-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------

### Endpoints

The API implements the following RESTful endpoints using Go 1.22's new pattern matching syntax:

#### 1. Authentication

```
POST /auth/register
Content-Type: application/json

{
  "username": "user123",
  "email": "user@example.com",
  "password": "securepassword"
}
```
Registers a new user account. Returns user info and an API key (HTTP 201).

Response:
```json
{
  "id": "user123",
  "username": "user123",
  "email": "user@example.com",
  "apiKey": "nb_a1b2c3d4e5f6g7h8i9j0..."
}
```

```
POST /auth/login
Content-Type: application/json

{
  "username": "user123",
  "password": "securepassword"
}
```
Logs in a user. Returns user info and a new API key (HTTP 200).

```
GET /auth/keys
Authorization: Bearer nb_your_api_key_here
```
Lists all API keys for the authenticated user.

```
POST /auth/keys
Authorization: Bearer nb_your_api_key_here
Content-Type: application/json

{
  "name": "My API Key",
  "duration": 7776000
}
```
Creates a new API key. Duration is in seconds (default: 90 days).

```
DELETE /auth/keys/{id}
Authorization: Bearer nb_your_api_key_here
```
Revokes an API key.

#### 2. Tables Management

```
GET /tables
Authorization: Bearer nb_your_api_key_here
```
Lists all tables for the authenticated user.

Response (HTTP 200):
```json
{
  "tables": [
    {
      "name": "table1",
      "createdAt": "2023-08-21T12:34:56Z"
    }
  ]
}
```

```
POST /tables
Authorization: Bearer nb_your_api_key_here
Content-Type: application/json

{
  "name": "my-table"
}
```
Creates a new table. Returns the created table info (HTTP 201).

#### 3. Row Operations

```
GET /tables/{table}/rows
Authorization: Bearer nb_your_api_key_here
```
Lists all rows in a table.

Response (HTTP 200):
```json
{
  "rows": [
    {
      "id": "row1",
      "timestamp": "2023-08-21T12:34:56Z",
      "values": { "key1": "value1", "key2": "value2" }
    }
  ]
}
```

```
GET /tables/{table}/rows/{id}
Authorization: Bearer nb_your_api_key_here
```
Gets a specific row by ID.

Response (HTTP 200):
```json
{
  "id": "row1",
  "timestamp": "2023-08-21T12:34:56Z",
  "values": { "key1": "value1", "key2": "value2" }
}
```

```
POST /tables/{table}/rows
Authorization: Bearer nb_your_api_key_here
Content-Type: application/json

{
  "id": "row1",
  "values": { "key1": "value1", "key2": "value2" }
}
```
Creates a new row. Returns the created row (HTTP 201).

```
PUT /tables/{table}/rows/{id}
Authorization: Bearer nb_your_api_key_here
Content-Type: application/json

{
  "values": { "key1": "updated", "key2": "updated" }
}
```
Updates an existing row. Returns the updated row (HTTP 200).

```
DELETE /tables/{table}/rows/{id}
Authorization: Bearer nb_your_api_key_here
```
Deletes a row (tombstone). Returns HTTP 204 No Content on success.

#### 4. Table Snapshot and History

```
GET /tables/{table}/snapshot?at=<RFC3339>
Authorization: Bearer nb_your_api_key_here
```
Gets a snapshot of the table at a specific time. If `at` is not provided, uses the current time.

Response (HTTP 200):
```json
{
  "rows": [
    {
      "id": "row1",
      "timestamp": "2023-08-21T12:34:56Z",
      "values": { "key1": "value1", "key2": "value2" }
    }
  ]
}
```

```
GET /tables/{table}/history?start=<RFC3339>&end=<RFC3339>
Authorization: Bearer nb_your_api_key_here
```
Gets the history of all row events within a time range.

Response (HTTP 200):
```json
{
  "events": [
    {
      "id": "row1",
      "timestamp": "2023-08-21T12:34:56Z",
      "values": { "key1": "value1", "key2": "value2" }
    }
  ]
}
```

-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------

## 2. Project Structure

The project has been reorganized into a cleaner structure:

```
notably/
  ├── cmd/                # Command-line applications
  │   └── server/         # Server CLI
  ├── internal/           # Private packages
  │   ├── db/             # Database interfaces and implementations
  │   └── dynamo/         # AWS DynamoDB client
  └── pkg/                # Public packages
      ├── auth/           # Authentication and user management
      └── server/         # HTTP server and API implementation
```

## 3. Summary of Implementation

    * The server is implemented using the standard `net/http` package with Go 1.22's new routing features.
    * User authentication with username/password and API key management
    * Go 1.22's pattern matching syntax for routing, simplifying the code and making it more maintainable
    * Wildcards like `/tables/{table}/rows` and `/tables/{table}/rows/{id}` to capture path parameters
    * HTTP method matching (`GET`, `POST`, `PUT`, `DELETE`) to restrict routes to specific methods
    * Request parameters accessed via the new `PathValue()` method on the `http.Request` object
    * DynamoDB integration for persistent storage via the internal packages
    * Uniform JSON error responses (`{"error":"..."}`) with appropriate HTTP status codes
    * RFC3339 timestamps for query parameters and data versioning
    * DynamoDB table name configured via the `DYNAMODB_TABLE_NAME` environment variable
    * Secure password hashing and API key management
    * Clean separation of concerns with modular package structure

You can build and run the server like so:

    # Set the required environment variable
    export DYNAMODB_TABLE_NAME=NotablyFacts
    
    # Optionally set a custom DynamoDB endpoint for local testing
    export DYNAMODB_ENDPOINT_URL=http://localhost:8000
    
    # Run the server
    go run cmd/server/main.go --addr :8080

Then interact with it via curl or any HTTP client using the design above.

-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------

## 4. Authentication Flow

1. Register a user account:
   ```
   curl -X POST http://localhost:8080/auth/register -d '{"username":"myuser","email":"user@example.com","password":"securepass"}'
   ```

2. Use the returned API key in subsequent requests:
   ```
   curl -X GET http://localhost:8080/tables -H "Authorization: Bearer nb_your_api_key_here"
   ```

3. Create additional API keys if needed:
   ```
   curl -X POST http://localhost:8080/auth/keys -H "Authorization: Bearer nb_your_api_key_here" -d '{"name":"My Development Key"}'
   ```

API keys are required for all endpoints except for registration and login.

-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------

## 5. Testing

A comprehensive test suite is available testing all endpoints and various scenarios. Run the tests with:

```
go test -v ./...
```

Feel free to review the code and let me know if you'd like any tweaks!
