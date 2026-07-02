package apierror

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestAPIError_ErrorFormatting(t *testing.T) {
	causeErr := errors.New("database connection failed")
	apiErr := New(http.StatusInternalServerError, "Internal failure", causeErr)

	assert.Equal(t, "Internal failure: database connection failed", apiErr.Error())
	assert.Equal(t, causeErr, apiErr.Unwrap())

	apiErrNoCause := New(http.StatusBadRequest, "Invalid request", nil)
	assert.Equal(t, "Invalid request", apiErrNoCause.Error())
}

func TestWrite_APIErrorType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	err := BadRequest("Invalid parameters", errors.New("bad field"))
	Write(c, err)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.JSONEq(t, `{"error":"Invalid parameters"}`, w.Body.String())
	assert.Len(t, c.Errors, 1)
	assert.Equal(t, err, c.Errors[0].Err)
}

func TestWrite_GenericErrorType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	err := errors.New("something went wrong")
	Write(c, err)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.JSONEq(t, `{"error":"something went wrong"}`, w.Body.String())
	assert.Len(t, c.Errors, 1)
	assert.Equal(t, err, c.Errors[0].Err)
}
