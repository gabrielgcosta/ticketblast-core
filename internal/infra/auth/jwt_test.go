package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTokenEngine_GenerateAndValidateToken(t *testing.T) {
	secret := "my_test_secret_key_long_enough_for_security"
	engine := NewTokenEngine(secret)

	userID := "user-123"
	role := "admin"

	// 1. Generate token
	tokenStr, err := engine.GenerateToken(userID, role)
	assert.NoError(t, err)
	assert.NotEmpty(t, tokenStr)

	// 2. Validate token
	claims, err := engine.ValidateToken(tokenStr)
	assert.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, role, claims.Role)
}

func TestTokenEngine_ValidateInvalidToken(t *testing.T) {
	engine := NewTokenEngine("secret")

	// 1. Invalid signature
	otherEngine := NewTokenEngine("different_secret")
	tokenStr, err := otherEngine.GenerateToken("user-1", "customer")
	assert.NoError(t, err)

	_, err = engine.ValidateToken(tokenStr)
	assert.Error(t, err)

	// 2. Garbage token
	_, err = engine.ValidateToken("this.is.garbage")
	assert.Error(t, err)
}
