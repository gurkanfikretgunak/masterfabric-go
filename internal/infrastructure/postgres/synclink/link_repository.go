package synclink

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/masterfabric-go/masterfabric/internal/domain/synclink/model"
	"github.com/masterfabric-go/masterfabric/internal/domain/synclink/repository"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
)

var _ repository.LinkRepository = (*LinkRepository)(nil)

type LinkRepository struct {
	db *pgxpool.Pool
}

func NewLinkRepository(db *pgxpool.Pool) *LinkRepository {
	return &LinkRepository{db: db}
}

func scanLink(row pgx.Row) (*model.Link, error) {
	var l model.Link
	err := row.Scan(&l.ID, &l.PageID, &l.Title, &l.URL, &l.Icon, &l.Order, &l.Active, &l.CreatedAt, &l.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainErr.New(domainErr.ErrNotFound, "link not found", nil)
		}
		return nil, domainErr.New(domainErr.ErrInternal, "failed to scan link", err)
	}
	return &l, nil
}

func (r *LinkRepository) Create(ctx context.Context, link *model.Link) error {
	now := time.Now().UTC()
	link.CreatedAt = now
	link.UpdatedAt = now
	_, err := r.db.Exec(ctx, `
		INSERT INTO synclink_links (id, page_id, title, url, icon, sort_order, active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, link.ID, link.PageID, link.Title, link.URL, link.Icon, link.Order, link.Active, link.CreatedAt, link.UpdatedAt)
	return err
}

func (r *LinkRepository) Update(ctx context.Context, link *model.Link) error {
	link.UpdatedAt = time.Now().UTC()
	_, err := r.db.Exec(ctx, `
		UPDATE synclink_links
		SET title = $2, url = $3, icon = $4, sort_order = $5, active = $6, updated_at = $7
		WHERE id = $1
	`, link.ID, link.Title, link.URL, link.Icon, link.Order, link.Active, link.UpdatedAt)
	return err
}

func (r *LinkRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM synclink_links WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domainErr.New(domainErr.ErrNotFound, "link not found", nil)
	}
	return nil
}

func (r *LinkRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Link, error) {
	return scanLink(r.db.QueryRow(ctx, `
		SELECT id, page_id, title, url, icon, sort_order, active, created_at, updated_at
		FROM synclink_links WHERE id = $1
	`, id))
}

func (r *LinkRepository) ListByPageID(ctx context.Context, pageID uuid.UUID) ([]*model.Link, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, page_id, title, url, icon, sort_order, active, created_at, updated_at
		FROM synclink_links WHERE page_id = $1
		ORDER BY sort_order ASC, created_at ASC
	`, pageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var links []*model.Link
	for rows.Next() {
		var l model.Link
		if err := rows.Scan(&l.ID, &l.PageID, &l.Title, &l.URL, &l.Icon, &l.Order, &l.Active, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, err
		}
		links = append(links, &l)
	}
	if links == nil {
		links = []*model.Link{}
	}
	return links, rows.Err()
}

func (r *LinkRepository) MaxOrder(ctx context.Context, pageID uuid.UUID) (int, error) {
	var max *int
	err := r.db.QueryRow(ctx, `SELECT MAX(sort_order) FROM synclink_links WHERE page_id = $1`, pageID).Scan(&max)
	if err != nil {
		return 0, err
	}
	if max == nil {
		return -1, nil
	}
	return *max, nil
}

func (r *LinkRepository) Reorder(ctx context.Context, pageID uuid.UUID, ids []uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	now := time.Now().UTC()
	for i, id := range ids {
		tag, err := tx.Exec(ctx, `
			UPDATE synclink_links SET sort_order = $1, updated_at = $2
			WHERE id = $3 AND page_id = $4
		`, i, now, id, pageID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return domainErr.New(domainErr.ErrNotFound, "link not found", nil)
		}
	}
	return tx.Commit(ctx)
}
