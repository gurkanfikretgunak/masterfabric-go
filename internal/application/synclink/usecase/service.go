package usecase

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/internal/application/synclink/dto"
	"github.com/masterfabric-go/masterfabric/internal/domain/synclink/model"
	"github.com/masterfabric-go/masterfabric/internal/domain/synclink/repository"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
)

var slugRE = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,62}[a-z0-9])?$`)

// Service implements SyncLink page and link use cases.
type Service struct {
	pages repository.PageRepository
	links repository.LinkRepository
}

func NewService(pages repository.PageRepository, links repository.LinkRepository) *Service {
	return &Service{pages: pages, links: links}
}

func normalizeSlug(slug string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(slug))
	if !slugRE.MatchString(s) {
		return "", domainErr.New(domainErr.ErrValidation, "slug must be 2-64 chars of lowercase letters, numbers, and hyphens", nil)
	}
	return s, nil
}

func toPageDTO(p *model.Page) dto.Page {
	return dto.Page{
		ID:          p.ID,
		Slug:        p.Slug,
		DisplayName: p.DisplayName,
		Bio:         p.Bio,
		AvatarURL:   p.AvatarURL,
		Theme:       p.Theme,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

func toLinkDTO(l *model.Link) dto.Link {
	return dto.Link{
		ID:        l.ID,
		Title:     l.Title,
		URL:       l.URL,
		Icon:      l.Icon,
		Order:     l.Order,
		Active:    l.Active,
		CreatedAt: l.CreatedAt,
		UpdatedAt: l.UpdatedAt,
	}
}

func (s *Service) GetPublicPage(ctx context.Context, slug string) (*dto.PublicPage, error) {
	normalized, err := normalizeSlug(slug)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrNotFound, "page not found", nil)
	}
	page, err := s.pages.GetBySlug(ctx, normalized)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrNotFound, "page not found", err)
	}
	links, err := s.links.ListByPageID(ctx, page.ID)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to list links", err)
	}
	public := make([]dto.PublicLink, 0, len(links))
	for _, l := range links {
		if !l.Active {
			continue
		}
		public = append(public, dto.PublicLink{
			ID:    l.ID,
			Title: l.Title,
			URL:   l.URL,
			Icon:  l.Icon,
			Order: l.Order,
		})
	}
	return &dto.PublicPage{
		Slug:        page.Slug,
		DisplayName: page.DisplayName,
		Bio:         page.Bio,
		AvatarURL:   page.AvatarURL,
		Theme:       page.Theme,
		Links:       public,
	}, nil
}

func (s *Service) GetMyPage(ctx context.Context, userID uuid.UUID) (*dto.Page, error) {
	page, err := s.pages.GetByUserID(ctx, userID)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrNotFound, "page not found", err)
	}
	out := toPageDTO(page)
	return &out, nil
}

func (s *Service) UpsertPage(ctx context.Context, userID uuid.UUID, in dto.UpsertPageInput) (*dto.Page, error) {
	slug, err := normalizeSlug(in.Slug)
	if err != nil {
		return nil, err
	}
	theme := strings.TrimSpace(in.Theme)
	if theme == "" {
		theme = model.ThemeDefault
	}
	if !model.ValidTheme(theme) {
		return nil, domainErr.New(domainErr.ErrValidation, "theme must be default, dark, light, or colorful", nil)
	}

	existingBySlug, slugErr := s.pages.GetBySlug(ctx, slug)
	if slugErr != nil && !isNotFound(slugErr) {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to check slug", slugErr)
	}
	mine, mineErr := s.pages.GetByUserID(ctx, userID)
	if mineErr != nil && !isNotFound(mineErr) {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to load page", mineErr)
	}

	if existingBySlug != nil && (mine == nil || existingBySlug.ID != mine.ID) {
		return nil, domainErr.New(domainErr.ErrConflict, "slug already taken", nil)
	}

	displayName := strings.TrimSpace(in.DisplayName)
	bio := strings.TrimSpace(in.Bio)

	if mineErr == nil && mine != nil {
		mine.Slug = slug
		mine.DisplayName = displayName
		mine.Bio = bio
		mine.AvatarURL = in.AvatarURL
		mine.Theme = theme
		if err := s.pages.Update(ctx, mine); err != nil {
			return nil, domainErr.New(domainErr.ErrInternal, "failed to update page", err)
		}
		updated, err := s.pages.GetByID(ctx, mine.ID)
		if err != nil {
			out := toPageDTO(mine)
			return &out, nil
		}
		out := toPageDTO(updated)
		return &out, nil
	}

	page := &model.Page{
		ID:          uuid.New(),
		UserID:      userID,
		Slug:        slug,
		DisplayName: displayName,
		Bio:         bio,
		AvatarURL:   in.AvatarURL,
		Theme:       theme,
	}
	if err := s.pages.Create(ctx, page); err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to create page", err)
	}
	created, err := s.pages.GetByID(ctx, page.ID)
	if err != nil {
		out := toPageDTO(page)
		return &out, nil
	}
	out := toPageDTO(created)
	return &out, nil
}

func (s *Service) requirePage(ctx context.Context, userID uuid.UUID) (*model.Page, error) {
	page, err := s.pages.GetByUserID(ctx, userID)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrNotFound, "page not found", err)
	}
	return page, nil
}

func (s *Service) ListLinks(ctx context.Context, userID uuid.UUID) ([]dto.Link, error) {
	page, err := s.requirePage(ctx, userID)
	if err != nil {
		return nil, err
	}
	links, err := s.links.ListByPageID(ctx, page.ID)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to list links", err)
	}
	out := make([]dto.Link, 0, len(links))
	for _, l := range links {
		out = append(out, toLinkDTO(l))
	}
	return out, nil
}

func (s *Service) CreateLink(ctx context.Context, userID uuid.UUID, in dto.CreateLinkInput) (*dto.Link, error) {
	page, err := s.requirePage(ctx, userID)
	if err != nil {
		return nil, err
	}
	maxOrder, err := s.links.MaxOrder(ctx, page.ID)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to assign link order", err)
	}
	active := true
	if in.Active != nil {
		active = *in.Active
	}
	link := &model.Link{
		ID:     uuid.New(),
		PageID: page.ID,
		Title:  strings.TrimSpace(in.Title),
		URL:    strings.TrimSpace(in.URL),
		Icon:   in.Icon,
		Order:  maxOrder + 1,
		Active: active,
	}
	if err := s.links.Create(ctx, link); err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to create link", err)
	}
	out := toLinkDTO(link)
	return &out, nil
}

func (s *Service) UpdateLink(ctx context.Context, userID uuid.UUID, linkID uuid.UUID, in dto.UpdateLinkInput) (*dto.Link, error) {
	page, err := s.requirePage(ctx, userID)
	if err != nil {
		return nil, err
	}
	link, err := s.links.GetByID(ctx, linkID)
	if err != nil || link.PageID != page.ID {
		return nil, domainErr.New(domainErr.ErrNotFound, "link not found", err)
	}
	if in.Title != nil {
		link.Title = strings.TrimSpace(*in.Title)
	}
	if in.URL != nil {
		link.URL = strings.TrimSpace(*in.URL)
	}
	if in.Icon != nil {
		link.Icon = in.Icon
	}
	if in.Active != nil {
		link.Active = *in.Active
	}
	if err := s.links.Update(ctx, link); err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to update link", err)
	}
	out := toLinkDTO(link)
	return &out, nil
}

func (s *Service) DeleteLink(ctx context.Context, userID uuid.UUID, linkID uuid.UUID) error {
	page, err := s.requirePage(ctx, userID)
	if err != nil {
		return err
	}
	link, err := s.links.GetByID(ctx, linkID)
	if err != nil || link.PageID != page.ID {
		return domainErr.New(domainErr.ErrNotFound, "link not found", err)
	}
	if err := s.links.Delete(ctx, link.ID); err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to delete link", err)
	}
	return nil
}

func (s *Service) ReorderLinks(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) ([]dto.Link, error) {
	page, err := s.requirePage(ctx, userID)
	if err != nil {
		return nil, err
	}
	existing, err := s.links.ListByPageID(ctx, page.ID)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to list links", err)
	}
	owned := make(map[uuid.UUID]struct{}, len(existing))
	for _, l := range existing {
		owned[l.ID] = struct{}{}
	}
	if len(ids) != len(existing) {
		return nil, domainErr.New(domainErr.ErrValidation, "reorder must include every link id exactly once", nil)
	}
	seen := make(map[uuid.UUID]struct{}, len(ids))
	for _, id := range ids {
		if _, ok := owned[id]; !ok {
			return nil, domainErr.New(domainErr.ErrNotFound, "link not found", nil)
		}
		if _, dup := seen[id]; dup {
			return nil, domainErr.New(domainErr.ErrValidation, "duplicate link id in reorder", nil)
		}
		seen[id] = struct{}{}
	}
	if err := s.links.Reorder(ctx, page.ID, ids); err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to reorder links", err)
	}
	return s.ListLinks(ctx, userID)
}

func isNotFound(err error) bool {
	return errors.Is(err, domainErr.ErrNotFound)
}
