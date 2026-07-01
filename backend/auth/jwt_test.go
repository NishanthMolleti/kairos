package auth

import (
	"testing"
	"github.com/google/uuid"
)

func TestJWTRoundTrip(t *testing.T) {
	secret := "test-secret-key-long-enough"
	userID := uuid.New()
	token, err := SignJWT(userID, secret)
	if err != nil {
		t.Fatalf("sign failed: %v", err)
	}
	claims, err := ValidateJWT(token, secret)
	if err != nil {
		t.Fatalf("validate failed: %v", err)
	}
	if claims.UserID != userID {
		t.Fatalf("want userID %s, got %s", userID, claims.UserID)
	}
}

func TestJWTWrongSecret(t *testing.T) {
	userID := uuid.New()
	token, _ := SignJWT(userID, "correct-secret")
	_, err := ValidateJWT(token, "wrong-secret")
	if err == nil {
		t.Fatal("expected error with wrong secret, got nil")
	}
}
