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

    optional := map[string]string{
        "VAPID_PRIVATE_KEY": "Push notifications disabled",
        "CLOUDINARY_URL":    "Photo uploads disabled",
        "PORT":              "Using default port 8080",
    }

    for _, env := range required {
        if os.Getenv(env) == "" {
            log.Printf("⚠️  Missing required environment variable: %s", env)
            
            switch env {
            case "JWT_SECRET":
                os.Setenv("JWT_SECRET", "dev-secret-key-change-this-in-production")
                log.Println("⚠️  Using default JWT_SECRET for development")
            case "MONGODB_URI":
                os.Setenv("MONGODB_URI", "mongodb://localhost:27017")
                log.Println("⚠️  Using default MONGODB_URI: mongodb://localhost:27017")
            }
        }
    }

    for env, message := range optional {
        if os.Getenv(env) == "" {
            log.Printf("ℹ️  %s: %s", env, message)
        }
    }
}

func PrintRoutes(router *gin.Engine) {
    log.Println("📋 Registered routes:")
    routes := router.Routes()
    for i, route := range routes {
        log.Printf("  %-6s %s", route.Method, route.Path)
        if i >= 20 && i < len(routes)-5 {
            log.Printf("  ... and %d more routes", len(routes)-i-1)
            break
        }
    }
}

