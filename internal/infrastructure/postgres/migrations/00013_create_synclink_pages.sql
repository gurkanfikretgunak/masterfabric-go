-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS synclink_pages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    slug VARCHAR(64) NOT NULL UNIQUE,
    display_name VARCHAR(80) NOT NULL,
    bio VARCHAR(280) NOT NULL DEFAULT '',
    avatar_url TEXT,
    theme VARCHAR(32) NOT NULL DEFAULT 'default'
        CHECK (theme IN ('default', 'dark', 'light', 'colorful')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_synclink_pages_slug ON synclink_pages(slug);
CREATE INDEX idx_synclink_pages_user_id ON synclink_pages(user_id);

CREATE TABLE IF NOT EXISTS synclink_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    page_id UUID NOT NULL REFERENCES synclink_pages(id) ON DELETE CASCADE,
    title VARCHAR(80) NOT NULL,
    url TEXT NOT NULL,
    icon TEXT,
    sort_order INT NOT NULL DEFAULT 0,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_synclink_links_page_id_order ON synclink_links(page_id, sort_order);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_synclink_links_page_id_order;
DROP TABLE IF EXISTS synclink_links;
DROP INDEX IF EXISTS idx_synclink_pages_user_id;
DROP INDEX IF EXISTS idx_synclink_pages_slug;
DROP TABLE IF EXISTS synclink_pages;
-- +goose StatementEnd
