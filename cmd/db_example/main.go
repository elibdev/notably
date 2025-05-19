package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"github.com/elibdev/notably/db"
)

func main() {
	// For a proper application, these would be configured via env vars or flags
	tableName := "NotablyFacts"
	userID := "example-user"

	// Initialize context
	ctx := context.Background()

	// Set up store with AWS config from environment
	store, err := setupStore(ctx, tableName, userID)
	if err != nil {
		log.Fatalf("Failed to set up store: %v", err)
	}

	// Create the DynamoDB table
	fmt.Println("Creating table...")
	if err := store.CreateTable(ctx); err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}
	fmt.Println("Table created successfully!")

	// Example workflow: tracking a user profile
	exampleUserProfileWorkflow(ctx, store)

	// Example workflow: tracking a product inventory
	exampleInventoryWorkflow(ctx, store)

	fmt.Println("\nExample completed successfully!")
}

// setupStore configures and returns a DynamoDB store
func setupStore(ctx context.Context, tableName, userID string) (db.Store, error) {
	// Check if we're using a local DynamoDB endpoint
	if ep := os.Getenv("DYNAMODB_ENDPOINT_URL"); ep != "" {
		fmt.Printf("Using local DynamoDB endpoint: %s\n", ep)
		
		// Load AWS configuration with the custom endpoint
		opts := []func(*config.LoadOptions) error{}
		resolver := aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
			if service == dynamodb.ServiceID {
				return aws.Endpoint{URL: ep, SigningRegion: "us-west-2"}, nil
			}
			return aws.Endpoint{}, &aws.EndpointNotFoundError{}
		})
		opts = append(opts, config.WithEndpointResolver(resolver))
		
		// Set some default AWS credentials if using local DynamoDB
		if os.Getenv("AWS_ACCESS_KEY_ID") == "" {
			os.Setenv("AWS_ACCESS_KEY_ID", "dummy")
			os.Setenv("AWS_SECRET_ACCESS_KEY", "dummy")
			os.Setenv("AWS_REGION", "us-west-2")
		}
		
		cfg, err := config.LoadDefaultConfig(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("loading AWS config: %w", err)
		}
		
		// Create and return the store
		return db.NewDynamoDBStore(&db.Config{
			TableName:    tableName,
			UserID:       userID,
			DynamoClient: dynamodb.NewFromConfig(cfg),
		}), nil
	}
	
	// For real AWS usage, get configuration from the environment
	return db.NewDynamoDBStoreFromEnv(ctx, tableName, userID)
}

