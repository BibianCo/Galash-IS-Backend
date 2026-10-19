package domain

import "time"

const (
	RegistrationFiled    = "Radicada"
	RegistrationReview   = "En revisión"
	RegistrationApproved = "Aprobada"
	RegistrationRejected = "Rechazada"
	RegistrationPending  = "Pendiente de información"
)

type RegistrationRequest struct {
	ID                   string    `json:"id"`
	UserID               string    `json:"userId,omitempty"`
	SeedbedID            string    `json:"seedbedId,omitempty"`
	Name                 string    `json:"nombre"`
	DocumentNumber       string    `json:"cc"`
	StudentCode          string    `json:"cod"`
	Phone                string    `json:"telefono"`
	BirthDate            string    `json:"fechaNacimiento"`
	BirthPlace           string    `json:"lugarNacimiento"`
	Address              string    `json:"direccionResidencia"`
	CvLAC                string    `json:"linkCvLAC"`
	ORCID                string    `json:"linkORCID"`
	HighestEducation     string    `json:"maximoNivelEstudio"`
	HighestDegree        string    `json:"tituloMaximoNivelEstudio"`
	UndergraduateDegree  string    `json:"tituloPregrado"`
	Affiliation          string    `json:"vinculacion"`
	ResearcherCategory   string    `json:"categoriaActualInvestigador"`
	MinCienciasCall      string    `json:"convocatoriaMinCiencias"`
	Status               string    `json:"estado"`
	Observation          string    `json:"observacion,omitempty"`
	CreatedAt            time.Time `json:"fechaRadicacion"`
	UpdatedAt            time.Time `json:"fechaActualizacion"`
	ResponsibleID        string    `json:"responsableId,omitempty"`
}

type RegistrationHistory struct {
	Status       string    `json:"estado"`
	Observation string    `json:"observacion,omitempty"`
	Responsible  string    `json:"responsableId,omitempty"`
	At           time.Time `json:"fecha"`
}

type RegistrationDetail struct {
	RegistrationRequest
	History []RegistrationHistory `json:"historial"`
}
