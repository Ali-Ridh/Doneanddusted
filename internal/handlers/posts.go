package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"forumapp/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// PostHandler handles post-related requests
type PostHandler struct {
	db *gorm.DB
}

// NewPostHandler creates a new PostHandler
func NewPostHandler(db *gorm.DB) *PostHandler {
	return &PostHandler{db: db}
}

// GetPosts returns paginated posts
func (h *PostHandler) GetPosts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 10
	}

	offset := (page - 1) * limit

	var posts []models.Post
	var total int64

	query := h.db.Model(&models.Post{}).Preload("User").Preload("Game").Preload("Game.Tags")

	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count posts"})
		return
	}

	if err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&posts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch posts"})
		return
	}

	// Get comment counts for each post
	for i := range posts {
		var count int64
		h.db.Model(&models.Comment{}).Where("post_id = ?", posts[i].ID).Count(&count)
		posts[i].CommentCount = int(count)
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
}

// GetPost returns a single post by ID
func (h *PostHandler) GetPost(c *gin.Context) {
	postID := c.Param("id")

	var post models.Post
	if err := h.db.Preload("User").Preload("Game").Preload("Game.Tags").First(&post, postID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "post not found"})
		return
	}

	// Get comment count
	var count int64
	h.db.Model(&models.Comment{}).Where("post_id = ?", post.ID).Count(&count)
	post.CommentCount = int(count)

	c.JSON(http.StatusOK, post)
}

// CreatePost creates a new post
func (h *PostHandler) CreatePost(c *gin.Context) {
	userID := c.GetUint("user_id")

	title := c.PostForm("title")
	content := c.PostForm("content")
	gameIDStr := c.PostForm("game_id")
	gameName := c.PostForm("game_name") // For backward compatibility

	if title == "" || content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title and content are required"})
		return
	}

	var gameID uint
	var game models.Game

	// Try to get game by ID first
	if gameIDStr != "" {
		id, err := strconv.ParseUint(gameIDStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid game_id"})
			return
		}
		gameID = uint(id)
		if err := h.db.First(&game, gameID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "game not found"})
			return
		}
	} else if gameName != "" {
		// Backward compatibility: find or create game by name
		slug := strings.ToLower(strings.ReplaceAll(gameName, " ", "-"))
		result := h.db.Where("slug = ?", slug).First(&game)
		if result.Error == gorm.ErrRecordNotFound {
			game = models.Game{
				Title:   gameName,
				Slug:    slug,
				IsLocal: true,
			}
			if err := h.db.Create(&game).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create game"})
				return
			}
		} else if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to find game"})
			return
		}
		gameID = game.ID
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "game_id or game_name is required"})
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
		filePath := filepath.Join("./uploads", filename)
		out, err := os.Create(filePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file"})
			return
		}
		defer out.Close()
		io.Copy(out, file)
		mediaURL = "/uploads/" + filename
	}

	post := models.Post{
		UserID:    userID,
		GameID:    &gameID,
		Title:     title,
		Content:   content,
		MediaURL:  mediaURL,
		MediaType: mediaType,
		GameTag:   gameName, // Keep for backward compatibility
	}

	if err := h.db.Create(&post).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create post"})
		return
	}

	// Load user and game for response
	h.db.Preload("User").Preload("Game").Preload("Game.Tags").First(&post, post.ID)
	c.JSON(http.StatusCreated, post)
}

// UpdatePost updates a post (only by the author)
func (h *PostHandler) UpdatePost(c *gin.Context) {
	userID := c.GetUint("user_id")
	postID := c.Param("id")

	var post models.Post
	if err := h.db.First(&post, postID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "post not found"})
		return
	}

	// Check ownership
	if post.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "you can only edit your own posts"})
		return
	}

	title := c.PostForm("title")
	content := c.PostForm("content")

	if title != "" {
		post.Title = title
	}
	if content != "" {
		post.Content = content
	}

	// Handle new file upload
	file, header, err := c.Request.FormFile("file")
	if err == nil {
		defer file.Close()

		// Determine media type
		contentType := header.Header.Get("Content-Type")
		var mediaType string
		if strings.HasPrefix(contentType, "image/") {
			mediaType = "image"
		} else if strings.HasPrefix(contentType, "video/") {
			mediaType = "video"
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported file type"})
			return
		}

		// Delete old file if exists
		if post.MediaURL != "" {
			oldPath := "." + post.MediaURL
			os.Remove(oldPath)
		}

		// Save new file
		filename := fmt.Sprintf("%d_%s", time.Now().Unix(), header.Filename)
		filePath := filepath.Join("./uploads", filename)
		out, err := os.Create(filePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file"})
			return
		}
		defer out.Close()
		io.Copy(out, file)
		post.MediaURL = "/uploads/" + filename
		post.MediaType = mediaType
	}

	if err := h.db.Save(&post).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update post"})
		return
	}

	h.db.Preload("User").Preload("Game").Preload("Game.Tags").First(&post, post.ID)
	c.JSON(http.StatusOK, post)
}

