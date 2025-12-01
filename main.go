package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Username  string    `gorm:"unique;not null" json:"username"`
	Email     string    `gorm:"unique;not null" json:"email"`
	Password  string    `gorm:"not null" json:"-"`
	CreatedAt time.Time `json:"created_at"`
}

type Post struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null" json:"user_id"`
	Title     string    `gorm:"not null" json:"title"`
	Content   string    `gorm:"not null" json:"content"`
	MediaURL  string    `json:"media_url"`
	MediaType string    `json:"media_type"` // 'image' or 'video'
	GameTag   string    `gorm:"not null" json:"game_tag"`
	CreatedAt time.Time `json:"created_at"`
	User      User      `gorm:"foreignKey:UserID" json:"user"`
}

type Game struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	Title       string `gorm:"unique;not null" json:"title"`
	Slug        string `gorm:"unique;not null" json:"slug"`
	CoverImage  string `json:"cover_image"`
}

type RAWGGame struct {
	ID               int    `json:"id"`
	Name             string `json:"name"`
	Slug             string `json:"slug"`
	BackgroundImage  string `json:"background_image"`
}

type RAWGSearchResponse struct {
	Results []RAWGGame `json:"results"`
}

type AuthClaims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
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

func saveGameWithCoverIfNotExists(db *gorm.DB, title, slug, coverImage string) error {
	var game Game
	result := db.Where("slug = ?", slug).First(&game)
	if result.Error == gorm.ErrRecordNotFound {
		game = Game{Title: title, Slug: slug, CoverImage: coverImage}
		return db.Create(&game).Error
	}
	return result.Error
}

func downloadAndCacheImage(imageURL, filename string) error {
	resp, err := http.Get(imageURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath.Join("./uploads", filename))
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			c.Abort()
			return
		}

		// Remove "Bearer " if present
		if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
			tokenString = tokenString[7:]
		}

		token, err := jwt.ParseWithClaims(tokenString, &AuthClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte("your-secret-key"), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(*AuthClaims); ok {
			c.Set("user_id", claims.UserID)
			c.Set("username", claims.Username)
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		c.Next()
	}
}

func generateJWT(userID uint, username string) (string, error) {
	claims := AuthClaims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte("your-secret-key"))
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
	db.AutoMigrate(&User{}, &Post{}, &Game{})

	// Authentication routes
	router.POST("/api/auth/register", func(c *gin.Context) {
		var req struct {
			Username string `json:"username" binding:"required"`
			Email    string `json:"email" binding:"required,email"`
			Password string `json:"password" binding:"required,min=6"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Check if user exists
		var existingUser User
		if err := db.Where("username = ? OR email = ?", req.Username, req.Email).First(&existingUser).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "user already exists"})
			return
		}

		// Hash password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
			return
		}

		user := User{
			Username: req.Username,
			Email:    req.Email,
			Password: string(hashedPassword),
		}

		if err := db.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
			return
		}

		token, err := generateJWT(user.ID, user.Username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"token": token, "user": gin.H{"id": user.ID, "username": user.Username, "email": user.Email}})
	})

	router.POST("/api/auth/login", func(c *gin.Context) {
		var req struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var user User
		if err := db.Where("username = ?", req.Username).First(&user).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}

		token, err := generateJWT(user.ID, user.Username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"token": token, "user": gin.H{"id": user.ID, "username": user.Username, "email": user.Email}})
	})

	// Protected routes
	router.GET("/api/dashboard", authMiddleware(), func(c *gin.Context) {
		userID := c.GetUint("user_id")

		// Get user stats
		var userPostCount int64
		db.Model(&Post{}).Where("user_id = ?", userID).Count(&userPostCount)

		// Get recent posts
		var recentPosts []Post
		db.Preload("User").Order("created_at DESC").Limit(10).Find(&recentPosts)

		// Get total users and posts
		var totalUsers, totalPosts int64
		db.Model(&User{}).Count(&totalUsers)
		db.Model(&Post{}).Count(&totalPosts)

		c.JSON(http.StatusOK, gin.H{
			"user_stats": gin.H{
				"post_count": userPostCount,
			},
			"recent_posts": recentPosts,
			"global_stats": gin.H{
				"total_users": totalUsers,
				"total_posts": totalPosts,
			},
		})
	})

	// Posts routes
	router.GET("/api/posts", func(c *gin.Context) {
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

		if page < 1 {
			page = 1
		}
		if limit < 1 || limit > 50 {
			limit = 10
		}

		offset := (page - 1) * limit

		var posts []Post
		var total int64

		query := db.Model(&Post{}).Preload("User")

		if err := query.Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count posts"})
			return
		}

		if err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&posts).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch posts"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"posts": posts,
			"pagination": gin.H{
				"page":  page,
				"limit": limit,
				"total": total,
				"pages": (total + int64(limit) - 1) / int64(limit),
			},
		})
	})

	router.POST("/api/posts", authMiddleware(), func(c *gin.Context) {
		userID := c.GetUint("user_id")

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
			UserID:    userID,
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

		// Load user for response
		db.Preload("User").First(&post, post.ID)
		c.JSON(http.StatusCreated, post)
	})

	router.GET("/api/posts/search", func(c *gin.Context) {
		query := c.Query("q")
		gameTag := c.Query("game_tag")
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

		if page < 1 {
			page = 1
		}
		if limit < 1 || limit > 50 {
			limit = 10
		}

		offset := (page - 1) * limit

		var posts []Post
		var total int64

		dbQuery := db.Model(&Post{}).Preload("User")

		// Apply filters
		if query != "" {
			dbQuery = dbQuery.Where("title LIKE ? OR content LIKE ?", "%"+query+"%", "%"+query+"%")
		}
		if gameTag != "" {
			dbQuery = dbQuery.Where("game_tag = ?", gameTag)
		}

		if err := dbQuery.Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count posts"})
			return
		}

		if err := dbQuery.Order("created_at DESC").Limit(limit).Offset(offset).Find(&posts).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to search posts"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"posts": posts,
			"pagination": gin.H{
				"page":  page,
				"limit": limit,
				"total": total,
				"pages": (total + int64(limit) - 1) / int64(limit),
			},
		})
	})

	// Games routes
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

		// Cache games and download cover images
		for _, rawgGame := range games {
			if rawgGame.BackgroundImage != "" {
				// Download and cache cover image
				imageFilename := fmt.Sprintf("game_%d_cover.jpg", rawgGame.ID)
				if err := downloadAndCacheImage(rawgGame.BackgroundImage, imageFilename); err == nil {
					rawgGame.BackgroundImage = "/uploads/" + imageFilename
				}
			}

			// Save game to database
			saveGameWithCoverIfNotExists(db, rawgGame.Name, rawgGame.Slug, rawgGame.BackgroundImage)
		}

		c.JSON(http.StatusOK, games)
	})

	router.GET("/api/games", func(c *gin.Context) {
		var games []Game
		if err := db.Find(&games).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch games"})
			return
		}
		c.JSON(http.StatusOK, games)
	})

	router.Run(":8080")
}