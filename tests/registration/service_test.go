package registration_test

import (
	"context"
	"testing"

	application "github.com/galash-uptc/backend/internal/application/registration"
	"github.com/galash-uptc/backend/internal/domain"
)

type repository struct{ created domain.RegistrationRequest; hash string }

func (r *repository) CreateRegistration(_ context.Context, request domain.RegistrationRequest, hash string) (domain.RegistrationRequest, error) {
	r.created, r.hash = request, hash
	r.created.ID, r.created.Status = "request-id", domain.RegistrationFiled
	return r.created, nil
}
func (r *repository) GetRegistrationStatus(context.Context, string, string, string) (domain.RegistrationRequest, error) { return domain.RegistrationRequest{}, nil }
func (r *repository) ListRegistrations(context.Context, string, string, string, string, string) ([]domain.RegistrationRequest, error) { return nil, nil }
func (r *repository) GetRegistrationDetail(context.Context, string) (domain.RegistrationDetail, error) { return domain.RegistrationDetail{}, nil }
func (r *repository) UpdateRegistrationStatus(context.Context, string, string, string, string) (domain.RegistrationRequest, error) { return domain.RegistrationRequest{}, nil }

func validRequest() domain.RegistrationRequest {
	return domain.RegistrationRequest{Name: "Ana", DocumentNumber: "123", StudentCode: "COD1", Phone: "300", BirthDate: "2000-01-01", BirthPlace: "Tunja", Address: "Calle 1", CvLAC: "https://cv", ORCID: "https://orcid", HighestEducation: "Maestría", HighestDegree: "Magíster", UndergraduateDegree: "Ingeniería", Affiliation: "Estudiante", ResearcherCategory: "Junior", MinCienciasCall: "2025"}
}

func TestCreateRejectsMissingRequiredField(t *testing.T) {
	request := validRequest()
	request.Phone = ""
	_, _, err := application.NewService(&repository{}).Create(context.Background(), request, "")
	if err != domain.ErrIncompleteRegistration { t.Fatalf("error=%v", err) }
}

func TestCreateReturnsOpaqueTrackingToken(t *testing.T) {
	repo := &repository{}
	result, token, err := application.NewService(repo).Create(context.Background(), validRequest(), "user-id")
	if err != nil || result.Status != domain.RegistrationFiled || token == "" || token == repo.hash { t.Fatalf("result=%+v token=%q hash=%q error=%v", result, token, repo.hash, err) }
}
