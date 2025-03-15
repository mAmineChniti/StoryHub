package database

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	_ "github.com/joho/godotenv/autoload"
	"github.com/mAmineChniti/StoryHub/internal/data"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Service interface {
	CreateStory(req *data.StoryDetails) (bool, error)
	GetStoryDetails(id primitive.ObjectID) (*data.StoryDetails, error)
	GetStoryContent(id primitive.ObjectID) (*data.StoryContent, error)
	GetStories(page, limit int) ([]data.StoryDetails, error)
	GetStoryCollaborators(id primitive.ObjectID) ([]primitive.ObjectID, error)
	GetStoriesByFilters(genres []string, page, limit int) ([]data.StoryDetails, error)
	GetStoriesByUser(userID primitive.ObjectID, page, limit int) ([]data.StoryDetails, error)
	GetCollaborations(userID primitive.ObjectID, page, limit int) ([]data.StoryDetails, error)
	EditStoryContent(id primitive.ObjectID, content string) (bool, error)
	DeleteStory(id primitive.ObjectID) (bool, error)
	ValidateToken(authHeader string) (primitive.ObjectID, error)
	Health() (map[string]string, error)
}

type service struct {
	db *mongo.Client
}

var (
	dbUsername       = os.Getenv("DB_USERNAME")
	dbPassword       = os.Getenv("DB_PASSWORD")
	connectionString = os.Getenv("DB_CONNECTION_STRING")
	jwtSecret        = []byte(os.Getenv("JWTSECRET"))
)

func New() Service {
	uri := fmt.Sprintf("mongodb+srv://%s:%s%s", dbUsername, dbPassword, connectionString)
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))

	if err != nil {
		log.Fatal(err)

	}
	return &service{
		db: client,
	}
}

func (s *service) CreateStory(req *data.StoryDetails) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()

	res, err := s.db.Database("storyhub").Collection("storydetails").InsertOne(ctx, req)
	if err != nil {
		return false, fmt.Errorf("error inserting story: %v", err)
	}

	req.ID = res.InsertedID.(primitive.ObjectID)
	return true, nil
}

func (s *service) GetStoryDetails(id primitive.ObjectID) (*data.StoryDetails, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var story data.StoryDetails
	err := s.db.Database("storyhub").Collection("storydetails").FindOne(ctx, primitive.M{"_id": id}).Decode(&story)
	if err != nil {
		return nil, fmt.Errorf("error fetching story: %v", err)
	}

	return &story, nil
}

func (s *service) GetStoryContent(id primitive.ObjectID) (*data.StoryContent, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var story data.StoryDetails
	err := s.db.Database("storyhub").Collection("storydetails").FindOne(ctx, primitive.M{"_id": id}).Decode(&story)
	if err != nil {
		return nil, fmt.Errorf("story not found")
	}

	var content data.StoryContent
	err = s.db.Database("storyhub").Collection("storycontent").FindOne(ctx, primitive.M{"story_id": id}).Decode(&content)
	if err != nil {
		return &data.StoryContent{StoryID: id}, nil
	}

	return &content, nil
}

func (s *service) GetStories(page, limit int) ([]data.StoryDetails, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	skip := (page - 1) * limit
	findOptions := options.Find().SetSkip(int64(skip)).SetLimit(int64(limit))

	cursor, err := s.db.Database("storyhub").Collection("storydetails").Find(ctx, primitive.M{}, findOptions)
	if err != nil {
		return nil, fmt.Errorf("error fetching stories: %v", err)
	}
	defer cursor.Close(ctx)

	var stories []data.StoryDetails
	if err := cursor.All(ctx, &stories); err != nil {
		return nil, fmt.Errorf("error decoding stories: %v", err)
	}

	return stories, nil
}

func (s *service) GetStoryCollaborators(id primitive.ObjectID) ([]primitive.ObjectID, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var story data.StoryDetails
	err := s.db.Database("storyhub").Collection("storydetails").FindOne(ctx, primitive.M{"_id": id}).Decode(&story)
	if err != nil {
		return nil, fmt.Errorf("error fetching story: %v", err)
	}
	return story.Collaborators, nil
}

func (s *service) GetStoriesByFilters(genres []string, page, limit int) ([]data.StoryDetails, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := primitive.M{}
	if len(genres) > 0 {
		filter["genre"] = primitive.M{"$in": genres}
	}

	skip := (page - 1) * limit
	findOptions := options.Find().SetSkip(int64(skip)).SetLimit(int64(limit))

	cursor, err := s.db.Database("storyhub").Collection("storydetails").Find(ctx, filter, findOptions)
	if err != nil {
		return nil, fmt.Errorf("error fetching stories: %v", err)
	}
	defer cursor.Close(ctx)

	var stories []data.StoryDetails
	if err := cursor.All(ctx, &stories); err != nil {
		return nil, fmt.Errorf("error decoding stories: %v", err)
	}

	return stories, nil
}

