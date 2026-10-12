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

var (
	errInvalidSessionTokenEncoding = errors.New("invalid session token encoding")
	errInvalidSessionTokenLength   = errors.New("invalid session token length")
	errSessionNotFoundOrExpired    = errors.New("session not found or expired")
)

func AuthenticatedUserID(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(userContextKey).(int64)
	return userID, ok
}

func getUserIDFromSession(r *http.Request, db *sql.DB) (int64, error) {
	cookie, err := r.Cookie("questionnaire_session")
	if err != nil {
		return 0, err
	}

	tokenBytes, err := base64.RawURLEncoding.DecodeString(cookie.Value)
	if err != nil {
		return 0, errInvalidSessionTokenEncoding
	}

	if len(tokenBytes) != 32 {
		return 0, errInvalidSessionTokenLength
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
			return 0, errSessionNotFoundOrExpired
		}
		return 0, err
	}

	return userID, nil
}

func Authenticate(db *sql.DB, logger *slog.Logger, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := getUserIDFromSession(r, db)
		if err != nil {
			switch {
			case errors.Is(err, http.ErrNoCookie):
				logger.InfoContext(r.Context(), "authentication failed: session cookie is missing")
			case errors.Is(err, errInvalidSessionTokenEncoding):
				logger.InfoContext(r.Context(), "authentication failed: session token encoding is invalid")
			case errors.Is(err, errInvalidSessionTokenLength):
				logger.InfoContext(r.Context(), "authentication failed: session token length is invalid")
			case errors.Is(err, errSessionNotFoundOrExpired):
				logger.InfoContext(r.Context(), "authentication failed: session was not found or has expired")
			default:
				logger.ErrorContext(r.Context(), "failed to retrieve authenticated session", "error", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, userID)
		next(w, r.WithContext(ctx))
	}
}

func OptionalAuthenticate(db *sql.DB, logger *slog.Logger, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := getUserIDFromSession(r, db)
		if err != nil {
			switch {
			case errors.Is(err, http.ErrNoCookie):
				logger.DebugContext(r.Context(), "optional authentication skipped: session cookie is missing")
			case errors.Is(err, errInvalidSessionTokenEncoding):
				logger.InfoContext(r.Context(), "optional authentication skipped: session token encoding is invalid")
			case errors.Is(err, errInvalidSessionTokenLength):
				logger.InfoContext(r.Context(), "optional authentication skipped: session token length is invalid")
			case errors.Is(err, errSessionNotFoundOrExpired):
				logger.InfoContext(r.Context(), "optional authentication skipped: session was not found or has expired")
			default:
				logger.ErrorContext(r.Context(), "failed to check optional session", "error", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			next(w, r)
			return
		}
		ctx := context.WithValue(r.Context(), userContextKey, userID)
		next(w, r.WithContext(ctx))
	}
}
