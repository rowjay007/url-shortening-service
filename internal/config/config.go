package config

import (
	"log"

	"github.com/rowjay/url-shortening-service/internal/constants"
	"github.com/spf13/viper"
)

type Config struct {
	BaseURL            string
	JWTSecret          string
	Environment        string
	CORSAllowedOrigins []string
	Port               string
	ShortCodeLength    int
	MaxRetries         int
}

func Load() *Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()
	viper.SetDefault("cors_allowed_origins", []string{"*"})
	viper.SetDefault("short_code_length", constants.DefaultShortCodeLength)
	viper.SetDefault("max_retries", constants.MaxRetries)

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Error reading config file, using defaults: %v", err)
	}

	corsAllowedOrigins := viper.GetStringSlice("cors_allowed_origins")
	if len(corsAllowedOrigins) == 0 {
		corsAllowedOrigins = []string{"*"}
	}

	return &Config{
		BaseURL:            viper.GetString("pocket_base_url"),
		JWTSecret:          viper.GetString("jwt_secret"),
		Environment:        viper.GetString("app_env"),
		CORSAllowedOrigins: corsAllowedOrigins,
		Port:               viper.GetString("port"),
		ShortCodeLength:    viper.GetInt("short_code_length"),
		MaxRetries:         viper.GetInt("max_retries"),
	}
}
