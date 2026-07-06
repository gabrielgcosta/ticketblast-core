package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/gabrielgcosta/ticketblast-core/internal/entity"
)

type CreateEventInput struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Location    string    `json:"location"`
	EventDate   time.Time `json:"event_date"`
	TicketPrice float64   `json:"ticket_price"`
	Stock       int       `json:"stock"`
}

type CreateEventOutput struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Location    string    `json:"location"`
	EventDate   time.Time `json:"event_date"`
	TicketID    string    `json:"ticket_id"`
	TicketPrice float64   `json:"ticket_price"`
	Stock       int       `json:"stock"`
}

type CreateEventUseCase struct {
	eventRepo  EventRepository
	ticketRepo TicketRepository
	cache      CacheService
	txManager  TxManager
}

func NewCreateEventUseCase(
	eventRepo EventRepository,
	ticketRepo TicketRepository,
	cache CacheService,
	txManager TxManager,
) *CreateEventUseCase {
	return &CreateEventUseCase{
		eventRepo:  eventRepo,
		ticketRepo: ticketRepo,
		cache:      cache,
		txManager:  txManager,
	}
}

func (uc *CreateEventUseCase) Execute(ctx context.Context, input CreateEventInput) (*CreateEventOutput, error) {
	var createdEvent *entity.Event
	var createdTicket *entity.Ticket

	err := uc.txManager.RunInTx(ctx, func(ctx context.Context) error {
		// 1. Create the event in the database
		event := &entity.Event{
			Title:       input.Title,
			Description: input.Description,
			Location:    input.Location,
			EventDate:   input.EventDate,
		}
		var err error
		createdEvent, err = uc.eventRepo.Create(ctx, event)
		if err != nil {
			return fmt.Errorf("failed to create event: %w", err)
		}

		// 2. Create the default ticket for the event
		ticket := &entity.Ticket{
			EventID:       createdEvent.ID,
			Name:          "General Admission",
			Price:         input.TicketPrice,
			TotalQuantity: input.Stock,
		}
		createdTicket, err = uc.ticketRepo.Create(ctx, ticket)
		if err != nil {
			return fmt.Errorf("failed to create ticket: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 3. Populate initial stock key in Redis
	// Ex: event:105:stock
	redisKey := fmt.Sprintf("event:%s:stock", createdEvent.ID)
	if err := uc.cache.Set(ctx, redisKey, input.Stock, 0); err != nil {
		return nil, fmt.Errorf("failed to populate inventory key in Redis: %w", err)
	}

	return &CreateEventOutput{
		ID:          createdEvent.ID,
		Title:       createdEvent.Title,
		Description: createdEvent.Description,
		Location:    createdEvent.Location,
		EventDate:   createdEvent.EventDate,
		TicketID:    createdTicket.ID,
		TicketPrice: createdTicket.Price,
		Stock:       createdTicket.TotalQuantity,
	}, nil
}
