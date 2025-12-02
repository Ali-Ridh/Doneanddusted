package config

import "os"

// Config holds all configuration for the application
type Config struct {
	Port         string
	DatabasePath string
	JWTSecret    string
	RAWGAPIKey   string
	UploadDir    string
	ModeratorKey string
}

// Load returns the application configuration
func Load() *Config {
	return &Config{
		Port:         getEnv("PORT", "8080"),
		DatabasePath: getEnv("DATABASE_PATH", "forum.db"),
		JWTSecret:    getEnv("JWT_SECRET", "your-secret-key"),
		RAWGAPIKey:   getEnv("RAWG_API_KEY", "5e3f8883fe504827bf672e7bc73cbdee"),
		UploadDir:    getEnv("UPLOAD_DIR", "./uploads"),
		ModeratorKey: getEnv("MODERATOR_KEY", "moderator123"),
	}
}

// getEnv returns the value of an environment variable or a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
