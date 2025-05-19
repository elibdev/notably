Below is a brief user-facing design for the new HTTP/JSON API, followed by a summary of the implementation you can review. The code has already been added under cmd/server/main.go.

-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------

## 1. API Design

### Base URL

    http://<host>:<port>

All endpoints are rooted at /.  (E.g. default :8080.)

-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------

### Authentication / Configuration

For simplicity this initial version is single-tenant: you supply a fixed DynamoDB table and user at startup via flags.  (Later you could extend it to pick up a User-ID header, etc.)

    go run cmd/server/main.go --table MyTableName --user myUserID [--addr :8080]

You may also point to a local DynamoDB emulator by setting DYNAMODB_ENDPOINT_URL.

-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------

### Endpoints

#### 1. Create/Update a fact

    POST /facts
    Content-Type: application/json

    {
      "id":        "fact-123",
      "timestamp": "2023-08-21T12:34:56Z",
      "namespace": "user-profile",
      "fieldName": "displayName",
      "dataType":  "string",
      "value":     "Alice"
    }

Returns the same fact back (HTTP 201).

-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------

#### 2. Get the latest version of a fact

    GET /facts/{id}

Example response (HTTP 200):

    {
      "id":        "fact-123",
      "timestamp": "2023-08-21T12:34:56Z",
      "namespace": "user-profile",
      "fieldName": "displayName",
      "dataType":  "string",
      "value":     "Alice"
    }

## If not found: HTTP 404 with body {"error":"..."}

#### 3. Delete (tombstone) a fact

    DELETE /facts/{id}

On success returns HTTP 204 No Content.

Internally this creates a “deleted” marker entry in DynamoDB with the current timestamp.

-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------

#### 4. Query by time range (all namespaces)

    GET /facts?start=<RFC3339>&end=<RFC3339>

Example:

    GET /facts?start=2023-08-20T00:00:00Z&end=2023-08-22T00:00:00Z

Response (HTTP 200):

    {
      "facts": [
        { "id":"fact-1", "timestamp":"2023-08-20T01:00:00Z", ... },
        { "id":"fact-2", "timestamp":"2023-08-21T05:00:00Z", ... }
      ]
    }

-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------

#### 5. Query by namespace

    GET /namespaces/{namespace}/facts?start=<RFC3339>&end=<RFC3339>

E.g.:

    GET /namespaces/user-profile/facts?start=2023-08-20T00:00:00Z&end=2023-08-22T00:00:00Z

Returns only facts in that namespace over the time range.

-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------

#### 6. Query by field (namespace + fieldName)

    GET /namespaces/{namespace}/fields/{fieldName}/facts?start=<RFC3339>&end=<RFC3339>

E.g.:

    GET /namespaces/user-profile/fields/displayName/facts?start=...&end=...

Returns the history (all versions) of that field.

-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------

#### 7. Snapshot at a point in time

    GET /snapshots?at=<RFC3339>

Response JSON is a nested object: namespace → fieldName → latest fact as of that time:

    {
      "user-profile": {
        "displayName": {
          "id":"fact-123",
          "timestamp":"2023-08-21T12:34:56Z",
          "namespace":"user-profile",
          "fieldName":"displayName",
          "dataType":"string",
          "value":"Alice"
        },
        "email": { ... }
      },
      "other-namespace": { ... }
    }

-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------

## 2. Summary of Implementation

    * **New file** `cmd/server/main.go` implementing the HTTP server using only the standard `net/http` package.
    * The server **wraps the tested adapter** (`db.NewStoreAdapter(db.CreateStoreFromClient(legacyClient))`), so it drives the same `Store` implementation you already have tests for.
    * All JSON request/response types are defined inline (`FactResponse`, `QueryResponse`), preserving the underlying `dynamo.Fact` shape.
    * Error responses are uniform JSON (`{"error":"..."}`) with appropriate HTTP status codes.
    * The code parses RFC3339 timestamps for `start`, `end`, and `at` parameters.
    * Flags `--table` and `--user` configure the DynamoDB table + user; you can override the DynamoDB endpoint via `DYNAMODB_ENDPOINT_URL` (for a local emulator).
    * No new external dependencies were added.

You can build and run the server like so:

    # (optionally start DynamoDB Local, set env vars)
    go run cmd/server/main.go --table NotablyFacts --user example-user --addr :8080

Then interact with it via curl or any HTTP client using the design above.

-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------

Feel free to review the code in cmd/server/main.go and let me know if you’d like any tweaks!
