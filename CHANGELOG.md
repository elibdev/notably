
# Changelog

# 2025-05-18

# Change API to Per-User Tables Abstraction

I’ve overhauled the HTTP server so that it now exposes a true “per-user table” interface—no more “facts” endpoints, no more table/user flags at startup. All data lives in one DynamoDB table (name taken from
$DYNAMODB_TABLE_NAME), and each request is scoped to the user identified by the X-User-ID header.

### What changed

    * **Removed** the `--table` and `--user` flags.
      The DynamoDB table name is now read from `DYNAMODB_TABLE_NAME`; the user is picked up on each request from the `X-User-ID` header.
    * **Startup** still ensures the underlying DynamoDB table exists (using an empty user context).
    * **New REST endpoints** under `/tables`:

      ┌────────┬───────────────────────────────────────┬──────────────────────────────────────────────────────────┐
      │ Method │ Path                                  │ Description                                              │
      ├────────┼───────────────────────────────────────┼──────────────────────────────────────────────────────────┤
      │ POST   │ /tables                               │ Create a new table for the user                          │
      ├────────┼───────────────────────────────────────┼──────────────────────────────────────────────────────────┤
      │ GET    │ /tables                               │ List all tables for the user                             │
      ├────────┼───────────────────────────────────────┼──────────────────────────────────────────────────────────┤
      │ POST   │ /tables/{table}/rows                  │ Insert (or upsert) a row in the named table              │
      ├────────┼───────────────────────────────────────┼──────────────────────────────────────────────────────────┤
      │ GET    │ /tables/{table}/rows                  │ List current rows of the table                           │
      ├────────┼───────────────────────────────────────┼──────────────────────────────────────────────────────────┤
      │ GET    │ /tables/{table}/rows/{rowID}          │ Get the latest version of a single row                   │
      ├────────┼───────────────────────────────────────┼──────────────────────────────────────────────────────────┤
      │ PUT    │ /tables/{table}/rows/{rowID}          │ Update (append a new version of) a single row            │
      ├────────┼───────────────────────────────────────┼──────────────────────────────────────────────────────────┤
      │ DELETE │ /tables/{table}/rows/{rowID}          │ “Tombstone” a row (marks it deleted at the current time) │
      ├────────┼───────────────────────────────────────┼──────────────────────────────────────────────────────────┤
      │ GET    │ /tables/{table}/snapshot?at=…         │ Snapshot of the table at a given RFC3339 timestamp (at)  │
      ├────────┼───────────────────────────────────────┼──────────────────────────────────────────────────────────┤
      │ GET    │ /tables/{table}/history?start=…&end=… │ Full row-event history in a time range                   │
      └────────┴───────────────────────────────────────┴──────────────────────────────────────────────────────────┘
    * **Data model**:


        * Table-creation events are recorded as “facts” under the user’s namespace.

        * Row events live under namespace `"{user}/{table}"` with one JSON-typed fact per row-version.
    * **Versioning & history**:


        * Every insert/update/delete is a new fact (with a timestamped ID).

        * The snapshot endpoint replays up-to-time events.

        * The history endpoint streams all events in the specified range.

You can review the full implementation in cmd/server/main.go.

-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------

Next steps (as a user of the API):

    export DYNAMODB_TABLE_NAME=NotablyFacts
    # (optional) export DYNAMODB_ENDPOINT_URL=http://localhost:8000
    go run cmd/server/main.go --addr :8080

Then talk to it with your HTTP client, remembering to set X-User-ID: yourUserID on each request.

# Scaffold time based database idea

I’ve scaffolded a minimal Go‐based “notably” DynamoDB fact store (module path github.com/elibdev/notably) exactly as laid out in the README:

    1: I want to build a database on Dynamo DB that will be a flexible time versioned database …
    2: The basic building block is a tuple … (id, timestamp, namespace/fieldName, dataType, value).
    4: I want to build different indexes for common access patterns for looking things up based on field or time …
    11: Everything should be partitioned by user, and every user will get their own namespace …

README.md (/Users/eli/projects/notably/README.md)README.md (/Users/eli/projects/notably/README.md)README.md (/Users/eli/projects/notably/README.md)

-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------

## What was added

### 1. Go module bootstrap (go.mod / go.sum)

A go.mod declaring module github.com/elibdev/notably and pulling in AWS SDK v2 dependencies.

### 2. dynamo package (dynamo/client.go)

Implements the core “fact” storage layer:

    * **Fact struct** capturing `(ID, Timestamp, Namespace, FieldName, DataType, Value)`.
    * **Client** bound to a `tableName` and `userID`, with methods to:
        * **CreateTable**: on-demand table creation with a GSI (`FieldIndex`) for per-field history queries.

        * **PutFact**: writes a versioned fact (`SK = timestamp#id`, composite `FieldKey = user#namespace#field`).

        * **QueryByField**: GSI-backed time-range queries for a single namespace/field.

        * **QueryByTimeRange**: primary-key time-range scan across all fields for a user.
    * Internal helper to unmarshal DynamoDB items back into high-level `Fact` values.

### 3. Example client (cmd/example/main.go)

A small CLI that demonstrates:

    1. Bootstrapping the table (`CreateTable`)
    2. Writing a couple of sample facts
    3. Querying the “profile/name” history and printing out the results

-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------

## Getting started

    # (1) Build the example
    go build ./cmd/example

    # (2) Run it (requires AWS credentials / permissions)
    ./example

Everything is formatted with gofmt and managed via Go modules. Let me know if you’d like any tweaks or additional access patterns!

# Add api package

I’ve added a first‐class Go “api” package on top of the low-level Dynamo client and updated the README with AWS vs. local-emulator setup instructions, plus a Go code snippet showing the new API in action.

-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------

### 1. New api package

File: api/api.go

    * **FactStore**: wraps your `dynamo.Client` and loads AWS config (credentials/region) from the environment.  If `DYNAMODB_ENDPOINT_URL` is set, it will automatically point at a local DynamoDB emulator.
    * **CreateSchema**, **AddFact**, **UpdateFact**, **DeleteFact**: unified methods for table-creation, inserts/updates, and “soft” deletes (tombstones).
    * **QueryByField**, **QueryByTimeRange**: direct access patterns on your facts.
    * **SnapshotAt**: time-travel “view” of all namespace/fields as of a given timestamp.

    // Example usage (see api/api.go for docs):
    ctx := context.Background()
    store, err := api.NewFactStore(ctx, "NotablyFacts", "user123")
    …
    store.CreateSchema(ctx)
    store.AddFact(ctx, Fact{ … })
    store.UpdateFact(ctx, Fact{ … })
    store.DeleteFact(ctx, "profile", "name", "1", time.Now())
    history, _ := store.QueryByField(ctx, "profile", "name", start, end)
    snapshot, _ := store.SnapshotAt(ctx, time.Now())

-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------

### 2. Example client now supports a local emulator

Modified cmd/example/main.go so you can point it at DynamoDB Local via the DYNAMODB_ENDPOINT_URL env var.

    export DYNAMODB_ENDPOINT_URL=http://localhost:8000
    go run ./cmd/example

-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------

### 3. README.md updated

File: README.md

    * **Environment Setup**: instructions for both real AWS and a local DynamoDB emulator (via Docker).
    * **Programmatic API (Go)**: shows how to call into the new `api` package to create schemas, add/update/delete facts, query history, and take snapshots.

-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------

Everything is go-fmt’ed and compiles cleanly. Let me know if you’d like any tweaks!
