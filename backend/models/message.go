package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Message struct {
    ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    ChatID    primitive.ObjectID `bson:"chatId" json:"chatId"`
    SenderID  primitive.ObjectID `bson:"senderId" json:"senderId"`
    Content   string             `bson:"content" json:"content"`
    Type      string             `bson:"type" json:"type"` // text, image, voice
    IsRead    bool               `bson:"isRead" json:"isRead"`
    CreatedAt int64              `bson:"createdAt" json:"createdAt"`
}
