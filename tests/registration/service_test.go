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
	return domain.RegistrationRequest{BirthPlace: "Tunja", Address: "Calle 1", CvLAC: "https://cv", ORCID: "https://orcid", HighestEducation: "Maestría", HighestDegree: "Magíster", UndergraduateDegree: "Ingeniería", Affiliation: "Estudiante", ResearcherCategory: "Junior", MinCienciasCall: "2025"}
}

func TestCreateRejectsMissingRequiredField(t *testing.T) {
	request := validRequest()
	request.Address = ""
	_, _, err := application.NewService(&repository{}).Create(context.Background(), request, "user-id")
	if err != domain.ErrIncompleteRegistration { t.Fatalf("error=%v", err) }
}

func TestCreateReturnsOpaqueTrackingToken(t *testing.T) {
	repo := &repository{}
	result, token, err := application.NewService(repo).Create(context.Background(), validRequest(), "user-id")
	if err != nil || result.Status != domain.RegistrationFiled || token == "" || token == repo.hash { t.Fatalf("result=%+v token=%q hash=%q error=%v", result, token, repo.hash, err) }
}

func TestCreateRequiresAccount(t *testing.T) {
	repo := &repository{}
	_, _, err := application.NewService(repo).Create(context.Background(), validRequest(), " ")
	if err != domain.ErrInvalidSession || repo.hash != "" { t.Fatalf("error=%v hash=%q", err, repo.hash) }
}

func TestCreateDropsClientPersonalDataAndKeepsStudentCode(t *testing.T) {
	repo := &repository{}
	request := validRequest()
	request.Name, request.DocumentNumber, request.Phone, request.BirthDate, request.StudentCode = "Otro", "999", "1", "1990-01-01", " COD9 "
	if _, _, err := application.NewService(repo).Create(context.Background(), request, "user-id"); err != nil { t.Fatal(err) }
	c := repo.created
	if c.UserID != "user-id" || c.Name != "" || c.DocumentNumber != "" || c.Phone != "" || c.BirthDate != "" || c.StudentCode != "COD9" || c.Address != "Calle 1" { t.Fatalf("created=%+v", c) }
}
