// @title           Notably API
// @version         1.0
// @description     # Notably API - Time-Series Data Management Platform
// @description
// @description     **Notably** is a powerful time-series database API that allows you to create dynamic data schemas, store entities with flexible fields, and query historical states of your data. Perfect for applications requiring audit trails, versioning, and time travel capabilities.
// @description
// @description     ## 🚀 Key Features
// @description
// @description     ### 📊 **Dynamic Table Management**
// @description     - Create tables with custom field schemas (string, int, float, bool, date, json, reference)
// @description     - Add/remove fields dynamically without downtime
// @description     - Full schema evolution tracking
// @description
// @description     ### 🗃️ **Flexible Entity Storage**
// @description     - Store entities with any combination of fields
// @description     - Real-time CRUD operations
// @description     - Soft delete with undelete capabilities
// @description     - Field-level operations (add, update, delete individual fields)
// @description
// @description     ### ⏰ **Time Travel & History**
// @description     - Query data as it existed at any point in time using `asOf` parameters
// @description     - Complete audit trail for all changes
// @description     - Field-level history tracking
// @description     - Entity and table history endpoints
// @description
// @description     ### 🔐 **Secure Authentication**
// @description     - JWT-based authentication
// @description     - User registration and login
// @description     - Per-user data isolation
// @description
// @description     ## 📖 Usage Examples
// @description
// @description     ### Basic Workflow
// @description     ```
// @description     1. Register/Login → Get JWT token
// @description     2. Create a table → Define your schema
// @description     3. Add entities → Store your data
// @description     4. Query & update → Manage your data
// @description     5. Time travel → View historical states
// @description     ```
// @description
// @description     ### Example: Creating a User Directory
// @description     ```json
// @description     // 1. Create table
// @description     POST /tables
// @description     {
// @description       "id": "users",
// @description       "fields": [
// @description         {"name": "name", "data_type": "string"},
// @description         {"name": "email", "data_type": "string"},
// @description         {"name": "age", "data_type": "int"},
// @description         {"name": "active", "data_type": "bool"}
// @description       ]
// @description     }
// @description
// @description     // 2. Add user
// @description     POST /tables/users/entities
// @description     {
// @description       "fields": {
// @description         "name": "John Doe",
// @description         "email": "john@example.com",
// @description         "age": 30,
// @description         "active": true
// @description       }
// @description     }
// @description
// @description     // 3. Query historical state
// @description     GET /tables/users/entities?asOf=2023-12-01T10:00:00Z
// @description     ```
// @description
// @description     ## 🔍 Time Travel Queries
// @description
// @description     Add `asOf` parameter to any entity query to see data as it existed at that time:
// @description     - `GET /tables/users/entities?asOf=2023-12-01T10:00:00Z`
// @description     - `GET /tables/users/entities/123?asOf=2023-12-01T10:00:00Z`
// @description
// @description     ## 📚 Response Formats
// @description
// @description     All timestamps use **RFC3339** format: `2023-12-31T23:59:59Z`
// @description
// @description     Standard response structure:
// @description     - **Success**: HTTP 200/201/204 with data
// @description     - **Error**: HTTP 4xx/5xx with `{"error": "description"}`
// @description
// @description     ## 🛠️ Development
// @description
// @description     - **Environment**: Go 1.23+
// @description     - **Database**: DynamoDB (local or AWS)
// @description     - **Auth**: JWT tokens (HS256)
// @description     - **Testing**: Comprehensive integration tests included
// @termsOfService  http://swagger.io/terms/

// @contact.name   Notably API Support
// @contact.url    https://github.com/elibdev/notably
// @contact.email  support@notably.dev

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.apikey  Bearer
// @in                          header
// @name                        Authorization
// @description                 JWT token format: `Bearer <token>`

// @externalDocs.description  Notably GitHub Repository
// @externalDocs.url          https://github.com/elibdev/notably
package main

//go:generate swag init -g main.go -o ../../docs

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
