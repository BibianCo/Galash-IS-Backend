package domain

import "time"

// AuditLog is an immutable record of a critical operation.
type AuditLog struct {
	ID       string    `json:"id"`
	ActorID  string    `json:"actorId"`
	Action   string    `json:"action"`
	Entity   string    `json:"entity"`
	EntityID string    `json:"entityId,omitempty"`
	Date     time.Time `json:"date"`
	Result   string    `json:"result"`
	Origin   string    `json:"origin,omitempty"`
}
