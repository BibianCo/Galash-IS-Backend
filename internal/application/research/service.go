package research

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/galash-uptc/backend/internal/domain"
	"github.com/galash-uptc/backend/internal/ports"
)

type Service struct {
	repo     ports.ResearchRepository
	seedbeds ports.SeedbedRepository
	users    ports.UserReader
}

func NewService(r ports.ResearchRepository, s ports.SeedbedRepository, u ports.UserReader) *Service {
	return &Service{r, s, u}
}

func role(u domain.User, roles ...string) bool {
	for _, r := range roles {
		if strings.EqualFold(u.Role, r) {
			return true
		}
	}
	return false
}

func (s *Service) manage(ctx context.Context, actor, seedbed string) error {
	u, e := s.users.FindByID(ctx, actor)
	if e != nil || (!role(u, "admin") && !role(u, "docente") && !role(u, "profesor")) {
		return domain.ErrForbidden
	}
	if role(u, "admin") {
		return nil
	}
	sb, e := s.seedbeds.Find(ctx, seedbed)
	if e != nil {
		return domain.ErrResearchNotFound
	}
	if sb.ResponsibleTeacherID != actor {
		return domain.ErrForbidden
	}
	return nil
}

func validStatus(v string) bool { return v == "Radicado" || v == "Activo" || v == "Finalizado" }

func (s *Service) CreateActivity(ctx context.Context, a domain.Activity, actor string) (domain.Activity, error) {
	if e := s.manage(ctx, actor, a.SeedbedID); e != nil {
		return a, e
	}
	if !validActivity(a) {
		return a, domain.ErrInvalidResearch
	}
	return s.repo.CreateActivity(ctx, a)
}

func validActivity(a domain.Activity) bool {
	return strings.TrimSpace(a.Title) != "" && (a.Status == "próxima" || a.Status == "en curso" || a.Status == "finalizada")
}

func (s *Service) UpdateActivity(ctx context.Context, id string, a domain.Activity, actor string) (domain.Activity, error) {
	old, e := s.repo.FindActivity(ctx, id)
	if e != nil {
		return a, e
	}
	if e = s.manage(ctx, actor, old.SeedbedID); e != nil {
		return a, e
	}
	a.SeedbedID = old.SeedbedID
	if !validActivity(a) {
		return a, domain.ErrInvalidResearch
	}
	return s.repo.UpdateActivity(ctx, id, a)
}

func (s *Service) CreateProject(ctx context.Context, p domain.Project, actor string) (domain.Project, error) {
	if e := s.manage(ctx, actor, p.SeedbedID); e != nil {
		return p, e
	}
	if strings.TrimSpace(p.Title) == "" || !validStatus(p.Status) || p.Progress < 0 || p.Progress > 100 {
		return p, domain.ErrInvalidResearch
	}
	p.CreatedBy = actor
	return s.repo.CreateProject(ctx, p)
}

func day(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02")
}

// projectChanges lists the relevant fields that differ, for the update history.
func projectChanges(old, p domain.Project) []domain.ProjectHistory {
	var out []domain.ProjectHistory
	add := func(field, before, after string) {
		if before != after {
			out = append(out, domain.ProjectHistory{Field: field, Value: after})
		}
	}
	add("titulo", old.Title, p.Title)
	add("resumen", old.Summary, p.Summary)
	add("linea", old.ResearchLine, p.ResearchLine)
	add("estado", old.Status, p.Status)
	add("avance", strconv.Itoa(old.Progress), strconv.Itoa(p.Progress))
	add("responsables", strings.Join(old.ResponsibleIDs, ","), strings.Join(p.ResponsibleIDs, ","))
	add("integrantes", strings.Join(old.MemberIDs, ","), strings.Join(p.MemberIDs, ","))
	add("fecha_inicio", day(old.StartDate), day(p.StartDate))
	add("fecha_fin", day(old.EndDate), day(p.EndDate))
	return out
}

func (s *Service) UpdateProject(ctx context.Context, id string, p domain.Project, actor string) (domain.Project, error) {
	old, e := s.repo.FindProject(ctx, id)
	if e != nil {
		return p, e
	}
	if e = s.manage(ctx, actor, old.SeedbedID); e != nil {
		return p, e
	}
	p.SeedbedID = old.SeedbedID
	if strings.TrimSpace(p.Title) == "" || !validStatus(p.Status) || p.Progress < 0 || p.Progress > 100 {
		return p, domain.ErrInvalidResearch
	}
	return s.repo.UpdateProject(ctx, id, p, actor, projectChanges(old, p))
}

func (s *Service) Progress(ctx context.Context, id string, n int, actor string) (domain.Project, error) {
	p, e := s.repo.FindProject(ctx, id)
	if e != nil {
		return p, e
	}
	if e = s.manage(ctx, actor, p.SeedbedID); e != nil {
		return p, e
	}
	if n < 0 || n > 100 {
		return p, domain.ErrInvalidResearch
	}
	return s.repo.UpdateProgress(ctx, id, n, actor)
}

