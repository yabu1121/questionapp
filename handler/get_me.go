package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"jev/middleware"
	"log/slog"
	"net/http"
)

type getMeResponse struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	DisplayName string  `json:"display_name"`
	Handle      string  `json:"handle"`
	Email       string  `json:"email"`
	Bio         *string `json:"bio"`
	AvatarURL   *string `json:"avatar_url"`
}

func GetMe(db *sql.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.AuthenticatedUserID(r.Context())
		if !ok {
			logger.ErrorContext(r.Context(), "authenticated user id is missing from request context")
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		query := `
			select
				id,
				name,
				display_name,
				handle,
				email,
				bio,
				avatar_url
			from users
			where id = ?
		`
		var user getMeResponse

		err := db.QueryRowContext(r.Context(), query, userID).Scan(
			&user.ID,
			&user.Name,
			&user.DisplayName,
			&user.Handle,
			&user.Email,
			&user.Bio,
			&user.AvatarURL,
		)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				logger.InfoContext(r.Context(), "authenticated user not found", "user_id", userID)
				http.Error(w, "authentication required", http.StatusUnauthorized)
				return
			}
			logger.ErrorContext(r.Context(), "failed to retrieve authenticated user", "user_id", userID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(user); err != nil {
			logger.ErrorContext(r.Context(), "failed to encode authenticated user response", "user_id", user.ID, "error", err)
			return
		}
	}
}
