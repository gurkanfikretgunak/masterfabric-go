package dto

import (
	"time"

	"github.com/google/uuid"
)

type Page struct {
	ID          uuid.UUID `json:"id"`
	Slug        string    `json:"slug"`
	DisplayName string    `json:"displayName"`
	Bio         string    `json:"bio"`
	AvatarURL   *string   `json:"avatarUrl"`
	Theme       string    `json:"theme"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Link struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	URL       string    `json:"url"`
	Icon      *string   `json:"icon"`
	Order     int       `json:"order"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type PublicLink struct {
	ID    uuid.UUID `json:"id"`
	Title string    `json:"title"`
	URL   string    `json:"url"`
	Icon  *string   `json:"icon"`
	Order int       `json:"order"`
}

type PublicPage struct {
	Slug        string       `json:"slug"`
	DisplayName string       `json:"displayName"`
	Bio         string       `json:"bio"`
	AvatarURL   *string      `json:"avatarUrl"`
	Theme       string       `json:"theme"`
	Links       []PublicLink `json:"links"`
}

type UpsertPageInput struct {
	Slug        string  `json:"slug" validate:"required,min=2,max=64"`
	DisplayName string  `json:"displayName" validate:"required,min=1,max=80"`
	Bio         string  `json:"bio" validate:"max=280"`
	AvatarURL   *string `json:"avatarUrl"`
	Theme       string  `json:"theme"`
}

type CreateLinkInput struct {
	Title  string  `json:"title" validate:"required,min=1,max=80"`
	URL    string  `json:"url" validate:"required,url"`
	Icon   *string `json:"icon"`
	Active *bool   `json:"active"`
}

type UpdateLinkInput struct {
	Title  *string `json:"title"`
	URL    *string `json:"url"`
	Icon   *string `json:"icon"`
	Active *bool   `json:"active"`
}

type ReorderLinksInput struct {
	IDs []uuid.UUID `json:"ids" validate:"required,min=1"`
}
