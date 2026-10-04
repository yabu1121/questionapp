package handler

import (
	"database/sql"
	"encoding/json"
	"jev/model"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

type getQuestionnaireResponse struct {
	Questionnaire questionnaireDetail `json:"questionnaire"`
	Choices       []choiceDetail      `json:"choices"`
}

type questionnaireDetail struct {
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
	CreatedAt        time.Time                 `json:"created_at"`
	UpdatedAt        *time.Time                `json:"updated_at"`
}

type choiceDetail struct {
	ChoiceID     int64  `json:"choice_id"`
	ChoiceTitle  string `json:"choice_title"`
	DisplayOrder int    `json:"display_order"`
}

func GetQuestionnaire(db *sql.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stringQuestionnaireID := r.PathValue("questionnaire_id")
		questionnaireID, err := strconv.ParseInt(stringQuestionnaireID, 10, 64)
		if err != nil || questionnaireID <= 0 {
			logger.InfoContext(r.Context(), "invalid questionnaire id for questionnaire retrieval", "questionnaire_id", stringQuestionnaireID)
			http.Error(w, "questionnaire_id must be a positive integer", http.StatusBadRequest)
			return
		}

		query := `
		select
			questionnaire.id,
			questionnaire.created_by,
			questionnaire.title,
			questionnaire.description,
			questionnaire.status,
			questionnaire.deadline,
			questionnaire.type,
			questionnaire.max_choices,
			questionnaire.language_code,
			questionnaire.language_detected,
			questionnaire.visibility,
			questionnaire.created_at,
			questionnaire.updated_at,
			choice.id,
			choice.title,
			choice.display_order
		from questionnaire
		left join choice on choice.questionnaire_id = questionnaire.id
		where questionnaire.id = ?
		`

		rows, err := db.QueryContext(r.Context(), query, questionnaireID)
		if err != nil {
			logger.ErrorContext(r.Context(), "failed to retrieve questionnaire", "questionnaire_id", questionnaireID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		defer func() {
			if cerr := rows.Close(); cerr != nil {
				logger.ErrorContext(r.Context(), "failed to close questionnaire detail rows", "questionnaire_id", questionnaireID, "error", cerr)
			}
		}()

		response := getQuestionnaireResponse{
			Choices: make([]choiceDetail, 0),
		}

		found := false

		for rows.Next() {
			var questionnaire questionnaireDetail
			var choice choiceDetail
			err = rows.Scan(
				&questionnaire.ID,
				&questionnaire.CreatedBy,
				&questionnaire.Title,
				&questionnaire.Description,
				&questionnaire.Status,
				&questionnaire.Deadline,
				&questionnaire.Type,
				&questionnaire.MaxChoices,
				&questionnaire.LanguageCode,
				&questionnaire.LanguageDetected,
				&questionnaire.Visibility,
				&questionnaire.CreatedAt,
				&questionnaire.UpdatedAt,
				&choice.ChoiceID,
				&choice.ChoiceTitle,
				&choice.DisplayOrder,
			)
			if err != nil {
				logger.ErrorContext(r.Context(), "failed to scan questionnaire detail", "questionnaire_id", questionnaireID, "error", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			if !found {
				response.Questionnaire = questionnaire
				found = true
			}

			response.Choices = append(response.Choices, choice)
		}

		if err := rows.Err(); err != nil {
			logger.ErrorContext(r.Context(), "failed while iterating over questionnaire detail", "questionnaire_id", questionnaireID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		if !found {
			logger.InfoContext(r.Context(), "questionnaire not found", "questionnaire_id", questionnaireID)
			http.Error(w, "questionnaire not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			logger.ErrorContext(r.Context(), "failed to encode questionnaire response", "questionnaire_id", questionnaireID, "error", err)
		}
	}
}
