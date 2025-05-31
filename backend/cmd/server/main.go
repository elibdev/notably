package main

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"github.com/elibdev/notably/internal/api"
	internalConfig "github.com/elibdev/notably/internal/config"
	"github.com/elibdev/notably/internal/repository"
)

func main() {
	// Load configuration
	cfg, err := internalConfig.LoadConfig()
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

func setupAWSConfig(cfg *internalConfig.Config) (aws.Config, error) {
	ctx := context.Background()

	if cfg.Database.EndpointURL != "" {
		// Local DynamoDB
		customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			if service == dynamodb.ServiceID {
				return aws.Endpoint{
					URL:               cfg.Database.EndpointURL,
					HostnameImmutable: true,
				}, nil
			}
			return aws.Endpoint{}, fmt.Errorf("unknown endpoint requested")
		})

		return config.LoadDefaultConfig(ctx,
			config.WithRegion(cfg.Database.Region),
			config.WithEndpointResolverWithOptions(customResolver),
		)
	}

	// Production AWS
	return config.LoadDefaultConfig(ctx, config.WithRegion(cfg.Database.Region))
}
