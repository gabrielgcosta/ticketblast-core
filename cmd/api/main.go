package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gabrielgcosta/ticketblast-core/db"
	"github.com/gabrielgcosta/ticketblast-core/internal/infra/web/middleware"
	"github.com/gabrielgcosta/ticketblast-core/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	logger.Init(os.Getenv("APP_ENV"))

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbHost := os.Getenv("DB_HOST")
		if dbHost == "" {
			dbHost = "localhost"
		}
		dbPort := os.Getenv("DB_PORT")
		if dbPort == "" {
			dbPort = "5432"
		}
		dbUser := os.Getenv("POSTGRES_USER")
		if dbUser == "" {
			dbUser = "ticketblast_user"
		}
		dbPassword := os.Getenv("POSTGRES_PASSWORD")
		if dbPassword == "" {
			dbPassword = "ticketblast_password"
		}
		dbName := os.Getenv("POSTGRES_DB")
		if dbName == "" {
			dbName = "ticketblast_db"
		}
		dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPassword, dbHost, dbPort, dbName)
	}

	if err := db.RunMigrations(dbURL); err != nil {
		log.Fatal("Critical: Database migration failed: ", err)
	}

	r := gin.New()
	r.Use(middleware.Logger())
	r.Use(gin.Recovery())

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	log.Println("Starting API server on port :8080...")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Critical: Server failed to start: ", err)
	}
}
