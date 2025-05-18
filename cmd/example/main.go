package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/elibdev/notably/dynamo"
)

func main() {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("unable to load AWS SDK config: %v", err)
	}

	tableName := "NotablyFacts"
	userID := "user123"
	client := dynamo.NewClient(cfg, tableName, userID)
	if err := client.CreateTable(ctx); err != nil {
		log.Fatalf("failed to create DynamoDB table: %v", err)
	}
	fmt.Println("table is ready")

	now := time.Now()
	f1 := dynamo.Fact{ID: "1", Timestamp: now.Add(-time.Hour), Namespace: "profile", FieldName: "name", DataType: "string", Value: "Alice"}
	f2 := dynamo.Fact{ID: "2", Timestamp: now, Namespace: "profile", FieldName: "name", DataType: "string", Value: "Alice Smith"}

	for _, f := range []dynamo.Fact{f1, f2} {
		if err := client.PutFact(ctx, f); err != nil {
			log.Fatalf("failed to put fact: %v", err)
		}
	}

	facts, err := client.QueryByField(ctx, "profile", "name", now.Add(-2*time.Hour), now.Add(time.Minute))
	if err != nil {
		log.Fatalf("failed to query by field: %v", err)
	}

	fmt.Println("name field history:")
	for _, f := range facts {
		fmt.Printf("- ID=%s Timestamp=%s Value=%v\n", f.ID, f.Timestamp.Format(time.RFC3339), f.Value)
	}
}
