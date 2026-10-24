package member_test

import (
	"context"
	"errors"
	"testing"

	memberapp "github.com/galash-uptc/backend/internal/application/member"
	"github.com/galash-uptc/backend/internal/domain"
	"github.com/galash-uptc/backend/internal/ports"
)

type fakeRepository struct {
	profile domain.MemberProfile
}

func (f *fakeRepository) List(context.Context, ports.MemberFilter) ([]domain.MemberProfile, error) {
	return []domain.MemberProfile{f.profile}, nil
}
func (f *fakeRepository) Find(context.Context, string) (domain.MemberProfile, error) {
	return f.profile, nil
}
func (f *fakeRepository) CreateMember(context.Context, ports.CreateMemberCommand) (domain.MemberProfile, error) {
	return f.profile, nil
}
func (f *fakeRepository) Update(context.Context, string, ports.UpdateMemberCommand) (domain.MemberProfile, error) {
	return f.profile, nil
}
func (f *fakeRepository) SetStatus(context.Context, string, string) (domain.MemberProfile, error) {
	return f.profile, nil
}
func (f *fakeRepository) Publications(context.Context, string) ([]domain.Publication, error) {
	return nil, nil
}

func TestGetHidesPrivateProfileFromVisitors(t *testing.T) {
	s := memberapp.NewService(&fakeRepository{profile: domain.MemberProfile{
		ID:              "896ac946-b4d3-4527-887d-be37a655a748",
		Status:          "activo",
		VisiblePublicly: false,
	}})

	_, err := s.Get(context.Background(), "896ac946-b4d3-4527-887d-be37a655a748", false)
	if !errors.Is(err, domain.ErrMemberNotFound) {
		t.Fatalf("Get() error = %v, want ErrMemberNotFound", err)
	}
}

func TestMemberOperationsRejectMalformedIDsAsNotFound(t *testing.T) {
	s := memberapp.NewService(&fakeRepository{})
	ctx := context.Background()
	if _, err := s.Get(ctx, "not-a-uuid", false); !errors.Is(err, domain.ErrMemberNotFound) {
		t.Fatalf("Get() error = %v, want ErrMemberNotFound", err)
	}
	if _, err := s.Update(ctx, "not-a-uuid", memberapp.UpdateCommand{}); !errors.Is(err, domain.ErrInvalidMember) && !errors.Is(err, domain.ErrMemberNotFound) {
		t.Fatalf("Update() error = %v, want a controlled not-found validation error", err)
	}
	if _, err := s.SetStatus(ctx, "not-a-uuid", "activo"); !errors.Is(err, domain.ErrMemberNotFound) {
		t.Fatalf("SetStatus() error = %v, want ErrMemberNotFound", err)
	}
	if _, err := s.Publications(ctx, "not-a-uuid"); !errors.Is(err, domain.ErrMemberNotFound) {
		t.Fatalf("Publications() error = %v, want ErrMemberNotFound", err)
	}
}
