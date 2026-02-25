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
)

func main() {

	log.Println("🚀 Starting Coded Backend Server")

	// ===== ENV CHECK =====
	if os.Getenv("JWT_SECRET") == "" {
		log.Fatal("JWT_SECRET is required")
	}

	if os.Getenv("MONGODB_URI") == "" {
		log.Fatal("MONGODB_URI is required")
	}

	// ===== DATABASE CONNECTION =====
	log.Println("🔌 Connecting MongoDB...")

	if err := database.ConnectDB(); err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := database.Client.Ping(ctx, nil); err != nil {
		log.Fatal("MongoDB ping failed:", err)
	}

	log.Println("✅ MongoDB Connected")

	// ===== GIN CONFIG =====
	gin.SetMode(gin.ReleaseMode)

	router := routes.SetupRouter()

	// ===== FRONTEND STATIC FILES =====

frontendPath := "frontend"

// Serve assets
router.Static("/asset", frontendPath+"/asset")
router.Static("/css", frontendPath+"/css")
router.Static("/js", frontendPath+"/js")

// Root page
router.GET("/", func(c *gin.Context) {

	indexFile := frontendPath + "/index.html"

	if _, err := os.Stat(indexFile); err == nil {
		c.File(indexFile)
		return
	}

	c.JSON(404, gin.H{
		"error": "Frontend index.html not found",
	})
})

// SPA fallback routing
router.NoRoute(func(c *gin.Context) {

	indexFile := frontendPath + "/index.html"

	if _, err := os.Stat(indexFile); err == nil {
		c.File(indexFile)
		return
	}

	c.JSON(404, gin.H{
		"error": "Page not found",
	})
})

	// ===== HEALTH CHECK =====
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "Coded Backend Running 🚀",
			"service": "healthy",
		})
	})

	// ===== WEBSOCKET =====
	wsManager := websocket.NewManager()
	go wsManager.Start()

	handlers.SetWebSocketManager(wsManager)

	router.GET("/ws", func(c *gin.Context) {
		websocket.WebSocketHandler(wsManager)(c.Writer, c.Request)
	})

	// ===== SERVER CONFIG =====
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		log.Println("🌐 Server running on port", port)

		if err := server.ListenAndServe(); err != nil &&
			err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	// ===== GRACEFUL SHUTDOWN =====
	quit := make(chan os.Signal, 1)

	signal.Notify(quit,
		syscall.SIGINT,
		syscall.SIGTERM,
	)

	<-quit

	log.Println("🛑 Shutting down server...")

	ctxShutdown, cancelShutdown := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)

	defer cancelShutdown()

	if err := server.Shutdown(ctxShutdown); err != nil {
		log.Println("Shutdown error:", err)
	}

	log.Println("👋 Server stopped")
}


