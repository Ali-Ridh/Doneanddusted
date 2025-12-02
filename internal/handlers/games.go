package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"forumapp/internal/config"
	"forumapp/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GameHandler handles game-related requests
type GameHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

// NewGameHandler creates a new GameHandler
func NewGameHandler(db *gorm.DB, cfg *config.Config) *GameHandler {
	return &GameHandler{
		db:  db,
		cfg: cfg,
	}
}

// SearchRAWGGames searches for games using the RAWG API (external)
// This is search-first: only fetches when user provides a query
func (h *GameHandler) SearchRAWGGames(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "search query 'q' is required"})
		return
	}

	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")

	games, err := h.fetchFromRAWG(query, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to search games from RAWG: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, games)
}

// GetRAWGGameDetails gets detailed info about a specific game from RAWG
func (h *GameHandler) GetRAWGGameDetails(c *gin.Context) {
	gameID := c.Param("id")
	if gameID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "game ID is required"})
		return
	}

	url := fmt.Sprintf("https://api.rawg.io/api/games/%s?key=%s", gameID, h.cfg.RAWGAPIKey)
	resp, err := http.Get(url)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch game details"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusNotFound, gin.H{"error": "game not found"})
		return
	}

	var game models.RAWGGame
	if err := json.NewDecoder(resp.Body).Decode(&game); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse game details"})
		return
	}

	c.JSON(http.StatusOK, game)
}

// CreateLocalGame creates a new game in the local database
func (h *GameHandler) CreateLocalGame(c *gin.Context) {
	var req models.CreateGameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create slug from title
	slug := strings.ToLower(strings.ReplaceAll(req.Title, " ", "-"))

	// Check if game already exists
	var existingGame models.Game
	if err := h.db.Where("slug = ?", slug).First(&existingGame).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "game with this title already exists"})
		return
	}

	game := models.Game{
		Title:       req.Title,
		Slug:        slug,
		Description: req.Description,
		CoverImage:  req.CoverImage,
		Released:    req.Released,
		IsLocal:     true,
	}

	// Parse and create tags
	if req.Tags != "" {
		tagNames := strings.Split(req.Tags, ",")
		for _, tagName := range tagNames {
			tagName = strings.TrimSpace(tagName)
			if tagName == "" {
				continue
			}
			tagSlug := strings.ToLower(strings.ReplaceAll(tagName, " ", "-"))

			var tag models.Tag
			// Find or create tag
			if err := h.db.Where("slug = ?", tagSlug).First(&tag).Error; err == gorm.ErrRecordNotFound {
				tag = models.Tag{Name: tagName, Slug: tagSlug}
				h.db.Create(&tag)
			}
			game.Tags = append(game.Tags, tag)
		}
	}

	if err := h.db.Create(&game).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create game"})
		return
	}

	// Reload with tags
	h.db.Preload("Tags").First(&game, game.ID)
	c.JSON(http.StatusCreated, game)
}

