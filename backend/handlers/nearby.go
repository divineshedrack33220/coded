package handlers

import (
    "context"
    "log"
    "math"
    "net/http"
    "time"

    "coded/database"
    "coded/models"

    "github.com/gin-gonic/gin"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"
)

// GetNearbyUsers finds users within a certain radius of the current user
func GetNearbyUsers(c *gin.Context) {
    log.Printf("[GetNearbyUsers] Request received")
    
    userIDStr := c.GetString("userId")
    userID, err := primitive.ObjectIDFromHex(userIDStr)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    usersColl := database.Client.Database("coded").Collection("users")

    // Get current user's location
    var currentUser models.User
    err = usersColl.FindOne(ctx, bson.M{"_id": userID}).Decode(&currentUser)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch current user"})
        return
    }

    // Check if current user has location data
    if currentUser.Latitude == nil || currentUser.Longitude == nil ||
        *currentUser.Latitude == 0 && *currentUser.Longitude == 0 {
        // User doesn't have location, return empty array
        log.Printf("[GetNearbyUsers] Current user has no location data")
        c.JSON(http.StatusOK, []interface{}{})
        return
    }

    // Get all users except current user
    cursor, err := usersColl.Find(ctx, bson.M{
        "_id": bson.M{"$ne": userID},
        "latitude": bson.M{"$exists": true, "$ne": nil},
        "longitude": bson.M{"$exists": true, "$ne": nil},
    })
    if err != nil {
        log.Printf("[GetNearbyUsers] Database error: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
        return
    }
    defer cursor.Close(ctx)

    var allUsers []models.User
    if err = cursor.All(ctx, &allUsers); err != nil {
        log.Printf("[GetNearbyUsers] Decode error: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode users"})
        return
    }

    var nearbyUsers []map[string]interface{}
    currentLat := *currentUser.Latitude
    currentLon := *currentUser.Longitude

    log.Printf("[GetNearbyUsers] Current location: %f, %f", currentLat, currentLon)
    log.Printf("[GetNearbyUsers] Found %d total users", len(allUsers))

    for _, user := range allUsers {
        if user.Latitude == nil || user.Longitude == nil ||
            *user.Latitude == 0 && *user.Longitude == 0 {
            continue
        }

        // Calculate distance using Haversine formula
        distance := calculateDistance(currentLat, currentLon, *user.Latitude, *user.Longitude)
        
        // Filter users within 50km radius (adjust this as needed)
        if distance <= 50.0 {
            distanceMeters := math.Round(distance * 1000)
            nearbyUsers = append(nearbyUsers, map[string]interface{}{
                "id":       user.ID.Hex(),
                "name":     user.Name,
                "avatar":   user.Avatar,
                "distance": distanceMeters,
                "status":   user.Status,
                "bio":      user.Bio,
            })
            log.Printf("[GetNearbyUsers] Found nearby user: %s (%fm)", user.Name, distanceMeters)
        }
    }

    log.Printf("[GetNearbyUsers] Returning %d nearby users", len(nearbyUsers))
    
    // If no nearby users found, return empty array
    if len(nearbyUsers) == 0 {
        c.JSON(http.StatusOK, []interface{}{})
        return
    }

    c.JSON(http.StatusOK, nearbyUsers)
}

// calculateDistance calculates distance in kilometers using Haversine formula
func calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
    const R = 6371 // Earth's radius in kilometers
    
    dLat := (lat2 - lat1) * math.Pi / 180
    dLon := (lon2 - lon1) * math.Pi / 180
    
    a := math.Sin(dLat/2)*math.Sin(dLat/2) +
        math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
            math.Sin(dLon/2)*math.Sin(dLon/2)
    
    c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
    
    return R * c
}