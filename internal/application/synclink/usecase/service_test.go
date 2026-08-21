package usecase

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/internal/application/synclink/dto"
	"github.com/masterfabric-go/masterfabric/internal/domain/synclink/model"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type memPages struct {
	mu     sync.Mutex
	byID   map[uuid.UUID]*model.Page
	byUser map[uuid.UUID]uuid.UUID
	bySlug map[string]uuid.UUID
}

func newMemPages() *memPages {
	return &memPages{
		byID:   map[uuid.UUID]*model.Page{},
		byUser: map[uuid.UUID]uuid.UUID{},
		bySlug: map[string]uuid.UUID{},
	}
}

func (m *memPages) clone(p *model.Page) *model.Page {
	cp := *p
	if p.AvatarURL != nil {
		v := *p.AvatarURL
		cp.AvatarURL = &v
	}
	return &cp
}

func (m *memPages) Create(ctx context.Context, page *model.Page) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.bySlug[page.Slug]; ok {
		return domainErr.New(domainErr.ErrConflict, "slug already taken", nil)
	}
	m.byID[page.ID] = m.clone(page)
	m.byUser[page.UserID] = page.ID
	m.bySlug[page.Slug] = page.ID
	return nil
}

func (m *memPages) Update(ctx context.Context, page *model.Page) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if other, ok := m.bySlug[page.Slug]; ok && other != page.ID {
		return domainErr.New(domainErr.ErrConflict, "slug already taken", nil)
	}
	old := m.byID[page.ID]
	if old != nil {
		delete(m.bySlug, old.Slug)
	}
	m.byID[page.ID] = m.clone(page)
	m.bySlug[page.Slug] = page.ID
	m.byUser[page.UserID] = page.ID
	return nil
}

func (m *memPages) GetByID(ctx context.Context, id uuid.UUID) (*model.Page, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.byID[id]
	if !ok {
		return nil, domainErr.New(domainErr.ErrNotFound, "page not found", nil)
	}
	return m.clone(p), nil
}

func (m *memPages) GetByUserID(ctx context.Context, userID uuid.UUID) (*model.Page, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.byUser[userID]
	if !ok {
		return nil, domainErr.New(domainErr.ErrNotFound, "page not found", nil)
	}
	return m.clone(m.byID[id]), nil
}

func (m *memPages) GetBySlug(ctx context.Context, slug string) (*model.Page, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.bySlug[slug]
	if !ok {
		return nil, domainErr.New(domainErr.ErrNotFound, "page not found", nil)
	}
	return m.clone(m.byID[id]), nil
}

type memLinks struct {
	mu   sync.Mutex
	byID map[uuid.UUID]*model.Link
}

func newMemLinks() *memLinks {
	return &memLinks{byID: map[uuid.UUID]*model.Link{}}
}

func (m *memLinks) clone(l *model.Link) *model.Link {
	cp := *l
	if l.Icon != nil {
		v := *l.Icon
		cp.Icon = &v
	}
	return &cp
}

func (m *memLinks) Create(ctx context.Context, link *model.Link) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.byID[link.ID] = m.clone(link)
	return nil
}

func (m *memLinks) Update(ctx context.Context, link *model.Link) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.byID[link.ID] = m.clone(link)
	return nil
}

func (m *memLinks) Delete(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.byID, id)
	return nil
}

func (m *memLinks) GetByID(ctx context.Context, id uuid.UUID) (*model.Link, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	l, ok := m.byID[id]
	if !ok {
		return nil, domainErr.New(domainErr.ErrNotFound, "link not found", nil)
	}
	return m.clone(l), nil
}

func (m *memLinks) ListByPageID(ctx context.Context, pageID uuid.UUID) ([]*model.Link, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []*model.Link
	for _, l := range m.byID {
		if l.PageID == pageID {
			out = append(out, m.clone(l))
		}
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].Order < out[i].Order {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out, nil
}

func (m *memLinks) MaxOrder(ctx context.Context, pageID uuid.UUID) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	max := -1
	for _, l := range m.byID {
		if l.PageID == pageID && l.Order > max {
			max = l.Order
		}
	}
	return max, nil
}

