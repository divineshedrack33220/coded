package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"os"
	"time"

	"coded/database"
	"coded/models"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

const fallbackAvatar = "https://upload.wikimedia.org/wikipedia/commons/8/89/Portrait_Placeholder.png"

type OnboardingData struct {
	Name         string   `json:"name" form:"name"`
	BirthDate    int64    `json:"birthDate,omitempty" form:"birthDate"`
	Gender       string   `json:"gender" form:"gender"`
	InterestedIn []string `json:"interestedIn" form:"interestedIn"`
	Bio          string   `json:"bio" form:"bio"`
	Status       string   `json:"status" form:"status"`
	Photos       []string `json:"photos" form:"photos"`
	Latitude     *float64 `json:"latitude,omitempty" form:"latitude"`
	Longitude    *float64 `json:"longitude,omitempty" form:"longitude"`
}

// Helper: generate a unique 8-character referral code
func generateReferralCode() (string, error) {
	b := make([]byte, 4) // 4 bytes → 8 hex chars
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func GetUser(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	usersColl := database.Client.Database("coded").Collection("users")

	var user models.User
	err = usersColl.FindOne(ctx, bson.M{"_id": userID}).Decode(&user)
	if err == mongo.ErrNoDocuments {
		c.JSON(http.StatusOK, gin.H{
			"id":     userID.Hex(),
			"name":   "Unknown User",
			"avatar": fallbackAvatar,
			"status": "offline",
		})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":     user.ID.Hex(),
		"name":   user.Name,
		"avatar": user.Avatar,
		"status": user.Status,
	})
}

func UpdateMyProfile(c *gin.Context) {
	userIDStr := c.GetString("userId")
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	usersColl := database.Client.Database("coded").Collection("users")

	update := bson.M{"$set": bson.M{}}

	contentType := c.ContentType()

	var data OnboardingData

	if contentType == "application/json" {
		if err := c.ShouldBindJSON(&data); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON data"})
			return
		}
	} else {
		if err := c.Request.ParseMultipartForm(10 << 20); err != nil && err != http.ErrNotMultipart {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse form data"})
			return
		}
		if err := c.ShouldBind(&data); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid form data"})
			return
		}
	}

	// Map fields to update
	if data.Name != "" {
		update["$set"].(bson.M)["name"] = data.Name
	}
	if data.BirthDate != 0 {
		update["$set"].(bson.M)["birthDate"] = data.BirthDate
	}
	if data.Gender != "" {
		update["$set"].(bson.M)["gender"] = data.Gender
	}
	if len(data.InterestedIn) > 0 {
		update["$set"].(bson.M)["interestedIn"] = data.InterestedIn
	}
	if data.Bio != "" {
		update["$set"].(bson.M)["bio"] = data.Bio
	}
	if data.Status != "" {
		update["$set"].(bson.M)["status"] = data.Status
	}
	if len(data.Photos) > 0 {
		update["$set"].(bson.M)["photos"] = data.Photos
	}
	if data.Latitude != nil {
		update["$set"].(bson.M)["latitude"] = *data.Latitude
	}
	if data.Longitude != nil {
		update["$set"].(bson.M)["longitude"] = *data.Longitude
	}

	if username := c.PostForm("username"); username != "" {
		update["$set"].(bson.M)["username"] = username
	}

	// Avatar upload (multipart only)
	avatarFile, _, err := c.Request.FormFile("avatar")
	if err == nil {
		defer avatarFile.Close()

		cld, err := cloudinary.NewFromURL(os.Getenv("CLOUDINARY_URL"))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Cloudinary configuration error"})
			return
		}

		uploadParams := uploader.UploadParams{
			Folder:         "coded/avatars",
			PublicID:       userID.Hex(),
			Transformation: "c_limit,w_400,h_400,q_auto",
		}

		uploadResult, err := cld.Upload.Upload(ctx, avatarFile, uploadParams)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload avatar to Cloudinary"})
			return
		}

		update["$set"].(bson.M)["avatar"] = uploadResult.SecureURL
	}

	if len(update["$set"].(bson.M)) == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "No changes to update"})
		return
	}

	result, err := usersColl.UpdateOne(ctx, bson.M{"_id": userID}, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
		return
	}
	if result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Profile updated successfully"})
}

func UploadPhoto(c *gin.Context) {
	userIDStr := c.GetString("userId")
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := c.Request.ParseMultipartForm(10 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse form data"})
		return
	}

	photoFile, _, err := c.Request.FormFile("photo")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No photo file provided"})
		return
	}
	defer photoFile.Close()

	cld, err := cloudinary.NewFromURL(os.Getenv("CLOUDINARY_URL"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Cloudinary configuration error"})
		return
	}

	uploadParams := uploader.UploadParams{
		Folder:         "coded/photos",
		PublicID:       userID.Hex() + "_" + time.Now().Format("20060102150405"),
		Transformation: "c_limit,w_800,h_800,q_auto",
	}

	uploadResult, err := cld.Upload.Upload(ctx, photoFile, uploadParams)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload photo to Cloudinary"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": uploadResult.SecureURL})
}

// Updated GetMyProfile – auto-generates referral code if missing
func GetMyProfile(c *gin.Context) {
	userIDStr := c.GetString("userId")
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	usersColl := database.Client.Database("coded").Collection("users")

	var user models.User
	err = usersColl.FindOne(ctx, bson.M{"_id": userID}).Decode(&user)
	if err == mongo.ErrNoDocuments {
		c.JSON(http.StatusNotFound, gin.H{"error": "Profile not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch profile"})
		return
	}

	// Auto-generate referral code if it doesn't exist
	if user.ReferralCode == "" {
		var code string
		for {
			code, err = generateReferralCode()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate referral code"})
				return
			}
			// Ensure uniqueness
			count, _ := usersColl.CountDocuments(ctx, bson.M{"referralCode": code})
			if count == 0 {
				break
			}
		}

		_, err = usersColl.UpdateOne(ctx, bson.M{"_id": userID}, bson.M{"$set": bson.M{"referralCode": code}})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save referral code"})
			return
		}
		user.ReferralCode = code
	}

	c.JSON(http.StatusOK, user)
}

// New endpoint: GET /me/referral
func GetReferral(c *gin.Context) {
	userIDStr := c.GetString("userId")
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	usersColl := database.Client.Database("coded").Collection("users")

	var user models.User
	err = usersColl.FindOne(ctx, bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch profile"})
		return
	}

	if user.ReferralCode == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Referral code not generated yet"})
		return
	}

	// Change this to your production domain later
	baseURL := "http://localhost:8080"
	referralURL := baseURL + "/register?ref=" + user.ReferralCode

	c.JSON(http.StatusOK, gin.H{
		"referralCode": user.ReferralCode,
		"referralUrl":  referralURL,
	})
}

func GetMatches(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "GetMatches - not implemented"})
}