package handlers

import (
	"context"
	"net/http"
	"time"

	"coded/database"
	"coded/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// REMOVE this line - fallbackAvatar is already declared in user.go
// const fallbackAvatar = "https://upload.wikimedia.org/wikipedia/commons/8/89/Portrait_Placeholder.png"

func AddFavorite(c *gin.Context) {
	var req struct {
		TargetUserID string `json:"targetUserId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userIDStr := c.GetString("userId")
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	targetID, err := primitive.ObjectIDFromHex(req.TargetUserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid target user ID"})
		return
	}

	if userID == targetID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot favorite yourself"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	favColl := database.Client.Database("coded").Collection("favorites")

	// Check if already favorited
	count, err := favColl.CountDocuments(ctx, bson.M{
		"userId":       userID,
		"targetUserId": targetID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Already favorited"})
		return
	}

	fav := models.Favorite{
		ID:           primitive.NewObjectID(),
		UserID:       userID,
		TargetUserID: targetID,
		CreatedAt:    time.Now().Unix(),
	}

	_, err = favColl.InsertOne(ctx, fav)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add favorite"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Favorite added"})
}

func RemoveFavorite(c *gin.Context) {
	var req struct {
		TargetUserID string `json:"targetUserId" binding:"required"`
	}
	
	// Try to parse from query parameter first (for backward compatibility)
	targetUserId := c.Query("targetUserId")
	if targetUserId == "" {
		// If not in query, try to parse from JSON body
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "targetUserId is required"})
			return
		}
		targetUserId = req.TargetUserID
	}

	userIDStr := c.GetString("userId")
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	targetID, err := primitive.ObjectIDFromHex(targetUserId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid target user ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	favColl := database.Client.Database("coded").Collection("favorites")

	result, err := favColl.DeleteOne(ctx, bson.M{
		"userId":       userID,
		"targetUserId": targetID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove favorite"})
		return
	}

	if result.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Favorite not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Favorite removed"})
}

func GetFavorites(c *gin.Context) {
	userIDStr := c.GetString("userId")
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	favColl := database.Client.Database("coded").Collection("favorites")
	usersColl := database.Client.Database("coded").Collection("users")

	// Fetch favorites
	findOptions := options.Find().SetSort(bson.D{{"createdAt", -1}})
	cursor, err := favColl.Find(ctx, bson.M{"userId": userID}, findOptions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch favorites"})
		return
	}
	defer cursor.Close(ctx)

	var favorites []models.Favorite
	if err := cursor.All(ctx, &favorites); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode favorites"})
		return
	}

	if len(favorites) == 0 {
		c.JSON(http.StatusOK, []map[string]interface{}{})
		return
	}

	// Collect target user IDs
	var targetIDs []primitive.ObjectID
	for _, f := range favorites {
		targetIDs = append(targetIDs, f.TargetUserID)
	}

	// Fetch user documents
	userCursor, err := usersColl.Find(ctx, bson.M{"_id": bson.M{"$in": targetIDs}})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}
	defer userCursor.Close(ctx)

	var users []models.User
	if err := userCursor.All(ctx, &users); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode users"})
		return
	}

	userMap := make(map[primitive.ObjectID]map[string]interface{})
	for _, u := range users {
		userMap[u.ID] = map[string]interface{}{
			"id":     u.ID.Hex(),
			"name":   u.Name,
			"avatar": u.Avatar,
			"status": u.Status,
			"bio":    u.Bio,
		}
	}

	// Use a local fallback variable
	const localFallbackAvatar = "https://upload.wikimedia.org/wikipedia/commons/8/89/Portrait_Placeholder.png"
	
	response := make([]map[string]interface{}, len(favorites))
	for i, f := range favorites {
		userData := map[string]interface{}{
			"id":     f.TargetUserID.Hex(),
			"name":   "Unknown User",
			"avatar": localFallbackAvatar,
			"status": "offline",
			"bio":    "",
		}

		if storedUser, exists := userMap[f.TargetUserID]; exists {
			userData = storedUser
		}

		response[i] = map[string]interface{}{
			"id":           f.ID.Hex(),
			"targetUserId": f.TargetUserID.Hex(),
			"createdAt":    f.CreatedAt,
			"user":         userData,
		}
	}

	c.JSON(http.StatusOK, response)
}