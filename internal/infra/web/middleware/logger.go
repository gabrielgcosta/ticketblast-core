package middleware

import (
	"errors"
	"time"

	"github.com/gabrielgcosta/ticketblast-core/internal/infra/web/apierror"
	"github.com/gabrielgcosta/ticketblast-core/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		fields := []zap.Field{
			zap.Int("status", status),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", c.ClientIP()),
			zap.Duration("latency", latency),
			zap.String("user-agent", c.Request.UserAgent()),
		}

		// Retrieve authenticated user ID if registered by auth middleware
		if userID, exists := c.Get("user_id"); exists {
			fields = append(fields, zap.Any("user_id", userID))
		}

		// Log errors registered in Gin context, extracting APIError root causes when present
		if len(c.Errors) > 0 {
			for _, ginErr := range c.Errors {
				errFields := append([]zap.Field{}, fields...)
				
				var apiErr *apierror.APIError
				if errors.As(ginErr.Err, &apiErr) {
					errFields = append(errFields,
						zap.String("api_error_message", apiErr.Message),
						zap.Int("api_error_status", apiErr.Status),
					)
					if apiErr.Cause != nil {
						errFields = append(errFields, zap.Error(apiErr.Cause))
					} else {
						errFields = append(errFields, zap.Error(ginErr.Err))
					}
				} else {
					errFields = append(errFields, zap.Error(ginErr.Err))
				}

				logger.Log.Error("HTTP request failed", errFields...)
			}
			return
		}

		if status >= 500 {
			logger.Log.Error("HTTP request completed with error", fields...)
		} else if status >= 400 {
			logger.Log.Warn("HTTP request completed with warning", fields...)
		} else {
			logger.Log.Info("HTTP request completed successfully", fields...)
		}
	}
}
