package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"jev/config"
	"jev/model"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type patchUserSettingRequest struct {
	UIMode   *string `json:"ui_mode"`
	Timezone *string `json:"timezone"`
}

func (r *patchUserSettingRequest) Normalize() {
	if r.UIMode != nil {
		value := strings.TrimSpace(*r.UIMode)
		r.UIMode = &value
	}
	if r.Timezone != nil {
		value := strings.TrimSpace(*r.Timezone)
		r.Timezone = &value
	}
}

func (r patchUserSettingRequest) Validate() error {
	if r.UIMode == nil && r.Timezone == nil {
		return errors.New("at least one user setting field is required")
	}

	if r.UIMode != nil {
		if _, err := model.ParseUIMode(*r.UIMode); err != nil {
			return errors.New("ui_mode must be light, dark, or system")
		}
	}

	if r.Timezone != nil {
		if *r.Timezone == "" {
			return errors.New("timezone must not be empty")
		}
		if len(*r.Timezone) > 20 {
			return errors.New("timezone must be 20 characters or less")
		}
		if _, err := time.LoadLocation(*r.Timezone); err != nil {
			return errors.New("timezone must be a valid IANA timezone")
		}
	}

	return nil
}

func PatchUserSetting(db *sql.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, config.MaxPatchUserSettingRequestBodyBytes)

		stringUserID := r.PathValue("user_id")

		userID, err := strconv.ParseInt(stringUserID, 10, 64)
		if err != nil || userID <= 0 {
			logger.InfoContext(r.Context(), "invalid user id for user setting update", "user_id", stringUserID)
			http.Error(w, "user_id must be a positive integer", http.StatusBadRequest)
			return
		}

		var req patchUserSettingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.InfoContext(r.Context(), "failed to decode user setting update request", "user_id", userID, "error", err)
			http.Error(w, "request body must be valid JSON", http.StatusBadRequest)
			return
		}

		req.Normalize()

		if err := req.Validate(); err != nil {
			logger.InfoContext(r.Context(), "user setting update request validation failed", "user_id", userID, "error", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		query := `
		update user_setting
		set
			ui_mode = coalesce(?, ui_mode),
			timezone = coalesce(?, timezone)
		where user_id = ?`

		result, err := db.ExecContext(r.Context(), query, req.UIMode, req.Timezone, userID)
		if err != nil {
			logger.ErrorContext(r.Context(), "failed to update user setting", "user_id", userID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		count, err := result.RowsAffected()
		if err != nil {
			logger.ErrorContext(r.Context(), "failed to read affected rows after updating user setting", "user_id", userID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		if count == 0 {
			var exists bool
			existsQuery := `select exists(select 1 from user_setting where user_id = ?)`
			if err := db.QueryRowContext(r.Context(), existsQuery, userID).Scan(&exists); err != nil {
				logger.ErrorContext(r.Context(), "failed to check user setting existence after update", "user_id", userID, "error", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			if !exists {
				logger.InfoContext(r.Context(), "user setting not found for update", "user_id", userID)
				http.Error(w, "user setting not found", http.StatusNotFound)
				return
			}
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
