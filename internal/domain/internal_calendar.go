package domain

import "time"

type InternalCalendarEvent struct {
	ID             string                   `json:"id"`
	SeedbedID      string                   `json:"seedbedId"`
	Type           string                   `json:"type"`
	Title          string                   `json:"title"`
	Description    string                   `json:"description,omitempty"`
	StartAt        time.Time                `json:"startAt"`
	EndAt          time.Time                `json:"endAt"`
	OrganizerID    string                   `json:"organizerId"`
	ParticipantIDs []string                 `json:"participantIds"`
	Participants   []InternalCalendarPerson `json:"participants,omitempty"`
	Status         string                   `json:"status"`
	CreatedAt      time.Time                `json:"createdAt"`
	UpdatedAt      time.Time                `json:"updatedAt"`
}

type InternalCalendarPerson struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	LastName string `json:"lastName"`
	Role     string `json:"role"`
}
