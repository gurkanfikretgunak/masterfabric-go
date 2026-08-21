package model

import (
	"time"

	"github.com/google/uuid"
)

const (
	ThemeDefault  = "default"
	ThemeDark     = "dark"
	ThemeLight    = "light"
	ThemeColorful = "colorful"
)

func ValidTheme(theme string) bool {
	switch theme {
	case ThemeDefault, ThemeDark, ThemeLight, ThemeColorful:
		return true
	default:
		return false
	}
}

// Page is a user's public SyncLink page.
type Page struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Slug        string
	DisplayName string
	Bio         string
	AvatarURL   *string
	Theme       string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