func (m *memLinks) Reorder(ctx context.Context, pageID uuid.UUID, ids []uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, id := range ids {
		l, ok := m.byID[id]
		if !ok || l.PageID != pageID {
			return domainErr.New(domainErr.ErrNotFound, "link not found", nil)
		}
		l.Order = i
	}
	return nil
}

func TestGetPublicPage_NotFound(t *testing.T) {
	svc := NewService(newMemPages(), newMemLinks())
	page, err := svc.GetPublicPage(context.Background(), "missing")
	assert.Nil(t, page)
	assert.True(t, errors.Is(err, domainErr.ErrNotFound))
}

func TestGetPublicPage_OmitsInactive(t *testing.T) {
	pages, links := newMemPages(), newMemLinks()
	svc := NewService(pages, links)
	user := uuid.New()
	created, err := svc.UpsertPage(context.Background(), user, dto.UpsertPageInput{
		Slug: "gurkan", DisplayName: "Gürkan",
	})
	require.NoError(t, err)
	active, err := svc.CreateLink(context.Background(), user, dto.CreateLinkInput{Title: "Site", URL: "https://ex.com"})
	require.NoError(t, err)
	off := false
	_, err = svc.CreateLink(context.Background(), user, dto.CreateLinkInput{Title: "Hidden", URL: "https://hid.com", Active: &off})
	require.NoError(t, err)

	pub, err := svc.GetPublicPage(context.Background(), created.Slug)
	require.NoError(t, err)
	require.Len(t, pub.Links, 1)
	assert.Equal(t, active.ID, pub.Links[0].ID)
}

func TestUpsertPage_SlugConflict(t *testing.T) {
	svc := NewService(newMemPages(), newMemLinks())
	_, err := svc.UpsertPage(context.Background(), uuid.New(), dto.UpsertPageInput{Slug: "taken", DisplayName: "A"})
	require.NoError(t, err)
	_, err = svc.UpsertPage(context.Background(), uuid.New(), dto.UpsertPageInput{Slug: "taken", DisplayName: "B"})
	assert.True(t, errors.Is(err, domainErr.ErrConflict))
}

func TestLinkCRUDAndReorder(t *testing.T) {
	svc := NewService(newMemPages(), newMemLinks())
	user := uuid.New()
	_, err := svc.UpsertPage(context.Background(), user, dto.UpsertPageInput{Slug: "me", DisplayName: "Me"})
	require.NoError(t, err)
	a, err := svc.CreateLink(context.Background(), user, dto.CreateLinkInput{Title: "A", URL: "https://a.com"})
	require.NoError(t, err)
	b, err := svc.CreateLink(context.Background(), user, dto.CreateLinkInput{Title: "B", URL: "https://b.com"})
	require.NoError(t, err)
	title := "A2"
	updated, err := svc.UpdateLink(context.Background(), user, a.ID, dto.UpdateLinkInput{Title: &title})
	require.NoError(t, err)
	assert.Equal(t, "A2", updated.Title)

	reordered, err := svc.ReorderLinks(context.Background(), user, []uuid.UUID{b.ID, a.ID})
	require.NoError(t, err)
	require.Len(t, reordered, 2)
	assert.Equal(t, b.ID, reordered[0].ID)
	assert.Equal(t, 0, reordered[0].Order)
	assert.Equal(t, a.ID, reordered[1].ID)

	require.NoError(t, svc.DeleteLink(context.Background(), user, a.ID))
	left, err := svc.ListLinks(context.Background(), user)
	require.NoError(t, err)
	require.Len(t, left, 1)
	assert.Equal(t, b.ID, left[0].ID)
}

func TestCreateLink_RequiresPage(t *testing.T) {
	svc := NewService(newMemPages(), newMemLinks())
	_, err := svc.CreateLink(context.Background(), uuid.New(), dto.CreateLinkInput{Title: "A", URL: "https://a.com"})
	assert.True(t, errors.Is(err, domainErr.ErrNotFound))
}
