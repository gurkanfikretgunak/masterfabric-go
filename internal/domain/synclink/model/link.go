package model

import (
	"time"

	"github.com/google/uuid"
)

// Link is a single item on a SyncLink page.
type Link struct {
	ID        uuid.UUID
	PageID    uuid.UUID
	Title     string
	URL       string
	Icon      *string
	Order     int
	Active    bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
