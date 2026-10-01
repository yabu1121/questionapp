package handler

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
)

type getQuestionnaireChoiceResponse struct {
	ID              int64  `json:"id"`
	QuestionnaireID int64  `json:"questionnaire_id"`
	Title           string `json:"title"`
	DisplayOrder    int    `json:"display_order"`
}

func GetQuestionnaireChoices(db *sql.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stringQuestionnaireID := r.PathValue("questionnaire_id")
		questionnaireID, err := strconv.ParseInt(stringQuestionnaireID, 10, 64)
		if err != nil || questionnaireID <= 0 {
			logger.InfoContext(r.Context(), "invalid questionnaire id for choice retrieval", "questionnaire_id", stringQuestionnaireID)
			http.Error(w, "questionnaire_id must be a positive integer", http.StatusBadRequest)
			return
		}

		query := `select id, questionnaire_id, title, display_order from choice where questionnaire_id = ? order by display_order asc`
		rows, err := db.QueryContext(r.Context(), query, questionnaireID)
		if err != nil {
			logger.ErrorContext(r.Context(), "failed to retrieve questionnaire choices", "questionnaire_id", questionnaireID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		questionnaireChoices := make([]getQuestionnaireChoiceResponse, 0)
		for rows.Next() {
			var questionnaireChoice getQuestionnaireChoiceResponse
			err = rows.Scan(
				&questionnaireChoice.ID,
				&questionnaireChoice.QuestionnaireID,
				&questionnaireChoice.Title,
				&questionnaireChoice.DisplayOrder,
			)
			if err != nil {
				logger.ErrorContext(r.Context(), "failed to scan questionnaire choice", "questionnaire_id", questionnaireID, "error", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			questionnaireChoices = append(questionnaireChoices, questionnaireChoice)
		}
		if err := rows.Err(); err != nil {
			logger.ErrorContext(r.Context(), "failed while iterating over questionnaire choices", "questionnaire_id", questionnaireID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(questionnaireChoices); err != nil {
			logger.ErrorContext(r.Context(), "failed to encode questionnaire choices response", "questionnaire_id", questionnaireID, "error", err)
		}
	}
}
