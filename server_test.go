package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"forumapp/internal/config"
	"forumapp/internal/database"
	"forumapp/internal/middleware"
	"forumapp/internal/router"

	"github.com/stretchr/testify/assert"
)

func setupTestRouter() http.Handler {
	// Load test configuration
	cfg := &config.Config{
		Port:         "8080",
		DatabasePath: ":memory:",
		JWTSecret:    "test-secret-key",
		RAWGAPIKey:   "test-api-key",
		UploadDir:    "./uploads",
	}

	// Initialize JWT secret
	middleware.SetJWTSecret(cfg)

	// Initialize database
	db := database.Initialize(cfg)

	// Setup router
	return router.Setup(db, cfg)
}

func TestRegister(t *testing.T) {
	r := setupTestRouter()

	user := map[string]string{
		"username": "testuser",
		"password": "testpass123",
		"email":    "test@example.com",
	}
	jsonData, _ := json.Marshal(user)

	req, _ := http.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "token")
}

func TestLogin(t *testing.T) {
	r := setupTestRouter()

	// First register a user
	user := map[string]string{
		"username": "loginuser",
		"password": "testpass123",
		"email":    "login@example.com",
	}
	jsonData, _ := json.Marshal(user)

	req, _ := http.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Now test login
	loginData := map[string]string{
		"username": "loginuser",
		"password": "testpass123",
	}
	jsonData, _ = json.Marshal(loginData)

	req, _ = http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "token")
}

func TestGetPosts(t *testing.T) {
	r := setupTestRouter()

	req, _ := http.NewRequest("GET", "/api/posts", nil)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "posts")
	assert.Contains(t, w.Body.String(), "pagination")
}

func TestGetGames(t *testing.T) {
	r := setupTestRouter()

	req, _ := http.NewRequest("GET", "/api/games", nil)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
