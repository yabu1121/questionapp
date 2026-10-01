package handler

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

type getUserLikesResponse struct {
	QuestionnaireID int64     `json:"questionnaire_id"`
	UserID          int64     `json:"user_id"`
	CreatedAt       time.Time `json:"created_at"`
}

func GetUserLikes(db *sql.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stringUserID := r.PathValue("user_id")
		userID, err := strconv.ParseInt(stringUserID, 10, 64)
		if err != nil || userID <= 0 {
			logger.InfoContext(r.Context(), "invalid user id for like retrieval", "user_id", stringUserID)
			http.Error(w, "user_id must be a positive integer", http.StatusBadRequest)
			return
		}

		query := `select questionnaire_id, user_id, created_at from questionnaire_like where user_id = ? order by created_at desc, questionnaire_id desc`
		rows, err := db.QueryContext(r.Context(), query, userID)
		if err != nil {
			logger.ErrorContext(r.Context(), "failed to retrieve user likes", "user_id", userID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		userLikes := make([]getUserLikesResponse, 0)
		for rows.Next() {
			var userLike getUserLikesResponse
			err = rows.Scan(
				&userLike.QuestionnaireID,
				&userLike.UserID,
				&userLike.CreatedAt,
			)
			if err != nil {
				logger.ErrorContext(r.Context(), "failed to scan user like", "user_id", userID, "error", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			userLikes = append(userLikes, userLike)
		}

		if err := rows.Err(); err != nil {
			logger.ErrorContext(r.Context(), "failed while iterating over user likes", "user_id", userID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(userLikes); err != nil {
			logger.ErrorContext(r.Context(), "failed to encode user likes response", "user_id", userID, "error", err)
		}
	}
}
