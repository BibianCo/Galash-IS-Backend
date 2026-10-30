package domain

// User contiene únicamente los datos del usuario que el caso de uso necesita.
// Los detalles de Firebase y PostgreSQL permanecen fuera del dominio.
type User struct {
	ID             string `json:"id"`
	FirebaseUID    string `json:"-"`
	Name           string `json:"name"`
	LastName       string `json:"last_name"`
	Email          string `json:"email"`
	DocumentType   string `json:"document_type"`
	DocumentNumber string `json:"document_number"`
	StudentCode    string `json:"student_code"`
	BirthDate      string `json:"birth_date"`
	Phone          string `json:"phone"`
	Role           string `json:"role"`
	Status         string `json:"status"`
}

// PublicProfile is the subset safe to expose in the public directory.
type PublicProfile struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	LastName string `json:"last_name"`
	Role     string `json:"role"`
}

type AdminProfile struct {
	ID             string `json:"id"`
	FirebaseUID    string `json:"firebase_uid"`
	Name           string `json:"name"`
	LastName       string `json:"last_name"`
	Email          string `json:"email"`
	DocumentType   string `json:"document_type"`
	DocumentNumber string `json:"document_number"`
	Role           string `json:"role"`
	Status         string `json:"status"`
}

type Identity struct {
	UID   string
	Name  string
	Email string
}
