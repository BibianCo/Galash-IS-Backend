package domain

import "time"

type Event struct {
	ID              string     `json:"id"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	Type            string     `json:"type"`
	StartAt         time.Time  `json:"startAt"`
	EndAt           *time.Time `json:"endAt,omitempty"`
	OrganizerID     string     `json:"organizerId"`
	OrganizerName   string     `json:"organizerName,omitempty"`
	Venue           string     `json:"venue"`
	MeetingURL      string     `json:"meetingUrl"`
	ImageURL        string     `json:"imageUrl"`
	RegistrationURL string     `json:"registrationUrl"`
	Status          string     `json:"status"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}
