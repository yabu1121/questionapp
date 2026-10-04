package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/go-sql-driver/mysql"
)

const maxPatchUserProfileRequestBodyBytes = 1 << 20

type patchUserProfileRequest struct {
	Name        *string `json:"name"`
	DisplayName *string `json:"display_name"`
	Handle      *string `json:"handle"`
	Bio         *string `json:"bio"`
	AvatarURL   *string `json:"avatar_url"`
}

func (r patchUserProfileRequest) Normalize() {
	if r.Name != nil {
		*r.Name = strings.TrimSpace(*r.Name)
	}

	if r.DisplayName != nil {
		*r.DisplayName = strings.TrimSpace(*r.DisplayName)
	}

	if r.Handle != nil {
		*r.Handle = strings.TrimSpace(*r.Handle)
	}

	if r.Bio != nil {
		*r.Bio = strings.TrimSpace(*r.Bio)
	}

	if r.AvatarURL != nil {
		*r.AvatarURL = strings.TrimSpace(*r.AvatarURL)
	}

}

func (r patchUserProfileRequest) Validate() error {
	if r.Name == nil &&
		r.DisplayName == nil &&
		r.Handle == nil &&
		r.Bio == nil &&
		r.AvatarURL == nil {
		return errors.New("at least one profile field is required")
	}
	if r.Name != nil && *r.Name == "" {
		return errors.New("name must not be empty")
	}
	if r.Name != nil && utf8.RuneCountInString(*r.Name) > 20 {
		return errors.New("name must be 20 characters or less")
	}
	if r.DisplayName != nil && *r.DisplayName == "" {
		return errors.New("display_name must not be empty")
	}
	if r.DisplayName != nil && utf8.RuneCountInString(*r.DisplayName) > 20 {
		return errors.New("display_name must be 20 characters or less")
	}
	if r.Handle != nil && *r.Handle == "" {
		return errors.New("handle must not be empty")
	}
	if r.Handle != nil && utf8.RuneCountInString(*r.Handle) > 20 {
		return errors.New("handle must be 20 characters or less")
	}
	if r.Handle != nil && strings.ContainsAny(*r.Handle, "/?#:@=&") {
		return errors.New("handle contains invalid characters")
	}
	if r.Bio != nil && utf8.RuneCountInString(*r.Bio) > 255 {
		return errors.New("bio must be 255 characters or less")
	}
	if r.AvatarURL != nil && utf8.RuneCountInString(*r.AvatarURL) > 2048 {
		return errors.New("avatar_url must be 2048 characters or less")
	}
	return nil
}

func PatchUserProfile(db *sql.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stringUserID := r.PathValue("user_id")
		userID, err := strconv.ParseInt(stringUserID, 10, 64)
		if err != nil || userID <= 0 {
			logger.InfoContext(r.Context(), "invalid user id for user profile update", "user_id", stringUserID)
			http.Error(w, "user_id must be a positive integer", http.StatusBadRequest)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxPatchUserProfileRequestBodyBytes)

		var req patchUserProfileRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.InfoContext(r.Context(), "failed to decode user profile update request", "user_id", userID, "error", err)
			http.Error(w, "request body must be valid JSON", http.StatusBadRequest)
			return
		}

		req.Normalize()

		if err := req.Validate(); err != nil {
			logger.InfoContext(r.Context(), "user profile update request validation failed", "user_id", userID, "error", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var exists bool
		checkQuery := `select exists(select 1 from users where id = ?)`

		if err := db.QueryRowContext(r.Context(), checkQuery, userID).Scan(&exists); err != nil {
			logger.ErrorContext(r.Context(), "failed to check user existence before updating profile", "user_id", userID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		if !exists {
			logger.InfoContext(r.Context(), "user not found for profile update", "user_id", userID)
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}

		query := `
		update users
		set
			name = coalesce(?, name),
			display_name = coalesce(?, display_name),
			handle = coalesce(?, handle),
			bio = coalesce(?, bio),
			avatar_url = coalesce(?, avatar_url)
		where id = ?
		`

		result, err := db.ExecContext(r.Context(), query, req.Name, req.DisplayName, req.Handle, req.Bio, req.AvatarURL, userID)
		if err != nil {
			var mysqlErr *mysql.MySQLError
			if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
				logger.InfoContext(r.Context(), "handle already exists during user profile update", "user_id", userID)
				http.Error(w, "handle already exists", http.StatusConflict)
				return
			}
			logger.ErrorContext(r.Context(), "failed to update user profile", "user_id", userID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		_, err = result.RowsAffected()
		if err != nil {
			logger.ErrorContext(r.Context(), "failed to read affected rows after updating user profile", "user_id", userID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