func main() {
    log.Println("🚀 Starting Coded Backend Server...")
    
    // Load .env file
    if err := godotenv.Load(); err != nil {
        log.Println("ℹ️  No .env file found or unable to load it")
    }

    // Validate environment variables with fallbacks
    validateEnv()

    // Connect to MongoDB with retry logic
    log.Println("🔌 Connecting to MongoDB...")
    var dbErr error
    for i := 1; i <= 3; i++ {
        if err := database.ConnectDB(); err != nil {
            dbErr = err
            log.Printf("❌ MongoDB connection attempt %d failed: %v", i, err)
            if i < 3 {
                time.Sleep(2 * time.Second)
                continue
            }
        } else {
            dbErr = nil
            break
        }
    }
    
    if dbErr != nil {
        log.Fatal("❌ Failed to connect to MongoDB after 3 attempts:", dbErr)
    }
    
    defer func() {
        if database.Client != nil {
            ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
            defer cancel()
            if err := database.Client.Disconnect(ctx); err != nil {
                log.Printf("⚠️ Error disconnecting MongoDB: %v", err)
            } else {
                log.Println("✅ MongoDB disconnected successfully")
            }
        }
    }()
    
    log.Println("✅ MongoDB connected successfully")

    // Ping the database to verify connection
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    if err := database.Client.Ping(ctx, nil); err != nil {
        log.Fatal("❌ MongoDB ping failed:", err)
    }
    log.Println("✅ MongoDB ping successful")

    // Initialize WebSocket Manager
    log.Println("🔌 Initializing WebSocket manager...")
    wsManager := websocket.NewManager()
    go wsManager.Start()
    log.Printf("✅ WebSocket manager started")

    // Pass WebSocket manager to handlers
    handlers.SetWebSocketManager(wsManager)

    // Set VAPID private key if available
    if vapidKey := os.Getenv("VAPID_PRIVATE_KEY"); vapidKey != "" {
        handlers.SetVAPIDPrivateKey(vapidKey)
        log.Println("✅ VAPID private key set")
    } else {
        log.Println("⚠️  VAPID_PRIVATE_KEY not set - push notifications disabled")
    }

    // Set Gin mode
    if os.Getenv("GIN_MODE") == "release" {
        gin.SetMode(gin.ReleaseMode)
        log.Println("⚙️  Running in RELEASE mode")
    } else {
        gin.SetMode(gin.DebugMode)
        log.Println("⚙️  Running in DEBUG mode")
    }

    // Setup router
    log.Println("🔄 Setting up routes...")
    router := routes.SetupRouter()
    
    // Add WebSocket endpoint - FIXED: Convert http.HandlerFunc to gin.HandlerFunc
    router.GET("/ws", func(c *gin.Context) {
        websocket.WebSocketHandler(wsManager)(c.Writer, c.Request)
    })
    log.Println("✅ WebSocket endpoint: /ws")
    
    // Print all registered routes
    PrintRoutes(router)

    // Static file serving - FRONTEND is at ../frontend (sibling directory)
    log.Println("📁 Configuring static file serving...")
    
    frontendPath := "../frontend"
    log.Printf("📂 Serving static files from: %s", frontendPath)

    // Check if frontend directory exists
    if _, err := os.Stat(frontendPath); os.IsNotExist(err) {
        log.Printf("❌ Frontend directory not found: %s", frontendPath)
        log.Println("⚠️  Static files will not be served - API only mode")
    } else {
        log.Println("✅ Frontend directory found")
        
        // Serve static assets
        router.Static("/asset", frontendPath+"/asset")
        router.Static("/css", frontendPath+"/css")
        router.Static("/js", frontendPath+"/js")
        router.StaticFile("/manifest.json", frontendPath+"/manifest.json")
        router.StaticFile("/sw.js", frontendPath+"/sw.js")
        router.StaticFile("/logo.jpeg", frontendPath+"/logo.jpeg")
        router.StaticFile("/logo.png", frontendPath+"/logo.png")
        
        // Serve individual HTML files
        htmlFiles := []string{
            "index.html",
            "login.html", 
            "signup.html",
            "live-requests.html",
            "my-profile.html",
            "profile-settings.html",
            "chats.html",
            "chat.html",
            "post.html",
            "favorites.html",
            "view-profile.html",
            "offline.html",
        }
        
        for _, htmlFile := range htmlFiles {
            filePath := frontendPath + "/" + htmlFile
            router.GET("/"+htmlFile, func(c *gin.Context) {
                c.File(filePath)
            })
        }
        log.Printf("✅ Serving %d HTML files", len(htmlFiles))
        
        // Serve index.html as the default route
        indexPath := frontendPath + "/index.html"
        router.GET("/", func(c *gin.Context) {
            c.File(indexPath)
        })
        log.Printf("✅ Serving: / -> %s", indexPath)
        
        // SPA fallback - serve index.html for any non-API route that doesn't exist
        router.NoRoute(func(c *gin.Context) {
            // Don't serve index.html for API routes
            if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] == "/api" {
                c.JSON(404, gin.H{
                    "error":   "API endpoint not found",
                    "path":    c.Request.URL.Path,
                    "message": "Check the API documentation for available endpoints",
                })
                return
            }
            
            // Don't serve index.html for WebSocket routes
            if c.Request.URL.Path == "/ws" {
                c.JSON(404, gin.H{
                    "error":   "WebSocket endpoint not found",
                    "path":    c.Request.URL.Path,
                })
                return
            }
            
            // For non-API routes, try to serve index.html (SPA behavior)
            if _, err := os.Stat(indexPath); err == nil {
                c.File(indexPath)
            } else {
                c.JSON(404, gin.H{
                    "error":   "Page not found",
                    "path":    c.Request.URL.Path,
                    "message": "Static file not found and no SPA fallback available",
                })
            }
        })
    }

    // Get port
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    // HTTP server configuration
    server := &http.Server{
        Addr:         ":" + port,
        Handler:      router,
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 15 * time.Second,
        IdleTimeout:  60 * time.Second,
    }

    // Channel to signal when server is ready
    serverReady := make(chan bool, 1)
    
    // Start server
    go func() {
        log.Printf("🌐 Server starting on http://localhost:%s", port)
        log.Println("")
        log.Println("🔗 Quick links:")
        log.Println("   📡 API Health:    GET  http://localhost:" + port + "/api/health")
        log.Println("   🔌 WebSocket:     GET  http://localhost:" + port + "/ws")
        log.Println("   🏠 Homepage:      GET  http://localhost:" + port + "/")
        log.Println("   🔐 Login page:    GET  http://localhost:" + port + "/login.html")
        log.Println("   💬 Chats page:    GET  http://localhost:" + port + "/chats.html")
        log.Println("")
        log.Println("📝 Test API with curl:")
        log.Println("   curl -X POST http://localhost:" + port + "/api/login \\")
        log.Println("     -H \"Content-Type: application/json\" \\")
        log.Println("     -d '{\"email\":\"test@example.com\",\"password\":\"password123\"}'")
        log.Println("")
        
        serverReady <- true
        
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatal("❌ Server error:", err)
        }
    }()

    // Wait a moment for server to start
    <-serverReady
    time.Sleep(100 * time.Millisecond)
    log.Println("✅ Server is ready and accepting connections")

    // Graceful shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    
    <-quit
    log.Println("\n🛑 Received shutdown signal...")
    
    shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer shutdownCancel()
    
    log.Println("🔄 Disconnecting WebSocket clients...")
    // WebSocket cleanup would go here if needed
    
    log.Println("🔄 Shutting down HTTP server...")
    if err := server.Shutdown(shutdownCtx); err != nil {
        log.Println("❌ Server forced to shutdown:", err)
    } else {
        log.Println("✅ Server stopped gracefully")
    }
    
    log.Println("👋 Application stopped")
}
