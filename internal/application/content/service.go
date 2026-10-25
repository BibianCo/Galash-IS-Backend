package content

import (
	"context"
	"regexp"
	"strings"

	"github.com/galash-uptc/backend/internal/domain"
	"github.com/galash-uptc/backend/internal/ports"
	"golang.org/x/net/html"
)

type Service struct {
	repo  ports.ContentPageRepository
	users ports.UserReader
}

func NewService(repo ports.ContentPageRepository, users ports.UserReader) *Service {
	return &Service{repo: repo, users: users}
}
func isAdmin(u domain.User) bool { return strings.EqualFold(u.Role, "admin") }

func (s *Service) authorize(ctx context.Context, actor string) error {
	u, err := s.users.FindByID(ctx, actor)
	if err != nil || !isAdmin(u) {
		return domain.ErrForbidden
	}
	return nil
}
func validSlug(v string) bool { return regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`).MatchString(v) }
func (s *Service) ListPublished(ctx context.Context) ([]domain.ContentPage, error) {
	return s.repo.List(ctx, false)
}
func (s *Service) FindPublished(ctx context.Context, slug string) (domain.ContentPage, error) {
	if !validSlug(slug) {
		return domain.ContentPage{}, domain.ErrContentNotFound
	}
	return s.repo.FindBySlug(ctx, slug)
}
func (s *Service) ListAdmin(ctx context.Context, actor string) ([]domain.ContentPage, error) {
	if err := s.authorize(ctx, actor); err != nil {
		return nil, err
	}
	return s.repo.List(ctx, true)
}
func (s *Service) Create(ctx context.Context, page domain.ContentPage, actor string) (domain.ContentPage, error) {
	if err := s.authorize(ctx, actor); err != nil {
		return page, err
	}
	page.AuthorID, page.Status = actor, "borrador"
	page.Content = Sanitize(page.Content)
	if strings.TrimSpace(page.Title) == "" || !validSlug(page.Slug) || strings.TrimSpace(page.Content) == "" {
		return page, domain.ErrInvalidContent
	}
	return s.repo.Create(ctx, page)
}
func (s *Service) Update(ctx context.Context, id string, title, content, slug, actor string) (domain.ContentPage, error) {
	if err := s.authorize(ctx, actor); err != nil {
		return domain.ContentPage{}, err
	}
	old, err := s.repo.Find(ctx, id)
	if err != nil {
		return old, err
	}
	if title == "" {
		title = old.Title
	}
	if content == "" {
		content = old.Content
	}
	if slug == "" {
		slug = old.Slug
	}
	content = Sanitize(content)
	if strings.TrimSpace(title) == "" || !validSlug(slug) || strings.TrimSpace(content) == "" {
		return domain.ContentPage{}, domain.ErrInvalidContent
	}
	return s.repo.Update(ctx, id, title, content, slug)
}
func (s *Service) Publish(ctx context.Context, id, actor string) (domain.ContentPage, error) {
	if err := s.authorize(ctx, actor); err != nil {
		return domain.ContentPage{}, err
	}
	p, err := s.repo.Find(ctx, id)
	if err != nil {
		return p, err
	}
	if p.Status == "publicado" {
		return p, domain.ErrInvalidContentTransition
	}
	return s.repo.Publish(ctx, id)
}

// Sanitize retains the small formatting vocabulary accepted by institutional pages.
func Sanitize(raw string) string {
	tokenizer := html.NewTokenizer(strings.NewReader(raw))
	allowed := map[string]bool{"p": true, "br": true, "strong": true, "em": true, "ul": true, "ol": true, "li": true, "a": true, "h2": true, "h3": true}
	var b strings.Builder
	for {
		tokenType := tokenizer.Next()
		if tokenType == html.ErrorToken {
			break
		}
		token := tokenizer.Token()
		switch tokenType {
		case html.TextToken:
			b.WriteString(token.String())
		case html.StartTagToken:
			if !allowed[token.Data] {
				continue
			}
			b.WriteString("<" + token.Data)
			if token.Data == "a" {
				for _, attr := range token.Attr {
					value := strings.TrimSpace(attr.Val)
					lower := strings.ToLower(value)
					if attr.Key == "href" && (strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")) {
						b.WriteString(` href="` + html.EscapeString(value) + `"`)
					}
				}
			}
			b.WriteString(">")
			if token.Data == "br" {
				b.WriteString("\n")
			}
		case html.EndTagToken:
			if allowed[token.Data] && token.Data != "br" {
				b.WriteString("</" + token.Data + ">")
			}
		}
	}
	return strings.TrimSpace(b.String())
}