func (s *service) GetStoriesByUser(userID primitive.ObjectID, page, limit int) ([]data.StoryDetails, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	skip := (page - 1) * limit
	findOptions := options.Find().SetSkip(int64(skip)).SetLimit(int64(limit))
	filter := primitive.M{"owner_id": userID}
	cursor, err := s.db.Database("storyhub").Collection("storydetails").Find(ctx, filter, findOptions)
	if err != nil {
		return nil, fmt.Errorf("error fetching stories: %v", err)
	}
	defer cursor.Close(ctx)
	var stories []data.StoryDetails
	if err := cursor.All(ctx, &stories); err != nil {
		return nil, fmt.Errorf("error decoding stories: %v", err)
	}
	return stories, nil
}

func (s *service) GetCollaborations(userID primitive.ObjectID, page, limit int) ([]data.StoryDetails, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	skip := (page - 1) * limit
	findOptions := options.Find().SetSkip(int64(skip)).SetLimit(int64(limit))

	filter := primitive.M{"collaborators": primitive.M{"$in": []primitive.ObjectID{userID}}}

	cursor, err := s.db.Database("storyhub").Collection("storydetails").Find(ctx, filter, findOptions)
	if err != nil {
		return nil, fmt.Errorf("error fetching stories: %v", err)
	}
	defer cursor.Close(ctx)

	var stories []data.StoryDetails
	if err := cursor.All(ctx, &stories); err != nil {
		return nil, fmt.Errorf("error decoding stories: %v", err)
	}

	return stories, nil
}

func (s *service) EditStoryContent(storyID primitive.ObjectID, newContent string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var story data.StoryDetails
	err := s.db.Database("storyhub").Collection("storydetails").FindOne(ctx, primitive.M{"_id": storyID}).Decode(&story)
	if err != nil {
		return false, fmt.Errorf("story not found")
	}

	filterContent := primitive.M{"story_id": storyID}
	updateContent := primitive.M{"$set": primitive.M{"content": newContent}}

	res, err := s.db.Database("storyhub").Collection("storycontent").UpdateOne(ctx, filterContent, updateContent)
	if err != nil {
		return false, fmt.Errorf("error updating story content: %v", err)
	}

	if res.MatchedCount == 0 {
		newStoryContent := data.StoryContent{
			StoryID: storyID,
			Content: newContent,
		}
		_, err := s.db.Database("storyhub").Collection("storycontent").InsertOne(ctx, newStoryContent)
		if err != nil {
			return false, fmt.Errorf("error inserting new story content: %v", err)
		}
	}

	_, err = s.db.Database("storyhub").Collection("storydetails").UpdateOne(ctx, primitive.M{"_id": storyID}, primitive.M{"$set": primitive.M{"updated_at": time.Now()}})
	if err != nil {
		return false, fmt.Errorf("error updating story details: %v", err)
	}

	return true, nil
}

func (s *service) DeleteStory(storyID primitive.ObjectID) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := primitive.M{"_id": storyID}
	res, err := s.db.Database("storyhub").Collection("storydetails").DeleteOne(ctx, filter)
	if err != nil {
		return false, fmt.Errorf("error deleting story: %v", err)
	}
	if res.DeletedCount == 0 {
		return false, fmt.Errorf("story not found")
	}

	filterContent := primitive.M{"story_id": storyID}
	contentDel, err := s.db.Database("storyhub").Collection("storycontent").DeleteOne(ctx, filterContent)
	if err != nil {
		return false, fmt.Errorf("error deleting story content: %v", err)
	}
	if contentDel.DeletedCount == 0 {
		log.Printf("story content not found")
	}

	return true, nil
}

func (s *service) ValidateToken(authHeader string) (primitive.ObjectID, error) {
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return primitive.ObjectID{}, fmt.Errorf("invalid token format")
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		return jwtSecret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return primitive.ObjectID{}, fmt.Errorf("token expired: %v", err)
		}
		return primitive.ObjectID{}, fmt.Errorf("invalid token: %v", err)
	}

	if !token.Valid {
		return primitive.ObjectID{}, fmt.Errorf("invalid token")
	}

	userID, err := primitive.ObjectIDFromHex(claims.Subject)
	if err != nil {
		return primitive.ObjectID{}, fmt.Errorf("invalid user ID: %v", err)
	}

	return userID, nil
}

func (s *service) Health() (map[string]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := s.db.Ping(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("db down: %v", err)
	}

	return map[string]string{"message": "It's healthy"}, nil
}
