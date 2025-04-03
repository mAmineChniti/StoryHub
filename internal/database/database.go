package database

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	_ "github.com/joho/godotenv/autoload"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/mAmineChniti/StoryHub/internal/data"
)

type Service interface {
	CreateStory(req *data.StoryDetails) (primitive.ObjectID, error)
	GetStoryDetails(id primitive.ObjectID) (*data.StoryDetails, error)
	GetStoryContent(id primitive.ObjectID) (*data.StoryContent, error)
	GetStories(page, limit int) ([]data.StoryDetails, error)
	GetStoryCollaborators(id primitive.ObjectID) ([]primitive.ObjectID, error)
	GetStoriesByFilters(genres []string, page, limit int) ([]data.StoryDetails, error)
	GetStoriesByUser(userID primitive.ObjectID, page, limit int) ([]data.StoryDetails, error)
	GetCollaborations(userID primitive.ObjectID, page, limit int) ([]data.StoryDetails, error)
	EditStoryContent(id primitive.ObjectID, content string) (bool, error)
	ForkStory(id primitive.ObjectID, userID primitive.ObjectID) (primitive.ObjectID, error)
	DeleteStory(id primitive.ObjectID) (bool, error)
	DeleteAllStoriesByUser(userID primitive.ObjectID) (bool, error)
	Health() (map[string]string, error)
	CleanupOrphanedStories() error
}

type service struct {
	db *mongo.Client
}

