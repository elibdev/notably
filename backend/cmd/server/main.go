package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/sirupsen/logrus"

	"github.com/elibdev/notably/internal/api"
	internalConfig "github.com/elibdev/notably/internal/config"
	"github.com/elibdev/notably/internal/repository"
)

func main() {
	// Load configuration
	cfg, err := internalConfig.LoadConfig()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to load config")
	}

	// Setup structured logging
	setupLogging(cfg)

	logrus.WithFields(logrus.Fields{
		"environment": os.Getenv("TIMEDB_ENV"),
		"log_level":   cfg.Logging.Level,
		"log_format":  cfg.Logging.Format,
	}).Info("Starting TimeDB server")

	// Setup AWS DynamoDB client
	logrus.Info("Setting up AWS DynamoDB client")
	awsCfg, err := setupAWSConfig(cfg)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to setup AWS config")
	}

	dynamoClient := dynamodb.NewFromConfig(awsCfg)
	logrus.WithFields(logrus.Fields{
		"endpoint": cfg.Database.EndpointURL,
		"region":   cfg.Database.Region,
		"table":    cfg.Database.TableName,
	}).Info("DynamoDB client configured")

	// Create user manager
	logrus.Info("Initializing user manager")
	userManager := repository.NewDynamoUserManager(dynamoClient, cfg.Database.TableName)

	// Create HTTP server
	logrus.Info("Creating HTTP server")
	server := api.NewServer(cfg, userManager)

	// Start server
	logrus.WithFields(logrus.Fields{
		"host": cfg.Server.Host,
		"port": cfg.Server.Port,
	}).Info("Starting TimeDB server")

	if err := server.Start(); err != nil {
		logrus.WithError(err).Fatal("Failed to start server")
	}
}

func setupLogging(cfg *internalConfig.Config) {
	// Set log level
	level, err := logrus.ParseLevel(cfg.Logging.Level)
	if err != nil {
		logrus.WithError(err).Warn("Invalid log level, defaulting to info")
		level = logrus.InfoLevel
	}
	logrus.SetLevel(level)

	// Set log format
	if cfg.Logging.Format == "json" {
		logrus.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		})
	} else {
		logrus.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		})
	}

	// Output to stdout
	logrus.SetOutput(os.Stdout)
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
