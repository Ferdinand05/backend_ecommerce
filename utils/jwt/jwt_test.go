package jwt

import (
	"testing"

	"github.com/google/uuid"
)

func TestGenerateAndValidateToken(t *testing.T) {
	svc := NewJWTService("test-secret-key")

	userID := uuid.New()
	token, err := svc.GenerateToken(userID, "user@example.com", "customer")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	claims, err := svc.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}

	if claims.Subject != userID.String() {
		t.Fatalf("Subject = %q, want %q", claims.Subject, userID.String())
	}

	if claims.Email != "user@example.com" {
		t.Fatalf("Email = %q, want %q", claims.Email, "user@example.com")
	}

	if claims.Role != "customer" {
		t.Fatalf("Role = %q, want %q", claims.Role, "customer")
	}
}

func TestValidateTokenErrors(t *testing.T) {
	svc := NewJWTService("test-secret-key")
	otherSvc := NewJWTService("other-secret-key")

	validToken, err := svc.GenerateToken(uuid.New(), "user@example.com", "customer")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	tampered := validToken + "x"

	otherToken, err := otherSvc.GenerateToken(uuid.New(), "user@example.com", "customer")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	tests := []struct {
		name  string
		token string
	}{
		{name: "random string", token: "not-a-jwt"},
		{name: "tampered token", token: tampered},
		{name: "wrong secret", token: otherToken},
		{name: "empty token", token: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := svc.ValidateToken(tt.token); err == nil {
				t.Fatal("ValidateToken() error = nil, want error")
			}
		})
	}
}
