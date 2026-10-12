package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"jev/config"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func generateSessionTokens() (string, [32]byte, error) {
	const sessionTokenBytes = 32
	tokenBytes := make([]byte, sessionTokenBytes)

	_, err := rand.Read(tokenBytes)
	if err != nil {
		return "", [32]byte{}, err
	}

	rawToken := base64.RawURLEncoding.EncodeToString(tokenBytes)
	tokenHash := sha256.Sum256(tokenBytes)

	return rawToken, tokenHash, nil
}

func (r loginRequest) Validate() error {
	if r.Email == "" {
		return errors.New("email is required")
	}
	if r.Password == "" {
		return errors.New("password is required")
	}

	if len([]byte(r.Password)) > 72 {
		return errors.New("password must be 72 bytes or less")
	}
	return nil
}

func (r *loginRequest) Normalize() {
	r.Email = strings.TrimSpace(r.Email)
}

func isDevelopment() bool {
	return os.Getenv("APP_ENV") == "development"
}

func Login(db *sql.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, config.MaxLoginRequestBodyBytes)
		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.InfoContext(r.Context(), "failed to decode login request", "error", err)
			http.Error(w, "request body must be valid JSON", http.StatusBadRequest)
			return
		}

		req.Normalize()

		if err := req.Validate(); err != nil {
			logger.InfoContext(r.Context(), "login request validation failed", "error", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		checkQuery := `select id, hashed_password from users where email = ?`

		var userID int64
		var hashedPassword string
		if err := db.QueryRowContext(r.Context(), checkQuery, req.Email).Scan(&userID, &hashedPassword); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				logger.InfoContext(r.Context(), "login failed due to invalid credentials")
				http.Error(w, "invalid email or password", http.StatusUnauthorized)
				return
			}
			logger.ErrorContext(r.Context(), "failed to retrieve user for login", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(req.Password)); err != nil {
			if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
				logger.InfoContext(r.Context(), "login failed due to invalid credentials", "user_id", userID)
				http.Error(w, "invalid email or password", http.StatusUnauthorized)
				return
			}
			logger.ErrorContext(r.Context(), "failed to compare password hash during login", "user_id", userID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		rawToken, tokenHash, err := generateSessionTokens()
		if err != nil {
			logger.ErrorContext(r.Context(), "failed to generate session token", "user_id", userID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		validTime := 72 * time.Hour
		expiresAt := time.Now().UTC().Add(validTime)
		query := `insert into sessions (token_hash, user_id, expires_at) values (?, ?, ?)`

		_, err = db.ExecContext(r.Context(), query, tokenHash[:], userID, expiresAt)
		if err != nil {
			logger.ErrorContext(r.Context(), "failed to create login session", "user_id", userID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		cookie := http.Cookie{
			Name:     "questionnaire_session",
			Value:    rawToken,
			Path:     "/",
			HttpOnly: true,
			Secure:   !isDevelopment(),
			SameSite: http.SameSiteLaxMode,
			Expires:  expiresAt,
			MaxAge:   int(validTime.Seconds()),
		}

		http.SetCookie(w, &cookie)
		logger.InfoContext(r.Context(), "user logged in", "user_id", userID)
		w.WriteHeader(http.StatusNoContent)
	}
}
