package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Post struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Title     string    `gorm:"not null" json:"title"`
	Content   string    `gorm:"not null" json:"content"`
	MediaURL  string    `json:"media_url"`
	MediaType string    `json:"media_type"` // 'image' or 'video'
	GameTag   string    `gorm:"not null" json:"game_tag"`
	CreatedAt time.Time `json:"created_at"`
}

type Game struct {
	ID    uint   `gorm:"primaryKey" json:"id"`
	Title string `gorm:"unique;not null" json:"title"`
	Slug  string `gorm:"unique;not null" json:"slug"`
}

type RAWGGame struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type RAWGSearchResponse struct {
	Results []RAWGGame `json:"results"`
}

func searchRAWGGames(query string) ([]RAWGGame, error) {
	url := fmt.Sprintf("https://api.rawg.io/api/games?key=YOUR_RAWG_API_KEY&search=%s", query)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data RAWGSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	return data.Results, nil
}

func saveGameIfNotExists(db *gorm.DB, title, slug string) error {
	var game Game
	result := db.Where("slug = ?", slug).First(&game)
	if result.Error == gorm.ErrRecordNotFound {
		game = Game{Title: title, Slug: slug}
		return db.Create(&game).Error
	}
	return result.Error
}

func main() {
	router := gin.Default()
	router.Use(cors.Default())

	// Create uploads directory if it doesn't exist
	os.MkdirAll("./uploads", os.ModePerm)

	// Serve static files
	router.Static("/uploads", "./uploads")
	router.StaticFile("/", "./public/index.html")

	db, err := gorm.Open(sqlite.Open("forum.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database")
	}

	// AutoMigrate the schema
	db.AutoMigrate(&Post{}, &Game{})

	// API routes
	router.GET("/api/posts", func(c *gin.Context) {
		var posts []Post
		if err := db.Order("created_at DESC").Find(&posts).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch posts"})
			return
		}
		c.JSON(http.StatusOK, posts)
	})

	router.POST("/api/posts", func(c *gin.Context) {
		title := c.PostForm("title")
		content := c.PostForm("content")
		gameName := c.PostForm("game_name")

		if title == "" || content == "" || gameName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "title, content, and game_name are required"})
			return
		}

		// Handle file upload
		var mediaURL, mediaType string
		file, header, err := c.Request.FormFile("file")
		if err == nil {
			defer file.Close()

			// Determine media type
			contentType := header.Header.Get("Content-Type")
			if strings.HasPrefix(contentType, "image/") {
				mediaType = "image"
			} else if strings.HasPrefix(contentType, "video/") {
				mediaType = "video"
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported file type"})
				return
			}

			// Save file
			filename := fmt.Sprintf("%d_%s", time.Now().Unix(), header.Filename)
			filepath := filepath.Join("./uploads", filename)
			out, err := os.Create(filepath)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file"})
				return
			}
			defer out.Close()
			io.Copy(out, file)
			mediaURL = "/uploads/" + filename
		}

		// Save game if not exists
		if err := saveGameIfNotExists(db, gameName, strings.ToLower(strings.ReplaceAll(gameName, " ", "-"))); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save game"})
			return
		}

		post := Post{
			Title:     title,
			Content:   content,
			MediaURL:  mediaURL,
			MediaType: mediaType,
			GameTag:   gameName,
		}

		if err := db.Create(&post).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create post"})
			return
		}

		c.JSON(http.StatusCreated, post)
	})

	router.GET("/api/games/search", func(c *gin.Context) {
		query := c.Query("q")
		if query == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter required"})
			return
		}

		games, err := searchRAWGGames(query)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to search games"})
			return
		}

		c.JSON(http.StatusOK, games)
	})

	router.Run(":8080")
}