// ImportFromRAWG imports a game from RAWG to local database
func (h *GameHandler) ImportFromRAWG(c *gin.Context) {
	var req struct {
		RawgID int `json:"rawg_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "rawg_id is required"})
		return
	}
	rawgID := strconv.Itoa(req.RawgID)

	// Fetch game details from RAWG
	url := fmt.Sprintf("https://api.rawg.io/api/games/%s?key=%s", rawgID, h.cfg.RAWGAPIKey)
	resp, err := http.Get(url)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch game from RAWG"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusNotFound, gin.H{"error": "game not found on RAWG"})
		return
	}

	var rawgGame models.RAWGGame
	if err := json.NewDecoder(resp.Body).Decode(&rawgGame); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse RAWG response"})
		return
	}

	// Check if already imported
	var existingGame models.Game
	if err := h.db.Where("rawg_id = ?", rawgGame.ID).First(&existingGame).Error; err == nil {
		c.JSON(http.StatusOK, gin.H{"message": "game already imported", "game": existingGame})
		return
	}

	// Create local game from RAWG data
	game := models.Game{
		RAWGId:      rawgGame.ID,
		Title:       rawgGame.Name,
		Slug:        rawgGame.Slug,
		Description: rawgGame.Description,
		CoverImage:  rawgGame.BackgroundImage,
		Released:    rawgGame.Released,
		Rating:      rawgGame.Rating,
		Metacritic:  rawgGame.Metacritic,
		Playtime:    rawgGame.Playtime,
		IsLocal:     false,
	}

	// Import tags from RAWG
	for _, rawgTag := range rawgGame.Tags {
		var tag models.Tag
		if err := h.db.Where("slug = ?", rawgTag.Slug).First(&tag).Error; err == gorm.ErrRecordNotFound {
			tag = models.Tag{Name: rawgTag.Name, Slug: rawgTag.Slug}
			h.db.Create(&tag)
		}
		game.Tags = append(game.Tags, tag)
	}

	// Also import genres as tags
	for _, genre := range rawgGame.Genres {
		var tag models.Tag
		if err := h.db.Where("slug = ?", genre.Slug).First(&tag).Error; err == gorm.ErrRecordNotFound {
			tag = models.Tag{Name: genre.Name, Slug: genre.Slug}
			h.db.Create(&tag)
		}
		game.Tags = append(game.Tags, tag)
	}

	if err := h.db.Create(&game).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to import game"})
		return
	}

	h.db.Preload("Tags").First(&game, game.ID)
	c.JSON(http.StatusCreated, game)
}

// GetLocalGames returns all games from the local database
func (h *GameHandler) GetLocalGames(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 20
	}

	offset := (page - 1) * limit

	var games []models.Game
	var total int64

	query := h.db.Model(&models.Game{}).Preload("Tags")

	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count games"})
		return
	}

	if err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&games).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch games"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"games": games,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// GetLocalGameByID returns a specific game from the local database
func (h *GameHandler) GetLocalGameByID(c *gin.Context) {
	id := c.Param("id")

	var game models.Game
	if err := h.db.Preload("Tags").Preload("Posts").Preload("Posts.User").First(&game, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "game not found"})
		return
	}

	c.JSON(http.StatusOK, game)
}

// GetGamesByTag returns games filtered by tag
func (h *GameHandler) GetGamesByTag(c *gin.Context) {
	tagSlug := c.Param("tag")
	if tagSlug == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tag is required"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 20
	}

	offset := (page - 1) * limit

	var tag models.Tag
	if err := h.db.Where("slug = ?", tagSlug).First(&tag).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tag not found"})
		return
	}

	var games []models.Game
	var total int64

	// Count games with this tag
	h.db.Model(&models.Game{}).
		Joins("JOIN game_tags ON game_tags.game_id = games.id").
		Where("game_tags.tag_id = ?", tag.ID).
		Count(&total)

	// Get games with this tag
	h.db.Preload("Tags").
		Joins("JOIN game_tags ON game_tags.game_id = games.id").
		Where("game_tags.tag_id = ?", tag.ID).
		Order("games.created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&games)

	c.JSON(http.StatusOK, gin.H{
		"tag":   tag,
		"games": games,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// GetAllTags returns all tags
func (h *GameHandler) GetAllTags(c *gin.Context) {
	var tags []models.Tag
	if err := h.db.Order("name ASC").Find(&tags).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tags"})
		return
	}
	c.JSON(http.StatusOK, tags)
}

// fetchFromRAWG fetches games from the RAWG API
func (h *GameHandler) fetchFromRAWG(query, page, pageSize string) (*models.RAWGSearchResponse, error) {
	// URL-encode the search query to handle spaces and special characters
	encodedQuery := url.QueryEscape(query)
	apiURL := fmt.Sprintf("https://api.rawg.io/api/games?key=%s&search=%s&page=%s&page_size=%s",
		h.cfg.RAWGAPIKey, encodedQuery, page, pageSize)

	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("RAWG API returned status %d", resp.StatusCode)
	}

	var data models.RAWGSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	return &data, nil
}
