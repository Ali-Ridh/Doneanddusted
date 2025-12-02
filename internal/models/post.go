package models

import "time"

// Post represents a forum post
type Post struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserID       uint      `gorm:"not null" json:"user_id"`
	GameID       *uint     `json:"game_id"` // Nullable for backward compatibility
	Title        string    `gorm:"not null" json:"title"`
	Content      string    `gorm:"not null" json:"content"`
	MediaURL     string    `json:"media_url"`
	MediaType    string    `json:"media_type"` // 'image' or 'video'
	GameTag      string    `json:"game_tag"`   // Legacy field for backward compatibility
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	User         User      `gorm:"foreignKey:UserID" json:"user"`
	Game         *Game     `gorm:"foreignKey:GameID" json:"game,omitempty"`
	Comments     []Comment `gorm:"foreignKey:PostID" json:"comments,omitempty"`
	CommentCount int       `gorm:"-" json:"comment_count"` // Not stored in DB, calculated
}

// Comment represents a comment on a post with support for nested replies (Reddit-style)
type Comment struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	PostID    uint      `gorm:"not null" json:"post_id"`
	UserID    uint      `gorm:"not null" json:"user_id"`
	ParentID  *uint     `json:"parent_id"` // nil for top-level comments, set for replies
	Content   string    `gorm:"not null" json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	User      User      `gorm:"foreignKey:UserID" json:"user"`
	Post      Post      `gorm:"foreignKey:PostID" json:"-"`
	Parent    *Comment  `gorm:"foreignKey:ParentID" json:"-"`
	Replies   []Comment `gorm:"foreignKey:ParentID" json:"replies,omitempty"`
}

// CreatePostRequest represents the request body for creating a post
type CreatePostRequest struct {
	GameID  uint   `json:"game_id" binding:"required"`
	Title   string `json:"title" binding:"required"`
	Content string `json:"content" binding:"required"`
}

// CreateCommentRequest represents the request body for creating a comment
type CreateCommentRequest struct {
	PostID   uint   `json:"post_id" binding:"required"`
	ParentID *uint  `json:"parent_id"` // Optional - nil for top-level, set for replies
	Content  string `json:"content" binding:"required"`
}
