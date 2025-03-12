package data

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type StoryDetails struct {
	ID            primitive.ObjectID   `json:"id" bson:"_id,omitempty"`
	Title         string               `json:"title" bson:"title" validate:"required,min=3,max=100"`
	Genre         string               `json:"genre" bson:"genre" validate:"required,min=3,max=100"`
	Description   string               `json:"description" bson:"description" validate:"required,min=10,max=500"`
	OwnerID       primitive.ObjectID   `json:"owner_id" bson:"owner_id" validate:"required"`
	Collaborators []primitive.ObjectID `json:"collaborators,omitempty" bson:"collaborators,omitempty"`
	CreatedAt     time.Time            `json:"created_at" bson:"created_at"`
	UpdatedAt     time.Time            `json:"updated_at" bson:"updated_at"`
	ForkedFrom    *primitive.ObjectID  `json:"forked_from,omitempty" bson:"forked_from,omitempty"`
}

type StoryContent struct {
	ID      primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	StoryID primitive.ObjectID `json:"story_id" bson:"story_id" validate:"required"`
	Content string             `json:"content" bson:"content" validate:"required,min=20"`
}

type ForkRequest struct {
	StoryID primitive.ObjectID `json:"story_id" validate:"required"`
	UserID  primitive.ObjectID `json:"user_id" validate:"required"`
}

type Stories struct {
	Stories []StoryDetails `json:"stories"`
}
