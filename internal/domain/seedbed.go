package domain

import "time"

const (
	SeedbedDraft     = "Borrador"
	SeedbedPublished = "Publicado"
	SeedbedHidden    = "Oculto"
	SeedbedInactive  = "Inactivo"
)

type Seedbed struct {
	ID                   string    `json:"id"`
	Name                 string    `json:"name"`
	Description          string    `json:"description"`
	ResearchLine         string    `json:"researchLine"`
	Objective            string    `json:"objective"`
	ResponsibleTeacherID string    `json:"responsibleTeacherId"`
	ContactEmail         string    `json:"contactEmail"`
	Status               string    `json:"status"`
	Visibility           string    `json:"visibility"`
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
}

type SeedbedMember struct {
	UserID   string     `json:"userId"`
	Name     string     `json:"name"`
	LastName string     `json:"lastName"`
	Role     string     `json:"role"`
	JoinedAt time.Time  `json:"joinedAt"`
	LeftAt   *time.Time `json:"leftAt,omitempty"`
}

type ActivitySummary struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	Status      string     `json:"status"`
	Date        *time.Time `json:"date,omitempty"`
}

type ProjectSummary struct {
	ID           string     `json:"id"`
	Title        string     `json:"title"`
	Summary      string     `json:"summary,omitempty"`
	ResearchArea string     `json:"researchArea,omitempty"`
	StartDate    *time.Time `json:"startDate,omitempty"`
	EndDate      *time.Time `json:"endDate,omitempty"`
}

type ResponsibleSummary struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	LastName string `json:"lastName"`
	Email    string `json:"email,omitempty"`
}

type SeedbedSpace struct {
	Seedbed     Seedbed            `json:"seedbed"`
	Purpose     string             `json:"purpose"`
	Responsible ResponsibleSummary `json:"responsible"`
	Activities  []ActivitySummary  `json:"activities"`
	Projects    []ProjectSummary   `json:"projects"`
}
