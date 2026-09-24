package model

import "time"

type Vote struct {
	ID              int64
	QuestionnaireID int64
	UserID          int64
	CreatedAt       time.Time
}
