package models

import (
	"time"
)

// User represents a user in the system
type User struct {
	ID        string    `json:"id"`
	PK        string    `json:"pk"`         // Partition key: USER#{userID}
	SK        string    `json:"sk"`         // Sort key: USER
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewUser creates a new user with the given ID
func NewUser(userID string) User {
	now := time.Now()
	return User{
		ID:        userID,
		PK:        "USER#" + userID,
		SK:        "USER",
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// UpdateTimestamp updates the UpdatedAt field to the current time
func (u *User) UpdateTimestamp() {
	u.UpdatedAt = time.Now()
}

// GetUserID extracts the user ID from the partition key
func (u *User) GetUserID() string {
	return u.ID
}

// IsValid checks if the user has required fields
func (u *User) IsValid() bool {
	return u.ID != "" && u.PK != "" && u.SK == "USER"
}