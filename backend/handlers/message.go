package handlers

import (
    "context"
    "encoding/json"
    "log"
    "net/http"
    "time"

    "coded/database"
    "coded/models"

    "github.com/gin-gonic/gin"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"
    "go.mongodb.org/mongo-driver/mongo"
    "github.com/SherClockHolmes/webpush-go"
)

func GetMessages(c *gin.Context) {
    chatIDStr := c.Param("chatId")
    chatID, err := primitive.ObjectIDFromHex(chatIDStr)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chat ID"})
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

    // First, verify user is in the chat
    chatsColl := database.Client.Database("coded").Collection("chats")
    var chat models.Chat
    err = chatsColl.FindOne(ctx, bson.M{"_id": chatID, "participants": userID}).Decode(&chat)
    if err == mongo.ErrNoDocuments {
        c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to chat"})
        return
    }
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify chat access"})
        return
    }

    messagesColl := database.Client.Database("coded").Collection("messages")

    // Fetch messages with sender user data
    pipeline := mongo.Pipeline{
        {{"$match", bson.D{{"chatId", chatID}}}},
        {{"$sort", bson.D{{"createdAt", 1}}}},
        {{"$lookup", bson.D{
            {"from", "users"},
            {"localField", "senderId"},
            {"foreignField", "_id"},
            {"as", "senderProfile"},
        }}},
        {{"$unwind", bson.D{
            {"path", "$senderProfile"},
            {"preserveNullAndEmptyArrays", true},
        }}},
    }

    cursor, err := messagesColl.Aggregate(ctx, pipeline)
    if err != nil {
        log.Printf("GetMessages aggregate error: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch messages"})
        return
    }
    defer cursor.Close(ctx)

    var rawMessages []bson.M
    if err := cursor.All(ctx, &rawMessages); err != nil {
        log.Printf("GetMessages decode error: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode messages"})
        return
    }

    // Build response with safe sender object (never null)
    response := make([]map[string]interface{}, len(rawMessages))
    for i, m := range rawMessages {
        senderProfile := m["senderProfile"]

        senderMap := map[string]interface{}{
            "id":     m["senderId"].(primitive.ObjectID).Hex(),
            "name":   "Unknown",
            "avatar": fallbackAvatar,
        }

        if profile, ok := senderProfile.(bson.M); ok && profile != nil {
            if name, _ := profile["name"].(string); name != "" {
                senderMap["name"] = name
            }
            if avatar, _ := profile["avatar"].(string); avatar != "" {
                senderMap["avatar"] = avatar
            }
        }

        response[i] = map[string]interface{}{
            "id":        m["_id"].(primitive.ObjectID).Hex(),
            "chatId":    m["chatId"].(primitive.ObjectID).Hex(),
            "senderId":  m["senderId"].(primitive.ObjectID).Hex(),
            "sender":    senderMap,
            "content":   m["content"],
            "type":      m["type"],
            "isRead":    m["isRead"],
            "createdAt": m["createdAt"],
        }
    }

    c.JSON(http.StatusOK, response)
}

func SendMessage(c *gin.Context) {
    var req struct {
        ChatID  string `json:"chatId" binding:"required"`
        Content string `json:"content" binding:"required"`
        Type    string `json:"type,omitempty"`
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

    chatID, err := primitive.ObjectIDFromHex(req.ChatID)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chat ID"})
        return
    }

    if req.Type == "" {
        req.Type = "text"
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    // Verify user is in the chat
    chatsColl := database.Client.Database("coded").Collection("chats")
    var chat models.Chat
    err = chatsColl.FindOne(ctx, bson.M{"_id": chatID, "participants": userID}).Decode(&chat)
    if err == mongo.ErrNoDocuments {
        c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to chat"})
        return
    }
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify chat access"})
        return
    }

    messagesColl := database.Client.Database("coded").Collection("messages")

    message := models.Message{
        ID:        primitive.NewObjectID(),
        ChatID:    chatID,
        SenderID:  userID,
        Content:   req.Content,
        Type:      req.Type,
        IsRead:    false,
        CreatedAt: time.Now().Unix(),
    }

    _, err = messagesColl.InsertOne(ctx, message)
    if err != nil {
        log.Printf("SendMessage insert error: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send message"})
        return
    }

    // Update chat's last message
    _, err = chatsColl.UpdateOne(
        ctx,
        bson.M{"_id": chatID},
        bson.M{
            "$set": bson.M{
                "lastMessage":   req.Content,
                "lastMessageAt": message.CreatedAt,
            },
        },
    )
    if err != nil {
        log.Printf("Update chat lastMessage error: %v", err)
        // Not critical – message was already saved
    }

    // Get sender info for WebSocket broadcast
    usersColl := database.Client.Database("coded").Collection("users")
    var sender models.User
    usersColl.FindOne(ctx, bson.M{"_id": userID}).Decode(&sender)

    // Prepare WebSocket message
    wsMessage := map[string]interface{}{
        "id":        message.ID.Hex(),
        "chatId":    message.ChatID.Hex(),
        "senderId":  message.SenderID.Hex(),
        "sender": map[string]interface{}{
            "id":     sender.ID.Hex(),
            "name":   sender.Name,
            "avatar": sender.Avatar,
        },
        "content":   message.Content,
        "type":      message.Type,
        "isRead":    message.IsRead,
        "createdAt": message.CreatedAt,
    }

    // Broadcast via WebSocket
    if wsManager != nil {
        wsManager.BroadcastNewMessage(wsMessage)
    }

    // Send push notification to the other participant(s)
    go func() {
        defer func() {
            if r := recover(); r != nil {
                log.Printf("Panic in push notification: %v", r)
            }
        }()

        subsColl := database.Client.Database("coded").Collection("subscriptions")
        usersColl := database.Client.Database("coded").Collection("users")

        for _, participantID := range chat.Participants {
            if participantID == userID {
                continue // Skip sender
            }

            // Get receiver's name for payload (optional)
            var sender models.User
            usersColl.FindOne(context.Background(), bson.M{"_id": userID}).Decode(&sender)

            payload := map[string]string{
                "title": sender.Name + " sent a message",
                "body":  req.Content,
                "icon":  sender.Avatar, // Optional
            }
            payloadBytes, _ := json.Marshal(payload)

            // Find subscription
            var sub PushSubscription
            err = subsColl.FindOne(context.Background(), bson.M{"userId": participantID}).Decode(&sub)
            if err == mongo.ErrNoDocuments {
                continue // No subscription
            }
            if err != nil {
                log.Printf("Failed to find subscription: %v", err)
                continue
            }

            // Send push
            _, err = webpush.SendNotification(payloadBytes, &sub.Sub, &webpush.Options{
                Subscriber:      "user@example.com", // Replace with actual if needed
                VAPIDPrivateKey: vapidPrivateKey,
                TTL:             30,
            })
            if err != nil {
                log.Printf("Failed to send push: %v", err)
            }
        }
    }()

    c.JSON(http.StatusCreated, gin.H{
        "message": "Message sent",
        "id":      message.ID.Hex(),
    })
}

