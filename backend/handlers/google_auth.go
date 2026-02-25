package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
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
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Google OAuth Config
var (
	googleOAuthConfig *oauth2.Config
)

// Initialize Google OAuth
func init() {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	
	if clientID != "" && clientSecret != "" {
		googleOAuthConfig = &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  "http://localhost:8080/api/google/callback",
			Scopes: []string{
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/userinfo.profile",
			},
			Endpoint: google.Endpoint,
		}
		log.Println("✅ Google OAuth configured successfully")
	} else {
		log.Println("⚠️  Google OAuth not configured - set GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET")
	}
}

// Google user info structure
type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
}

// Google Auth Request
type GoogleAuthRequest struct {
	Credential string `json:"credential" binding:"required"`
}

// Generate username from email
func generateUsernameFromEmail(email string) string {
	// Take the part before @ and clean it up
	for i := 0; i < len(email); i++ {
		if email[i] == '@' {
			username := email[:i]
			// Remove any dots and make lowercase
			cleanUsername := ""
			for _, ch := range username {
				if ch != '.' {
					cleanUsername += string(ch)
				}
			}
			return cleanUsername + "_" + primitive.NewObjectID().Hex()[:4]
		}
	}
	return "user_" + primitive.NewObjectID().Hex()[:8]
}

// Handle Google OAuth callback (for traditional OAuth flow)
func GoogleOAuthCallback(c *gin.Context) {
	fmt.Printf("[%s] 🔐 GET /api/google/callback received\n", time.Now().Format("15:04:05"))
	
	code := c.Query("code")
	if code == "" {
		log.Printf("❌ Authorization code missing")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Authorization code missing"})
		return
	}

	if googleOAuthConfig == nil {
		log.Printf("❌ Google OAuth not configured")
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Google OAuth not configured"})
		return
	}

	ctx := context.Background()
	token, err := googleOAuthConfig.Exchange(ctx, code)
	if err != nil {
		log.Printf("❌ Google OAuth token exchange failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange authorization code"})
		return
	}

	// Get user info from Google
	client := googleOAuthConfig.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		log.Printf("❌ Failed to get user info from Google: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user information"})
		return
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("❌ Failed to read Google user info: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read user information"})
		return
	}

	var googleUser GoogleUserInfo
	if err := json.Unmarshal(data, &googleUser); err != nil {
		log.Printf("❌ Failed to parse Google user info: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse user information"})
		return
	}

	log.Printf("✅ Google user info retrieved: %s (%s)", googleUser.Email, googleUser.Name)
	handleGoogleUser(c, googleUser, token)
}

// Handle Google Sign-In with Credential (Google Identity Services)
func GoogleAuthWithCredential(c *gin.Context) {
	fmt.Printf("[%s] 🔐 POST /api/google-auth received\n", time.Now().Format("15:04:05"))
	
	var req GoogleAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("❌ Invalid Google auth request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Verify the Google credential (in production, you should verify the JWT)
	// For now, we'll parse the JWT to get user info
	token, _, err := new(jwt.Parser).ParseUnverified(req.Credential, jwt.MapClaims{})
	if err != nil {
		log.Printf("❌ Failed to parse Google credential: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Google credential"})
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		log.Printf("❌ Invalid Google credential claims")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Google credential"})
		return
	}

	// Extract user info from claims
	googleUser := GoogleUserInfo{
		ID:      getStringClaim(claims, "sub"),
		Email:   getStringClaim(claims, "email"),
		Name:    getStringClaim(claims, "name"),
		Picture: getStringClaim(claims, "picture"),
	}

	if googleUser.Email == "" {
		log.Printf("❌ Google credential missing email")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email not provided by Google"})
		return
	}

	log.Printf("✅ Google credential parsed: %s (%s)", googleUser.Email, googleUser.Name)
	handleGoogleUser(c, googleUser, nil)
}

