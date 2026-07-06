package entity

import "time"

type Ticket struct {
	ID            string    `json:"id"`
	EventID       string    `json:"event_id"`
	Name          string    `json:"name"`
	Price         float64   `json:"price"`
	TotalQuantity int       `json:"total_quantity"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
