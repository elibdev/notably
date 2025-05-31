package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func OpenAPISpec(c *gin.Context) {
	spec := map[string]interface{}{
		"openapi": "3.0.3",
		"info": map[string]interface{}{
			"title":       "TimeDB API",
			"description": "Time-versioned database API for personal information management",
			"version":     "1.0.0",
		},
		"servers": []map[string]interface{}{
			{"url": "/api/v1", "description": "API v1"},
		},
		"paths": map[string]interface{}{
			"/auth/login": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "User login",
					"requestBody": map[string]interface{}{
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"user_id":  map[string]interface{}{"type": "string"},
										"password": map[string]interface{}{"type": "string"},
									},
									"required": []string{"user_id", "password"},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Login successful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"token":      map[string]interface{}{"type": "string"},
											"expires_at": map[string]interface{}{"type": "string", "format": "date-time"},
											"user_id":    map[string]interface{}{"type": "string"},
										},
									},
								},
							},
						},
					},
				},
			},
			"/tables": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "List tables",
					"security": []map[string]interface{}{
						{"bearerAuth": []string{}},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "List of tables",
						},
					},
				},
				"post": map[string]interface{}{
					"summary": "Create table",
					"security": []map[string]interface{}{
						{"bearerAuth": []string{}},
					},
				},
			},
		},
		"components": map[string]interface{}{
			"securitySchemes": map[string]interface{}{
				"bearerAuth": map[string]interface{}{
					"type":   "http",
					"scheme": "bearer",
					"bearerFormat": "JWT",
				},
			},
		},
	}

	c.JSON(http.StatusOK, spec)
}