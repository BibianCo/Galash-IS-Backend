package domain

import "time"

type MemberProfile struct {
	ID              string   `json:"id"`
	FirebaseUID     string   `json:"firebase_uid,omitempty"`
	Name            string   `json:"name"`
	LastName        string   `json:"last_name"`
	Email           string   `json:"email,omitempty"`
	DocumentType    string   `json:"document_type,omitempty"`
	DocumentNumber  string   `json:"document_number,omitempty"`
	Role            string   `json:"role"`
	Status          string   `json:"status"`
	Area            string   `json:"area,omitempty"`
	Formation       string   `json:"formation,omitempty"`
	ORCID           string   `json:"orcid,omitempty"`
	CvLAC           string   `json:"cvlac,omitempty"`
	VisiblePublicly bool     `json:"visible_publicly"`
	SeedbedIDs      []string `json:"seedbed_ids,omitempty"`
}

type Publication struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	URL         string    `json:"url,omitempty"`
	PublishedAt time.Time `json:"published_at,omitempty"`
}