func (s *Service) CreateProposal(ctx context.Context, p domain.ResearchProposal, actor string) (domain.ResearchProposal, error) {
	u, e := s.users.FindByID(ctx, actor)
	if e != nil || !role(u, "estudiante", "docente", "profesor", "admin") {
		return p, domain.ErrForbidden
	}
	if !validProposal(p) {
		return p, domain.ErrInvalidResearch
	}
	// Every proposal enters review as Radicada; only reviewers change it later.
	p.AuthorID, p.Status = actor, "Radicada"
	return s.repo.CreateProposal(ctx, p)
}

func (s *Service) UpdateProposal(ctx context.Context, id string, p domain.ResearchProposal, actor string) (domain.ResearchProposal, error) {
	old, e := s.repo.FindProposal(ctx, id)
	if e != nil {
		return p, e
	}
	u, e := s.users.FindByID(ctx, actor)
	reviewer := e == nil && role(u, "admin", "docente", "profesor")
	if e != nil || old.AuthorID != actor && !reviewer {
		return p, domain.ErrForbidden
	}
	// Authors edit content only; status and review observations belong to reviewers.
	if !reviewer || p.Status == "" {
		p.Status = old.Status
	}
	if !reviewer {
		p.Observations = old.Observations
	}
	if p.Status != old.Status && (p.Status == "Convertida" || old.Status == "Convertida") {
		return p, domain.ErrInvalidResearch
	}
	if !validProposal(p) {
		return p, domain.ErrInvalidResearch
	}
	p.AuthorID = old.AuthorID
	return s.repo.UpdateProposal(ctx, id, p, actor)
}

func validProposal(p domain.ResearchProposal) bool {
	return strings.TrimSpace(p.Title) != "" && strings.TrimSpace(p.Problem) != "" && strings.TrimSpace(p.Objectives) != "" && strings.TrimSpace(p.Methodology) != ""
}

func (s *Service) ListActivities(c context.Context, id, q, st string) ([]domain.Activity, error) {
	if _, e := s.seedbeds.Find(c, id); e != nil {
		return nil, domain.ErrSeedbedNotFound
	}
	return s.repo.ListActivities(c, id, q, st)
}

func (s *Service) FindActivity(c context.Context, id string) (domain.Activity, error) {
	return s.repo.FindActivity(c, id)
}

func (s *Service) ListProjects(c context.Context, sb, st, p string) ([]domain.Project, error) {
	return s.repo.ListProjects(c, sb, st, p)
}

func (s *Service) FindProject(c context.Context, id string) (domain.Project, error) {
	return s.repo.FindProject(c, id)
}

func (s *Service) ProgressList(c context.Context, sb, p string) ([]domain.Project, error) {
	return s.repo.ProjectProgress(c, sb, p)
}

func (s *Service) History(c context.Context, id string) ([]domain.ProjectHistory, error) {
	return s.repo.ProjectHistory(c, id)
}

// Proposals lists every proposal for reviewers; others see only proposals they author or are responsible for.
func (s *Service) Proposals(c context.Context, actor string) ([]domain.ResearchProposal, error) {
	u, e := s.users.FindByID(c, actor)
	if e != nil || !role(u, "admin", "docente", "profesor", "estudiante") {
		return nil, domain.ErrForbidden
	}
	if role(u, "admin", "docente", "profesor") {
		actor = ""
	}
	return s.repo.ListProposals(c, actor)
}

func (s *Service) Proposal(c context.Context, id, actor string) (domain.ResearchProposal, error) {
	p, e := s.repo.FindProposal(c, id)
	if e != nil {
		return p, e
	}
	u, e := s.users.FindByID(c, actor)
	if e != nil || p.AuthorID != actor && !role(u, "admin", "docente", "profesor") {
		return p, domain.ErrForbidden
	}
	p.History, e = s.repo.ProposalHistory(c, id)
	return p, e
}

func (s *Service) Convert(c context.Context, id, actor string) (domain.Project, error) {
	p, e := s.repo.FindProposal(c, id)
	if e != nil {
		return domain.Project{}, e
	}
	u, e := s.users.FindByID(c, actor)
	if e != nil || !role(u, "admin", "docente", "profesor") {
		return domain.Project{}, domain.ErrForbidden
	}
	if strings.ToLower(p.Status) != "aprobada" || p.SeedbedID == "" {
		return domain.Project{}, domain.ErrInvalidResearch
	}
	return s.repo.ConvertProposal(c, id, actor)
}

func (s *Service) Review(c context.Context, v domain.SeedbedReview) (domain.SeedbedReview, error) {
	u, e := s.users.FindByID(c, v.AuthorID)
	if e != nil || !role(u, "admin", "estudiante", "docente", "profesor") {
		return v, domain.ErrForbidden
	}
	if _, e := s.seedbeds.Find(c, v.SeedbedID); e != nil {
		return v, domain.ErrSeedbedNotFound
	}
	return s.repo.CreateReview(c, v)
}

// Reviews defaults to the moderation queue (pendiente) when no status is given.
func (s *Service) Reviews(c context.Context, status string) ([]domain.SeedbedReview, error) {
	if status == "" {
		status = "pendiente"
	}
	return s.repo.ListReviews(c, status)
}

func (s *Service) Moderate(c context.Context, id, status, note string) (domain.SeedbedReview, error) {
	if status != "publicada" && status != "oculta" && status != "rechazada" {
		return domain.SeedbedReview{}, domain.ErrInvalidResearch
	}
	return s.repo.ModerateReview(c, id, status, note)
}
