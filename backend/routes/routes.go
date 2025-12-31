// File: routes/routes.go

package routes

import (
	"coded/handlers"
	"coded/middleware"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	router := gin.Default()

	// CORS configuration - FIXED
	router.Use(cors.New(cors.Config{
		// Explicitly list allowed origins (no wildcard when AllowCredentials is true)
		AllowOrigins:     []string{"http://localhost:8080", "http://127.0.0.1:8080", "http://localhost:5500", "http://127.0.0.1:5500"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,                  // Important for JWT in cookies or Authorization header
		MaxAge:           12 * time.Hour,
	}))

	// Public routes (no auth required)
	router.POST("/signup", handlers.Signup)
	router.POST("/login", handlers.Login)

	// Protected routes group
	protected := router.Group("/")
	protected.Use(middleware.JWTAuthMiddleware())

	// Profile
	protected.GET("/me", handlers.GetMyProfile)
	protected.PUT("/me", handlers.UpdateMyProfile)
	protected.GET("/user/:id", handlers.GetUser)

	// Posts
	protected.POST("/post", handlers.CreatePost)
	protected.GET("/feed", handlers.GetFeed)
	protected.GET("/user/:id/posts", handlers.GetUserPosts)
	protected.GET("/my/posts", handlers.GetMyPosts) // Optional: if you want a direct /my/posts

	// Favorites
	protected.POST("/favorite", handlers.AddFavorite)
	protected.DELETE("/favorite", handlers.RemoveFavorite)
	protected.GET("/favorites", handlers.GetFavorites)

	// Matches (placeholder)
	protected.GET("/matches", handlers.GetMatches)

	// Chats
	protected.GET("/chats", handlers.GetChatList)
	protected.POST("/chats", handlers.CreateChat)
	protected.GET("/chats/:id", handlers.GetChat)

	// Messages
	protected.POST("/message", handlers.SendMessage)
	protected.GET("/messages/:chatId", handlers.GetMessages)
	protected.POST("/messages/:id/read", handlers.MarkAsRead)

	// Photo upload (used in onboarding/profile)
	protected.POST("/upload-photo", handlers.UploadPhoto)

	// Referral (optional but useful)
	protected.GET("/me/referral", handlers.GetReferral)

	return router
}