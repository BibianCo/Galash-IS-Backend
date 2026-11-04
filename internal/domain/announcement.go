package domain

import "time"

type AnnouncementAttachment struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type Announcement struct {
	ID            string                   `json:"id"`
	Title         string                   `json:"title"`
	Description   string                   `json:"description"`
	Attachments   []AnnouncementAttachment `json:"attachments"`
	Status        string                   `json:"status"`
	AuthorID      string                   `json:"authorId"`
	AuthorName    string                   `json:"authorName,omitempty"`
	CreatedAt     time.Time                `json:"createdAt"`
	UpdatedAt     time.Time                `json:"updatedAt"`
	PublishedAt   *time.Time               `json:"publishedAt,omitempty"`
	ReadConfirmed bool                     `json:"readConfirmed"`
}

type AnnouncementReadStatus struct {
	UserID      string     `json:"userId"`
	Name        string     `json:"name"`
	Email       string     `json:"email"`
	Status      string     `json:"status"` // Leído o Pendiente
	ConfirmedAt *time.Time `json:"confirmedAt,omitempty"`
}
