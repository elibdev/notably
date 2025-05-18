I want to build a database on Dynamo DB that will be a flexible time versioned database kind of similar to datomic or XTDB in its architecture. 
The basic building block is a tuple that represents the value of a single field at a given time, something like: (id, timestamp, namespace/fieldName, dataType, value).

I want to build different indexes for common access patterns for looking things up based on field or time, or maybe even on value for certain data types.

The idea is to build a versioned flexible database platform to be at the core of a personal information management system to keep track of your life, or any other thing.
Kind of inspired by notion or by spreadsheets.

This database platform will be at the core of this broader tool and will enable it to scale.

Everything should be partitioned by user, and every user will get their own namespace so their data is all in their own partition and they can make their own field types.

The main UI will be an app / web app that will essentially look like some tables like airtable or something where each data type is rendered in a useful way.

The table abstraction is something that will have to be built after, but it will be based on this fundamental database platform.

## Environment Setup

Notably supports both AWS DynamoDB and a local DynamoDB emulator for development.

### AWS DynamoDB

Ensure you have valid AWS credentials and region configured. For example:

```bash
export AWS_ACCESS_KEY_ID=YOUR_ACCESS_KEY_ID
export AWS_SECRET_ACCESS_KEY=YOUR_SECRET_ACCESS_KEY
export AWS_REGION=us-west-2
```

### Local DynamoDB Emulator

You can run DynamoDB Local via Docker:

```bash
docker run --name dynamodb-local -p 8000:8000 amazon/dynamodb-local
```

Point the SDK to your local emulator by setting:

```bash
export DYNAMODB_ENDPOINT_URL=http://localhost:8000
export AWS_REGION=us-west-2      # region is still required
export AWS_ACCESS_KEY_ID=foo     # use dummy credentials
export AWS_SECRET_ACCESS_KEY=bar
```

## Programmatic API (Go)

Use the `api` package for a convenient, unified API to manage schemas, facts, and time-travel queries:

```go
package main

import (
   "context"
   "fmt"
   "time"

   "github.com/elibdev/notably/api"
)

func main() {
   ctx := context.Background()
   store, err := api.NewFactStore(ctx, "NotablyFacts", "user123")
   if err != nil {
       panic(err)
   }

   // Create the table schema (if not already exists)
   if err := store.CreateSchema(ctx); err != nil {
       panic(err)
   }

   // Add a fact (initial value)
   fact := api.Fact{
       ID:        "1",
       Timestamp: time.Now().Add(-time.Hour),
       Namespace: "profile",
       FieldName: "name",
       DataType:  "string",
       Value:     "Alice",
   }
   if err := store.AddFact(ctx, fact); err != nil {
       panic(err)
   }

   // Update the same fact (new version)
   fact.Timestamp = time.Now()
   fact.Value = "Alice Smith"
   if err := store.UpdateFact(ctx, fact); err != nil {
       panic(err)
   }

   // Delete the fact (tombstone)
   if err := store.DeleteFact(ctx, "profile", "name", "1", time.Now()); err != nil {
       panic(err)
   }

   // Query by field history
   start := time.Now().Add(-2 * time.Hour)
   end := time.Now()
   history, err := store.QueryByField(ctx, "profile", "name", start, end)
   if err != nil {
       panic(err)
   }
   fmt.Println("Field history:", history)

   // Take a snapshot at a point in time
   snapshot, err := store.SnapshotAt(ctx, time.Now())
   if err != nil {
       panic(err)
   }
   fmt.Println("Snapshot:", snapshot)
}
```
