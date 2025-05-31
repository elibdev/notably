package main

import (
	"log"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"github.com/elibdev/notably/internal/api"
	"github.com/elibdev/notably/internal/repository"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Setup AWS DynamoDB client
	awsCfg, err := setupAWSConfig(cfg)
	if err != nil {
		log.Fatalf("Failed to setup AWS config: %v", err)
	}

	dynamoClient := dynamodb.NewFromConfig(awsCfg)

	// Create user manager
	userManager := repository.NewDynamoUserManager(dynamoClient, cfg.Database.TableName)

	// Create HTTP server
	server := api.NewServer(cfg, userManager)

	// Start server
	log.Printf("Starting TimeDB server on %s:%d", cfg.Server.Host, cfg.Server.Port)
	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
