package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Chat struct {
	ID            primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	Participants  []primitive.ObjectID `bson:"participants" json:"participants"`
	LastMessage   interface{}          `bson:"lastMessage,omitempty" json:"lastMessage,omitempty"`
	LastMessageAt int64                `bson:"lastMessageAt" json:"lastMessageAt"`
	CreatedAt     int64                `bson:"createdAt,omitempty" json:"createdAt,omitempty"`
}