package handler

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

type getUserCommentResponse struct {
	ID              int64     `json:"id"`
	QuestionnaireID int64     `json:"questionnaire_id"`
	UserID          int64     `json:"user_id"`
	Content         string    `json:"content"`
	ParentCommentID *int64    `json:"parent_comment_id"`
	CreatedAt       time.Time `json:"created_at"`
}

func GetUserComments(db *sql.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stringUserID := r.PathValue("user_id")
		userID, err := strconv.ParseInt(stringUserID, 10, 64)
		if err != nil || userID <= 0 {
			logger.InfoContext(r.Context(), "invalid user id for comment retrieval", "user_id", stringUserID)
			http.Error(w, "user_id must be a positive integer", http.StatusBadRequest)
			return
		}
		query := `select id, questionnaire_id, user_id, content, parent_comment_id, created_at from comments where user_id = ? order by created_at desc, id desc`
		rows, err := db.QueryContext(r.Context(), query, userID)
		if err != nil {
			logger.ErrorContext(r.Context(), "failed to retrieve user comments", "user_id", userID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		getComments := make([]getUserCommentResponse, 0)
		for rows.Next() {
			var getComment getUserCommentResponse
			if err := rows.Scan(
				&getComment.ID,
				&getComment.QuestionnaireID,
				&getComment.UserID,
				&getComment.Content,
				&getComment.ParentCommentID,
				&getComment.CreatedAt,
			); err != nil {
				logger.ErrorContext(r.Context(), "failed to scan user comment", "user_id", userID, "error", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			getComments = append(getComments, getComment)
		}
		if err := rows.Err(); err != nil {
			logger.ErrorContext(r.Context(), "failed while iterating over user comments", "user_id", userID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(getComments); err != nil {
			logger.ErrorContext(r.Context(), "failed to encode user comments response", "user_id", userID, "error", err)
		}
	}
}
