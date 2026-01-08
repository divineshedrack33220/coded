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
	"coded/routes"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or unable to load it")
	}

	// Connect to MongoDB
	if err := database.ConnectMongo(); err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}

	// Set Gin to release mode
	gin.SetMode(gin.ReleaseMode)

	// Setup Gin router using the routes package
	router := routes.SetupRouter()

	// === STATIC FILE SERVING ===
	router.Static("/asset", "../frontend/asset")
	router.StaticFile("/manifest.json", "../frontend/manifest.json")
	router.StaticFile("/sw.js", "../frontend/sw.js")

	// Serve all HTML pages
	htmlFiles := []string{
		"/", "/index.html",
		"/login.html", "/signup.html",
		"/live-requests.html", "/my-profile.html",
		"/profile-settings.html", "/chats.html",
		"/chat.html", "/post.html", "/favorites.html",
		"/view-profile.html",
	}
	for _, path := range htmlFiles {
		router.StaticFile(path, "../frontend"+path)
	}

	// Optional: SPA fallback (uncomment only if you use client-side routing later)
	// router.NoRoute(func(c *gin.Context) {
	// 	c.File("../frontend/index.html")
	// })

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
		log.Printf("Server starting on http://localhost:%s", port)
		log.Printf("Frontend served from ../frontend")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server error:", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Println("Server forced to shutdown:", err)
	} else {
		log.Println("Server stopped gracefully")
	}

	if err := database.DisconnectMongo(); err != nil {
		log.Println("Error disconnecting MongoDB:", err)
	} else {
		log.Println("MongoDB disconnected successfully")
	}

	log.Println("Application stopped")
}
