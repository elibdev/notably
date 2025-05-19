Below is a brief user-facing design for the new HTTP/JSON API, followed by a summary of the implementation you can review. The code has already been added under cmd/server/main.go.

-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------

## 1. API Design

### Base URL

    http://<host>:<port>

All endpoints are rooted at /.  (E.g. default :8080.)

-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------

### Authentication / Configuration

The server requires a DynamoDB table name which is provided via the `DYNAMODB_TABLE_NAME` environment variable. Authentication is done via the `X-User-ID` HTTP header which must be included with each request.

    go run cmd/server/main.go [--addr :8080]

You may also point to a local DynamoDB emulator by setting DYNAMODB_ENDPOINT_URL.

-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------

### Endpoints

The API implements the following RESTful endpoints using Go 1.22's new pattern matching syntax:

#### 1. Tables Management

```
GET /tables
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
Content-Type: application/json

{
  "name": "my-table"
}
```
Creates a new table. Returns the created table info (HTTP 201).

#### 2. Row Operations

```
GET /tables/{table}/rows
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
Content-Type: application/json

{
  "id": "row1",
  "values": { "key1": "value1", "key2": "value2" }
}
```
Creates a new row. Returns the created row (HTTP 201).

```
PUT /tables/{table}/rows/{id}
Content-Type: application/json

{
  "values": { "key1": "updated", "key2": "updated" }
}
```
Updates an existing row. Returns the updated row (HTTP 200).

```
DELETE /tables/{table}/rows/{id}
```
Deletes a row (tombstone). Returns HTTP 204 No Content on success.

#### 3. Table Snapshot and History

```
GET /tables/{table}/snapshot?at=<RFC3339>
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

## 2. Summary of Implementation

    * The server is implemented in `cmd/server/main.go` using the standard `net/http` package with Go 1.22's new routing features.
    * The server uses Go 1.22's pattern matching syntax for routing, simplifying the code and making it more maintainable.
    * Wildcards like `/tables/{table}/rows` and `/tables/{table}/rows/{id}` are used to capture path parameters.
    * HTTP method matching (`GET`, `POST`, `PUT`, `DELETE`) is used to restrict routes to specific methods.
    * Request parameters can be accessed via the new `PathValue()` method on the `http.Request` object.
    * The server wraps the tested adapter (`db.NewStoreAdapter(db.CreateStoreFromClient(client))`).
    * Error responses are uniform JSON (`{"error":"..."}`) with appropriate HTTP status codes.
    * Authentication is handled via the `X-User-ID` header.
    * The code parses RFC3339 timestamps for query parameters.
    * The DynamoDB table name is configured via the `DYNAMODB_TABLE_NAME` environment variable.
    * No new external dependencies were added.

You can build and run the server like so:

    # Set the required environment variable
    export DYNAMODB_TABLE_NAME=NotablyFacts
    
    # Optionally set a custom DynamoDB endpoint for local testing
    export DYNAMODB_ENDPOINT_URL=http://localhost:8000
    
    # Run the server
    go run cmd/server/main.go --addr :8080

Then interact with it via curl or any HTTP client using the design above.

-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------

## 3. Testing

A comprehensive test suite is available in `cmd/server/tests/api_test.go`, testing all endpoints and various scenarios. Run the tests with:

```
cd notably/cmd/server/tests
go test -v
```

Feel free to review the code in cmd/server/main.go and let me know if you'd like any tweaks!
