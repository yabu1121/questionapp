package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"jev/config"
	"jev/model"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

type createQuestionnaireRequest struct {
	CreatedBy    int64                 `json:"created_by"`
	Title        string                `json:"title"`
	Description  string                `json:"description"`
	Deadline     time.Time             `json:"deadline"`
	Type         string                `json:"type"`
	MaxChoices   int                   `json:"max_choices"`
	LanguageCode model.LanguageCode    `json:"language_code"`
	Visibility   string                `json:"visibility"`
	Choices      []createChoiceRequest `json:"choices"`
}

type createChoiceRequest struct {
	Title string `json:"title"`
}

type createQuestionnaireResponse struct {
	ID               int64                     `json:"id"`
	CreatedBy        int64                     `json:"created_by"`
	Title            string                    `json:"title"`
	Description      string                    `json:"description"`
	Status           model.QuestionnaireStatus `json:"status"`
	Deadline         time.Time                 `json:"deadline"`
	Type             model.QuestionType        `json:"type"`
	MaxChoices       int                       `json:"max_choices"`
	LanguageCode     model.LanguageCode        `json:"language_code"`
	LanguageDetected bool                      `json:"language_detected"`
	Visibility       model.ResultVisibility    `json:"visibility"`
	Choices          []createChoiceResponse    `json:"choices"`
}

type createChoiceResponse struct {
	ID           int64  `json:"id"`
	Title        string `json:"title"`
	DisplayOrder int    `json:"display_order"`
}

func (r createQuestionnaireRequest) Validate() error {
	if r.CreatedBy <= 0 {
		return errors.New("created_by must be a positive integer")
	}

	if r.Title == "" {
		return errors.New("title is required")
	}
	if utf8.RuneCountInString(r.Title) > 100 {
		return errors.New("title must be 100 characters or less")
	}

	if utf8.RuneCountInString(r.Description) > 255 {
		return errors.New("description must be 255 characters or less")
	}

	if r.Deadline.Before(time.Now()) {
		return errors.New("deadline must be in the future")
	}

	if r.LanguageCode == "" {
		return errors.New("language_code is required")
	}

	if len(r.Choices) < 2 {
		return errors.New("at least two choices are required")
	}

	for _, choice := range r.Choices {
		if choice.Title == "" {
			return errors.New("choice title is required")
		}

		if utf8.RuneCountInString(choice.Title) > 20 {
			return errors.New("choice title must be 20 characters or less")
		}
	}

	if r.MaxChoices <= 0 {
		return errors.New("max_choices must be a positive integer")
	}
	if r.MaxChoices > len(r.Choices) {
		return errors.New("max_choices cannot exceed the number of choices")
	}

	if r.Type == string(model.QuestionTypeSingle) && r.MaxChoices != 1 {
		return errors.New("max_choices must be 1 for single_choice")
	}
	return nil
}

func (r *createQuestionnaireRequest) Normalize() {
	r.Title = strings.TrimSpace(r.Title)
	r.Description = strings.TrimSpace(r.Description)
	r.Type = strings.TrimSpace(r.Type)
	r.Visibility = strings.TrimSpace(r.Visibility)

	for i := range r.Choices {
		r.Choices[i].Title = strings.TrimSpace(
			r.Choices[i].Title,
		)
	}
}

func CreateQuestionnaire(db *sql.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, config.MaxCreateQuestionnaireRequestBodyBytes)
		var req createQuestionnaireRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.InfoContext(r.Context(), "failed to decode create questionnaire request", "error", err)
			http.Error(w, "request body must be valid JSON", http.StatusBadRequest)
			return
		}

		status, err := model.ParseQuestionnaireStatus(r.PathValue("status"))
		if err != nil {
			logger.InfoContext(r.Context(), "failed to parse questionnaire status", "error", err)
			http.Error(w, "invalid questionnaire status", http.StatusBadRequest)
			return
		}

		if status == model.QuestionnaireStatusClosed {
			logger.InfoContext(r.Context(), "closed questionnaire cannot be created", "status", status)
			http.Error(w, "questionnaire cannot be created with closed status", http.StatusBadRequest)
			return
		}

		req.Normalize()

		questionType, err := model.ParseQuestionType(req.Type)
		if err != nil {
			logger.InfoContext(r.Context(), "invalid question type", "error", err)
			http.Error(w, "type must be single_choice or multi_choices", http.StatusBadRequest)
			return
		}

		visibility, err := model.ParseResultVisibility(req.Visibility)
		if err != nil {
			logger.InfoContext(r.Context(), "invalid result visibility", "error", err)
			http.Error(w, "visibility must be always, after_vote, or closed", http.StatusBadRequest)
			return
		}

		if err := req.Validate(); err != nil {
			logger.InfoContext(r.Context(), "create questionnaire request validation failed", "error", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		tx, err := db.BeginTx(r.Context(), nil)
		if err != nil {
			logger.ErrorContext(r.Context(), "failed to begin create questionnaire transaction", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		defer func() {
			if rerr := tx.Rollback(); rerr != nil && !errors.Is(rerr, sql.ErrTxDone) {
				logger.ErrorContext(r.Context(), "failed to roll back create questionnaire transaction", "created_by", req.CreatedBy, "error", rerr)
			}
		}()

		// insert questionnaire
		questionnaireQuery := `
			insert into questionnaire (created_by, title, description, status, deadline, type, max_choices, language_code, language_detected, visibility)
			values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`
		questionnaireResult, err := tx.ExecContext(r.Context(), questionnaireQuery, req.CreatedBy, req.Title, req.Description, status, req.Deadline, questionType, req.MaxChoices, req.LanguageCode, false, visibility)
		if err != nil {
			logger.ErrorContext(r.Context(), "failed to insert questionnaire", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		questionnaireID, err := questionnaireResult.LastInsertId()
		if err != nil {
			logger.ErrorContext(r.Context(), "failed to read created questionnaire id", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		choiceQuery := `insert into choice (questionnaire_id, title, display_order) values (?, ?, ?)`
		choiceResponses := make([]createChoiceResponse, 0, len(req.Choices))
		for index, choice := range req.Choices {
			displayOrder := index + 1

			choiceResult, err := tx.ExecContext(r.Context(), choiceQuery, questionnaireID, choice.Title, displayOrder)
			if err != nil {
				logger.ErrorContext(r.Context(), "failed to insert questionnaire choice", "questionnaire_id", questionnaireID, "display_order", displayOrder, "error", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			choiceID, err := choiceResult.LastInsertId()
			if err != nil {
				logger.ErrorContext(r.Context(), "failed to read created choice id", "questionnaire_id", questionnaireID, "display_order", displayOrder, "error", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			choiceResponse := createChoiceResponse{
				ID:           choiceID,
				Title:        choice.Title,
				DisplayOrder: displayOrder,
			}

			choiceResponses = append(choiceResponses, choiceResponse)
		}

		if err := tx.Commit(); err != nil {
			logger.ErrorContext(r.Context(), "failed to commit create questionnaire transaction", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		res := createQuestionnaireResponse{
			ID:               questionnaireID,
			CreatedBy:        req.CreatedBy,
			Title:            req.Title,
			Description:      req.Description,
			Status:           status,
			Deadline:         req.Deadline,
			Type:             questionType,
			MaxChoices:       req.MaxChoices,
			LanguageCode:     req.LanguageCode,
			LanguageDetected: false,
			Visibility:       visibility,
			Choices:          choiceResponses,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		if err := json.NewEncoder(w).Encode(res); err != nil {
			logger.ErrorContext(r.Context(), "failed to encode create questionnaire response", "error", err)
		}
	}
}