func MarkAsRead(c *gin.Context) {
    messageIDStr := c.Param("id")
    messageID, err := primitive.ObjectIDFromHex(messageIDStr)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message ID"})
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

    messagesColl := database.Client.Database("coded").Collection("messages")

    // Get the chat ID from the message and verify access
    var msg models.Message
    err = messagesColl.FindOne(ctx, bson.M{"_id": messageID}).Decode(&msg)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Message not found"})
        return
    }

    chatsColl := database.Client.Database("coded").Collection("chats")
    count, err := chatsColl.CountDocuments(ctx, bson.M{"_id": msg.ChatID, "participants": userID})
    if err != nil || count == 0 {
        c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to chat"})
        return
    }

    // Mark all unread messages from the partner in this chat as read
    result, err := messagesColl.UpdateMany(
        ctx,
        bson.M{
            "chatId":   msg.ChatID,
            "senderId": bson.M{"$ne": userID},
            "isRead":   false,
        },
        bson.M{"$set": bson.M{"isRead": true}},
    )
    if err != nil {
        log.Printf("MarkAsRead error: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark as read"})
        return
    }

    // Broadcast read receipt via WebSocket
    if wsManager != nil && result.ModifiedCount > 0 {
        // Get all message IDs that were marked as read
        cursor, err := messagesColl.Find(ctx, bson.M{
            "chatId":   msg.ChatID,
            "senderId": bson.M{"$ne": userID},
            "isRead":   true,
        })
        if err == nil {
            var messages []models.Message
            if err = cursor.All(ctx, &messages); err == nil {
                var messageIds []string
                for _, msg := range messages {
                    messageIds = append(messageIds, msg.ID.Hex())
                }
                
                wsReadReceipt := map[string]interface{}{
                    "chatId":     msg.ChatID.Hex(),
                    "userId":     userID.Hex(),
                    "messageIds": messageIds,
                    "timestamp":  time.Now().Unix(),
                }
                
                wsManager.BroadcastMessageRead(wsReadReceipt)
            }
        }
    }

    c.JSON(http.StatusOK, gin.H{
        "message":      "Marked as read",
        "updatedCount": result.ModifiedCount,
    })
}

// New function to send typing indicator via WebSocket
func SendTypingIndicator(c *gin.Context) {
    var req struct {
        ChatID string `json:"chatId" binding:"required"`
        Typing bool   `json:"typing"`
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

    chatID, err := primitive.ObjectIDFromHex(req.ChatID)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chat ID"})
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // Verify user is in the chat
    chatsColl := database.Client.Database("coded").Collection("chats")
    count, err := chatsColl.CountDocuments(ctx, bson.M{"_id": chatID, "participants": userID})
    if err != nil || count == 0 {
        c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to chat"})
        return
    }

    // Broadcast typing indicator via WebSocket
    if wsManager != nil {
        typingMsg := map[string]interface{}{
            "chatId":    chatID.Hex(),
            "userId":    userID.Hex(),
            "typing":    req.Typing,
            "timestamp": time.Now().Unix(),
        }
        
        if req.Typing {
            wsManager.BroadcastTypingStart(typingMsg)
        } else {
            wsManager.BroadcastTypingEnd(typingMsg)
        }
    }

    c.JSON(http.StatusOK, gin.H{
        "message": "Typing indicator sent",
        "typing":  req.Typing,
    })
}