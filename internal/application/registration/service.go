package registration

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"time"

	"github.com/galash-uptc/backend/internal/domain"
	"github.com/galash-uptc/backend/internal/ports"
)

type Service struct {
	repository ports.RegistrationRepository
}

func NewService(repository ports.RegistrationRepository) *Service { return &Service{repository: repository} }

func (s *Service) Create(ctx context.Context, request domain.RegistrationRequest, userID string) (domain.RegistrationRequest, string, error) {
	if err := validate(request); err != nil {
		return domain.RegistrationRequest{}, "", err
	}
	request.UserID = userID
	token, err := randomToken()
	if err != nil {
		return domain.RegistrationRequest{}, "", err
	}
	created, err := s.repository.CreateRegistration(ctx, request, hashToken(token))
	return created, token, err
}

func (s *Service) Status(ctx context.Context, id, token, userID string) (domain.RegistrationRequest, error) {
	if strings.TrimSpace(token) == "" && strings.TrimSpace(userID) == "" {
		return domain.RegistrationRequest{}, domain.ErrInvalidTrackingToken
	}
	return s.repository.GetRegistrationStatus(ctx, id, hashToken(token), userID)
}

func (s *Service) List(ctx context.Context, status, from, to, seedbedID, responsibleID string) ([]domain.RegistrationRequest, error) {
	return s.repository.ListRegistrations(ctx, status, from, to, seedbedID, responsibleID)
}

func (s *Service) Detail(ctx context.Context, id string) (domain.RegistrationDetail, error) {
	return s.repository.GetRegistrationDetail(ctx, id)
}

func (s *Service) UpdateStatus(ctx context.Context, id, status, observation, responsibleID string) (domain.RegistrationRequest, error) {
	status = strings.TrimSpace(status)
	if !validStatus(status) || strings.TrimSpace(observation) == "" {
		return domain.RegistrationRequest{}, domain.ErrInvalidRegistrationStatus
	}
	return s.repository.UpdateRegistrationStatus(ctx, id, status, strings.TrimSpace(observation), strings.TrimSpace(responsibleID))
}

func validate(r domain.RegistrationRequest) error {
	fields := []string{r.Name, r.DocumentNumber, r.StudentCode, r.Phone, r.BirthDate, r.BirthPlace, r.Address, r.CvLAC, r.ORCID, r.HighestEducation, r.HighestDegree, r.UndergraduateDegree, r.Affiliation, r.ResearcherCategory, r.MinCienciasCall}
	for _, field := range fields {
		if strings.TrimSpace(field) == "" {
			return domain.ErrIncompleteRegistration
		}
	}
	if _, err := time.Parse("2006-01-02", strings.TrimSpace(r.BirthDate)); err != nil {
		return domain.ErrInvalidRegistrationData
	}
	return nil
}

func validStatus(status string) bool {
	for _, value := range []string{domain.RegistrationReview, domain.RegistrationApproved, domain.RegistrationRejected, domain.RegistrationPending} {
		if status == value {
			return true
		}
	}
	return false
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func hashToken(token string) string {
	digest := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(digest[:])
}
