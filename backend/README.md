# ⏰ TimeDB - Time-Versioned Database API

**Because every moment in your data's life matters.** ✨

**TimeDB** is a powerful time-versioned database system built on DynamoDB with a clean HTTP API. It's ideal for personal information management systems that need to track data changes over time.

---

## 🚀 Quick Start

Start everything (DynamoDB Local + API server):

```bash
make dev
```

* API available at: [http://localhost:8080](http://localhost:8080)
* DynamoDB Admin UI: [http://localhost:8001](http://localhost:8001)

---

## ✨ Features

* **⏰ Time Travel**: Query data as it existed at any point in time
* **🔄 Full History**: Every change is preserved — nothing is lost
* **🛡️ Soft Deletes**: Delete data while preserving history and enabling recovery
* **🏗️ Flexible Schema**: Each user can define their own table structures
* **🔐 User Isolation**: Complete data separation between users
* **🎯 Type Safety**: Support for strings, numbers, booleans, dates, and JSON
* **⚡ High Performance**: Optimized DynamoDB access patterns with GSIs
* **🧪 Fully Tested**: Comprehensive integration test suite

---

## 📊 Architecture

### Core Components

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   HTTP API      │    │   Repository     │    │   DynamoDB      │
│                 │───▶│                  │───▶│                 │
│ • REST Endpoints│    │ • User-Scoped    │    │ • Single Table  │
│ • JWT Auth      │    │ • Type Safety    │    │ • 3 GSIs        │
│ • OpenAPI Docs  │    │ • Soft Deletes   │    │ • Time-Ordered  │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

### Data Model

```
User
├── Tables (contacts, notes, projects, etc.)
│   ├── Schema (field definitions)
│   └── Entities (rows/records)
│       ├── Fields (typed values)
│       ├── History (all changes)
│       └── Metadata (created, deleted, etc.)
```

---

## 🏃 Getting Started

### Prerequisites

* Go 1.21+
* Docker & Docker Compose
* AWS CLI (for cloud deployment)

### Development Setup

1. Clone and start services:

   ```bash
   git clone <your-repo>
   cd timedb
   make dev
   ```

2. Test the API:

   ```bash
   make api-example
   curl http://localhost:8080/health
   ```

3. Explore the API:

   * 📖 [API Docs](http://localhost:8080/docs)
   * 🔍 [OpenAPI Spec](http://localhost:8080/openapi.json)
   * 📊 [DynamoDB Admin](http://localhost:8001)

---

## 📚 API Usage

### 🔐 Authentication

Register a user:

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"user_id": "john_doe", "password": "secure123", "email": "john@example.com"}'
```

Example response:

```json
{"token": "eyJ...", "expires_at": "...", "user_id": "john_doe"}
```

---

### 🧱 Table Management

Create a table:

```bash
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
```

List tables:

```bash
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/tables
```

---

### 📄 Entity Operations

Create an entity:

```bash
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
```

Get entity (latest version):

```bash
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/tables/contacts/entities/$ENTITY_ID
```

Update entity:

```bash
curl -X PUT http://localhost:8080/api/v1/tables/contacts/entities/$ENTITY_ID \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"fields": {"email": "john.doe@newcompany.com", "salary": 85000.00}}'
```

---

### ⏳ Time Travel

Get entity as of a specific time:

```bash
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/api/v1/tables/contacts/entities/$ENTITY_ID?asOf=2024-01-14T10:00:00Z"
```

Query past data:

```bash
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/api/v1/tables/contacts/entities?asOf=2024-01-08T10:00:00Z"
```

View entity history:

```bash
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/tables/contacts/entities/$ENTITY_ID/history
```

---

### 🗑️ Soft Deletes

Soft delete entity:

```bash
curl -X DELETE http://localhost:8080/api/v1/tables/contacts/entities/$ENTITY_ID \
  -H "Authorization: Bearer $TOKEN"
```

Undelete entity:

```bash
curl -X POST http://localhost:8080/api/v1/tables/contacts/entities/$ENTITY_ID/undelete \
  -H "Authorization: Bearer $TOKEN"
```

Check admin view (includes deleted entities):

```bash
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/admin/tables/contacts/entities
```

---

## 🗄️ Database Schema

### Single Table Design

| PK              | SK                                        | Purpose       |
| --------------- | ----------------------------------------- | ------------- |
| `USER#{userID}` | `USER`                                    | User metadata |
| `USER#{userID}` | `TABLE#{tableID}`                         | Table schema  |
| `USER#{userID}` | `TUPLE#{tableID}#{entityID}#{ts}#{field}` | Data values   |

### Global Secondary Indexes

* **GSI1**: Field-based queries
* **GSI2**: Entity-based queries
* **GSI3**: Table-wide queries

---

## 🧪 Testing

Run all tests:

```bash
make test
```

Integration tests:

```bash
make test-integration
```

Load test:

```bash
make load-test
```

Health checks:

```bash
make health-check
make ready-check
```

---

## 🚀 Deployment

### Local Development

```bash
make dev
```

### Development Environment

```bash
make setup-dev
make deploy-dev
```

### Production

```bash
cd infrastructure/terraform
terraform init
terraform apply -var="environment=prod"
make deploy-prod
```

---

## 🛠️ Configuration

**config/environments/local.yaml**:

```yaml
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
  require_auth: false

logging:
  level: "debug"
  format: "text"
```

Environment Variables:

* `TIMEDB_ENV`
* `JWT_SECRET`
* `AWS_REGION`
* `AWS_ACCESS_KEY_ID`
* `AWS_SECRET_ACCESS_KEY`

---

## 📈 Performance

### Capacity Planning

| Environment | RCU/WCU   | Cost/Month | Use Case             |
| ----------- | --------- | ---------- | -------------------- |
| Local       | N/A       | \$0        | Development          |
| Dev         | 5/5       | \~\$3      | Testing              |
| Staging     | On-Demand | \~\$20–50  | Load testing         |
| Production  | On-Demand | \$100–500+ | Production workloads |

### Optimization Tips

1. Use batch operations
2. Query specific time ranges
3. Select only needed fields
4. Paginate large result sets
5. Use Redis for caching frequently accessed data

---

## 🔒 Security

* JWT authentication with configurable expiry
* User-scoped data isolation
* KMS encryption at rest (production)
* VPC & security groups
* Full audit history

---

## 🐛 Troubleshooting

### Common Issues

**Connection Refused**

```bash
docker-compose ps
make health-check
```

**Authentication Errors**

```bash
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/users/me
```

**DynamoDB Errors**

```bash
curl http://localhost:8000
aws dynamodb list-tables --endpoint-url http://localhost:8000 --region us-east-1
```

### Debug Mode

```bash
TIMEDB_ENV=local go run cmd/server/main.go
docker-compose logs timedb
```

---

## 🤝 Contributing

1. Fork the repo
2. Create a feature branch
3. Commit your changes
4. Push your branch
5. Open a Pull Request

### Development Workflow

```bash
make dev
make test
make api-example
make format
make lint
```

---

## 📄 License

*TODO: Add license info.*

---

## 🙏 Acknowledgments

* Inspired by [Datomic](https://www.datomic.com/) and [XTDB](https://xtdb.com/)
* Built with [Gin](https://github.com/gin-gonic/gin)
* Powered by [AWS DynamoDB](https://aws.amazon.com/dynamodb/)