var (
	dbUsername       = os.Getenv("DB_USERNAME")
	dbPassword       = os.Getenv("DB_PASSWORD")
	connectionString = os.Getenv("DB_CONNECTION_STRING")
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

func (s *service) CreateStory(req *data.StoryDetails) (primitive.ObjectID, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()

	res, err := s.db.Database("storyhub").Collection("storydetails").InsertOne(ctx, req)
	if err != nil {
		return primitive.NilObjectID, fmt.Errorf("error inserting story: %v", err)
	}

	return res.InsertedID.(primitive.ObjectID), nil
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
		if err == mongo.ErrNoDocuments {
			return &data.StoryContent{
				StoryID: id,
				Content: "",
			}, nil
		}
		return nil, fmt.Errorf("error finding story content: %v", err)
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

func (s *service) ForkStory(storyID, userID primitive.ObjectID) (primitive.ObjectID, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	story, err := s.GetStoryDetails(storyID)
	if err != nil {
		return primitive.NilObjectID, fmt.Errorf("story not found: %v", err)
	}

	filter := bson.M{
		"forked_from": storyID,
		"owner_id":    userID,
	}
	var existingFork data.StoryDetails
	err = s.db.Database("storyhub").Collection("storydetails").FindOne(ctx, filter).Decode(&existingFork)
	if err == nil {
		return primitive.NilObjectID, fmt.Errorf("you have already forked this story")
	}
	if err != mongo.ErrNoDocuments {
		return primitive.NilObjectID, fmt.Errorf("error checking existing forks: %v", err)
	}

	forkedStory := &data.StoryDetails{
		ID:            primitive.ObjectID{},
		OwnerID:       userID,
		Title:         story.Title,
		Description:   story.Description,
		Genre:         story.Genre,
		Collaborators: []primitive.ObjectID{},
		ForkedFrom:    story.ID,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	inserted_story_id, err := s.CreateStory(forkedStory)
	if err != nil {
		return primitive.NilObjectID, fmt.Errorf("error creating forked story: %v", err)
	}

	storyContent, err := s.GetStoryContent(storyID)
	if err != nil {
		return primitive.NilObjectID, fmt.Errorf("error getting story content: %v", err)
	}

	if storyContent.Content != "" {
		forkedStoryContent := &data.StoryContent{
			ID:      primitive.ObjectID{},
			StoryID: inserted_story_id,
			Content: storyContent.Content,
		}

		_, err = s.db.Database("storyhub").Collection("storycontent").InsertOne(ctx, forkedStoryContent)
		if err != nil {
			return inserted_story_id, fmt.Errorf("error inserting story content: %v", err)
		}
	}
	return inserted_story_id, nil
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

func (s *service) DeleteAllStoriesByUser(userID primitive.ObjectID) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := primitive.M{"owner_id": userID}
	cursor, err := s.db.Database("storyhub").Collection("storydetails").Find(ctx, filter)
	if err != nil {
		return false, fmt.Errorf("error finding user stories: %v", err)
	}
	defer cursor.Close(ctx)

	var storyIDs []primitive.ObjectID
	for cursor.Next(ctx) {
		var story data.StoryDetails
		if err := cursor.Decode(&story); err != nil {
			return false, fmt.Errorf("error decoding story: %v", err)
		}
		storyIDs = append(storyIDs, story.ID)
	}

	if len(storyIDs) == 0 {
		return true, nil
	}

	_, err = s.db.Database("storyhub").Collection("storydetails").DeleteMany(ctx, primitive.M{"owner_id": userID})
	if err != nil {
		return false, fmt.Errorf("error deleting story details: %v", err)
	}

	_, err = s.db.Database("storyhub").Collection("storycontent").DeleteMany(ctx, primitive.M{"story_id": primitive.M{"$in": storyIDs}})
	if err != nil {
		return false, fmt.Errorf("error deleting story contents: %v", err)
	}

	return true, nil
}

func (s *service) Health() (map[string]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := s.db.Ping(ctx, nil)
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"status": "ok",
	}, nil
}

func (s *service) CleanupOrphanedStories() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	distinctResult, err := s.db.Database("storyhub").Collection("storydetails").Distinct(ctx, "owner_id", bson.M{})
	if err != nil {
		return fmt.Errorf("error fetching unique owner IDs: %v", err)
	}

	var ownerIDs []primitive.ObjectID
	for _, id := range distinctResult {
		if objID, ok := id.(primitive.ObjectID); ok {
			ownerIDs = append(ownerIDs, objID)
		}
	}

	var orphanedOwnerIDs []primitive.ObjectID
	for _, ownerID := range ownerIDs {
		userCheckURL := "https://gordian.onrender.com/api/v1/fetchuserbyid"
		req, err := http.NewRequestWithContext(ctx, "POST", userCheckURL, nil)
		if err != nil {
			return fmt.Errorf("error creating request: %v", err)
		}

		reqBody, err := json.Marshal(map[string]string{
			"user_id": ownerID.Hex(),
		})
		if err != nil {
			return fmt.Errorf("error marshaling request body: %v", err)
		}
		req.Body = io.NopCloser(bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")

		now := time.Now()
		accessClaims := &jwt.RegisteredClaims{
			Subject:   ownerID.Hex(),
			ExpiresAt: jwt.NewNumericDate(now.Add(5 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        "access",
		}

		jwtSecret := []byte(os.Getenv("JWTSECRET"))
		tokenString, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString(jwtSecret)
		if err != nil {
			return fmt.Errorf("error generating JWT token: %v", err)
		}

		req.Header.Set("Authorization", "Bearer "+tokenString)

		client := &http.Client{
			Timeout: 10 * time.Second,
		}
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Error checking user %s: %v", ownerID.Hex(), err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			orphanedOwnerIDs = append(orphanedOwnerIDs, ownerID)
		} else if resp.StatusCode != http.StatusOK {
			log.Printf("Unexpected status checking user %s: %d", ownerID.Hex(), resp.StatusCode)
		}
	}

	if len(orphanedOwnerIDs) > 0 {
		cursor, err := s.db.Database("storyhub").Collection("storydetails").Find(ctx,
			bson.M{"owner_id": bson.M{"$in": orphanedOwnerIDs}},
			options.Find().SetProjection(bson.M{"_id": 1}),
		)
		if err != nil {
			return fmt.Errorf("error finding orphaned story IDs: %v", err)
		}
		defer cursor.Close(ctx)

		var orphanedStoryIDs []primitive.ObjectID
		for cursor.Next(ctx) {
			var result bson.M
			if err := cursor.Decode(&result); err != nil {
				return fmt.Errorf("error decoding cursor result: %v", err)
			}
			if id, ok := result["_id"].(primitive.ObjectID); ok {
				orphanedStoryIDs = append(orphanedStoryIDs, id)
			}
		}

		if err := cursor.Err(); err != nil {
			return fmt.Errorf("cursor error: %v", err)
		}

		_, err = s.db.Database("storyhub").Collection("storydetails").DeleteMany(ctx, bson.M{"owner_id": bson.M{"$in": orphanedOwnerIDs}})
		if err != nil {
			return fmt.Errorf("error deleting orphaned stories: %v", err)
		}

		_, err = s.db.Database("storyhub").Collection("storycontent").DeleteMany(ctx, bson.M{"story_id": bson.M{"$in": orphanedStoryIDs}})
		if err != nil {
			return fmt.Errorf("error deleting orphaned story contents: %v", err)
		}

		_, err = s.db.Database("storyhub").Collection("storydetails").UpdateMany(ctx,
			bson.M{"collaborators": bson.M{"$in": orphanedOwnerIDs}},
			bson.M{"$pull": bson.M{"collaborators": bson.M{"$in": orphanedOwnerIDs}}},
		)
		if err != nil {
			return fmt.Errorf("error removing orphaned users from collaborators: %v", err)
		}
	}
	return nil
}
