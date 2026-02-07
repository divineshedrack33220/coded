package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Favorite struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID       primitive.ObjectID `bson:"userId" json:"userId"`
	TargetUserID primitive.ObjectID `bson:"targetUserId" json:"targetUserId"`
	CreatedAt    int64              `bson:"createdAt" json:"createdAt"`
}