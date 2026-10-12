package handler

import (
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"log/slog"
	"net/http"
	"time"
)

func Logout(db *sql.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, cookieErr := r.Cookie("questionnaire_session")

		if cookieErr == nil {
			tokenBytes, decodeErr := base64.RawURLEncoding.DecodeString(cookie.Value)

			if decodeErr == nil && len(tokenBytes) == 32 {
				tokenHash := sha256.Sum256(tokenBytes)

				query := `
					delete from sessions
					where token_hash = ?
				`

				if _, err := db.ExecContext(r.Context(), query, tokenHash[:]); err != nil {
					logger.ErrorContext(r.Context(), "failed to delete session during logout", "error", err)
					http.Error(w, "internal server error", http.StatusInternalServerError)
					return
				}
			} else {
				logger.InfoContext(r.Context(), "discarding invalid session cookie during logout")
			}
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "questionnaire_session",
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Secure:   !isDevelopment(),
			SameSite: http.SameSiteLaxMode,
			Expires:  time.Unix(0, 0).UTC(),
			MaxAge:   -1,
		})

		logger.InfoContext(r.Context(), "user logged out")
		w.WriteHeader(http.StatusNoContent)
	}
}
