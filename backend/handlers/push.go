// handlers/push.go (Updated with import "log")
package handlers

import (
	"context"
	"net/http"
	"os"
	"time"
	"log"  // Added this import

	"coded/database"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var vapidPublicKey string
var vapidPrivateKey string

func init() {
	vapidPublicKey = os.Getenv("VAPID_PUBLIC_KEY")
	vapidPrivateKey = os.Getenv("VAPID_PRIVATE_KEY")

	if vapidPublicKey == "" || vapidPrivateKey == "" {
		var err error
		vapidPublicKey, vapidPrivateKey, err = webpush.GenerateVAPIDKeys()
		if err != nil {
			panic(err)
		}
		// Optionally save to env or db - for now, log
		log.Println("Generated new VAPID keys - set in env for persistence")
		log.Println("VAPID_PUBLIC_KEY:", vapidPublicKey)
		log.Println("VAPID_PRIVATE_KEY:", vapidPrivateKey)
	}
}

func GetVapidPublicKey(c *gin.Context) {
	c.String(http.StatusOK, vapidPublicKey)
}

type PushSubscription struct {
	ID     primitive.ObjectID   `bson:"_id"`
	UserID primitive.ObjectID   `bson:"userId"`
	Sub    webpush.Subscription `bson:"sub"`
}

func SubscribePush(c *gin.Context) {
	userIDStr := c.GetString("userId")
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	var sub webpush.Subscription
	if err := c.ShouldBindJSON(&sub); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	subsColl := database.Client.Database("coded").Collection("subscriptions")

	_, err = subsColl.UpdateOne(
		ctx,
		bson.M{"userId": userID},
		bson.M{"$set": bson.M{"sub": sub}},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save subscription"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Subscribed successfully"})
}
