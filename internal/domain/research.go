package domain

import "time"

type Activity struct {
	ID             string     `json:"id"`
	SeedbedID      string     `json:"seedbedId"`
	Title          string     `json:"title"`
	Description    string     `json:"description"`
	Status         string     `json:"status"`
	Date           *time.Time `json:"date"`
	ResponsibleIDs []string   `json:"responsibleIds"`
	Evidence       []string   `json:"evidence"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

type Project struct {
	ID             string     `json:"id"`
	SeedbedID      string     `json:"seedbedId"`
	Title          string     `json:"title"`
	Summary        string     `json:"summary"`
	ResearchLine   string     `json:"researchLine"`
	Status         string     `json:"status"`
	ResponsibleIDs []string   `json:"responsibleIds"`
	MemberIDs      []string   `json:"memberIds"`
	StartDate      *time.Time `json:"startDate"`
	EndDate        *time.Time `json:"endDate"`
	Progress       int        `json:"progress"`
	CreatedBy      string     `json:"createdBy"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

type ProjectHistory struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"projectId"`
	AuthorID  string    `json:"authorId"`
	Field     string    `json:"field"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"createdAt"`
}

type ResearchProposal struct {
	ID             string            `json:"id"`
	SeedbedID      string            `json:"seedbedId"`
	Title          string            `json:"title"`
	Problem        string            `json:"problem"`
	Objectives     string            `json:"objectives"`
	Methodology    string            `json:"methodology"`
	Status         string            `json:"status"`
	ResponsibleIDs []string          `json:"responsibleIds"`
	Attachments    []string          `json:"attachments"`
	AuthorID       string            `json:"authorId"`
	Observations   string            `json:"observations"`
	ProjectID      *string           `json:"projectId"`
	CreatedAt      time.Time         `json:"createdAt"`
	UpdatedAt      time.Time         `json:"updatedAt"`
	History        []ProposalHistory `json:"history"`
}

type ProposalHistory struct {
	ID          string    `json:"id"`
	ProposalID  string    `json:"proposalId"`
	AuthorID    string    `json:"authorId"`
	Status      string    `json:"status"`
	Observation string    `json:"observation"`
	CreatedAt   time.Time `json:"createdAt"`
}

type SeedbedReview struct {
	ID             string     `json:"id"`
	SeedbedID      string     `json:"seedbedId"`
	AuthorID       string     `json:"authorId"`
	Text           string     `json:"text"`
	Status         string     `json:"status"`
	ModerationNote string     `json:"moderationNote"`
	CreatedAt      *time.Time `json:"createdAt"`
	ModeratedAt    *time.Time `json:"moderatedAt"`
}
