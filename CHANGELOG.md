
# Changelog

# 2025-05-18

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

