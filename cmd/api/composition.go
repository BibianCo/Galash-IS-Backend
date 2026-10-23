package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	firebase "firebase.google.com/go/v4"
	httpadapter "github.com/galash-uptc/backend/internal/adapters/inbound/http"
	firebaseadapter "github.com/galash-uptc/backend/internal/adapters/outbound/firebase"
	postgresadapter "github.com/galash-uptc/backend/internal/adapters/outbound/postgres"
	application "github.com/galash-uptc/backend/internal/application/auth"
	memberapp "github.com/galash-uptc/backend/internal/application/member"
	registrationapp "github.com/galash-uptc/backend/internal/application/registration"
	seedbedapp "github.com/galash-uptc/backend/internal/application/seedbed"
	"github.com/galash-uptc/backend/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/api/option"
)

func Run() error {
	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	db, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("crear pool de PostgreSQL: %w", err)
	}
	defer db.Close()
	if err := db.Ping(ctx); err != nil {
		return fmt.Errorf("conectar con PostgreSQL: %w", err)
	}

	firebaseApp, err := firebase.NewApp(ctx, nil, option.WithCredentialsFile(cfg.FirebaseCredFile))
	if err != nil {
		return fmt.Errorf("inicializar Firebase Admin: %w", err)
	}
	firebaseAuth, err := firebaseApp.Auth(ctx)
	if err != nil {
		return fmt.Errorf("inicializar Firebase Auth: %w", err)
	}

	users := postgresadapter.NewUserRepository(db)
	seedbeds := postgresadapter.NewSeedbedRepository(db)
	registrations := postgresadapter.NewRegistrationRepository(db)
	service := application.NewService(firebaseadapter.NewVerifier(firebaseAuth), users)
	handler := httpadapter.NewHandlerWithMembers(service, users, seedbedapp.NewService(seedbeds, users), registrationapp.NewService(registrations), memberapp.NewService(users))
	go purgeExpiredCredentials(ctx, service, time.Hour)
	log.Printf("Auth escuchando en :%s", cfg.Port)
	return http.ListenAndServe(":"+cfg.Port, handler.Routes())
}

// purgeExpiredCredentials runs once at startup and then periodically, so expired
// or revoked sessions and recovery tokens do not accumulate in the database.
func purgeExpiredCredentials(ctx context.Context, service *application.Service, every time.Duration) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		if err := service.PurgeExpiredCredentials(ctx); err != nil {
			log.Printf("limpieza de sesiones: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
