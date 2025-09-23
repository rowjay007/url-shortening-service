package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func Logger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		logger := log.Info().
			Str("client_ip", param.ClientIP).
			Str("method", param.Method).
			Str("path", param.Path).
			Str("protocol", param.Request.Proto).
			Int("status_code", param.StatusCode).
			Dur("latency", param.Latency).
			Str("user_agent", param.Request.UserAgent()).
			Time("timestamp", param.TimeStamp)
		
		if param.ErrorMessage != "" {
			logger = logger.Str("error", param.ErrorMessage)
		}
		
		logger.Msg("HTTP Request")
		
		return ""
	})
}
