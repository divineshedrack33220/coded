package handlers

import (
	"context"
	"log"
	"net/http"
	"time"

	"coded/database"
	"coded/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type CreatePostRequest struct {
	Content  string   `json:"content" binding:"required"`
	Media    []string `json:"media"`
	Category string   `json:"category,omitempty"`
}

func CreatePost(c *gin.Context) {
	var req CreatePostRequest
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	postsColl := database.Client.Database("coded").Collection("posts")

	post := models.Post{
		ID:        primitive.NewObjectID(),
		UserID:    userID,               // <-- This matches the field name in your DB schema: "userId"
		Content:   req.Content,
		Media:     req.Media,
		Category:  req.Category,
		CreatedAt: time.Now().Unix(),
	}

	_, err = postsColl.InsertOne(ctx, post)
	if err != nil {
		log.Printf("CreatePost error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create post"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Post created successfully",
		"postId":  post.ID.Hex(),
	})
}

func GetFeed(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	postsColl := database.Client.Database("coded").Collection("posts")

	// IMPORTANT: The localField must match the exact field name in the posts collection ("userId", not "UserID")
	pipeline := mongo.Pipeline{
		{{"$sort", bson.D{{"createdAt", -1}}}},
		{{"$limit", 20}},
		{{"$lookup", bson.D{
			{"from", "users"},
			{"localField", "userId"},      // <-- Fixed: was "UserID", now "userId"
			{"foreignField", "_id"},
			{"as", "user"},
		}}},
		{{"$unwind", bson.D{
			{"path", "$user"},
			{"preserveNullAndEmptyArrays", true},
		}}},
	}

	cursor, err := postsColl.Aggregate(ctx, pipeline)
	if err != nil {
		log.Printf("GetFeed aggregate error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch feed"})
		return
	}
	defer cursor.Close(ctx)

	var posts []struct {
		models.Post         `bson:",inline"`
		User                *models.User `bson:"user"`
	}
	if err := cursor.All(ctx, &posts); err != nil {
		log.Printf("GetFeed decode error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode feed"})
		return
	}

	const fallbackAvatar = "https://upload.wikimedia.org/wikipedia/commons/8/89/Portrait_Placeholder.png"

	response := make([]map[string]interface{}, len(posts))
	for i, p := range posts {
		userMap := map[string]interface{}{
			"id":     p.UserID.Hex(),
			"name":   "Unknown User",
			"avatar": fallbackAvatar,
			"status": "offline",
			"bio":    "",
		}

		if p.User != nil {
			if p.User.Name != "" {
				userMap["name"] = p.User.Name
			}
			if p.User.Avatar != "" {
				userMap["avatar"] = p.User.Avatar
			}
			if p.User.Status != "" {
				userMap["status"] = p.User.Status
			}
			if p.User.Bio != "" {
				userMap["bio"] = p.User.Bio
			}
		}

		response[i] = map[string]interface{}{
			"id":        p.ID.Hex(),
			"userId":    p.UserID.Hex(),
			"content":   p.Content,
			"media":     p.Media,
			"category":  p.Category,
			"createdAt": p.CreatedAt,
			"user":      userMap,
			"distance":  "Nearby",
		}
	}

	c.JSON(http.StatusOK, response)
}

// Apply the same fix to GetUserPosts and GetMyPosts if you have them
// (change "UserID" → "userId" in the $lookup localField)

