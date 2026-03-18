package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestJWTManagerGenerateAndParseRoundTrip(t *testing.T) {
	manager := NewJWTManager("secret", time.Hour)
	userID := uuid.New()

	token, err := manager.Generate(userID, "test@example.com")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	claims, err := manager.Parse(token)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if claims.UserID != userID {
		t.Fatalf("claims.UserID = %s, want %s", claims.UserID, userID)
	}
	if claims.Email != "test@example.com" {
		t.Fatalf("claims.Email = %q, want %q", claims.Email, "test@example.com")
	}
}

func TestJWTManagerParseRejectsUnexpectedAlgorithm(t *testing.T) {
	manager := NewJWTManager("secret", time.Hour)
	now := time.Now().UTC()
	token := jwt.NewWithClaims(jwt.SigningMethodNone, Claims{
		UserID: uuid.New(),
		Email:  "test@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   uuid.NewString(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	})

	tokenString, err := token.SigningString()
	if err != nil {
		t.Fatalf("SigningString() error = %v", err)
	}
	tokenString += "."

	if _, err := manager.Parse(tokenString); err == nil {
		t.Fatal("Parse() error = nil, want signature/algorithm rejection")
	}
}
