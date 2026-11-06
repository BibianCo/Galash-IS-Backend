package domain

import "time"

type OfficialLink struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	URL         string    `json:"url"`
	Category    string    `json:"category"`
	Visible     bool      `json:"visible"`
	Order       int       `json:"order"`
	Status      string    `json:"status"`
	UpdatedBy   string    `json:"updatedBy,omitempty"`
	UpdatedAt   time.Time `json:"updatedAt"`
	CreatedAt   time.Time `json:"createdAt"`
}