// exampleUserProfileWorkflow demonstrates managing user profile data
func exampleUserProfileWorkflow(ctx context.Context, store db.Store) {
	fmt.Println("\n--- User Profile Workflow Example ---")
	
	// 1. Add initial user profile facts
	fmt.Println("Adding user profile facts...")
	
	// Current time as a baseline for all operations
	now := time.Now().UTC()
	
	// Add basic profile information
	profileFacts := []*db.Fact{
		{
			ID:        "profile-1",
			Timestamp: now,
			Namespace: "user-profile",
			FieldName: "displayName",
			DataType:  db.DataTypeString,
			Value:     "John Doe",
			UserID:    "example-user",
		},
		{
			ID:        "profile-2",
			Timestamp: now,
			Namespace: "user-profile",
			FieldName: "email",
			DataType:  db.DataTypeString,
			Value:     "johndoe@example.com",
			UserID:    "example-user",
		},
		{
			ID:        "profile-3",
			Timestamp: now,
			Namespace: "user-profile",
			FieldName: "age",
			DataType:  db.DataTypeNumber,
			Value:     "30",
			UserID:    "example-user",
		},
		{
			ID:        "profile-4",
			Timestamp: now,
			Namespace: "user-settings",
			FieldName: "theme",
			DataType:  db.DataTypeString,
			Value:     "dark",
			UserID:    "example-user",
		},
	}
	
	for _, fact := range profileFacts {
		if err := store.PutFact(ctx, fact); err != nil {
			log.Fatalf("Failed to add fact: %v", err)
		}
	}
	
	// 2. Query all profile fields
	fmt.Println("Querying profile namespace...")
	
	startTime := now.Add(-time.Minute)
	endTime := now.Add(time.Minute)
	
	profileResult, err := store.QueryByNamespace(ctx, "user-profile", db.QueryOptions{
		StartTime:     &startTime,
		EndTime:       &endTime,
		SortAscending: true,
	})
	
	if err != nil {
		log.Fatalf("Failed to query profile: %v", err)
	}
	
	fmt.Printf("Found %d profile facts:\n", len(profileResult.Facts))
	for i, fact := range profileResult.Facts {
		fmt.Printf("  %d. %s = %s (%s)\n", i+1, fact.FieldName, fact.Value, fact.DataType)
	}
	
	// 3. Update email
	fmt.Println("\nUpdating email address...")
	
	updatedEmail := &db.Fact{
		ID:        "profile-2",
		Timestamp: now.Add(time.Second),
		Namespace: "user-profile",
		FieldName: "email",
		DataType:  db.DataTypeString,
		Value:     "john.doe.updated@example.com",
		UserID:    "example-user",
	}
	
	if err := store.PutFact(ctx, updatedEmail); err != nil {
		log.Fatalf("Failed to update email: %v", err)
	}
	
	// 4. Get latest fact directly
	fmt.Println("Getting latest email fact...")
	emailFact, err := store.GetFact(ctx, "profile-2")
	if err != nil {
		log.Fatalf("Failed to get email fact: %v", err)
	}
	
	fmt.Printf("Latest email: %s\n", emailFact.Value)
	
	// 5. Query email history
	fmt.Println("\nQuerying email history...")
	
	emailHistory, err := store.QueryByField(ctx, "user-profile", "email", db.QueryOptions{
		StartTime:     &startTime,
		EndTime:       &now.Add(time.Minute * 5),
		SortAscending: false, // newest first
	})
	
	if err != nil {
		log.Fatalf("Failed to query email history: %v", err)
	}
	
	fmt.Printf("Email change history (newest first):\n")
	for i, fact := range emailHistory.Facts {
		fmt.Printf("  %d. %s at %s\n", i+1, fact.Value, fact.Timestamp.Format(time.RFC3339))
	}
	
	// 6. Take a snapshot at current time
	fmt.Println("\nTaking profile snapshot...")
	
	snapshotTime := now.Add(2 * time.Second) // After the email update
	snapshot, err := store.GetSnapshotAtTime(ctx, "user-profile", snapshotTime)
	
	if err != nil {
		log.Fatalf("Failed to get snapshot: %v", err)
	}
	
	fmt.Printf("Profile snapshot at %s:\n", snapshotTime.Format(time.RFC3339))
	for key, fact := range snapshot {
		fmt.Printf("  %s = %s\n", fact.FieldName, fact.Value)
	}
	
	// 7. Delete a fact
	fmt.Println("\nDeleting age fact...")
	
	if err := store.DeleteFact(ctx, "profile-3"); err != nil {
		log.Fatalf("Failed to delete age fact: %v", err)
	}
	
	// Verify deletion with a new snapshot
	deletionTime := now.Add(3 * time.Second) // After deletion
	snapshotAfterDelete, err := store.GetSnapshotAtTime(ctx, "user-profile", deletionTime)
	
	if err != nil {
		log.Fatalf("Failed to get snapshot after deletion: %v", err)
	}
	
	fmt.Printf("Profile snapshot after deletion at %s:\n", deletionTime.Format(time.RFC3339))
	for key, fact := range snapshotAfterDelete {
		fmt.Printf("  %s = %s\n", fact.FieldName, fact.Value)
	}
	
	fmt.Println("User profile workflow completed successfully!")
}

