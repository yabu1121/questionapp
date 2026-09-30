package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

type getUserResponse struct {
	ID          int64     `json:"id"`
	DisplayName string    `json:"display_name"`
	Handle      string    `json:"handle"`
	Bio         *string   `json:"bio"`
	AvatarURL   string    `json:"avatar_url"`
	CreatedAt   time.Time `json:"created_at"`
}

func GetUser(db *sql.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := strconv.ParseInt(r.PathValue("user_id"), 10, 64)
		if err != nil || userID <= 0 {
			logger.InfoContext(r.Context(), "failed to parse user id for user retrieval", "user_id", r.PathValue("user_id"), "error", err)
			http.Error(w, "user_id must be a positive integer", http.StatusBadRequest)
			return
		}

		query := `select id, display_name, handle, bio, avatar_url, created_at from users where id = ?`
		var res getUserResponse
		err = db.QueryRowContext(r.Context(), query, userID).Scan(
			&res.ID,
			&res.DisplayName,
			&res.Handle,
			&res.Bio,
			&res.AvatarURL,
			&res.CreatedAt,
		)

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				logger.InfoContext(r.Context(), "user not found", "user_id", userID)
				http.Error(w, "user not found", http.StatusNotFound)
				return
			}
			logger.ErrorContext(r.Context(), "failed to get user", "user_id", userID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(res); err != nil {
			logger.ErrorContext(r.Context(), "failed to encode get user response", "user_id", userID, "error", err)
		}
	}
}
