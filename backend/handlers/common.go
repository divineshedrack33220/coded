package handlers

import (
    "coded/websocket"
    "github.com/SherClockHolmes/webpush-go"
    "go.mongodb.org/mongo-driver/bson/primitive"
)

// Common constants and variables shared across all handler files
const fallbackAvatar = "https://upload.wikimedia.org/wikipedia/commons/8/89/Portrait_Placeholder.png"

var wsManager *websocket.Manager
var vapidPrivateKey string

// PushSubscription struct for push notifications
type PushSubscription struct {
    ID     primitive.ObjectID      `bson:"_id,omitempty"`
    UserID primitive.ObjectID      `bson:"userId"`
    Sub    webpush.Subscription    `bson:"sub"`
}

// SetWebSocketManager sets the global WebSocket manager
func SetWebSocketManager(manager *websocket.Manager) {
    wsManager = manager
}

// SetVAPIDPrivateKey sets the VAPID private key
func SetVAPIDPrivateKey(key string) {
    vapidPrivateKey = key
}