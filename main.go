package main

import (
	"fmt"
	"log"

	"forumapp/internal/config"
	"forumapp/internal/database"
	"forumapp/internal/middleware"
	"forumapp/internal/router"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize JWT secret
	middleware.SetJWTSecret(cfg)

	// Initialize database
	db := database.Initialize(cfg)

	// Setup router
	r := router.Setup(db, cfg)

	// Start server
	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("Server starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal("Failed to start server: ", err)
	}
}