func GetUserPosts(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	postsColl := database.Client.Database("coded").Collection("posts")

	pipeline := mongo.Pipeline{
		{{"$match", bson.D{{"userId", userID}}}}, // <-- Also fixed here if needed
		{{"$sort", bson.D{{"createdAt", -1}}}},
		{{"$lookup", bson.D{
			{"from", "users"},
			{"localField", "userId"},      // <-- Fixed: was "UserID"
			{"foreignField", "_id"},
			{"as", "user"},
		}}},
		{{"$unwind", bson.D{
			{"path", "$user"},
			{"preserveNullAndEmptyArrays", true},
		}}},
	}

	// ... rest remains the same as before (with fallback userMap)
	cursor, err := postsColl.Aggregate(ctx, pipeline)
	if err != nil {
		log.Printf("GetUserPosts aggregate error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch posts"})
		return
	}
	defer cursor.Close(ctx)

	var posts []struct {
		models.Post         `bson:",inline"`
		User                *models.User `bson:"user"`
	}
	if err := cursor.All(ctx, &posts); err != nil {
		log.Printf("GetUserPosts decode error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode posts"})
		return
	}

	const fallbackAvatar = "https://upload.wikimedia.org/wikipedia/commons/8/89/Portrait_Placeholder.png"

	response := make([]map[string]interface{}, len(posts))
	for i, p := range posts {
		userMap := map[string]interface{}{
			"id":     p.UserID.Hex(),
			"name":   "Unknown User",
			"avatar": fallbackAvatar,
			"status": "offline",
			"bio":    "",
		}

		if p.User != nil {
			if p.User.Name != "" {
				userMap["name"] = p.User.Name
			}
			if p.User.Avatar != "" {
				userMap["avatar"] = p.User.Avatar
			}
			if p.User.Status != "" {
				userMap["status"] = p.User.Status
			}
			if p.User.Bio != "" {
				userMap["bio"] = p.User.Bio
			}
		}

		response[i] = map[string]interface{}{
			"id":        p.ID.Hex(),
			"content":   p.Content,
			"media":     p.Media,
			"category":  p.Category,
			"createdAt": p.CreatedAt,
			"user":      userMap,
		}
	}

	c.JSON(http.StatusOK, response)
}

func GetMyPosts(c *gin.Context) {
	userIDStr := c.GetString("userId")
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	postsColl := database.Client.Database("coded").Collection("posts")

	pipeline := mongo.Pipeline{
		{{"$match", bson.D{{"userId", userID}}}}, // <-- Fixed if needed
		{{"$sort", bson.D{{"createdAt", -1}}}},
		{{"$lookup", bson.D{
			{"from", "users"},
			{"localField", "userId"},      // <-- Fixed: was "UserID"
			{"foreignField", "_id"},
			{"as", "user"},
		}}},
		{{"$unwind", bson.D{
			{"path", "$user"},
			{"preserveNullAndEmptyArrays", true},
		}}},
	}

	// ... rest identical to GetUserPosts (with fallback userMap)
	cursor, err := postsColl.Aggregate(ctx, pipeline)
	if err != nil {
		log.Printf("GetMyPosts aggregate error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch posts"})
		return
	}
	defer cursor.Close(ctx)

	var posts []struct {
		models.Post         `bson:",inline"`
		User                *models.User `bson:"user"`
	}
	if err := cursor.All(ctx, &posts); err != nil {
		log.Printf("GetMyPosts decode error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode posts"})
		return
	}

	const fallbackAvatar = "https://upload.wikimedia.org/wikipedia/commons/8/89/Portrait_Placeholder.png"

	response := make([]map[string]interface{}, len(posts))
	for i, p := range posts {
		userMap := map[string]interface{}{
			"id":     p.UserID.Hex(),
			"name":   "Unknown User",
			"avatar": fallbackAvatar,
			"status": "offline",
			"bio":    "",
		}

		if p.User != nil {
			if p.User.Name != "" {
				userMap["name"] = p.User.Name
			}
			if p.User.Avatar != "" {
				userMap["avatar"] = p.User.Avatar
			}
			if p.User.Status != "" {
				userMap["status"] = p.User.Status
			}
			if p.User.Bio != "" {
				userMap["bio"] = p.User.Bio
			}
		}

		response[i] = map[string]interface{}{
			"id":        p.ID.Hex(),
			"content":   p.Content,
			"media":     p.Media,
			"category":  p.Category,
			"createdAt": p.CreatedAt,
			"user":      userMap,
		}
	}

	c.JSON(http.StatusOK, response)
}