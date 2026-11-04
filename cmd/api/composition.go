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
	announcementapp "github.com/galash-uptc/backend/internal/application/announcement"
	application "github.com/galash-uptc/backend/internal/application/auth"
	contentapp "github.com/galash-uptc/backend/internal/application/content"
	eventapp "github.com/galash-uptc/backend/internal/application/event"
	calendarapp "github.com/galash-uptc/backend/internal/application/internalcalendar"
	memberapp "github.com/galash-uptc/backend/internal/application/member"
	officiallinkapp "github.com/galash-uptc/backend/internal/application/officiallink"
	registrationapp "github.com/galash-uptc/backend/internal/application/registration"
	researchapp "github.com/galash-uptc/backend/internal/application/research"
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
	research := postgresadapter.NewResearchRepository(db)
	events := postgresadapter.NewEventRepository(db)
	content := postgresadapter.NewContentPageRepository(db)
	announcements := postgresadapter.NewAnnouncementRepository(db)
	calendar := postgresadapter.NewInternalCalendarRepository(db)
	officialLinks := postgresadapter.NewOfficialLinkRepository(db)
	service := application.NewService(firebaseadapter.NewVerifier(firebaseAuth), users)
	handler := httpadapter.NewHandlerWithOfficialLinks(service, users, seedbedapp.NewService(seedbeds, users), registrationapp.NewService(registrations), researchapp.NewService(research, seedbeds, users), memberapp.NewService(users), eventapp.NewService(events, users, seedbeds), contentapp.NewService(content, users), announcementapp.NewService(announcements, users), calendarapp.NewService(calendar), officiallinkapp.NewService(officialLinks, users))
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
