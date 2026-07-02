package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gabrielgcosta/ticketblast-core/internal/infra/auth"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestAuthMiddleware_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	secret := "my_test_secret_key_long_enough_for_security"
	tokenEngine := auth.NewTokenEngine(secret)
	token, err := tokenEngine.GenerateToken("user-123", "admin")
	assert.NoError(t, err)

	r := gin.New()
	r.Use(Auth(tokenEngine))
	r.GET("/test", func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userRole, _ := c.Get("user_role")
		c.JSON(http.StatusOK, gin.H{
			"user_id":   userID,
			"user_role": userRole,
		})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"user_id":"user-123"`)
	assert.Contains(t, w.Body.String(), `"user_role":"admin"`)
}

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tokenEngine := auth.NewTokenEngine("secret")
	r := gin.New()
	r.Use(Auth(tokenEngine))
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Missing authorization header")
}

func TestAuthMiddleware_InvalidHeaderFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tokenEngine := auth.NewTokenEngine("secret")
	r := gin.New()
	r.Use(Auth(tokenEngine))
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "InvalidFormat token")

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid authorization header format")
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tokenEngine := auth.NewTokenEngine("secret")
	r := gin.New()
	r.Use(Auth(tokenEngine))
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token-string")

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid or expired token")
}