// Helper function to get string claim from JWT
func getStringClaim(claims jwt.MapClaims, key string) string {
	if val, ok := claims[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// Handle Google user authentication/registration
func handleGoogleUser(c *gin.Context, googleUser GoogleUserInfo, token *oauth2.Token) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	usersColl := database.Client.Database("coded").Collection("users")

	// Check if user already exists
	var user models.User
	err := usersColl.FindOne(ctx, bson.M{"email": googleUser.Email}).Decode(&user)

	if err == mongo.ErrNoDocuments {
		// New user - create account
		log.Printf("📝 Creating new user from Google: %s", googleUser.Email)
		user = createUserFromGoogle(googleUser)
		
		// Insert new user
		_, err = usersColl.InsertOne(ctx, user)
		if err != nil {
			log.Printf("❌ Failed to insert Google user: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user account"})
			return
		}

		log.Printf("✅ New Google user created: %s (ID: %s)", googleUser.Email, user.ID.Hex())

	} else if err != nil {
		// Database error
		log.Printf("❌ Database error checking Google user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	} else {
		// Existing user - update last seen and possibly profile picture
		log.Printf("📝 Existing Google user logging in: %s", googleUser.Email)
		
		// Update last seen time
		updateData := bson.M{
			"$set": bson.M{
				"lastSeen": time.Now().Unix(),
				"authProvider": "google",
			},
		}
		
		// Add GoogleID if not set
		if user.GoogleID == nil && googleUser.ID != "" {
			updateData["$set"].(bson.M)["googleId"] = googleUser.ID
		}
		
		// Update avatar if it's the default and Google has a better one
		if (user.Avatar == "" || user.Avatar == fallbackAvatar) && googleUser.Picture != "" {
			updateData["$set"].(bson.M)["avatar"] = googleUser.Picture
			user.Avatar = googleUser.Picture
		}
		
		_, err = usersColl.UpdateOne(ctx, bson.M{"_id": user.ID}, updateData)
		if err != nil {
			log.Printf("⚠️ Failed to update user last seen: %v", err)
		}
	}

	// Generate JWT token for the user
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &middleware.Claims{
		UserID: user.ID.Hex(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "your-secret-key-change-this-in-production"
	}
	
	tokenString, err := jwtToken.SignedString([]byte(jwtSecret))
	if err != nil {
		log.Printf("❌ Failed to generate JWT token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate authentication token"})
		return
	}

	// Check if user has completed onboarding
	hasCompletedOnboarding := user.Name != "" && user.Name != user.Username && user.Gender != "" && len(user.InterestedIn) > 0

	log.Printf("✅ Google authentication successful for: %s", googleUser.Email)

	// Return response
	c.JSON(http.StatusOK, gin.H{
		"token":                 tokenString,
		"userId":                user.ID.Hex(),
		"email":                 user.Email,
		"username":              user.Username,
		"avatar":                user.Avatar,
		"name":                  user.Name,
		"isNewUser":             err == mongo.ErrNoDocuments,
		"hasCompletedOnboarding": hasCompletedOnboarding,
		"message":               "Authentication successful",
		"expires":               expirationTime.Unix(),
	})
}

// Create user from Google info
func createUserFromGoogle(googleUser GoogleUserInfo) models.User {
	username := generateUsernameFromEmail(googleUser.Email)
	
	// Use Google picture if available, otherwise use default
	avatar := googleUser.Picture
	if avatar == "" {
		avatar = fallbackAvatar
	}

	// Generate name from Google info
	name := googleUser.Name
	if name == "" {
		// Try to combine given and family name
		if googleUser.GivenName != "" || googleUser.FamilyName != "" {
			name = googleUser.GivenName + " " + googleUser.FamilyName
		} else {
			name = username
		}
	}

	return models.User{
		ID:            primitive.NewObjectID(),
		Email:         googleUser.Email,
		PasswordHash:  nil, // Google users don't have password
		AuthProvider:  "google",
		GoogleID:      &googleUser.ID,
		CreatedAt:     time.Now().Unix(),
		LastSeen:      time.Now().Unix(),
		Username:      username,
		Name:          name,
		Avatar:        avatar,
		Bio:           "",
		Gender:        "",
		InterestedIn:  []string{},
		Photos:        []string{},
		Status:        "offline",
		BirthDate:     0,
		ReferralCode:  "",
		Latitude:      nil,
		Longitude:     nil,
	}
}

// Get Google OAuth URL (for traditional OAuth flow)
func GetGoogleAuthURL(c *gin.Context) {
	if googleOAuthConfig == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Google OAuth not configured"})
		return
	}

	// Generate state token for security
	state := primitive.NewObjectID().Hex()
	
	url := googleOAuthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
	c.JSON(http.StatusOK, gin.H{"url": url})
}