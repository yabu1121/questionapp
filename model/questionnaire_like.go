package model

import (
	"time"
)

type QuestionnaireLike struct {
	QuestionnaireID int64
	UserID          int64
	CreatedAt       time.Time
}
