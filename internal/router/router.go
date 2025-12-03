package router

import (
	"os"

	"forumapp/internal/config"
	"forumapp/internal/handlers"
	"forumapp/internal/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Setup configures and returns the Gin router
func Setup(db *gorm.DB, cfg *config.Config) *gin.Engine {
	router := gin.Default()
	router.Use(cors.Default())

	// Create uploads directory if it doesn't exist
	os.MkdirAll(cfg.UploadDir, os.ModePerm)

	// Serve static files
	router.Static("/uploads", cfg.UploadDir)
	router.Static("/static", "./static")
	router.StaticFile("/", "./public/index.html")
	router.StaticFile("/favicon.svg", "./public/favicon.svg")

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(db, cfg)
	postHandler := handlers.NewPostHandler(db)
	gameHandler := handlers.NewGameHandler(db, cfg)
	dashboardHandler := handlers.NewDashboardHandler(db)
	commentHandler := handlers.NewCommentHandler(db)

	// API routes
	api := router.Group("/api")
	{
		// Authentication routes
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/appoint-moderator", middleware.AuthMiddleware(), authHandler.AppointModerator)
		}

		// Dashboard routes (protected)
		api.GET("/dashboard", middleware.AuthMiddleware(), dashboardHandler.GetDashboard)

		// Posts routes
		posts := api.Group("/posts")
		{
			posts.GET("", postHandler.GetPosts)
			posts.GET("/:id", postHandler.GetPost)
			posts.POST("", middleware.AuthMiddleware(), postHandler.CreatePost)
			posts.PUT("/:id", middleware.AuthMiddleware(), postHandler.UpdatePost)
			posts.DELETE("/:id", middleware.AuthMiddleware(), postHandler.DeletePost)
			posts.GET("/search", postHandler.SearchPosts)
			posts.GET("/game/:game_id", postHandler.GetPostsByGame)
			posts.GET("/user/:user_id", postHandler.GetUserPosts)
		}

		// Comments routes
		comments := api.Group("/comments")
		{
			comments.GET("/post/:post_id", commentHandler.GetCommentsByPost)
			comments.GET("/post/:post_id/count", commentHandler.GetCommentCount)
			comments.GET("/:id", commentHandler.GetCommentThread)
			comments.POST("", middleware.AuthMiddleware(), commentHandler.CreateComment)
			comments.PUT("/:id", middleware.AuthMiddleware(), commentHandler.UpdateComment)
			comments.DELETE("/:id", middleware.AuthMiddleware(), commentHandler.DeleteComment)
			comments.GET("/recent", commentHandler.GetRecentComments)
		}

		// Games routes - RAWG API (search-first)
		games := api.Group("/games")
		{
			// RAWG API routes (search-first, no initial load)
			games.GET("/rawg/search", gameHandler.SearchRAWGGames)
			games.GET("/rawg/:id", gameHandler.GetRAWGGameDetails)
			games.POST("/rawg/import", middleware.AuthMiddleware(), gameHandler.ImportFromRAWG)

			// Local games routes
			games.GET("", gameHandler.GetLocalGames)
			games.POST("", middleware.AuthMiddleware(), gameHandler.CreateLocalGame)
			games.GET("/tag/:tag_slug", gameHandler.GetGamesByTag)

			// Tags routes
			games.GET("/tags", gameHandler.GetAllTags)
		}
	}

	// Catch-all route for SPA - serve index.html for any unmatched routes
	router.NoRoute(func(c *gin.Context) {
		c.File("./public/index.html")
	})

	return router
}
