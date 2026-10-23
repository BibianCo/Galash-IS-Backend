package ports

import("context"; "github.com/galash-uptc/backend/internal/domain")
type ResearchRepository interface {
 ListActivities(context.Context,string,string,string)([]domain.Activity,error); CreateActivity(context.Context,domain.Activity)(domain.Activity,error); FindActivity(context.Context,string)(domain.Activity,error); UpdateActivity(context.Context,string,domain.Activity)(domain.Activity,error)
 ListProjects(context.Context,string,string,string)([]domain.Project,error); CreateProject(context.Context,domain.Project)(domain.Project,error); FindProject(context.Context,string)(domain.Project,error); UpdateProject(context.Context,string,domain.Project)(domain.Project,error); UpdateProgress(context.Context,string,int,string)(domain.Project,error); ProjectProgress(context.Context,string,string)([]domain.Project,error); ProjectHistory(context.Context,string)([]domain.ProjectHistory,error)
 ListProposals(context.Context,string)([]domain.ResearchProposal,error); CreateProposal(context.Context,domain.ResearchProposal)(domain.ResearchProposal,error); FindProposal(context.Context,string)(domain.ResearchProposal,error); UpdateProposal(context.Context,string,domain.ResearchProposal)(domain.ResearchProposal,error); ConvertProposal(context.Context,string,domain.Project)(domain.Project,error)
 CreateReview(context.Context,domain.SeedbedReview)(domain.SeedbedReview,error); ListReviews(context.Context,string)([]domain.SeedbedReview,error); ModerateReview(context.Context,string,string,string)(domain.SeedbedReview,error)
}
