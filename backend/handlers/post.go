// handlers/post.go  (Replace the entire content of post.go with this)

package handlers

import (
	"context"
	"fmt"
	"log"
	"math"
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
		UserID:    userID,
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

func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // Earth radius in km
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

func GetFeed(c *gin.Context) {
	userIDStr := c.GetString("userId")
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	usersColl := database.Client.Database("coded").Collection("users")

	var currentUser models.User
	err = usersColl.FindOne(ctx, bson.M{"_id": userID}).Decode(&currentUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch current user"})
		return
	}

	hasLocation := currentUser.Latitude != nil && currentUser.Longitude != nil && *currentUser.Latitude != 0 && *currentUser.Longitude != 0

	postsColl := database.Client.Database("coded").Collection("posts")

	cursor, err := postsColl.Find(ctx, bson.M{"userId": bson.M{"$ne": userID}})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch posts"})
		return
	}
	defer cursor.Close(ctx)

	var posts []bson.M
	if err = cursor.All(ctx, &posts); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode posts"})
		return
	}

	var result []map[string]interface{}
	for _, post := range posts {
		userIDObj, ok := post["userId"].(primitive.ObjectID)
		if !ok {
			continue
		}

		var user models.User
		err = usersColl.FindOne(ctx, bson.M{"_id": userIDObj}).Decode(&user)
		if err != nil {
			continue
		}

		var distStr string
		if !hasLocation {
			distStr = "Nearby"
		} else if user.Latitude == nil || user.Longitude == nil || *user.Latitude == 0 && *user.Longitude == 0 {
			distStr = "Unknown"
		} else {
			distance := haversine(*currentUser.Latitude, *currentUser.Longitude, *user.Latitude, *user.Longitude)
			distStr = fmt.Sprintf("%.0f km away", distance)
		}

		postMap := map[string]interface{}{
			"id":        post["_id"],
			"user":      user,
			"content":   post["content"],
			"category":  post["category"],
			"createdAt": post["createdAt"],
			"distance":  distStr,
		}
		result = append(result, postMap)
	}

	c.JSON(http.StatusOK, result)
}

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
		{{"$match", bson.D{{"userId", userID}}}},
		{{"$sort", bson.D{{"createdAt", -1}}}},
		{{"$lookup", bson.D{
			{"from", "users"},
			{"localField", "userId"},
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
		{{"$match", bson.D{{"userId", userID}}}},
		{{"$sort", bson.D{{"createdAt", -1}}}},
		{{"$lookup", bson.D{
			{"from", "users"},
			{"localField", "userId"},
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