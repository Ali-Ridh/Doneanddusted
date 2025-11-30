package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRegister(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := setupRouter()

	user := map[string]string{
		"username": "testuser",
		"password": "testpass",
		"email":    "test@example.com",
	}
	jsonData, _ := json.Marshal(user)

	req, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Registration successful")
}

func setupRouter() *gin.Engine {
	router := gin.Default()
	router.Use(cors.Default())
	router.Static("/static", "./static")

	// Add routes similar to main
	router.POST("/register", func(c *gin.Context) {
		var user User
		if err := c.ShouldBindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
			return
		}

		user.Password = string(hashedPassword)

		// Mock save
		c.JSON(http.StatusOK, gin.H{"message": "Registration successful"})
	})

	return router
}