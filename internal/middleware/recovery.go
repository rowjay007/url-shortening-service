package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/rowjay/url-shortening-service/internal/models"
)

func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if err, ok := recovered.(string); ok {
			log.Error().
				Str("method", c.Request.Method).
				Str("path", c.Request.RequestURI).
				Str("client_ip", c.ClientIP()).
				Str("panic", err).
				Msg("Panic recovered")
			
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error:   "Internal server error",
				Message: err,
			})
		} else {
			log.Error().
				Str("method", c.Request.Method).
				Str("path", c.Request.RequestURI).
				Str("client_ip", c.ClientIP()).
				Interface("panic", recovered).
				Msg("Panic recovered")
			
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error: "Internal server error",
			})
		}
		c.AbortWithStatus(http.StatusInternalServerError)
	})
}
