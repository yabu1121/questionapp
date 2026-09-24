package model

import "time"

type QuestionnaireStatus string

const (
	QuestionnaireStatusDraft     QuestionnaireStatus = "draft"
	QuestionnaireStatusPublished QuestionnaireStatus = "published"
	QuestionnaireStatusClosed    QuestionnaireStatus = "closed"
)

type QuestionType string

const (
	QuestionTypeSingle QuestionType = "single_choice"
	QuestionTypeMulti  QuestionType = "multi_choices"
)

type ResultVisibility string

const (
	ResultVisibilityAlways    ResultVisibility = "always"
	ResultVisibilityAfterVote ResultVisibility = "after_vote"
	ResultVisibilityClosed    ResultVisibility = "closed"
)

type Questionnaire struct {
	ID          int64
	CreatedBy   int64
	Title       string
	Description string
	Status      QuestionnaireStatus
	Deadline    time.Time

	Type       QuestionType
	MaxChoices int

	LanguageCode     LanguageCode
	LanguageDetected bool

	Visibility ResultVisibility

	CreatedAt time.Time
	UpdatedAt *time.Time
}
