package ports

import (
	"context"
	"github.com/galash-uptc/backend/internal/domain"
)

type MemberFilter struct {
	Query, Role, SeedbedID, Area string
	Limit                        int
}
type CreateMemberCommand struct {
	FirebaseUID, Name, LastName, Email, DocumentType, DocumentNumber, Role string
	Area, Formation, ORCID, CvLAC                                          string
	VisiblePublicly                                                        bool
	SeedbedIDs                                                             []string
}
type UpdateMemberCommand struct {
	Name, LastName, Email, DocumentType, DocumentNumber, Role string
	Area, Formation, ORCID, CvLAC                             string
	VisiblePublicly                                           *bool
	SeedbedIDs                                                []string
}

type MemberRepository interface {
	List(context.Context, MemberFilter) ([]domain.MemberProfile, error)
	Find(context.Context, string) (domain.MemberProfile, error)
	CreateMember(context.Context, CreateMemberCommand) (domain.MemberProfile, error)
	Update(context.Context, string, UpdateMemberCommand) (domain.MemberProfile, error)
	SetStatus(context.Context, string, string) (domain.MemberProfile, error)
	Publications(context.Context, string) ([]domain.Publication, error)
}
