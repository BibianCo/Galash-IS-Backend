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
