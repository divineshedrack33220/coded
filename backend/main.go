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
	jwt := os.Getenv("JWT_SECRET")
	mongo := os.Getenv("MONGODB_URI")

	if jwt == "" || mongo == "" {
		log.Fatal("❌ JWT_SECRET and MONGODB_URI must be set in Render Environment Variables")
	}

	// ===== CONNECT TO MONGODB WITH RETRY =====
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
		log.Fatal("❌ Failed to connect to MongoDB:", dbErr)
	}

	log.Println("✅ MongoDB connected successfully")

	// Ping DB
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := database.Client.Ping(ctx, nil); err != nil {
		log.Fatal("❌ MongoDB ping failed:", err)
	}

	log.Println("✅ MongoDB ping successful")

	// ===== GIN MODE =====
	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
		log.Println("⚙️ Running in RELEASE mode")
	} else {
		gin.SetMode(gin.DebugMode)
		log.Println("⚙️ Running in DEBUG mode")
	}

	// ===== ROUTER =====
	router := routes.SetupRouter()

	// Health check (IMPORTANT for Render)
	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "Coded Backend Running 🚀",
			"service": "healthy",
		})
	})

	router.GET("/health", func(c *gin.Context) {
		c.String(200, "OK")
	})

	// ===== WEBSOCKET =====
	log.Println("🔌 Initializing WebSocket manager...")
	wsManager := websocket.NewManager()
	go wsManager.Start()

	handlers.SetWebSocketManager(wsManager)

	router.GET("/ws", func(c *gin.Context) {
		websocket.WebSocketHandler(wsManager)(c.Writer, c.Request)
	})

	log.Println("✅ WebSocket endpoint: /ws")

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
		IdleTimeout:  60 * time.Second,
	}

	// Start server
	go func() {
		log.Printf("🌐 Server running on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("❌ Server error:", err)
		}
	}()

	log.Println("✅ Server is ready and accepting connections")

	// ===== GRACEFUL SHUTDOWN =====
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Println("❌ Forced shutdown:", err)
	}

	log.Println("👋 Server stopped gracefully")
}
