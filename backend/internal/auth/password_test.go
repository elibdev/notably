package auth

import (
	"strings"
	"testing"
)

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "valid password",
			password: "securepassword123",
			wantErr:  false,
		},
		{
			name:     "short password",
			password: "short",
			wantErr:  false, // Hashing should work regardless of length
		},
		{
			name:     "long password",
			password: strings.Repeat("a", 200),
			wantErr:  true, // bcrypt has 72-byte limit
		},
		{
			name:     "empty password",
			password: "",
			wantErr:  false, // Hashing should work, validation happens elsewhere
		},
		{
			name:     "password with special characters",
			password: "p@ssw0rd!#$%^&*()",
			wantErr:  false,
		},
		{
			name:     "unicode password",
			password: "пароль123",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashPassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("HashPassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Hash should not be empty
				if hash == "" {
					t.Errorf("HashPassword() returned empty hash")
				}

				// Hash should not be the same as the original password
				if hash == tt.password {
					t.Errorf("HashPassword() returned plaintext password")
				}

				// Hash should start with bcrypt prefix
				if !strings.HasPrefix(hash, "$2a$") && !strings.HasPrefix(hash, "$2b$") && !strings.HasPrefix(hash, "$2y$") {
					t.Errorf("HashPassword() returned invalid bcrypt hash format: %s", hash)
				}
			}
		})
	}
}

func TestVerifyPassword(t *testing.T) {
	// First, create a known hash for testing
	testPassword := "testpassword123"
	validHash, err := HashPassword(testPassword)
	if err != nil {
		t.Fatalf("Failed to create test hash: %v", err)
	}

	tests := []struct {
		name     string
		password string
		hash     string
		want     bool
	}{
		{
			name:     "correct password",
			password: testPassword,
			hash:     validHash,
			want:     true,
		},
		{
			name:     "incorrect password",
			password: "wrongpassword",
			hash:     validHash,
			want:     false,
		},
		{
			name:     "empty password",
			password: "",
			hash:     validHash,
			want:     false,
		},
		{
			name:     "empty hash",
			password: testPassword,
			hash:     "",
			want:     false,
		},
		{
			name:     "invalid hash format",
			password: testPassword,
			hash:     "invalidhash",
			want:     false,
		},
		{
			name:     "case sensitive password",
			password: "TESTPASSWORD123",
			hash:     validHash,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := VerifyPassword(tt.password, tt.hash)
			if got != tt.want {
				t.Errorf("VerifyPassword() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidatePasswordStrength(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
		errType  error
	}{
		{
			name:     "valid 8 character password",
			password: "password",
			wantErr:  false,
		},
		{
			name:     "valid long password",
			password: "thisIsAVeryLongPasswordThatShouldBeValid",
			wantErr:  false,
		},
		{
			name:     "valid password with numbers",
			password: "password123",
			wantErr:  false,
		},
		{
			name:     "valid password with special chars",
			password: "pass@word!",
			wantErr:  false,
		},
		{
			name:     "too short - 7 characters",
			password: "1234567",
			wantErr:  true,
			errType:  ErrPasswordTooShort,
		},
		{
			name:     "too short - 1 character",
			password: "a",
			wantErr:  true,
			errType:  ErrPasswordTooShort,
		},
		{
			name:     "empty password",
			password: "",
			wantErr:  true,
			errType:  ErrPasswordTooShort,
		},
		{
			name:     "exactly 8 characters",
			password: "12345678",
			wantErr:  false,
		},
		{
			name:     "too long - exceeds 72 bytes",
			password: strings.Repeat("a", 73),
			wantErr:  true,
			errType:  ErrPasswordTooLong,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePasswordStrength(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePasswordStrength() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errType != nil && err != tt.errType {
				t.Errorf("ValidatePasswordStrength() error = %v, want %v", err, tt.errType)
			}
		})
	}
}

func TestHashAndVerifyRoundTrip(t *testing.T) {
	// Test that hashing and verifying work together correctly
	passwords := []string{
		"simplepassword",
		"complex!P@ssw0rd#123",
		"пароль测试密码",
		"😀🔐🛡️password",
		"validlengthpassword",
	}

	for _, password := range passwords {
		t.Run("roundtrip_"+password[:min(len(password), 10)], func(t *testing.T) {
			// Hash the password
			hash, err := HashPassword(password)
			if err != nil {
				t.Fatalf("HashPassword() failed: %v", err)
			}

			// Verify the password
			if !VerifyPassword(password, hash) {
				t.Errorf("VerifyPassword() failed for password that was just hashed")
			}

			// Verify wrong password fails
			if VerifyPassword(password+"wrong", hash) {
				t.Errorf("VerifyPassword() succeeded for wrong password")
			}
		})
	}
}

func TestHashPasswordConsistency(t *testing.T) {
	// Test that hashing the same password twice produces different hashes
	// (due to bcrypt salt), but both verify correctly
	password := "testpassword123"

	hash1, err := HashPassword(password)
	if err != nil {
		t.Fatalf("First HashPassword() failed: %v", err)
	}

	hash2, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Second HashPassword() failed: %v", err)
	}

	// Hashes should be different (due to salt)
	if hash1 == hash2 {
		t.Errorf("HashPassword() produced identical hashes, salt may not be working")
	}

	// Both hashes should verify the original password
	if !VerifyPassword(password, hash1) {
		t.Errorf("First hash failed to verify")
	}

	if !VerifyPassword(password, hash2) {
		t.Errorf("Second hash failed to verify")
	}
}

func BenchmarkHashPassword(b *testing.B) {
	password := "benchmarkpassword123"
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := HashPassword(password)
		if err != nil {
			b.Fatalf("HashPassword() failed: %v", err)
		}
	}
}

func BenchmarkVerifyPassword(b *testing.B) {
	password := "benchmarkpassword123"
	hash, err := HashPassword(password)
	if err != nil {
		b.Fatalf("Failed to create test hash: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		VerifyPassword(password, hash)
	}
}

// Helper function for Go versions that don't have min function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
