package model

import (
	"errors"
	"time"
)

type QuestionnaireStatus string

const (
	QuestionnaireStatusDraft     QuestionnaireStatus = "draft"
	QuestionnaireStatusPublished QuestionnaireStatus = "published"
	QuestionnaireStatusClosed    QuestionnaireStatus = "closed"
)

func ParseQuestionnaireStatus(value string) (QuestionnaireStatus, error) {
	status := QuestionnaireStatus(value)
	switch status {
	case QuestionnaireStatusDraft, QuestionnaireStatusPublished, QuestionnaireStatusClosed:
		return status, nil
	default:
		return "", errors.New("invalid questionnaire status")
	}
}

func CheckStatusIsValid(current, next QuestionnaireStatus) bool {
	switch current {
	case QuestionnaireStatusDraft:
		if next == QuestionnaireStatusPublished {
			return true
		}
		return false
	case QuestionnaireStatusPublished:
		if next == QuestionnaireStatusClosed {
			return true
		}
		return false
	case QuestionnaireStatusClosed:
		return false
	}
	return false
}

type QuestionType string

const (
	QuestionTypeSingle QuestionType = "single_choice"
	QuestionTypeMulti  QuestionType = "multi_choices"
)

func ParseQuestionType(value string) (QuestionType, error) {
	questionType := QuestionType(value)
	switch questionType {
	case QuestionTypeSingle, QuestionTypeMulti:
		return questionType, nil
	default:
		return "", errors.New("invalid question type")
	}
}

type ResultVisibility string

const (
	ResultVisibilityAlways    ResultVisibility = "always"
	ResultVisibilityAfterVote ResultVisibility = "after_vote"
	ResultVisibilityClosed    ResultVisibility = "closed"
)

func ParseResultVisibility(value string) (ResultVisibility, error) {
	visibility := ResultVisibility(value)
	switch visibility {
	case ResultVisibilityAlways, ResultVisibilityAfterVote, ResultVisibilityClosed:
		return visibility, nil
	default:
		return "", errors.New("invalid result visibility")
	}
}

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
