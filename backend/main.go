package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"coded/database"
	"coded/handlers"
	"coded/routes"
	"coded/websocket"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func validateEnv() {
	required := []string{"JWT_SECRET"}

	for _, env := range required {
		if os.Getenv(env) == "" {
			log.Printf("⚠️  Missing required environment variable: %s", env)
			switch env {
			case "JWT_SECRET":
				os.Setenv("JWT_SECRET", "dev-secret-key-change-this-in-production")
				log.Println("⚠️  Using default JWT_SECRET for development")
			}
		}
	}

	if os.Getenv("MONGODB_URI") == "" {
		log.Println("⚠️  MONGODB_URI not set - using default")
		os.Setenv("MONGODB_URI", "mongodb://localhost:27017")
	}
}

func main() {
	log.Println("🚀 Starting InstaPing Backend Server...")

	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("ℹ️  No .env file found or unable to load it")
	}

	// Validate environment variables
	validateEnv()

	// Set Gin mode
	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
		log.Println("⚙️  Running in RELEASE mode")
	} else {
		gin.SetMode(gin.DebugMode)
		log.Println("⚙️  Running in DEBUG mode")
	}

	// Connect to MongoDB
	log.Println("🔌 Connecting to MongoDB...")
	if err := database.ConnectDB(); err != nil {
		log.Printf("⚠️  MongoDB connection failed: %v", err)
		log.Println("⚠️  Running without MongoDB (some features may not work)")
	} else {
		defer func() {
			if database.Client != nil {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				database.Client.Disconnect(ctx)
				log.Println("✅ MongoDB disconnected")
			}
		}()
		log.Println("✅ MongoDB connected successfully")
	}

	// Initialize WebSocket Manager
	log.Println("🔌 Initializing WebSocket manager...")
	wsManager := websocket.NewManager()
	go wsManager.Start()
	handlers.SetWebSocketManager(wsManager)
	log.Println("✅ WebSocket manager started")

	// Setup router
	log.Println("🔄 Setting up routes...")
	router := routes.SetupRouter()

	// Add WebSocket endpoint
	router.GET("/ws", websocket.WebSocketHandler(wsManager))

	// Home route
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": "InstaPing Backend API",
			"version": "1.0.0",
			"status":  "running",
			"endpoints": gin.H{
				"health":   "/api/health",
				"login":    "/api/login",
				"signup":   "/api/signup",
				"websocket": "/ws",
			},
		})
	})

	// Get port
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// HTTP server
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server
	go func() {
		log.Printf("🌐 Server starting on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("❌ Server error:", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("\n🛑 Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Println("❌ Server forced to shutdown:", err)
	} else {
		log.Println("✅ Server stopped gracefully")
	}
}
