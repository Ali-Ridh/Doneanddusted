package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()
	quit := make(chan bool)

	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Welcome to the forum app!")
	})

	router.POST("/register", func(c *gin.Context) {
		c.String(http.StatusOK, "Register endpoint")
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

	<-quit
	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown: ", err)
	}

	log.Println("Server exiting")
}