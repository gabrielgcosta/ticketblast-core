package apierror

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type APIError struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Cause   error  `json:"-"`
}

func (e *APIError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *APIError) Unwrap() error {
	return e.Cause
}

func New(status int, message string, cause error) *APIError {
	return &APIError{
		Status:  status,
		Message: message,
		Cause:   cause,
	}
}

func BadRequest(message string, cause error) *APIError {
	return New(http.StatusBadRequest, message, cause)
}

func Unauthorized(message string, cause error) *APIError {
	return New(http.StatusUnauthorized, message, cause)
}

func Forbidden(message string, cause error) *APIError {
	return New(http.StatusForbidden, message, cause)
}

func NotFound(message string, cause error) *APIError {
	return New(http.StatusNotFound, message, cause)
}

func Conflict(message string, cause error) *APIError {
	return New(http.StatusConflict, message, cause)
}

func UnprocessableEntity(message string, cause error) *APIError {
	return New(http.StatusUnprocessableEntity, message, cause)
}

func Internal(message string, cause error) *APIError {
	return New(http.StatusInternalServerError, message, cause)
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func Write(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	message := "Internal server error"

	// Register error in Gin context so that monitoring/logging middleware can intercept it.
	_ = c.Error(err)

	var apiErr *APIError
	if errors.As(err, &apiErr) {
		status = apiErr.Status
		message = apiErr.Message
	} else if err != nil {
		message = err.Error()
	}

	c.JSON(status, ErrorResponse{Error: message})
}
