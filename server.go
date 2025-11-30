package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/driver/sqlite"
)

type User struct {
	ID        uint   `gorm:"primaryKey"`
	Username  string `gorm:"unique;not null"`
	Password  string `gorm:"not null"`
	Email     string `gorm:"unique;not null"`
	CreatedAt time.Time
}

func main() {
	router := gin.Default()

	db, err := gorm.Open(sqlite.Open("forum.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database")
	}

	// AutoMigrate the schema
	db.AutoMigrate(&User{})

	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Welcome to the forum app!")
	})

	router.POST("/register", func(c *gin.Context) {
		var user User
		if err := c.ShouldBindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// TODO: Validate user data

		// TODO: Hash password

		// TODO: Save user to database
		result := db.Create(&user)
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Registration successful"})
	})

	router.POST("/login", func(c *gin.Context) {
		c.String(http.StatusOK, "Login endpoint")
	})

	router.POST("/shutdown", func(c *gin.Context) {
		c.String(http.StatusOK, "Shutting down server")
		log.Println("Shutting down server...")
		os.Exit(0)
	})
	srv := &http.Server{
		Addr:    ":8082",
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan bool)
	router.POST("/shutdown", func(c *gin.Context) {
		c.String(http.StatusOK, "Shutting down server")
		log.Println("Shutting down server...")
		close(quit)
	})

	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown: ", err)
	}

	log.Println("Server exiting")
}