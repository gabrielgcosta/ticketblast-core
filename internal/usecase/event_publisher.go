package usecase

import "context"

type OrderCreatedEvent struct {
	OrderID  string  `json:"order_id"`
	UserID   string  `json:"user_id"`
	EventID  string  `json:"event_id"`
	TicketID string  `json:"ticket_id"`
	Quantity int     `json:"quantity"`
	Price    float64 `json:"price"`
}

type EventPublisher interface {
	PublishOrderCreated(ctx context.Context, event *OrderCreatedEvent) error
}
