package server

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/mAmineChniti/StoryHub/internal/data"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"slices"
)

func (s *Server) RegisterRoutes() http.Handler {
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{"https://*", "http://*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	e.Logger.SetLevel(log.INFO)
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "method=${method}, uri=${uri}, status=${status}\n",
	}))

	DEBUG(e)

	e.GET("/", func(c echo.Context) error {
		return c.Redirect(http.StatusMovedPermanently, "/api/v1")
	})
	e.POST("/api/v1/create-story", s.CreateStory, s.JWTMiddleware())
	e.GET("/api/v1/get-story-details/:story_id", s.GetStoryDetails)
	e.GET("/api/v1/get-story-content/:story_id", s.GetStoryContent)
	e.POST("/api/v1/get-stories", s.GetStories)
	e.GET("/api/v1/get-story-collaborators/:story_id", s.GetStoryCollaborators)
	e.POST("/api/v1/get-stories-by-filters", s.GetStoriesByFilters)
	e.POST("/api/v1/get-stories-by-user", s.GetStoriesByUser)
	e.POST("/api/v1/collaborations", s.GetCollaborations, s.JWTMiddleware())
	e.POST("/api/v1/edit-story", s.EditStory, s.JWTMiddleware())
	e.DELETE("/api/v1/delete-story/:story_id", s.DeleteStory, s.JWTMiddleware())
	e.GET("/api/v1/health", s.healthHandler)
	e.RouteNotFound("/*", func(c echo.Context) error {
		return c.JSON(http.StatusNotFound, map[string]string{"message": "Not found"})
	})

	return e
}

var (
	debug = os.Getenv("DEBUG") == "true"
)

func DEBUG(e *echo.Echo) {
	if debug {
		e.Use(middleware.BodyDump(func(c echo.Context, reqBody, resBody []byte) {
			if len(reqBody) > 0 {
				var formattedReq any
				if err := json.Unmarshal(reqBody, &formattedReq); err != nil {
					log.Printf("Request Body (raw): \n%s\n", string(reqBody))
					c.Logger().Error("Error parsing request body: " + err.Error())
				} else {
					reqBodyJson, err := json.MarshalIndent(formattedReq, "", "  ")
					if err != nil {
						log.Printf("Request Body (raw): \n%s\n", string(reqBody))
						c.Logger().Error("Error marshaling request body: " + err.Error())
					} else {
						c.Logger().Debug("Request Body:\n" + string(reqBodyJson))
					}
				}
			}

			if len(resBody) > 0 {
				var formattedRes any
				if err := json.Unmarshal(resBody, &formattedRes); err != nil {
					log.Printf("Response Body (raw): \n%s\n", string(resBody))
					c.Logger().Error("Error parsing response body: " + err.Error())
				} else {
					resBodyJson, err := json.MarshalIndent(formattedRes, "", "  ")
					if err != nil {
						log.Printf("Response Body (raw): \n%s\n", string(resBody))
						c.Logger().Error("Error marshaling response body: " + err.Error())
					} else {
						c.Logger().Debug("Response Body:\n" + string(resBodyJson))
					}
				}
			}
		}))
	}
}

func (s *Server) CreateStory(c echo.Context) error {
	var story data.StoryDetails
	story.OwnerID = c.Get("user_id").(primitive.ObjectID)
	if err := c.Bind(&story); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "Invalid request body"})
	}
	created, err := s.db.CreateStory(&story)
	if err != nil {
		c.Logger().Error(err.Error())
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "Internal server error"})
	}
	if !created {
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "Failed to create story"})
	}
	return c.JSON(http.StatusCreated, map[string]string{"message": "Story created successfully"})
}

func (s *Server) GetStoryDetails(c echo.Context) error {
	story_id, err := primitive.ObjectIDFromHex(c.Param("story_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "Invalid story ID"})
	}
	story, err := s.db.GetStoryDetails(story_id)
	if err != nil || story == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"message": "Story not found"})
	}
	return c.JSON(http.StatusOK, map[string]any{"message": "Story found", "story": story})
}

func (s *Server) GetStoryContent(c echo.Context) error {
	story_id, err := primitive.ObjectIDFromHex(c.Param("story_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "Invalid story ID"})
	}
	content, err := s.db.GetStoryContent(story_id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"message": "Story content not found"})
	}
	return c.JSON(http.StatusOK, map[string]any{"message": "Story content found", "content": content})
}

