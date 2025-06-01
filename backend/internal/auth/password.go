package auth

import (
	"golang.org/x/crypto/bcrypt"
)

const (
	// DefaultCost is the default bcrypt cost factor for password hashing
	// Cost 12 provides a good balance between security and performance
	DefaultCost = 12
)

// HashPassword takes a plaintext password and returns a bcrypt hash
func HashPassword(password string) (string, error) {
	// Check bcrypt 72-byte limit before attempting to hash
	if len([]byte(password)) > 72 {
		return "", ErrPasswordTooLong
	}

	bytes, err := bcrypt.GenerateFromPassword([]byte(password), DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// VerifyPassword compares a plaintext password with a bcrypt hash
// Returns true if the password matches the hash, false otherwise
func VerifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// ValidatePasswordStrength checks if a password meets minimum requirements
func ValidatePasswordStrength(password string) error {
	if len(password) < 8 {
		return ErrPasswordTooShort
	}

	// Check bcrypt 72-byte limit
	if len([]byte(password)) > 72 {
		return ErrPasswordTooLong
	}

	// Add more validation rules as needed:
	// - Contains uppercase letter
	// - Contains lowercase letter
	// - Contains number
	// - Contains special character

	return nil
}
