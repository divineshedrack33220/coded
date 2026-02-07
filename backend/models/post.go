package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Post struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    primitive.ObjectID `bson:"userId" json:"userId"`
	Content   string             `bson:"content" json:"content"`
	Media     []string           `bson:"media" json:"media"`
	Category  string             `bson:"category,omitempty" json:"category"` // Optional
	CreatedAt int64              `bson:"createdAt" json:"createdAt"`
	User      *Profile           `bson:"-" json:"user,omitempty"` // Populated in response only
}