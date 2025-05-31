package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Example API usage
func main() {
	baseURL := "http://localhost:8080/api/v1"
	
	// 1. Register a user
	registerReq := map[string]interface{}{
		"user_id":  "user_123",
		"password": "secure_password",
		"email":    "user@example.com",
	}
	
	authResp, err := makeRequest("POST", baseURL+"/auth/register", registerReq, "")
	if err != nil {
		panic(err)
	}
	
	token := authResp["token"].(string)
	fmt.Printf("✅ User registered, token: %s\n", token[:20]+"...")
	
	// 2. Create a contacts table
	createTableReq := map[string]interface{}{
		"id": "contacts",
		"fields": []map[string]interface{}{
			{"name": "name", "data_type": "string"},
			{"name": "email", "data_type": "string"},
			{"name": "age", "data_type": "int"},
			{"name": "salary", "data_type": "float"},
			{"name": "active", "data_type": "bool"},
			{"name": "joined", "data_type": "date"},
			{"name": "metadata", "data_type": "json"},
		},
	}
	
	tableResp, err := makeRequest("POST", baseURL+"/tables", createTableReq, token)
	if err != nil {
		panic(err)
	}
	fmt.Printf("✅ Table created: %s\n", tableResp["id"])
	
	// 3. Create a contact entity
	createEntityReq := map[string]interface{}{
		"fields": map[string]interface{}{
			"name":   "John Doe",
			"email":  "john@example.com",
			"age":    30,
			"salary": 75000.50,
			"active": true,
			"joined": time.Now().Format(time.RFC3339),
			"metadata": map[string]interface{}{
				"department": "engineering",
				"level":      "senior",
			},
		},
	}
	
	entityResp, err := makeRequest("POST", baseURL+"/tables/contacts/entities", createEntityReq, token)
	if err != nil {
		panic(err)
	}
	
	entityID := entityResp["entity_id"].(string)
	fmt.Printf("✅ Entity created: %s\n", entityID)
	
	// 4. Update the entity
	updateEntityReq := map[string]interface{}{
		"fields": map[string]interface{}{
			"email":  "john.doe@newcompany.com",
			"salary": 85000.00,
			"metadata": map[string]interface{}{
				"department": "management",
				"level":      "director",
			},
		},
	}
	
	_, err = makeRequest("PUT", fmt.Sprintf("%s/tables/contacts/entities/%s", baseURL, entityID), updateEntityReq, token)
	if err != nil {
		panic(err)
	}
	fmt.Printf("✅ Entity updated\n")
	
	// 5. Get entity current state
	currentEntity, err := makeRequest("GET", fmt.Sprintf("%s/tables/contacts/entities/%s", baseURL, entityID), nil, token)
	if err != nil {
		panic(err)
	}
	fmt.Printf("✅ Current entity: %v\n", currentEntity["fields"])
	
	// 6. Get entity history
	history, err := makeRequest("GET", fmt.Sprintf("%s/tables/contacts/entities/%s/history", baseURL, entityID), nil, token)
	if err != nil {
		panic(err)
	}
	tuples := history["tuples"].([]interface{})
	fmt.Printf("✅ Entity history: %d changes\n", len(tuples))
	
	// 7. Delete a field
	_, err = makeRequest("DELETE", fmt.Sprintf("%s/tables/contacts/entities/%s/fields/age", baseURL, entityID), nil, token)
	if err != nil {
		panic(err)
	}
	fmt.Printf("✅ Field 'age' deleted\n")
	
	// 8. Delete the entity
	_, err = makeRequest("DELETE", fmt.Sprintf("%s/tables/contacts/entities/%s", baseURL, entityID), nil, token)
	if err != nil {
		panic(err)
	}
	fmt.Printf("✅ Entity soft deleted\n")
	
	// 9. Verify entity is gone from current state
	_, err = makeRequest("GET", fmt.Sprintf("%s/tables/contacts/entities/%s", baseURL, entityID), nil, token)
	if err == nil {
		fmt.Printf("❌ Expected entity to be deleted\n")
	} else {
		fmt.Printf("✅ Entity correctly shows as deleted\n")
	}
	
	// 10. Undelete the entity
	_, err = makeRequest("POST", fmt.Sprintf("%s/tables/contacts/entities/%s/undelete", baseURL, entityID), nil, token)
	if err != nil {
		panic(err)
	}
	fmt.Printf("✅ Entity undeleted\n")
	
	// 11. Get all entities
	entities, err := makeRequest("GET", baseURL+"/tables/contacts/entities", nil, token)
	if err != nil {
		panic(err)
	}
	entityList := entities["entities"].([]interface{})
	fmt.Printf("✅ All entities: %d found\n", len(entityList))
	
	// 12. Time travel - get state from 5 minutes ago
	past := time.Now().Add(-5 * time.Minute).Format(time.RFC3339)
	pastEntities, err := makeRequest("GET", baseURL+"/tables/contacts/entities?asOf="+past, nil, token)
	if err != nil {
		panic(err)
	}
	pastEntityList := pastEntities["entities"].([]interface{})
	fmt.Printf("✅ Past entities (5min ago): %d found\n", len(pastEntityList))
	
	fmt.Println("🎉 All API operations completed successfully!")
}

func makeRequest(method, url string, body interface{}, token string) (map[string]interface{}, error) {
	var reqBody []byte
	var err error
	
	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}
	
	req, err := http.NewRequest(method, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	
	var result map[string]interface{}
	if resp.StatusCode != http.StatusNoContent {
		err = json.NewDecoder(resp.Body).Decode(&result)
		if err != nil {
			return nil, err
		}
	}
	
	return result, nil
}

/*
Example Dockerfile for deployment:

# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main cmd/server/main.go

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/main .
COPY --from=builder /app/config ./config

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the binary
CMD ["./main"]
*/