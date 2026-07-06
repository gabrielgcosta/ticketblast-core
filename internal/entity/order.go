package entity

import "time"

type OrderStatus string

const (
	OrderStatusDraft     OrderStatus = "draft"
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusCompleted OrderStatus = "completed"
	OrderStatusFailed    OrderStatus = "failed"
)

type Order struct {
	ID          string       `json:"id"`
	UserID      string       `json:"user_id"`
	Status      OrderStatus  `json:"status"`
	TotalAmount float64      `json:"total_amount"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	Items       []*OrderItem `json:"items,omitempty"`
}

type OrderItem struct {
	ID        string    `json:"id"`
	OrderID   string    `json:"order_id"`
	TicketID  string    `json:"ticket_id"`
	Quantity  int       `json:"quantity"`
	UnitPrice float64   `json:"unit_price"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
