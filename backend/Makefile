.PHONY: build run test clean docker-build docker-run setup-local api-example

# Go commands
build:
	go build -o bin/timedb cmd/server/main.go

run: build
	./bin/timedb

run-local:
	TIMEDB_ENV=local go run cmd/server/main.go

test:
	go test ./...

test-integration:
	go test ./tests/integration/...

clean:
	rm -rf bin/
	docker compose down -v

# Docker commands
docker-build:
	docker build -t timedb:latest .

docker-run: docker-build
	docker compose up -d

# Development setup
setup-local:
	@if [ ! "$$(docker ps -q -f name=dynamodb-local)" ]; then \
		if [ "$$(docker ps -aq -f name=dynamodb-local)" ]; then \
			echo "🛠️  Starting existing DynamoDB container..."; \
			docker start dynamodb-local; \
		else \
			echo "🚀 Creating new DynamoDB container..."; \
			docker run -d --name dynamodb-local -p 8000:8000 amazon/dynamodb-local; \
		fi \
	else \
		echo "✅ DynamoDB container already running."; \
	fi
	sleep 5
	# Example AWS CLI to create table manually (edit as needed):
	# aws dynamodb create-table \
	#   --table-name timedb \
	#   --attribute-definitions \
	#     AttributeName=PK,AttributeType=S \
	#     AttributeName=SK,AttributeType=S \
	#     AttributeName=GSI1PK,AttributeType=S \
	#     AttributeName=GSI1SK,AttributeType=S \
	#   --key-schema \
	#     AttributeName=PK,KeyType=HASH \
	#     AttributeName=SK,KeyType=RANGE \
	#   --global-secondary-indexes \
	#     "IndexName=GSI1,KeySchema=[{AttributeName=GSI1PK,KeyType=HASH},{AttributeName=GSI1SK,KeyType=RANGE}],Projection={ProjectionType=ALL},ProvisionedThroughput={ReadCapacityUnits=5,WriteCapacityUnits=5}" \
	#   --billing-mode PROVISIONED \
	#   --endpoint-url http://localhost:8000

setup-full: setup-local
	docker compose up -d

# API testing
api-example: setup-full
	sleep 10
	go run examples/api_usage.go

# Health checks
health-check:
	curl -f http://localhost:8080/health || exit 1

ready-check:
	curl -f http://localhost:8080/ready || exit 1

# Development workflow
dev: clean setup-full
	@echo "🚀 TimeDB development environment ready!"
	@echo "📊 DynamoDB Admin: http://localhost:8001"
	@echo "🔗 API Server: http://localhost:8080"
	@echo "📖 API Docs: http://localhost:8080/docs"
	@echo "❤️  Health: http://localhost:8080/health"

# Install dependencies
deps:
	go mod tidy
	go mod download

# Code quality
lint:
	golangci-lint run

format:
	go fmt ./...

# Database operations
db-setup:
	@echo "Run the following AWS CLI to create the table manually:"
	@echo ""
	@echo "aws dynamodb create-table \\"
	@echo "  --table-name timedb \\"
	@echo "  --attribute-definitions \\"
	@echo "    AttributeName=PK,AttributeType=S \\"
	@echo "    AttributeName=SK,AttributeType=S \\"
	@echo "    AttributeName=GSI1PK,AttributeType=S \\"
	@echo "    AttributeName=GSI1SK,AttributeType=S \\"
	@echo "  --key-schema \\"
	@echo "    AttributeName=PK,KeyType=HASH \\"
	@echo "    AttributeName=SK,KeyType=RANGE \\"
	@echo "  --global-secondary-indexes \\"
	@echo '    IndexName=GSI1,KeySchema=[{AttributeName=GSI1PK,KeyType=HASH},{AttributeName=GSI1SK,KeyType=RANGE}],Projection={ProjectionType=ALL},ProvisionedThroughput={ReadCapacityUnits=5,WriteCapacityUnits=5}'
	@echo "  --billing-mode PROVISIONED \\"
	@echo "  --endpoint-url http://localhost:8000"

db-migrate:
	@echo "Running database migrations..."
	# Add migration commands here

# Load testing
load-test:
	@echo "Running load tests..."
	# Add load testing with tools like k6 or vegeta

# Documentation
docs:
	@echo "Generating API documentation..."
	# Add swagger generation commands

# Production deployment
deploy-dev:
	@echo "Deploying to development..."
	# Add deployment commands

deploy-prod:
	@echo "Deploying to production..."
	# Add production deployment commands
