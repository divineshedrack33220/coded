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

    // Add health check endpoint for testing
    router.GET("/api/health", func(c *gin.Context) {
        c.JSON(200, gin.H{
            "status":  "ok",
            "message": "Coded API is running",
            "time":    time.Now().Unix(),
            "ws":      "WebSocket available at /ws",
            "google":  "Google OAuth available",
        })
    })

    // CORS configuration - FIXED with WebSocket support
    router.Use(cors.New(cors.Config{
        AllowOrigins:     []string{"http://localhost:8080", "http://127.0.0.1:8080", "http://localhost:5500", "http://127.0.0.1:5500", "http://localhost:3000"},
        AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
        AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Accept", "X-Requested-With"},
        ExposeHeaders:    []string{"Content-Length", "Content-Type"},
        AllowCredentials: true,
        MaxAge:           12 * time.Hour,
    }))

    // Public routes (no auth required)
    router.POST("/api/signup", handlers.Signup)
    router.POST("/api/login", handlers.Login)
    router.GET("/api/vapid-public-key", handlers.GetVapidPublicKey)
    
    // Google OAuth routes
    router.GET("/api/google/auth-url", handlers.GetGoogleAuthURL)
    router.GET("/api/google/callback", handlers.GoogleOAuthCallback)
    router.POST("/api/google-auth", handlers.GoogleAuthWithCredential)

    // Protected routes group
    protected := router.Group("/api")
    protected.Use(middleware.JWTAuthMiddleware())

    // Profile
    protected.GET("/me", handlers.GetMyProfile)
    protected.PUT("/me", handlers.UpdateMyProfile)
    protected.GET("/user/:id", handlers.GetUser)
    protected.PUT("/me/status", handlers.UpdateUserStatus)

    // Test endpoint
    protected.GET("/test-auth", handlers.TestAuth)

    // Nearby users
    protected.GET("/users/nearby", handlers.GetNearbyUsers)

    // Posts
    protected.POST("/post", handlers.CreatePost)
    protected.GET("/feed", handlers.GetFeed)
    protected.GET("/user/:id/posts", handlers.GetUserPosts)
    protected.GET("/my/posts", handlers.GetMyPosts)

    // Favorites
    protected.POST("/favorite", handlers.AddFavorite)
    protected.DELETE("/favorite", handlers.RemoveFavorite)
    protected.GET("/favorites", handlers.GetFavorites)

    // Matches
    protected.GET("/matches", handlers.GetMatches)

    // Chats
    protected.GET("/chats", handlers.GetChatList)
    protected.POST("/chats", handlers.CreateChat)
    protected.GET("/chats/:id", handlers.GetChat)

    // Messages
    protected.POST("/message", handlers.SendMessage)
    protected.GET("/messages/:chatId", handlers.GetMessages)
    protected.POST("/messages/:id/read", handlers.MarkAsRead)
    protected.POST("/typing", handlers.SendTypingIndicator) // New endpoint

    // Photo upload
    protected.POST("/upload-photo", handlers.UploadPhoto)

    // Referral
    protected.GET("/me/referral", handlers.GetReferral)

    // Push subscriptions
    protected.POST("/subscribe", handlers.SubscribePush)

    // Add a catch-all for undefined API routes
    router.NoRoute(func(c *gin.Context) {
        // If it's an API route, return JSON 404
        if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] == "/api" {
            c.JSON(404, gin.H{
                "error":   "Endpoint not found",
                "path":    c.Request.URL.Path,
                "message": "Check the API documentation for available endpoints",
            })
            return
        }
        // For WebSocket routes
        if c.Request.URL.Path == "/ws" {
            c.JSON(404, gin.H{
                "error":   "WebSocket endpoint not found",
                "path":    c.Request.URL.Path,
            })
            return
        }
        // For non-API routes, let Gin handle it
        c.Next()
    })

    return router
}