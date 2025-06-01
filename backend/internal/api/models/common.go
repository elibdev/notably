package models

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Error string `json:"error" example:"Invalid request"`
}

// SuccessResponse represents a standard success response
type SuccessResponse struct {
	Message string `json:"message" example:"Operation completed successfully"`
}

// ValidationErrorResponse represents validation error details
type ValidationErrorResponse struct {
	Error  string            `json:"error" example:"Validation failed"`
	Fields map[string]string `json:"fields,omitempty" example:"{\"email\":\"invalid email format\",\"password\":\"password too short\"}"`
}

// UserInfoResponse represents user information in responses
type UserInfoResponse struct {
	UserID    string `json:"user_id" example:"user123"`
	CreatedAt string `json:"created_at" example:"2023-12-31T23:59:59Z"`
	UpdatedAt string `json:"updated_at" example:"2023-12-31T23:59:59Z"`
}

// ListResponse represents a paginated list response
type ListResponse struct {
	Data   interface{} `json:"data"`
	Total  int         `json:"total" example:"100"`
	Offset int         `json:"offset" example:"0"`
	Limit  int         `json:"limit" example:"20"`
}

// TableListResponse represents a list of tables
type TableListResponse struct {
	Tables []TableResponse `json:"tables"`
}

// EntityListResponse represents a list of entities
type EntityListResponse struct {
	Entities []EntityResponse `json:"entities"`
}

// HealthResponse represents health check response
type HealthResponse struct {
	Status    string `json:"status" example:"healthy"`
	Timestamp string `json:"timestamp" example:"2023-12-31T23:59:59Z"`
	Version   string `json:"version,omitempty" example:"1.0.0"`
}

// EmptyResponse represents an empty success response (for 204 responses)
type EmptyResponse struct{}
