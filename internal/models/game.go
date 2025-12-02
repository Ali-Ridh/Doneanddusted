package models

import "time"

// Game represents a game stored in the local database
type Game struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	RAWGId      int       `json:"rawg_id"`
	Title       string    `gorm:"not null" json:"title"`
	Slug        string    `gorm:"unique;not null" json:"slug"`
	Description string    `json:"description"`
	CoverImage  string    `json:"cover_image"`
	Released    string    `json:"released"`
	Rating      float64   `json:"rating"`
	Metacritic  int       `json:"metacritic"`
	Playtime    int       `json:"playtime"`
	IsLocal     bool      `gorm:"default:false" json:"is_local"` // true if created locally, false if from RAWG
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Tags        []Tag     `gorm:"many2many:game_tags;" json:"tags"`
	Posts       []Post    `gorm:"foreignKey:GameID" json:"posts,omitempty"`
}

// Tag represents a tag that can be associated with games
type Tag struct {
	ID    uint   `gorm:"primaryKey" json:"id"`
	Name  string `gorm:"unique;not null" json:"name"`
	Slug  string `gorm:"unique;not null" json:"slug"`
	Games []Game `gorm:"many2many:game_tags;" json:"games,omitempty"`
}

// RAWGGame represents a game from the RAWG API
type RAWGGame struct {
	ID              int        `json:"id"`
	Name            string     `json:"name"`
	Slug            string     `json:"slug"`
	Description     string     `json:"description_raw"`
	BackgroundImage string     `json:"background_image"`
	Released        string     `json:"released"`
	Rating          float64    `json:"rating"`
	Metacritic      int        `json:"metacritic"`
	Playtime        int        `json:"playtime"`
	Genres          []RAWGTag  `json:"genres"`
	Tags            []RAWGTag  `json:"tags"`
	Platforms       []Platform `json:"platforms"`
}

// RAWGTag represents a tag/genre from RAWG API
type RAWGTag struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// Platform represents a platform from RAWG API
type Platform struct {
	Platform struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
		Slug string `json:"slug"`
	} `json:"platform"`
}

// RAWGSearchResponse represents the response from RAWG API search
type RAWGSearchResponse struct {
	Count    int        `json:"count"`
	Next     string     `json:"next"`
	Previous string     `json:"previous"`
	Results  []RAWGGame `json:"results"`
}

// CreateGameRequest represents the request body for creating a local game
type CreateGameRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
	CoverImage  string `json:"cover_image"`
	Released    string `json:"released"`
	Tags        string `json:"tags"` // Comma-separated tags
}
