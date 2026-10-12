package middleware

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"log/slog"
	"net/http"
)

type contextKey string

const userContextKey contextKey = "authenticated_user"

func AuthenticatedUserID(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(userContextKey).(int64)
	return userID, ok
}

func Authenticate(db *sql.DB, logger *slog.Logger, next http.HandlerFunc) http.HandlerFunc {
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
			select user_id
			from sessions
			where sessions.token_hash = ? and sessions.expires_at > current_timestamp(6)
		`
		var userID int64

		err = db.QueryRowContext(r.Context(), query, tokenHash[:]).Scan(&userID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				logger.InfoContext(r.Context(), "failed to authenticate request because session is invalid or expired")
				http.Error(w, "authentication required", http.StatusUnauthorized)
				return
			}
			logger.ErrorContext(r.Context(), "failed to retrieve authenticated session", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, userID)
		next(w, r.WithContext(ctx))
	}
}