func (s *Server) GetStories(c echo.Context) error {
	var request struct {
		Page  int `json:"page"`
		Limit int `json:"limit"`
	}
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "Invalid request body"})
	}
	stories, err := s.db.GetStories(request.Page, request.Limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "Internal server error"})
	}
	return c.JSON(http.StatusOK, map[string]any{"message": "Stories found", "stories": stories})
}

func (s *Server) GetStoryCollaborators(c echo.Context) error {
	story_id, err := primitive.ObjectIDFromHex(c.Param("story_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "Invalid story ID"})
	}

	collaborators, err := s.db.GetStoryCollaborators(story_id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "Internal server error"})
	}

	return c.JSON(http.StatusOK, map[string]any{"message": "Collaborators found", "collaborators": collaborators})
}

func (s *Server) GetStoriesByFilters(c echo.Context) error {
	var request struct {
		Genres []string `json:"genres"`
		Page   int      `json:"page"`
		Limit  int      `json:"limit"`
	}
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "Invalid request body"})
	}
	stories, err := s.db.GetStoriesByFilters(request.Genres, request.Page, request.Limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "Internal server error"})
	}
	return c.JSON(http.StatusOK, map[string]any{"message": "Stories found", "stories": stories})
}

func (s *Server) GetStoriesByUser(c echo.Context) error {
	var request struct {
		UserID string `json:"user_id"`
		Page   int    `json:"page"`
		Limit  int    `json:"limit"`
	}
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "Invalid request body"})
	}
	userID, err := primitive.ObjectIDFromHex(request.UserID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "Invalid user ID"})
	}
	stories, err := s.db.GetStoriesByUser(userID, request.Page, request.Limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "Internal server error"})
	}
	return c.JSON(http.StatusOK, map[string]any{"message": "Stories found", "stories": stories})
}

func (s *Server) GetCollaborations(c echo.Context) error {
	var request struct {
		Page  int `json:"page"`
		Limit int `json:"limit"`
	}
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "Invalid request body"})
	}
	userID := c.Get("user_id").(primitive.ObjectID)
	collaborations, err := s.db.GetCollaborations(userID, request.Page, request.Limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "Internal server error"})
	}
	return c.JSON(http.StatusOK, map[string]any{"message": "Collaborations found", "collaborations": collaborations})
}

func (s *Server) EditStory(c echo.Context) error {
	var updatedStory struct {
		ID      string `json:"story_id"`
		Content string `json:"content"`
	}
	if err := c.Bind(&updatedStory); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "Invalid request body"})
	}
	storyId, err := primitive.ObjectIDFromHex(updatedStory.ID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "Invalid story ID"})
	}
	story, err := s.db.GetStoryDetails(storyId)
	if err != nil || story == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"message": "Story not found"})
	}
	userId, ok := c.Get("user_id").(primitive.ObjectID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
	}

	isAuthorized := false
	if userId == story.OwnerID {
		isAuthorized = true
	} else {
		if slices.Contains(story.Collaborators, userId) {
			isAuthorized = true
		}
	}
	if !isAuthorized {
		return c.JSON(http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
	}
	updated, err := s.db.EditStoryContent(storyId, updatedStory.Content)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "Internal server error"})
	}
	if !updated {
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "Failed to update story content"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Story content updated successfully"})

}

func (s *Server) DeleteStory(c echo.Context) error {
	storyId, err := primitive.ObjectIDFromHex(c.Param("story_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "Invalid story ID"})
	}
	story, err := s.db.GetStoryDetails(storyId)
	if err != nil || story == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"message": "Story not found"})
	}
	userId, ok := c.Get("user_id").(primitive.ObjectID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
	}
	if userId != story.OwnerID {
		return c.JSON(http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
	}
	deleted, err := s.db.DeleteStory(storyId)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "Internal server error"})
	}
	if !deleted {
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "Failed to delete story"})
	}
	return c.JSON(http.StatusOK, map[string]string{"message": "Story deleted successfully"})
}

func (s *Server) JWTMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				return c.JSON(http.StatusUnauthorized, map[string]string{"message": "Invalid token format"})
			}

			userID, err := s.db.ValidateToken(authHeader)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"message": "Unauthorized: Invalid or expired token"})
			}

			c.Set("user_id", userID)
			return next(c)
		}
	}
}

func (s *Server) healthHandler(c echo.Context) error {
	health, err := s.db.Health()
	if err != nil {
		c.Logger().Error(err.Error())
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "Internal server error"})
	}
	return c.JSON(http.StatusOK, health)
}
