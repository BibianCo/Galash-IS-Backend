package ports

import (
	"context"

	"github.com/galash-uptc/backend/internal/domain"
)

type ResearchRepository interface {
	ListActivities(ctx context.Context, seedbedID, period, status string) ([]domain.Activity, error)
	CreateActivity(ctx context.Context, activity domain.Activity) (domain.Activity, error)
	FindActivity(ctx context.Context, id string) (domain.Activity, error)
	UpdateActivity(ctx context.Context, id string, activity domain.Activity) (domain.Activity, error)

	ListProjects(ctx context.Context, seedbedID, status, period string) ([]domain.Project, error)
	CreateProject(ctx context.Context, project domain.Project) (domain.Project, error)
	FindProject(ctx context.Context, id string) (domain.Project, error)
	// UpdateProject stores the project and one history row per change, atomically.
	UpdateProject(ctx context.Context, id string, project domain.Project, actorID string, changes []domain.ProjectHistory) (domain.Project, error)
	UpdateProgress(ctx context.Context, id string, progress int, actorID string) (domain.Project, error)
	ProjectProgress(ctx context.Context, seedbedID, period string) ([]domain.Project, error)
	ProjectHistory(ctx context.Context, id string) ([]domain.ProjectHistory, error)

	ListProposals(ctx context.Context, actorID string) ([]domain.ResearchProposal, error)
	CreateProposal(ctx context.Context, proposal domain.ResearchProposal) (domain.ResearchProposal, error)
	FindProposal(ctx context.Context, id string) (domain.ResearchProposal, error)
	UpdateProposal(ctx context.Context, id string, proposal domain.ResearchProposal, actorID string) (domain.ResearchProposal, error)
	ProposalHistory(ctx context.Context, id string) ([]domain.ProposalHistory, error)
	ConvertProposal(ctx context.Context, id, actorID string) (domain.Project, error)

	CreateReview(ctx context.Context, review domain.SeedbedReview) (domain.SeedbedReview, error)
	ListReviews(ctx context.Context, status string) ([]domain.SeedbedReview, error)
	ModerateReview(ctx context.Context, id, status, note string) (domain.SeedbedReview, error)
}
