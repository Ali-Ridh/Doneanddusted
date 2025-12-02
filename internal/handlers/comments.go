package handlers

import (
	"net/http"
	"strconv"

	"forumapp/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CommentHandler handles comment-related requests
type CommentHandler struct {
	db *gorm.DB
}

// NewCommentHandler creates a new CommentHandler
func NewCommentHandler(db *gorm.DB) *CommentHandler {
	return &CommentHandler{db: db}
}

// GetCommentsByPost returns all comments for a post with nested replies
func (h *CommentHandler) GetCommentsByPost(c *gin.Context) {
	postID := c.Param("post_id")
	if postID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "post_id is required"})
		return
	}

	// Get only top-level comments (ParentID is null)
	var comments []models.Comment
	if err := h.db.Where("post_id = ? AND parent_id IS NULL", postID).
		Preload("User").
		Preload("Replies").
		Preload("Replies.User").
		Preload("Replies.Replies").
		Preload("Replies.Replies.User").
		Preload("Replies.Replies.Replies").
		Preload("Replies.Replies.Replies.User").
		Order("created_at DESC").
		Find(&comments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch comments"})
		return
	}

	c.JSON(http.StatusOK, comments)
}

// GetCommentThread returns a specific comment with all its nested replies
func (h *CommentHandler) GetCommentThread(c *gin.Context) {
	commentID := c.Param("id")
	if commentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "comment id is required"})
		return
	}

	var comment models.Comment
	if err := h.db.Where("id = ?", commentID).
		Preload("User").
		Preload("Replies").
		Preload("Replies.User").
		Preload("Replies.Replies").
		Preload("Replies.Replies.User").
		Preload("Replies.Replies.Replies").
		Preload("Replies.Replies.Replies.User").
		First(&comment).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "comment not found"})
		return
	}

	c.JSON(http.StatusOK, comment)
}

// CreateComment creates a new comment or reply
func (h *CommentHandler) CreateComment(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req models.CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify post exists
	var post models.Post
	if err := h.db.First(&post, req.PostID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "post not found"})
		return
	}

	// If this is a reply, verify parent comment exists and belongs to same post
	if req.ParentID != nil {
		var parentComment models.Comment
		if err := h.db.First(&parentComment, *req.ParentID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "parent comment not found"})
			return
		}
		if parentComment.PostID != req.PostID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "parent comment does not belong to this post"})
			return
		}
	}

	comment := models.Comment{
		PostID:   req.PostID,
		UserID:   userID,
		ParentID: req.ParentID,
		Content:  req.Content,
	}

	if err := h.db.Create(&comment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create comment"})
		return
	}

	// Load user for response
	h.db.Preload("User").First(&comment, comment.ID)
	c.JSON(http.StatusCreated, comment)
}

// UpdateComment updates a comment (only by the author)
func (h *CommentHandler) UpdateComment(c *gin.Context) {
	userID := c.GetUint("user_id")
	commentID := c.Param("id")

	var comment models.Comment
	if err := h.db.First(&comment, commentID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "comment not found"})
		return
	}

	// Check ownership
	if comment.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "you can only edit your own comments"})
		return
	}

	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	comment.Content = req.Content
	if err := h.db.Save(&comment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update comment"})
		return
	}

	h.db.Preload("User").First(&comment, comment.ID)
	c.JSON(http.StatusOK, comment)
}

// DeleteComment deletes a comment (only by the author)
func (h *CommentHandler) DeleteComment(c *gin.Context) {
	userID := c.GetUint("user_id")
	commentID := c.Param("id")

	var comment models.Comment
	if err := h.db.First(&comment, commentID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "comment not found"})
		return
	}

	// Check ownership
	if comment.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "you can only delete your own comments"})
		return
	}

	// Delete all replies recursively
	h.deleteCommentAndReplies(comment.ID)

	c.JSON(http.StatusOK, gin.H{"message": "comment deleted"})
}

// deleteCommentAndReplies recursively deletes a comment and all its replies
func (h *CommentHandler) deleteCommentAndReplies(commentID uint) {
	// Find all direct replies
	var replies []models.Comment
	h.db.Where("parent_id = ?", commentID).Find(&replies)

	// Recursively delete replies
	for _, reply := range replies {
		h.deleteCommentAndReplies(reply.ID)
	}

	// Delete the comment itself
	h.db.Delete(&models.Comment{}, commentID)
}

// GetCommentCount returns the total number of comments for a post
func (h *CommentHandler) GetCommentCount(c *gin.Context) {
	postID := c.Param("post_id")
	if postID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "post_id is required"})
		return
	}

	var count int64
	if err := h.db.Model(&models.Comment{}).Where("post_id = ?", postID).Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count comments"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}

// GetRecentComments returns recent comments across all posts
func (h *CommentHandler) GetRecentComments(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if limit < 1 || limit > 50 {
		limit = 10
	}

	var comments []models.Comment
	if err := h.db.Preload("User").Preload("Post").
		Order("created_at DESC").
		Limit(limit).
		Find(&comments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch comments"})
		return
	}

	c.JSON(http.StatusOK, comments)
}
