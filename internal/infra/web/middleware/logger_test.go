package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gabrielgcosta/ticketblast-core/internal/infra/web/apierror"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestLoggerMiddleware_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Logger())
	r.GET("/success", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/success", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLoggerMiddleware_ClientError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Logger())
	r.GET("/client-error", func(c *gin.Context) {
		c.Status(http.StatusBadRequest)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/client-error", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoggerMiddleware_ServerError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Logger())
	r.GET("/server-error", func(c *gin.Context) {
		c.Status(http.StatusInternalServerError)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/server-error", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestLoggerMiddleware_WithAPIError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Logger())
	r.GET("/api-error", func(c *gin.Context) {
		err := apierror.BadRequest("Bad parameter", errors.New("underlying DB error"))
		apierror.Write(c, err)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api-error", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