// exampleInventoryWorkflow demonstrates inventory tracking
func exampleInventoryWorkflow(ctx context.Context, store db.Store) {
	fmt.Println("\n--- Inventory Tracking Workflow Example ---")
	
	// 1. Add initial inventory items
	fmt.Println("Adding inventory items...")
	
	// Current time as a baseline
	now := time.Now().UTC()
	
	inventory := []*db.Fact{
		{
			ID:        "item-1",
			Timestamp: now,
			Namespace: "inventory",
			FieldName: "product-1001",
			DataType:  db.DataTypeJSON,
			Value:     `{"name":"Widget A","count":100,"price":9.99}`,
			UserID:    "example-user",
		},
		{
			ID:        "item-2",
			Timestamp: now,
			Namespace: "inventory",
			FieldName: "product-1002",
			DataType:  db.DataTypeJSON,
			Value:     `{"name":"Widget B","count":50,"price":19.99}`,
			UserID:    "example-user",
		},
		{
			ID:        "item-3",
			Timestamp: now,
			Namespace: "inventory",
			FieldName: "product-1003",
			DataType:  db.DataTypeJSON,
			Value:     `{"name":"Widget C","count":25,"price":29.99}`,
			UserID:    "example-user",
		},
	}
	
	for _, fact := range inventory {
		if err := store.PutFact(ctx, fact); err != nil {
			log.Fatalf("Failed to add inventory item: %v", err)
		}
	}
	
	// 2. Simulate inventory changes over time
	fmt.Println("Updating inventory quantities...")
	
	// First update - 5 minutes later
	time1 := now.Add(5 * time.Minute)
	update1 := &db.Fact{
		ID:        "item-1",
		Timestamp: time1,
		Namespace: "inventory",
		FieldName: "product-1001",
		DataType:  db.DataTypeJSON,
		Value:     `{"name":"Widget A","count":95,"price":9.99}`, // 5 sold
		UserID:    "example-user",
	}
	
	if err := store.PutFact(ctx, update1); err != nil {
		log.Fatalf("Failed to update inventory: %v", err)
	}
	
	// Second update - 10 minutes later
	time2 := now.Add(10 * time.Minute)
	update2 := &db.Fact{
		ID:        "item-1",
		Timestamp: time2,
		Namespace: "inventory",
		FieldName: "product-1001",
		DataType:  db.DataTypeJSON,
		Value:     `{"name":"Widget A","count":80,"price":9.99}`, // 15 more sold
		UserID:    "example-user",
	}
	
	if err := store.PutFact(ctx, update2); err != nil {
		log.Fatalf("Failed to update inventory: %v", err)
	}
	
	// Price change update - 15 minutes later
	time3 := now.Add(15 * time.Minute)
	update3 := &db.Fact{
		ID:        "item-1",
		Timestamp: time3,
		Namespace: "inventory",
		FieldName: "product-1001",
		DataType:  db.DataTypeJSON,
		Value:     `{"name":"Widget A","count":80,"price":7.99}`, // Price reduced
		UserID:    "example-user",
	}
	
	if err := store.PutFact(ctx, update3); err != nil {
		log.Fatalf("Failed to update inventory: %v", err)
	}
	
	// 3. Query product history
	fmt.Println("\nQuerying product history...")
	
	productHistory, err := store.QueryByField(ctx, "inventory", "product-1001", db.QueryOptions{
		StartTime:     &now,
		EndTime:       &time3.Add(time.Minute),
		SortAscending: true, // oldest first
	})
	
	if err != nil {
		log.Fatalf("Failed to query product history: %v", err)
	}
	
	fmt.Printf("Product-1001 history (oldest first):\n")
	for i, fact := range productHistory.Facts {
		fmt.Printf("  %d. %s at %s\n", i+1, fact.Value, fact.Timestamp.Format(time.RFC3339))
	}
	
	// 4. Get inventory snapshots at different times
	fmt.Println("\nViewing inventory snapshots at different times...")
	
	// Check inventory at start
	snapshotStart, err := store.GetSnapshotAtTime(ctx, "inventory", now)
	if err != nil {
		log.Fatalf("Failed to get initial inventory snapshot: %v", err)
	}
	
	fmt.Printf("Initial inventory (t=%s):\n", now.Format(time.RFC3339))
	printInventorySnapshot(snapshotStart)
	
	// Check inventory after first update
	snapshotTime1, err := store.GetSnapshotAtTime(ctx, "inventory", time1.Add(time.Second))
	if err != nil {
		log.Fatalf("Failed to get inventory snapshot at time1: %v", err)
	}
	
	fmt.Printf("\nInventory after first update (t=%s):\n", time1.Format(time.RFC3339))
	printInventorySnapshot(snapshotTime1)
	
	// Check inventory at end (after price change)
	snapshotTime3, err := store.GetSnapshotAtTime(ctx, "inventory", time3.Add(time.Second))
	if err != nil {
		log.Fatalf("Failed to get final inventory snapshot: %v", err)
	}
	
	fmt.Printf("\nFinal inventory (t=%s):\n", time3.Format(time.RFC3339))
	printInventorySnapshot(snapshotTime3)
	
	fmt.Println("Inventory workflow completed successfully!")
}

// printInventorySnapshot formats and prints inventory data
func printInventorySnapshot(snapshot map[string]db.Fact) {
	for _, fact := range snapshot {
		fmt.Printf("  %s: %s\n", fact.FieldName, fact.Value)
	}
}