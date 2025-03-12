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
	e.POST("/api/v1/get-story-details", s.GetStoryDetails)
	e.GET("/api/v1/get-story-content", s.GetStoryContent)
	e.POST("/api/v1/get-stories", s.GetStories)
	e.GET("/api/v1/get-story-collaborators", s.GetStoryCollaborators)
	e.POST("/api/v1/get-stories-by-filters", s.GetStoriesByFilters)
	e.POST("/api/v1/get-stories-by-user", s.GetStoriesByUser)
	e.GET("/api/v1/collaborations", s.GetCollaborations, s.JWTMiddleware())
	// e.PUT("/api/v1/update", s.Update, s.JWTMiddleware())
	// e.PATCH("/api/v1/update", s.Update, s.JWTMiddleware())
	// e.DELETE("/api/v1/delete", s.Delete, s.JWTMiddleware())
	// e.GET("/api/v1/refresh", s.RefreshTokenHandler, s.JWTMiddleware())
	e.GET("/api/v1/health", s.healthHandler)
	e.RouteNotFound("/*", func(c echo.Context) error {
		return c.JSON(http.StatusNotFound, map[string]string{"message": "Not found"})
	})

	return e
}

var (
	jwtSecret = []byte(os.Getenv("JWTSECRET"))
	debug     = os.Getenv("DEBUG") == "true"
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
	var request struct {
		ID string `json:"id"`
	}
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "Invalid request body"})
	}
	id, err := primitive.ObjectIDFromHex(request.ID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "Invalid story ID"})
	}
	story, err := s.db.GetStoryDetails(id)
	if err != nil || story == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"message": "Story not found"})
	}
	return c.JSON(http.StatusOK, map[string]any{"message": "Story found", "story": story})
}

func (s *Server) GetStoryContent(c echo.Context) error {
	var request struct {
		ID string `json:"id"`
	}
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "Invalid request body"})
	}
	id, err := primitive.ObjectIDFromHex(request.ID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "Invalid story ID"})
	}
	content, err := s.db.GetStoryContent(id)
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
	var request struct {
		ID string `json:"id"`
	}

	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "Invalid request body"})
	}

	id, err := primitive.ObjectIDFromHex(request.ID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "Invalid story ID"})
	}

	collaborators, err := s.db.GetStoryCollaborators(id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "Internal server error"})
	}

	var response struct {
		Message       string               `json:"message"`
		Collaborators []primitive.ObjectID `json:"collaborators"`
	}

	response.Message = "Collaborators fetched successfully"
	response.Collaborators = collaborators

	return c.JSON(http.StatusOK, response)
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
