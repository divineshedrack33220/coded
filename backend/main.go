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
	required := []string{
		"JWT_SECRET",
		"MONGODB_URI",
	}

	for _, env := range required {
		if os.Getenv(env) == "" {
			log.Printf("⚠️ Missing required environment variable: %s", env)

			switch env {
			case "JWT_SECRET":
				os.Setenv("JWT_SECRET", "dev-secret-key-change-this-in-production")
				log.Println("⚠️ Using default JWT_SECRET (development only)")
			case "MONGODB_URI":
				os.Setenv("MONGODB_URI", "mongodb://localhost:27017")
				log.Println("⚠️ Using default local MongoDB URI")
			}
		}
	}
}

func findFrontendPath() string {
	paths := []string{
		"../frontend",      // local dev (running from backend)
		"./frontend",       // docker build context root
		"/app/frontend",    // common docker path
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func main() {
	log.Println("🚀 Starting Coded Backend Server...")

	// Load .env
	_ = godotenv.Load()

	validateEnv()

	// Connect MongoDB
	log.Println("🔌 Connecting to MongoDB...")
	if err := database.ConnectDB(); err != nil {
		log.Fatal("❌ MongoDB connection failed:", err)
	}
	log.Println("✅ MongoDB connected")

	defer func() {
		if database.Client != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			database.Client.Disconnect(ctx)
		}
	}()

	// Setup WebSocket manager
	wsManager := websocket.NewManager()
	go wsManager.Start()
	handlers.SetWebSocketManager(wsManager)

	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := routes.SetupRouter()

	// WebSocket route
	router.GET("/ws", func(c *gin.Context) {
		websocket.WebSocketHandler(wsManager)(c.Writer, c.Request)
	})

	// ----------------------------
	// STATIC FILE SERVING (FIXED)
	// ----------------------------

	log.Println("📁 Configuring static file serving...")

	frontendPath := findFrontendPath()

	if frontendPath == "" {
		log.Println("⚠️ Frontend folder not found. Running API-only mode.")
	} else {
		log.Printf("✅ Serving frontend from: %s", frontendPath)

		// Static folders
		router.Static("/asset", frontendPath+"/asset")
		router.Static("/css", frontendPath+"/css")
		router.Static("/js", frontendPath+"/js")

		// Static files
		router.StaticFile("/manifest.json", frontendPath+"/manifest.json")
		router.StaticFile("/sw.js", frontendPath+"/sw.js")
		router.StaticFile("/logo.jpeg", frontendPath+"/logo.jpeg")
		router.StaticFile("/logo.png", frontendPath+"/logo.png")

		// Root -> index.html
		router.GET("/", func(c *gin.Context) {
			c.File(frontendPath + "/index.html")
		})

		// Serve html pages dynamically
		router.GET("/:file", func(c *gin.Context) {
			file := c.Param("file")
			if len(file) > 5 && file[len(file)-5:] == ".html" {
				fullPath := frontendPath + "/" + file
				if _, err := os.Stat(fullPath); err == nil {
					c.File(fullPath)
					return
				}
			}
			c.Next()
		})

		// SPA fallback
		router.NoRoute(func(c *gin.Context) {
			if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] == "/api" {
				c.JSON(404, gin.H{
					"error": "API endpoint not found",
					"path":  c.Request.URL.Path,
				})
				return
			}

			indexPath := frontendPath + "/index.html"
			if _, err := os.Stat(indexPath); err == nil {
				c.File(indexPath)
			} else {
				c.JSON(404, gin.H{
					"error": "Page not found",
				})
			}
		})
	}

	// ----------------------------
	// SERVER CONFIG
	// ----------------------------

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("🌐 Server running on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("❌ Server error:", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Println("❌ Forced shutdown:", err)
	} else {
		log.Println("✅ Server stopped gracefully")
	}
}
