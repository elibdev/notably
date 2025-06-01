package auth

import "errors"

var (
	// Password validation errors
	ErrPasswordTooShort = errors.New("password must be at least 8 characters long")
	ErrPasswordTooLong  = errors.New("password cannot exceed 72 bytes")
	ErrPasswordTooWeak  = errors.New("password does not meet strength requirements")

	// Authentication errors
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user already exists")

	// Input validation errors
	ErrInvalidEmail  = errors.New("invalid email format")
	ErrInvalidUserID = errors.New("invalid user ID format")
	ErrEmptyPassword = errors.New("password cannot be empty")
	ErrEmptyEmail    = errors.New("email cannot be empty")
	ErrEmptyUserID   = errors.New("user ID cannot be empty")

	// Account state errors
	ErrAccountDisabled = errors.New("account is disabled")
	ErrAccountLocked   = errors.New("account is locked")

	// Token errors
	ErrInvalidToken   = errors.New("invalid or expired token")
	ErrTokenExpired   = errors.New("token has expired")
	ErrTokenMalformed = errors.New("token is malformed")
)
