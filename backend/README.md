/*
# TimeDB - Time-Versioned Database API

A powerful, time-versioned database system built on DynamoDB with a clean HTTP API. Perfect for personal information management systems that need to track changes over time.

## 🚀 Quick Start

# Start everything (DynamoDB Local + API Server)
make dev

# API will be available at http://localhost:8080
# DynamoDB Admin UI at http://localhost:8001

## ✨ Features

- **⏰ Time Travel**: Query data as it existed at any point in time
- **🔄 Full History**: Every change is preserved, nothing is ever lost
- **🛡️ Soft Deletes**: Delete data while preserving history and enabling recovery
- **🏗️ Flexible Schema**: Each user can define their own table structures
- **🔐 User Isolation**: Complete data separation between users
- **🎯 Type Safety**: Support for strings, numbers, booleans, dates, and JSON
- **⚡ High Performance**: Optimized DynamoDB access patterns with GSIs
- **🧪 Fully Tested**: Comprehensive test suite with integration tests

## 📊 Architecture

### Core Components

┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   HTTP API      │    │   Repository     │    │   DynamoDB      │
│                 │───▶│                  │───▶│                 │
│ • REST Endpoints│    │ • User-Scoped    │    │ • Single Table  │
│ • JWT Auth      │    │ • Type Safety    │    │ • 3 GSIs        │
│ • OpenAPI Docs  │    │ • Soft Deletes   │    │ • Time-Ordered  │
└─────────────────┘    └──────────────────┘    └─────────────────┘

### Data Model

User
├── Tables (contacts, notes, projects, etc.)
│   ├── Schema (field definitions)
│   └── Entities (rows/records)
│       ├── Fields (typed values)
│       ├── History (all changes)
│       └── Metadata (created, deleted, etc.)

## 🏃‍♂️ Getting Started

### Prerequisites

- Go 1.21+
- Docker & Docker Compose
- AWS CLI (for cloud deployment)

### Development Setup

1. **Clone and start services**:
git clone <your-repo>
cd timedb
make dev

2. **Test the API**:
# Run example usage
make api-example

# Or test manually
curl http://localhost:8080/health

3. **Explore the API**:
- 📖 **API Docs**: http://localhost:8080/docs
- 🔍 **OpenAPI Spec**: http://localhost:8080/openapi.json
- 📊 **DynamoDB Admin**: http://localhost:8001

## 📚 API Usage

### Authentication

# Register a user
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"user_id": "john_doe", "password": "secure123", "email": "john@example.com"}'

# Response: {"token": "eyJ...", "expires_at": "...", "user_id": "john_doe"}

### Table Management

# Create a table with typed fields
curl -X POST http://localhost:8080/api/v1/tables \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "contacts",
    "fields": [
      {"name": "name", "data_type": "string"},
      {"name": "email", "data_type": "string"},
      {"name": "age", "data_type": "int"},
      {"name": "salary", "data_type": "float"},
      {"name": "active", "data_type": "bool"},
      {"name": "joined", "data_type": "date"},
      {"name": "metadata", "data_type": "json"}
    ]
  }'

# List tables
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/tables

### Entity Operations

# Create an entity with typed values
curl -X POST http://localhost:8080/api/v1/tables/contacts/entities \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "fields": {
      "name": "John Doe",
      "email": "john@example.com",
      "age": 30,
      "salary": 75000.50,
      "active": true,
      "joined": "2024-01-15T10:00:00Z",
      "metadata": {"department": "engineering", "level": "senior"}
    }
  }'

# Get entity current state
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/tables/contacts/entities/$ENTITY_ID

# Update entity (creates new version)
curl -X PUT http://localhost:8080/api/v1/tables/contacts/entities/$ENTITY_ID \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"fields": {"email": "john.doe@newcompany.com", "salary": 85000.00}}'

### Time Travel

# Get entity as it existed yesterday
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/api/v1/tables/contacts/entities/$ENTITY_ID?asOf=2024-01-14T10:00:00Z"

# Get all entities from last week
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/api/v1/tables/contacts/entities?asOf=2024-01-08T10:00:00Z"

# Get entity history
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/tables/contacts/entities/$ENTITY_ID/history

### Soft Deletes

# Delete entity (soft delete - preserves history)
curl -X DELETE http://localhost:8080/api/v1/tables/contacts/entities/$ENTITY_ID \
  -H "Authorization: Bearer $TOKEN"

# Entity won't appear in current queries
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/tables/contacts/entities/$ENTITY_ID
# Returns: 404 Not Found

# But history is still accessible
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/tables/contacts/entities/$ENTITY_ID/history

# Undelete entity
curl -X POST http://localhost:8080/api/v1/tables/contacts/entities/$ENTITY_ID/undelete \
  -H "Authorization: Bearer $TOKEN"

# Admin view (includes deleted entities)
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/admin/tables/contacts/entities

## 🗄️ Database Schema

### Single Table Design

| PK | SK | Purpose |
|----|----|----- |
| `USER#{userID}` | `USER` | User metadata |
| `USER#{userID}` | `TABLE#{tableID}` | Table schemas |
| `USER#{userID}` | `TUPLE#{tableID}#{entityID}#{timestamp}#{fieldName}` | Data tuples |

### Global Secondary Indexes

- **GSI1**: Field-based queries (`USER#{userID}#{tableID}#{fieldName}` → `{timestamp}#{entityID}`)
- **GSI2**: Entity-based queries (`USER#{userID}#{tableID}#{entityID}` → `{timestamp}#{fieldName}`)
- **GSI3**: Table-wide queries (`USER#{userID}#{tableID}` → `{entityID}#{timestamp}#{fieldName}`)

## 🧪 Testing

# Run all tests
make test

# Run integration tests
make test-integration

# Load testing
make load-test

# Health checks
make health-check
make ready-check

## 🚀 Deployment

### Local Development
make dev

### Development Environment
# Setup DynamoDB table
make setup-dev

# Deploy application
make deploy-dev

### Production
# Setup infrastructure with Terraform
cd infrastructure/terraform
terraform init
terraform apply -var="environment=prod"

# Deploy application
make deploy-prod

## 🛠️ Configuration

Configuration is environment-based using YAML files:

# config/environments/local.yaml
database:
  provider: "dynamodb"
  table_name: "timedb"
  region: "us-east-1"
  endpoint_url: "http://localhost:8000"

server:
  port: 8080
  host: "localhost"

auth:
  jwt_secret: "local-development-secret"
  token_expiry: "24h"
  require_auth: false  # Set to true in production

logging:
  level: "debug"
  format: "text"

Environment variables:
- `TIMEDB_ENV`: Environment name (local, dev, staging, prod)
- `JWT_SECRET`: JWT signing secret (production)
- `AWS_REGION`: AWS region
- `AWS_ACCESS_KEY_ID`: AWS credentials
- `AWS_SECRET_ACCESS_KEY`: AWS credentials

## 📈 Performance

### Capacity Planning

| Environment | RCU/WCU | Cost/Month | Use Case |
|-------------|---------|------------|----------|
| **Local** | N/A | $0 | Development |
| **Dev** | 5/5 | ~$3 | Testing |
| **Staging** | On-Demand | ~$20-50 | Load testing |
| **Production** | On-Demand | $100-500+ | Production workloads |

### Optimization Tips

1. **Batch Operations**: Use batch endpoints for bulk operations
2. **Time Range Queries**: Use specific time ranges to limit data
3. **Field Selection**: Only query needed fields
4. **Pagination**: Use pagination for large result sets
5. **Caching**: Implement Redis caching for frequently accessed data

## 🔒 Security

- **Authentication**: JWT tokens with configurable expiry
- **Authorization**: User-scoped data access only
- **Encryption**: KMS encryption at rest (production)
- **Network**: VPC endpoints and security groups
- **Audit**: Complete history of all changes

## 🐛 Troubleshooting

### Common Issues

1. **Connection Refused**
   ```bash
   # Check if services are running
   docker-compose ps
   make health-check
   ```

2. **Authentication Errors**
   ```bash
   # Check token expiry
   curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/users/me
   ```

3. **DynamoDB Errors**
   ```bash
   # Check DynamoDB Local
   curl http://localhost:8000
   
   # Check table setup
   aws dynamodb list-tables --endpoint-url http://localhost:8000 --region us-east-1
   ```

### Debug Mode

# Enable debug logging
TIMEDB_ENV=local go run cmd/server/main.go

# Check logs
docker-compose logs timedb

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Workflow

# Setup development environment
make dev

# Make changes and test
make test
make api-example

# Format code
make format
make lint

# Submit PR

## 📄 License


## 🙏 Acknowledgments

- Inspired by [Datomic](https://www.datomic.com/) and [XTDB](https://xtdb.com/)
- Built with [Gin](https://github.com/gin-gonic/gin) web framework
- Powered by [AWS DynamoDB](https://aws.amazon.com/dynamodb/)

---

**TimeDB** - Because every moment in your data's life matters. ⏰✨
*/