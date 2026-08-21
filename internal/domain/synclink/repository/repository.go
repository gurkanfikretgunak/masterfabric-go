package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/internal/domain/synclink/model"
)

// PageRepository persists SyncLink pages.
type PageRepository interface {
	Create(ctx context.Context, page *model.Page) error
	Update(ctx context.Context, page *model.Page) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Page, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) (*model.Page, error)
	GetBySlug(ctx context.Context, slug string) (*model.Page, error)
}

// LinkRepository persists SyncLink links.
type LinkRepository interface {
	Create(ctx context.Context, link *model.Link) error
	Update(ctx context.Context, link *model.Link) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Link, error)
	ListByPageID(ctx context.Context, pageID uuid.UUID) ([]*model.Link, error)
	MaxOrder(ctx context.Context, pageID uuid.UUID) (int, error)
	Reorder(ctx context.Context, pageID uuid.UUID, ids []uuid.UUID) error
}
