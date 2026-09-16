package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"

	"firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/api/option"
)

type registerRequest struct {
	Nombre   string `json:"nombre"`
	Apellido string `json:"apellido"`
}

type server struct {
	db   *pgxpool.Pool
	auth *auth.Client
}

func main() {
	ctx := context.Background()
	databaseURL := requiredEnv("DATABASE_URL")
	credentialsFile := requiredEnv("FIREBASE_CREDENTIALS_FILE")

	db, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatalf("crear pool de PostgreSQL: %v", err)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		log.Fatalf("conectar con PostgreSQL: %v", err)
	}

	firebaseApp, err := firebase.NewApp(ctx, nil, option.WithCredentialsFile(credentialsFile))
	if err != nil {
		log.Fatalf("inicializar Firebase Admin: %v", err)
	}
	firebaseAuth, err := firebaseApp.Auth(ctx)
	if err != nil {
		log.Fatalf("inicializar Firebase Auth: %v", err)
	}

	s := &server{db: db, auth: firebaseAuth}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("POST /auth/register", s.register)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Auth escuchando en :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func (s *server) register(w http.ResponseWriter, r *http.Request) {
	token, err := bearerToken(r.Header.Get("Authorization"))
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	firebaseUser, err := s.auth.VerifyIDToken(r.Context(), token)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "token de Firebase inválido")
		return
	}

	var request registerRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "cuerpo JSON inválido")
		return
	}

	name := request.Nombre
	if name == "" {
		name, _ = firebaseUser.Claims["name"].(string)
	}
	email, _ := firebaseUser.Claims["email"].(string)
	if name == "" || email == "" {
		writeError(w, http.StatusBadRequest, "el token debe incluir nombre y correo electrónico")
		return
	}

	var id string
	err = s.db.QueryRow(r.Context(), `
		INSERT INTO usuarios (firebase_uid, nombre, apellido, email)
		VALUES ($1, $2, NULLIF($3, ''), $4)
		ON CONFLICT (firebase_uid) DO UPDATE SET
			nombre = EXCLUDED.nombre,
			apellido = COALESCE(EXCLUDED.apellido, usuarios.apellido),
			email = EXCLUDED.email
		RETURNING id`, firebaseUser.UID, name, request.Apellido, email).Scan(&id)
	if err != nil {
		log.Printf("guardar usuario Firebase %s: %v", firebaseUser.UID, err)
		writeError(w, http.StatusInternalServerError, "no fue posible guardar el usuario")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"id": id, "firebase_uid": firebaseUser.UID})
}

func (s *server) health(w http.ResponseWriter, r *http.Request) {
	if err := s.db.Ping(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, "base de datos no disponible")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func bearerToken(header string) (string, error) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return "", errors.New("se requiere Authorization: Bearer <token>")
	}
	return parts[1], nil
}

func requiredEnv(name string) string {
	value := os.Getenv(name)
	if value == "" {
		log.Fatalf("la variable %s es obligatoria", name)
	}
	return value
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
