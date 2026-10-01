package handler

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

type getQuestionnaireCommentResponse struct {
	ID              int64     `json:"id"`
	QuestionnaireID int64     `json:"questionnaire_id"`
	UserID          int64     `json:"user_id"`
	Content         string    `json:"content"`
	ParentCommentID *int64    `json:"parent_comment_id"`
	CreatedAt       time.Time `json:"created_at"`
}

func GetQuestionnaireComments(db *sql.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stringQuestionnaireID := r.PathValue("questionnaire_id")
		questionnaireID, err := strconv.ParseInt(stringQuestionnaireID, 10, 64)
		if err != nil || questionnaireID <= 0 {
			logger.InfoContext(r.Context(), "invalid questionnaire id for comment retrieval", "questionnaire_id", stringQuestionnaireID)
			http.Error(w, "questionnaire_id must be a positive integer", http.StatusBadRequest)
			return
		}
		query := `select id, questionnaire_id, user_id, content, parent_comment_id, created_at from comments where questionnaire_id = ? order by created_at desc, id desc`
		rows, err := db.QueryContext(r.Context(), query, questionnaireID)
		if err != nil {
			logger.ErrorContext(r.Context(), "failed to retrieve questionnaire comments", "questionnaire_id", questionnaireID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		getComments := make([]getQuestionnaireCommentResponse, 0)
		for rows.Next() {
			var getComment getQuestionnaireCommentResponse
			if err := rows.Scan(
				&getComment.ID,
				&getComment.QuestionnaireID,
				&getComment.UserID,
				&getComment.Content,
				&getComment.ParentCommentID,
				&getComment.CreatedAt,
			); err != nil {
				logger.ErrorContext(r.Context(), "failed to scan questionnaire comment", "questionnaire_id", questionnaireID, "error", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			getComments = append(getComments, getComment)
		}
		if err := rows.Err(); err != nil {
			logger.ErrorContext(r.Context(), "failed while iterating over questionnaire comments", "questionnaire_id", questionnaireID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(getComments); err != nil {
			logger.ErrorContext(r.Context(), "failed to encode questionnaire comments response", "questionnaire_id", questionnaireID, "error", err)
		}
	}
}
