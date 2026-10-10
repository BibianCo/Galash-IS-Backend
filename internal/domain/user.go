package domain

// User contiene únicamente los datos del usuario que el caso de uso necesita.
// Los detalles de Firebase y PostgreSQL permanecen fuera del dominio.
type User struct {
	ID          string `json:"id"`
	FirebaseUID string `json:"-"`
	Name        string `json:"name"`
	LastName    string `json:"last_name"`
	Email       string `json:"email"`
	DocumentType string `json:"document_type"`
	DocumentNumber string `json:"document_number"`
	Role        string `json:"role"`
	Status      string `json:"status"`
}

type Identity struct {
	UID   string
	Name  string
	Email string
}
