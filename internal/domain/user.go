package domain

// User contiene únicamente los datos del usuario que el caso de uso necesita.
// Los detalles de Firebase y PostgreSQL permanecen fuera del dominio.
type User struct {
	ID          string
	FirebaseUID string
	Name        string
	LastName    string
	Email       string
}

type Identity struct {
	UID   string
	Name  string
	Email string
}
