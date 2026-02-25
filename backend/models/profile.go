package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Profile struct {
	ID           primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	UserID       primitive.ObjectID   `bson:"userId" json:"userId"`
	Username     string               `bson:"username" json:"username"`
	Name         string               `bson:"name" json:"name"`
	Avatar       string               `bson:"avatar" json:"avatar"`
	Bio          string               `bson:"bio" json:"bio"`
	Interests    []string             `bson:"interests" json:"interests"`
	Gender       string               `bson:"gender" json:"gender"`             // male, female, other
	InterestedIn []string             `bson:"interestedIn" json:"interestedIn"` // men, women, everyone
	Photos       []string             `bson:"photos" json:"photos"`             // array of image URLs
	Location     *Location            `bson:"location,omitempty" json:"location"`
	Status       string               `bson:"status" json:"status"`             // available, busy
	BirthDate    int64                `bson:"birthDate" json:"birthDate"`       // Unix timestamp
	LastSeen     int64                `bson:"lastSeen" json:"lastSeen"`
}

type Location struct {
	Latitude  float64 `bson:"latitude" json:"latitude"`
	Longitude float64 `bson:"longitude" json:"longitude"`
}