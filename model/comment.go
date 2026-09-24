package model

import "time"

type Comment struct {
	ID              int64
	QuestionnaireID int64
	UserID          int64
	Content         string
	ParentCommentID *int64
	CreatedAt       time.Time
}
