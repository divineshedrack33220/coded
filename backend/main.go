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
	log.Println("🚀 Starting Coded Backend Server...")

	// ===== REQUIRED ENV VARIABLES =====
	if os.Getenv("JWT_SECRET") == "" || os.Getenv("MONGODB_URI") == "" {
		log.Fatal("❌ JWT_SECRET and MONGODB_URI must be set in Render Environment Variables")
	}

	// ===== CONNECT DATABASE =====
	log.Println("🔌 Connecting to MongoDB...")

	var dbErr error
	for i := 1; i <= 3; i++ {
		if err := database.ConnectDB(); err != nil {
			dbErr = err
			log.Printf("❌ MongoDB connection attempt %d failed: %v", i, err)
			time.Sleep(2 * time.Second)
			continue
		}
		dbErr = nil
		break
	}

	if dbErr != nil {
		log.Fatal("❌ MongoDB connection failed:", dbErr)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := database.Client.Ping(ctx, nil); err != nil {
		log.Fatal("❌ MongoDB ping failed:", err)
	}

	log.Println("✅ MongoDB connected")

	// ===== GIN MODE =====
	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := routes.SetupRouter()

	// ===== SERVE FRONTEND STATIC FILES =====
	// IMPORTANT: Your repo must contain frontend/index.html

	router.Static("/assets", "./frontend/assets")

	router.GET("/", func(c *gin.Context) {
		c.File("./frontend/index.html")
	})

	// SPA fallback
	router.NoRoute(func(c *gin.Context) {
		c.File("./frontend/index.html")
	})

	// ===== HEALTH ROUTES =====

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

	// ===== SERVER START =====

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
		log.Printf("🌐 Server running on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	// ===== GRACEFUL SHUTDOWN =====

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down server...")

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := server.Shutdown(ctxShutdown); err != nil {
		log.Println("❌ Shutdown error:", err)
	}

	log.Println("👋 Server stopped")
}
