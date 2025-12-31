package database

import (
	"context"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var Client *mongo.Client
var Users *mongo.Collection
var Posts *mongo.Collection
var Favorites *mongo.Collection
var PushSubs *mongo.Collection
var Chats *mongo.Collection

func ConnectMongo() error {
	// Read MongoDB URI from environment variable
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		log.Println("MONGODB_URI not set, using default localhost")
		uri = "mongodb://127.0.0.1:27017"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var err error
	Client, err = mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return err
	}

	// Ping MongoDB
	if err := Client.Ping(ctx, nil); err != nil {
		return err
	}

	db := Client.Database("coded")
	Users = db.Collection("users")
	Posts = db.Collection("posts")
	Favorites = db.Collection("favorites")
	PushSubs = db.Collection("push_subscriptions")
	Chats = db.Collection("chats")

	log.Println("Connected to MongoDB successfully")
	return nil
}

func DisconnectMongo() error {
	if Client == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := Client.Disconnect(ctx); err != nil {
		return err
	}

	log.Println("Disconnected from MongoDB")
	return nil
}