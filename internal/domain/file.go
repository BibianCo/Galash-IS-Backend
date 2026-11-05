package domain

import "time"

type FileAttachment struct {
	ID          string    `json:"id"`
	EntityType  string    `json:"entityType"`
	EntityID    string    `json:"entityId"`
	Name        string    `json:"name"`
	ContentType string    `json:"contentType"`
	Size        int64     `json:"size"`
	UploadedBy  string    `json:"uploadedBy"`
	CreatedAt   time.Time `json:"createdAt"`
}
