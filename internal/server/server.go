package server

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/joho/godotenv/autoload"

	"github.com/mAmineChniti/StoryHub/internal/database"
)

type Server struct {
	port int

	db database.Service
}

func NewServer() *http.Server {
	envPort := os.Getenv("PORT")
	if envPort == "" {
		envPort = "8080"
	}
	port, _ := strconv.Atoi(envPort)
	NewServer := &Server{
		port: port,

		db: database.New(),
	}

	// Declare Server config
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", NewServer.port),
		Handler:      NewServer.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return server
}
