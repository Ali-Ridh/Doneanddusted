package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
	"gorm.io/driver/sqlite"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID        uint   `gorm:"primaryKey"`
	Username  string `gorm:"unique;not null"`
	Password  string `gorm:"not null"`
	Email     string `gorm:"unique;not null"`
	CreatedAt time.Time
}

type SteamAppList struct {
	Applist struct {
		Apps []SteamApp `json:"apps"`
	} `json:"applist"`
}

type SteamApp struct {
	Appid uint   `json:"appid"`
	Name  string `json:"name"`
}

func fetchSteamGames() ([]SteamApp, error) {
	resp, err := http.Get("https://api.steampowered.com/ISteamApps/GetAppList/v2/")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data SteamAppList
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	return data.Applist.Apps, nil
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
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte("your-secret-key"), nil
		})
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}
		userID := uint(claims["user_id"].(float64))
		c.Set("user_id", userID)
		c.Next()
	}
}

func parseUint(s string) uint {
	u, _ := strconv.ParseUint(s, 10, 32)
	return uint(u)
}

func parseInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

type Forum struct {
	ID        uint   `gorm:"primaryKey"`
	GameID    uint   `gorm:"unique;not null"`
	Name      string `gorm:"not null"`
	CreatedAt time.Time
}

type Post struct {
	ID        uint   `gorm:"primaryKey"`
	ForumID   uint   `gorm:"not null"`
	UserID    uint   `gorm:"not null"`
	Title     string `gorm:"not null"`
	Content   string `gorm:"not null"`
	CreatedAt time.Time
	User      User   `gorm:"foreignKey:UserID"`
}

type Comment struct {
	ID        uint   `gorm:"primaryKey"`
	PostID    uint   `gorm:"not null"`
	UserID    uint   `gorm:"not null"`
	Content   string `gorm:"not null"`
	CreatedAt time.Time
	User      User   `gorm:"foreignKey:UserID"`
}

func main() {
	router := gin.Default()
	router.Use(cors.Default())
	router.Static("/static", "./static")

	db, err := gorm.Open(sqlite.Open("forum.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database")
	}

	// AutoMigrate the schema
	db.AutoMigrate(&User{}, &Forum{}, &Post{}, &Comment{})

	// Fetch Steam games and populate forums
	// games, err := fetchSteamGames()
	// if err != nil {
	// 	log.Fatal("failed to fetch steam games: ", err)
	// }
	// for _, game := range games {
	// 	forum := Forum{GameID: game.Appid, Name: game.Name}
	// 	db.FirstOrCreate(&forum, Forum{GameID: game.Appid})
	// }

	// Add dummy forums for testing
	db.FirstOrCreate(&Forum{}, Forum{GameID: 1, Name: "Dummy Game 1"})
	db.FirstOrCreate(&Forum{}, Forum{GameID: 2, Name: "Dummy Game 2"})

	router.StaticFile("/", "./static/index.html")

	router.POST("/register", func(c *gin.Context) {
		var user User
		if err := c.ShouldBindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// TODO: Validate user data

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
			return
		}

		user.Password = string(hashedPassword)

		// Save user to database
		result := db.Create(&user)
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Registration successful"})
	})

	router.POST("/login", func(c *gin.Context) {
		var loginReq struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := c.ShouldBindJSON(&loginReq); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var user User
		if err := db.Where("username = ?", loginReq.Username).First(&user).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginReq.Password)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}

		// Generate JWT token
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user_id": user.ID,
			"exp":     time.Now().Add(time.Hour * 24).Unix(),
		})
		tokenString, err := token.SignedString([]byte("your-secret-key"))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"token": tokenString})
	})

	router.GET("/forums", func(c *gin.Context) {
		var forums []Forum
		if err := db.Find(&forums).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch forums"})
			return
		}
		c.JSON(http.StatusOK, forums)
	})

	router.GET("/forums/:forum_id/posts", func(c *gin.Context) {
		forumID := c.Param("forum_id")
		page := parseInt(c.DefaultQuery("page", "1"))
		limit := parseInt(c.DefaultQuery("limit", "10"))
		offset := (page - 1) * limit

		var posts []Post
		var total int64
		if err := db.Model(&Post{}).Where("forum_id = ?", forumID).Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count posts"})
			return
		}
		if err := db.Where("forum_id = ?", forumID).Preload("User").Limit(limit).Offset(offset).Find(&posts).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch posts"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"posts": posts, "total": total, "page": page, "limit": limit})
	})

	router.POST("/forums/:forum_id/posts", authMiddleware(), func(c *gin.Context) {
		forumID := c.Param("forum_id")
		userID := c.GetUint("user_id")
		var postReq struct {
			Title   string `json:"title"`
			Content string `json:"content"`
		}
		if err := c.ShouldBindJSON(&postReq); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		post := Post{
			ForumID: parseUint(forumID),
			UserID:  userID,
			Title:   postReq.Title,
			Content: postReq.Content,
		}
		if err := db.Create(&post).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create post"})
			return
		}
		c.JSON(http.StatusCreated, post)
	})

	router.GET("/posts/:post_id/comments", func(c *gin.Context) {
		postID := c.Param("post_id")
		page := parseInt(c.DefaultQuery("page", "1"))
		limit := parseInt(c.DefaultQuery("limit", "10"))
		offset := (page - 1) * limit

		var comments []Comment
		var total int64
		if err := db.Model(&Comment{}).Where("post_id = ?", postID).Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count comments"})
			return
		}
		if err := db.Where("post_id = ?", postID).Preload("User").Limit(limit).Offset(offset).Find(&comments).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch comments"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"comments": comments, "total": total, "page": page, "limit": limit})
	})

	router.POST("/posts/:post_id/comments", authMiddleware(), func(c *gin.Context) {
		postID := c.Param("post_id")
		userID := c.GetUint("user_id")
		var commentReq struct {
			Content string `json:"content"`
		}
		if err := c.ShouldBindJSON(&commentReq); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		comment := Comment{
			PostID:  parseUint(postID),
			UserID:  userID,
			Content: commentReq.Content,
		}
		if err := db.Create(&comment).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create comment"})
			return
		}
		c.JSON(http.StatusCreated, comment)
	})

	router.GET("/search", func(c *gin.Context) {
		query := c.Query("q")
		if query == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter required"})
			return
		}

		var posts []Post
		var comments []Comment

		// Search posts
		db.Where("title LIKE ? OR content LIKE ?", "%"+query+"%", "%"+query+"%").Preload("User").Find(&posts)

		// Search comments
		db.Where("content LIKE ?", "%"+query+"%").Preload("User").Find(&comments)

		c.JSON(http.StatusOK, gin.H{"posts": posts, "comments": comments})
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
	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown: ", err)
	}

	log.Println("Server exiting")
}