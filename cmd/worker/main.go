package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/gabrielgcosta/ticketblast-core/db/sqlc"
	"github.com/gabrielgcosta/ticketblast-core/internal/entity"
	infraDB "github.com/gabrielgcosta/ticketblast-core/internal/infra/db"
	"github.com/gabrielgcosta/ticketblast-core/internal/usecase"
	"github.com/gabrielgcosta/ticketblast-core/pkg/logger"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize Pgx connection pool
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		logger.Log.Fatal("Critical: Failed to create database connection pool", zap.Error(err))
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		logger.Log.Fatal("Critical: Failed to ping database", zap.Error(err))
	}

	queries := sqlc.New(pool)
	ticketRepo := infraDB.NewPostgresTicketRepository(queries)
	orderRepo := infraDB.NewPostgresOrderRepository(queries)
	txManager := infraDB.NewPostgresTxManager(pool, queries)

	// RabbitMQ Connection Setup
	rabbitmqURL := os.Getenv("RABBITMQ_URL")
	if rabbitmqURL == "" {
		rabbitmqUser := os.Getenv("RABBITMQ_DEFAULT_USER")
		if rabbitmqUser == "" {
			rabbitmqUser = "guest"
		}
		rabbitmqPass := os.Getenv("RABBITMQ_DEFAULT_PASS")
		if rabbitmqPass == "" {
			rabbitmqPass = "guest"
		}
		rabbitmqHost := os.Getenv("RABBITMQ_HOST")
		if rabbitmqHost == "" {
			rabbitmqHost = "localhost"
		}
		rabbitmqPort := os.Getenv("RABBITMQ_PORT")
		if rabbitmqPort == "" {
			rabbitmqPort = "5672"
		}
		rabbitmqURL = fmt.Sprintf("amqp://%s:%s@%s:%s/", rabbitmqUser, rabbitmqPass, rabbitmqHost, rabbitmqPort)
	}

	conn, err := amqp.Dial(rabbitmqURL)
	if err != nil {
		logger.Log.Fatal("Critical: Failed to connect to RabbitMQ", zap.Error(err))
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		logger.Log.Fatal("Critical: Failed to open RabbitMQ channel", zap.Error(err))
	}
	defer ch.Close()

	// Declare exchange and queue just in case worker runs first
	err = ch.ExchangeDeclare(
		"orders_exchange",
		"direct",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		logger.Log.Fatal("Critical: Failed to declare exchange", zap.Error(err))
	}

	q, err := ch.QueueDeclare(
		"orders_queue",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		logger.Log.Fatal("Critical: Failed to declare queue", zap.Error(err))
	}

	err = ch.QueueBind(
		q.Name,
		"order.created",
		"orders_exchange",
		false,
		nil,
	)
	if err != nil {
		logger.Log.Fatal("Critical: Failed to bind queue", zap.Error(err))
	}

	// Set Qos prefetch count to 1 to evenly distribute messages and avoid overloading worker
	err = ch.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	if err != nil {
		logger.Log.Fatal("Critical: Failed to set channel QoS", zap.Error(err))
	}

	msgs, err := ch.Consume(
		q.Name,
		"ticketblast-worker", // consumer tag
		false,                // autoAck set to false (resilient manual acknowledgments)
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		logger.Log.Fatal("Critical: Failed to register queue consumer", zap.Error(err))
	}

	forever := make(chan bool)

	go func() {
		for msg := range msgs {
			logger.Log.Info("Worker received a message from RabbitMQ")

			var event usecase.OrderCreatedEvent
			if err := json.Unmarshal(msg.Body, &event); err != nil {
				logger.Log.Error("Worker failed to unmarshal JSON payload. Dropping malformed message.", zap.Error(err))
				// Nack and discard since JSON is malformed
				_ = msg.Nack(false, false)
				continue
			}

			// Open transaction, verify stock, process dummy payment, and persist
			dbErr := txManager.RunInTx(ctx, func(ctx context.Context) error {
				ticket, err := ticketRepo.GetByID(ctx, event.TicketID)
				if err != nil {
					return fmt.Errorf("failed to fetch ticket details: %w", err)
				}

				if ticket.TotalQuantity < event.Quantity {
					return fmt.Errorf("insufficient database inventory (total=%d, requested=%d)", ticket.TotalQuantity, event.Quantity)
				}

				logger.Log.Info("Processing payment (fictitious)", zap.String("order_id", event.OrderID), zap.Float64("amount", ticket.Price*float64(event.Quantity)))

				order := &entity.Order{
					ID:          event.OrderID,
					UserID:      event.UserID,
					Status:      entity.OrderStatusCompleted,
					TotalAmount: ticket.Price * float64(event.Quantity),
				}
				createdOrder, err := orderRepo.Create(ctx, order)
				if err != nil {
					return fmt.Errorf("failed to save order: %w", err)
				}

				item := &entity.OrderItem{
					OrderID:   createdOrder.ID,
					TicketID:  event.TicketID,
					Quantity:  event.Quantity,
					UnitPrice: ticket.Price,
				}
				_, err = orderRepo.CreateItem(ctx, item)
				if err != nil {
					return fmt.Errorf("failed to save order item: %w", err)
				}

				err = ticketRepo.UpdateStock(ctx, event.TicketID, event.Quantity)
				if err != nil {
					return fmt.Errorf("failed to update ticket stock: %w", err)
				}

				return nil
			})

			if dbErr != nil {
				logger.Log.Error("Worker failed to persist order. Requeueing message.", zap.String("order_id", event.OrderID), zap.Error(dbErr))
				// Redeliver message to the queue to try again
				_ = msg.Nack(false, true)
			} else {
				logger.Log.Info("Worker successfully persisted order and acknowledged message", zap.String("order_id", event.OrderID))
				_ = msg.Ack(false)
			}
		}
		forever <- true
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	logger.Log.Info("Worker loop started successfully. Listening for order messages...")

	select {
	case sig := <-sigChan:
		logger.Log.Info("Worker shutting down gracefully", zap.String("signal", sig.String()))
		cancel()
	case <-forever:
		logger.Log.Info("RabbitMQ channel closed, stopping worker")
	}
}
