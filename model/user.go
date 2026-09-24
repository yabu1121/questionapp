package model

import "time"

type User struct {
	ID             int64
	Name           string
	DisplayName    string
	Handle         string
	Email          string
	HashedPassword string
	Bio            *string
	AvatarURL      *string
	Birthday       *time.Time
	CreatedAt      time.Time
	UpdatedAt      *time.Time
}
