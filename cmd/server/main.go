package main

import (
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rowjay/url-shortening-service/internal/config"
	"github.com/rowjay/url-shortening-service/internal/database"
	"github.com/rowjay/url-shortening-service/internal/dto"
	"github.com/rowjay/url-shortening-service/internal/handlers"
	"github.com/rowjay/url-shortening-service/internal/middleware"
	"github.com/rowjay/url-shortening-service/internal/repository"
	"github.com/rowjay/url-shortening-service/internal/services"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	if os.Getenv("ENVIRONMENT") == "development" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}
	
	if err := godotenv.Load(); err != nil {
		log.Info().Msg("No .env file found")
	}

	cfg := config.Load()
	
	log.Info().Str("pocketbase_url", cfg.BaseURL).Msg("Starting URL shortening service")

	pb, err := database.Initialize(cfg.BaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize PocketBase")
	}

	if err := pb.CreateCollection(); err != nil {
		log.Warn().Err(err).Msg("Collection might already exist or there was an issue creating it")
	}

	urlRepo := repository.NewURLRepository(pb)
	urlService := services.NewURLService(urlRepo)
	urlHandler := handlers.NewURLHandler(urlService)

	r := gin.Default()
	r.Use(middleware.Logger())
	r.Use(middleware.CORS())

	r.POST("/api/v1/shorten", urlHandler.CreateShortURL)
	r.GET("/api/v1/shorten/:shortCode", urlHandler.GetOriginalURL)
	r.PUT("/api/v1/shorten/:shortCode", urlHandler.UpdateShortURL)
	r.DELETE("/api/v1/shorten/:shortCode", urlHandler.DeleteShortURL)
	r.GET("/api/v1/shorten/:shortCode/stats", urlHandler.GetStatistics)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, dto.HealthResponse{Status: "ok"})
	})

	log.Info().Str("port", cfg.Port).Msg("Server starting")
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal().Err(err).Msg("Failed to start server")
	}
}
