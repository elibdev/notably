#!/usr/bin/env bash
set -e

CONFIG_PATH="config/environments/local.yaml"

# Extract values from YAML
TABLE_NAME=$(yq '.database.table_name' "$CONFIG_PATH")
REGION=$(yq '.database.region' "$CONFIG_PATH")
ENDPOINT_URL=$(yq '.database.endpoint_url' "$CONFIG_PATH")

echo "Creating DynamoDB table '$TABLE_NAME' at $ENDPOINT_URL..."

aws dynamodb create-table \
  --table-name "$TABLE_NAME" \
  --attribute-definitions \
    AttributeName=PK,AttributeType=S \
    AttributeName=SK,AttributeType=S \
    AttributeName=GSI1PK,AttributeType=S \
    AttributeName=GSI1SK,AttributeType=S \
  --key-schema \
    AttributeName=PK,KeyType=HASH \
    AttributeName=SK,KeyType=RANGE \
  --global-secondary-indexes \
    "IndexName=GSI1,KeySchema=[{AttributeName=GSI1PK,KeyType=HASH},{AttributeName=GSI1SK,KeyType=RANGE}],Projection={ProjectionType=ALL},ProvisionedThroughput={ReadCapacityUnits=5,WriteCapacityUnits=5}" \
  --billing-mode PROVISIONED \
  --provisioned-throughput ReadCapacityUnits=5,WriteCapacityUnits=5 \
  --endpoint-url "$ENDPOINT_URL" \
  --region "$REGION" || echo "⚠️ Table may already exist — skipping."

echo "✅ DynamoDB setup complete using values from $CONFIG_PATH"
