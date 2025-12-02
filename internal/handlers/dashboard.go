package handlers

import (
	"net/http"

	"forumapp/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// DashboardHandler handles dashboard-related requests
type DashboardHandler struct {
	db *gorm.DB
}

// NewDashboardHandler creates a new DashboardHandler
func NewDashboardHandler(db *gorm.DB) *DashboardHandler {
	return &DashboardHandler{db: db}
}

// GetDashboard returns dashboard data for the authenticated user
func (h *DashboardHandler) GetDashboard(c *gin.Context) {
	userID := c.GetUint("user_id")

	// Get user stats
	var userPostCount int64
	h.db.Model(&models.Post{}).Where("user_id = ?", userID).Count(&userPostCount)

	// Get recent posts
	var recentPosts []models.Post
	h.db.Preload("User").Order("created_at DESC").Limit(10).Find(&recentPosts)

	// Get total users and posts
	var totalUsers, totalPosts int64
	h.db.Model(&models.User{}).Count(&totalUsers)
	h.db.Model(&models.Post{}).Count(&totalPosts)

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
}
