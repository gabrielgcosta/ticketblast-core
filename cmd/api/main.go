package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/gabrielgcosta/ticketblast-core/db"
	"github.com/gabrielgcosta/ticketblast-core/db/sqlc"
	"github.com/gabrielgcosta/ticketblast-core/internal/infra/auth"
	"github.com/gabrielgcosta/ticketblast-core/internal/infra/cache"
	infraDB "github.com/gabrielgcosta/ticketblast-core/internal/infra/db"
	"github.com/gabrielgcosta/ticketblast-core/internal/infra/web/handlers"
	"github.com/gabrielgcosta/ticketblast-core/internal/infra/web/middleware"
	"github.com/gabrielgcosta/ticketblast-core/internal/usecase"
	"github.com/gabrielgcosta/ticketblast-core/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func main() {
	if err := godotenv.Load(); err != nil {
		logger.Log.Info("No .env file found, using system environment variables")
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

	// Run migrations
	if err := db.RunMigrations(dbURL); err != nil {
		logger.Log.Fatal("Critical: Database migration failed", zap.Error(err))
	}

	ctx := context.Background()

	// Initialize Pgx connection pool
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		logger.Log.Fatal("Critical: Failed to create database connection pool", zap.Error(err))
	}
	defer pool.Close()

	// Ping database to verify connection
	if err := pool.Ping(ctx); err != nil {
		logger.Log.Fatal("Critical: Failed to ping database", zap.Error(err))
	}

	// Initialize SQLC queries
	queries := sqlc.New(pool)

	// Initialize Redis Cache Service
	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "localhost"
	}
	redisPort := os.Getenv("REDIS_PORT")
	if redisPort == "" {
		redisPort = "6379"
	}
	redisAddr := fmt.Sprintf("%s:%s", redisHost, redisPort)
	redisPassword := os.Getenv("REDIS_PASSWORD")
	redisDBStr := os.Getenv("REDIS_DB")
	redisDB := 0
	if redisDBStr != "" {
		if val, err := strconv.Atoi(redisDBStr); err == nil {
			redisDB = val
		}
	}

	redisCache := cache.NewRedisCacheService(redisAddr, redisPassword, redisDB)
	defer redisCache.Close()

	// Ping Redis to verify connection
	if err := redisCache.Ping(ctx); err != nil {
		logger.Log.Fatal("Critical: Failed to ping Redis", zap.Error(err))
	}

	// Initialize repositories
	userRepo := infraDB.NewPostgresUserRepository(queries)
	eventRepo := infraDB.NewPostgresEventRepository(queries)

	// Initialize use cases
	registerUC := usecase.NewRegisterUserUseCase(userRepo)
	loginUC := usecase.NewLoginUserUseCase(userRepo)
	listActiveEventsUC := usecase.NewListActiveEventsUseCase(eventRepo, redisCache)

	// Initialize Auth Token Engine
	jwtSecret := os.Getenv("JWT_SECRET")
	tokenEngine := auth.NewTokenEngine(jwtSecret)

	// Initialize handlers
	userHandler := handlers.NewUserHandler(registerUC, loginUC, tokenEngine)
	eventHandler := handlers.NewEventHandler(listActiveEventsUC)

	r := gin.New()
	r.Use(middleware.Logger())
	r.Use(gin.Recovery())

	// Public routes
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	r.POST("/register", userHandler.Register)
	r.POST("/login", userHandler.Login)
	r.GET("/events/active", eventHandler.ListActive)

	// Private routes
	private := r.Group("/")
	private.Use(middleware.Auth(tokenEngine))
	{
		private.GET("/protected-ping", func(c *gin.Context) {
			userID, _ := c.Get("user_id")
			role, _ := c.Get("user_role")
			c.JSON(http.StatusOK, gin.H{
				"message": "pong from protected area",
				"user_id": userID,
				"role":    role,
			})
		})
	}

	logger.Log.Info("Starting API server on port :8080...")
	if err := r.Run(":8080"); err != nil {
		logger.Log.Fatal("Critical: Server failed to start", zap.Error(err))
	}
}
