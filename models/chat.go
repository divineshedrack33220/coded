package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Chat struct {
    ID            primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
    Participants  []primitive.ObjectID `bson:"participants" json:"participants"`
    LastMessage   string               `bson:"lastMessage" json:"lastMessage"`
    LastMessageAt int64                `bson:"lastMessageAt" json:"lastMessageAt"`
}
