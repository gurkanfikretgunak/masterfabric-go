package synclink

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/masterfabric-go/masterfabric/internal/domain/synclink/model"
	"github.com/masterfabric-go/masterfabric/internal/domain/synclink/repository"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
)

var _ repository.PageRepository = (*PageRepository)(nil)

type PageRepository struct {
	db *pgxpool.Pool
}

func NewPageRepository(db *pgxpool.Pool) *PageRepository {
	return &PageRepository{db: db}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func scanPage(row pgx.Row) (*model.Page, error) {
	var p model.Page
	err := row.Scan(&p.ID, &p.UserID, &p.Slug, &p.DisplayName, &p.Bio, &p.AvatarURL, &p.Theme, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainErr.New(domainErr.ErrNotFound, "page not found", nil)
		}
		return nil, domainErr.New(domainErr.ErrInternal, "failed to scan page", err)
	}
	return &p, nil
}

func (r *PageRepository) Create(ctx context.Context, page *model.Page) error {
	now := time.Now().UTC()
	page.CreatedAt = now
	page.UpdatedAt = now
	_, err := r.db.Exec(ctx, `
		INSERT INTO synclink_pages (id, user_id, slug, display_name, bio, avatar_url, theme, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, page.ID, page.UserID, page.Slug, page.DisplayName, page.Bio, page.AvatarURL, page.Theme, page.CreatedAt, page.UpdatedAt)
	if isUniqueViolation(err) {
		return domainErr.New(domainErr.ErrConflict, "slug already taken", err)
	}
	return err
}

func (r *PageRepository) Update(ctx context.Context, page *model.Page) error {
	page.UpdatedAt = time.Now().UTC()
	_, err := r.db.Exec(ctx, `
		UPDATE synclink_pages
		SET slug = $2, display_name = $3, bio = $4, avatar_url = $5, theme = $6, updated_at = $7
		WHERE id = $1
	`, page.ID, page.Slug, page.DisplayName, page.Bio, page.AvatarURL, page.Theme, page.UpdatedAt)
	if isUniqueViolation(err) {
		return domainErr.New(domainErr.ErrConflict, "slug already taken", err)
	}
	return err
}

func (r *PageRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Page, error) {
	return scanPage(r.db.QueryRow(ctx, `
		SELECT id, user_id, slug, display_name, bio, avatar_url, theme, created_at, updated_at
		FROM synclink_pages WHERE id = $1
	`, id))
}

func (r *PageRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*model.Page, error) {
	return scanPage(r.db.QueryRow(ctx, `
		SELECT id, user_id, slug, display_name, bio, avatar_url, theme, created_at, updated_at
		FROM synclink_pages WHERE user_id = $1
	`, userID))
}

func (r *PageRepository) GetBySlug(ctx context.Context, slug string) (*model.Page, error) {
	return scanPage(r.db.QueryRow(ctx, `
		SELECT id, user_id, slug, display_name, bio, avatar_url, theme, created_at, updated_at
		FROM synclink_pages WHERE slug = $1
	`, slug))
}
