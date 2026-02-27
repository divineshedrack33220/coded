package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"coded/database"
	"coded/middleware"
	"coded/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

type SignupRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func Signup(c *gin.Context) {
	// Add request logging
	fmt.Printf("[%s] 📝 POST /api/signup received\n", time.Now().Format("15:04:05"))
	
	var req SignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fmt.Printf("❌ Bad request: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	fmt.Printf("📝 Signup attempt for email: %s\n", req.Email)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	usersColl := database.Client.Database("coded").Collection("users")

	// Check if user already exists
	var existingUser models.User
	err := usersColl.FindOne(ctx, bson.M{"email": req.Email}).Decode(&existingUser)
	if err == nil {
		fmt.Printf("⚠️  Email already in use: %s\n", req.Email)
		c.JSON(http.StatusConflict, gin.H{
			"error":   "Email already registered",
			"message": "Please use a different email or login instead",
		})
		return
	}
	if err != mongo.ErrNoDocuments {
		fmt.Printf("❌ Database error checking email: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Database error",
			"message": "Please try again later",
		})
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Printf("❌ Failed to hash password: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Server error",
			"message": "Failed to process password",
		})
		return
	}
	hashed := string(hashedPassword)

	// Create new user
	user := models.User{
		ID:           primitive.NewObjectID(),
		Email:        req.Email,
		PasswordHash: &hashed,
		AuthProvider: "email",
		CreatedAt:    time.Now().Unix(),
		LastSeen:     time.Now().Unix(),
		Username:     "user_" + primitive.NewObjectID().Hex()[:8],
		Name:         "",
		Avatar:       "https://upload.wikimedia.org/wikipedia/commons/8/89/Portrait_Placeholder.png",
		Bio:          "",
		Gender:       "",
		InterestedIn: []string{},
		Photos:       []string{},
		Status:       "offline",
		BirthDate:    0,
	}

	// Insert user
	_, err = usersColl.InsertOne(ctx, user)
	if err != nil {
		fmt.Printf("❌ Failed to insert user: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Database error",
			"message": "Failed to create user account",
		})
		return
	}

	fmt.Printf("✅ User created: %s (ID: %s)\n", req.Email, user.ID.Hex())

	// Generate JWT token
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &middleware.Claims{
		UserID: user.ID.Hex(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "your-secret-key-change-this-in-production"
		fmt.Println("⚠️  Using default JWT secret. Set JWT_SECRET environment variable!")
	}
	
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		fmt.Printf("❌ Failed to generate token: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Server error",
			"message": "Failed to generate authentication token",
		})
		return
	}

	fmt.Printf("✅ Signup completed successfully for: %s\n", req.Email)

	c.JSON(http.StatusCreated, gin.H{
		"message":  "User created successfully",
		"token":    tokenString,
		"userId":   user.ID.Hex(),
		"email":    user.Email,
		"username": user.Username,
	})
}

func Login(c *gin.Context) {
	// Add request logging
	fmt.Printf("[%s] 🔐 POST /api/login received\n", time.Now().Format("15:04:05"))
	
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fmt.Printf("❌ Bad request: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	fmt.Printf("📝 Login attempt for email: %s\n", req.Email)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	usersColl := database.Client.Database("coded").Collection("users")

	// Find user by email
	var user models.User
	err := usersColl.FindOne(ctx, bson.M{"email": req.Email}).Decode(&user)
	if err == mongo.ErrNoDocuments {
		fmt.Printf("❌ User not found: %s\n", req.Email)
		// Use same error message for security (don't reveal if email exists)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Authentication failed",
			"message": "Invalid email or password",
		})
		return
	}
	if err != nil {
		fmt.Printf("❌ Database error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Database error",
			"message": "Please try again later",
		})
		return
	}

	fmt.Printf("✅ User found: %s (ID: %s)\n", req.Email, user.ID.Hex())

	// Check password
	if user.PasswordHash == nil {
		fmt.Printf("❌ No password hash for user: %s\n", req.Email)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Authentication failed",
			"message": "Invalid email or password",
		})
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(req.Password))
	if err != nil {
		fmt.Printf("❌ Invalid password for: %s\n", req.Email)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Authentication failed",
			"message": "Invalid email or password",
		})
		return
	}

	fmt.Printf("✅ Password correct for: %s\n", req.Email)

	// Update last seen time
	usersColl.UpdateOne(ctx, bson.M{"_id": user.ID}, bson.M{
		"$set": bson.M{"lastSeen": time.Now().Unix()},
	})

	// Generate JWT token
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &middleware.Claims{
		UserID: user.ID.Hex(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "your-secret-key-change-this-in-production"
		fmt.Println("⚠️  Using default JWT secret. Set JWT_SECRET environment variable!")
	}
	
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		fmt.Printf("❌ Failed to generate token: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Server error",
			"message": "Failed to generate authentication token",
		})
		return
	}

	fmt.Printf("✅ Login successful for: %s, token generated\n", req.Email)

	c.JSON(http.StatusOK, gin.H{
		"token":    tokenString,
		"userId":   user.ID.Hex(),
		"email":    user.Email,
		"username": user.Username,
		"avatar":   user.Avatar,
		"message":  "Login successful",
		"expires":  expirationTime.Unix(),
	})
}

// Add this test endpoint to verify handlers are working
func TestHandler(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "Handlers are working correctly",
		"time":    time.Now().Unix(),
	})
}