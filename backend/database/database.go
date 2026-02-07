package database

import (
    "context"
    "log"
    "os"
    "time"

    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
)

var (
    Client *mongo.Client
    DB     *mongo.Database
)

func ConnectDB() error {
    mongoURI := os.Getenv("MONGODB_URI")
    if mongoURI == "" {
        mongoURI = "mongodb://localhost:27017"
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    clientOptions := options.Client().ApplyURI(mongoURI)
    client, err := mongo.Connect(ctx, clientOptions)
    if err != nil {
        return err
    }

    // Ping the database
    err = client.Ping(ctx, nil)
    if err != nil {
        return err
    }

    Client = client
    DB = client.Database("coded")
    
    log.Println("Connected to MongoDB successfully")
    
    // Create indexes
    CreateIndexes()
    
    return nil
}

func GetCollection(collectionName string) *mongo.Collection {
    return DB.Collection(collectionName)
}

func CreateIndexes() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // Users collection indexes
    usersColl := DB.Collection("users")
    usersIndexes := []mongo.IndexModel{
        {
            Keys:    bson.D{{Key: "email", Value: 1}},
            Options: options.Index().SetUnique(true),
        },
        {
            Keys:    bson.D{{Key: "username", Value: 1}},
            Options: options.Index().SetUnique(true).SetSparse(true),
        },
        {
            Keys: bson.D{{Key: "location", Value: "2dsphere"}},
        },
        {
            Keys: bson.D{{Key: "lastSeen", Value: -1}},
        },
    }

    // Chats collection indexes
    chatsColl := DB.Collection("chats")
    chatsIndexes := []mongo.IndexModel{
        {
            Keys:    bson.D{{Key: "participants", Value: 1}},
            Options: options.Index().SetUnique(true),
        },
        {
            Keys: bson.D{{Key: "lastMessageAt", Value: -1}},
        },
    }

    // Messages collection indexes
    messagesColl := DB.Collection("messages")
    messagesIndexes := []mongo.IndexModel{
        {
            Keys: bson.D{{Key: "chatId", Value: 1}, {Key: "createdAt", Value: -1}},
        },
        {
            Keys: bson.D{{Key: "senderId", Value: 1}},
        },
        {
            Keys: bson.D{{Key: "createdAt", Value: -1}},
        },
    }

    // Favorites collection indexes
    favoritesColl := DB.Collection("favorites")
    favoritesIndexes := []mongo.IndexModel{
        {
            Keys:    bson.D{{Key: "userId", Value: 1}, {Key: "targetUserId", Value: 1}},
            Options: options.Index().SetUnique(true),
        },
        {
            Keys: bson.D{{Key: "createdAt", Value: -1}},
        },
    }

    // Posts collection indexes
    postsColl := DB.Collection("posts")
    postsIndexes := []mongo.IndexModel{
        {
            Keys: bson.D{{Key: "userId", Value: 1}},
        },
        {
            Keys: bson.D{{Key: "createdAt", Value: -1}},
        },
        {
            Keys: bson.D{{Key: "category", Value: 1}},
        },
    }

    // Create all indexes
    if _, err := usersColl.Indexes().CreateMany(ctx, usersIndexes); err != nil {
        log.Printf("Error creating users indexes: %v", err)
    }

    if _, err := chatsColl.Indexes().CreateMany(ctx, chatsIndexes); err != nil {
        log.Printf("Error creating chats indexes: %v", err)
    }

    if _, err := messagesColl.Indexes().CreateMany(ctx, messagesIndexes); err != nil {
        log.Printf("Error creating messages indexes: %v", err)
    }

    if _, err := favoritesColl.Indexes().CreateMany(ctx, favoritesIndexes); err != nil {
        log.Printf("Error creating favorites indexes: %v", err)
    }

    if _, err := postsColl.Indexes().CreateMany(ctx, postsIndexes); err != nil {
        log.Printf("Error creating posts indexes: %v", err)
    }

    log.Println("Database indexes created successfully")
}