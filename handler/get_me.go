package handler

import (
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
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
		cookie, err := r.Cookie("questionnaire_session")
		if err != nil {
			logger.InfoContext(r.Context(), "failed to authenticate request because session cookie is missing or invalid")
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}

		rawToken := cookie.Value

		tokenBytes, err := base64.RawURLEncoding.DecodeString(rawToken)
		if err != nil {
			logger.InfoContext(r.Context(), "failed to authenticate request because session token encoding is invalid")
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}

		if len(tokenBytes) != 32 {
			logger.InfoContext(r.Context(), "failed to authenticate request because session token length is invalid")
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}

		tokenHash := sha256.Sum256(tokenBytes)

		query := `
			select
				users.id,
				users.name,
				users.display_name,
				users.handle,
				users.email,
				users.bio,
				users.avatar_url
			from sessions
			join users on sessions.user_id = users.id
			where sessions.token_hash = ? and sessions.expires_at > current_timestamp(6)
		`
		var user getMeResponse

		err = db.QueryRowContext(r.Context(), query, tokenHash[:]).Scan(
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
				logger.InfoContext(r.Context(), "failed to authenticate request because session is invalid or expired")
				http.Error(w, "authentication required", http.StatusUnauthorized)
				return
			}
			logger.ErrorContext(r.Context(), "failed to retrieve authenticated user", "error", err)
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