// DeletePost deletes a post (only by the author)
func (h *PostHandler) DeletePost(c *gin.Context) {
	userID := c.GetUint("user_id")
	postID := c.Param("id")

	var post models.Post
	if err := h.db.First(&post, postID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "post not found"})
		return
	}

	// Check ownership
	if post.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "you can only delete your own posts"})
		return
	}

	// Delete associated media file
	if post.MediaURL != "" {
		oldPath := "." + post.MediaURL
		os.Remove(oldPath)
	}

	// Delete all comments for this post
	h.db.Where("post_id = ?", post.ID).Delete(&models.Comment{})

	// Delete the post
	if err := h.db.Delete(&post).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete post"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "post deleted"})
}

// SearchPosts searches posts by query and/or game
func (h *PostHandler) SearchPosts(c *gin.Context) {
	query := c.Query("q")
	gameID := c.Query("game_id")
	gameTag := c.Query("game_tag") // Backward compatibility
	tagSlug := c.Query("tag")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 10
	}

	offset := (page - 1) * limit

	var posts []models.Post
	var total int64

	dbQuery := h.db.Model(&models.Post{}).Preload("User").Preload("Game").Preload("Game.Tags")

	// Apply filters
	if query != "" {
		dbQuery = dbQuery.Where("title LIKE ? OR content LIKE ?", "%"+query+"%", "%"+query+"%")
	}
	if gameID != "" {
		dbQuery = dbQuery.Where("game_id = ?", gameID)
	}
	if gameTag != "" {
		// Backward compatibility: search by game title
		dbQuery = dbQuery.Joins("JOIN games ON games.id = posts.game_id").
			Where("games.title = ?", gameTag)
	}
	if tagSlug != "" {
		// Filter by tag
		dbQuery = dbQuery.Joins("JOIN games ON games.id = posts.game_id").
			Joins("JOIN game_tags ON game_tags.game_id = games.id").
			Joins("JOIN tags ON tags.id = game_tags.tag_id").
			Where("tags.slug = ?", tagSlug)
	}

	if err := dbQuery.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count posts"})
		return
	}

	if err := dbQuery.Order("created_at DESC").Limit(limit).Offset(offset).Find(&posts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to search posts"})
		return
	}

	// Get comment counts for each post
	for i := range posts {
		var count int64
		h.db.Model(&models.Comment{}).Where("post_id = ?", posts[i].ID).Count(&count)
		posts[i].CommentCount = int(count)
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
}

// GetPostsByGame returns posts for a specific game
func (h *PostHandler) GetPostsByGame(c *gin.Context) {
	gameID := c.Param("game_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 10
	}

	offset := (page - 1) * limit

	var posts []models.Post
	var total int64

	query := h.db.Model(&models.Post{}).Where("game_id = ?", gameID).
		Preload("User").Preload("Game").Preload("Game.Tags")

	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count posts"})
		return
	}

	if err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&posts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch posts"})
		return
	}

	// Get comment counts for each post
	for i := range posts {
		var count int64
		h.db.Model(&models.Comment{}).Where("post_id = ?", posts[i].ID).Count(&count)
		posts[i].CommentCount = int(count)
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
}

// GetUserPosts returns posts by a specific user
func (h *PostHandler) GetUserPosts(c *gin.Context) {
	userID := c.Param("user_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 10
	}

	offset := (page - 1) * limit

	var posts []models.Post
	var total int64

	query := h.db.Model(&models.Post{}).Where("user_id = ?", userID).
		Preload("User").Preload("Game").Preload("Game.Tags")

	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count posts"})
		return
	}

	if err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&posts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch posts"})
		return
	}

	// Get comment counts for each post
	for i := range posts {
		var count int64
		h.db.Model(&models.Comment{}).Where("post_id = ?", posts[i].ID).Count(&count)
		posts[i].CommentCount = int(count)
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
}
