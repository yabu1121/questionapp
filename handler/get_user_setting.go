package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"jev/middleware"
	"jev/model"
	"log/slog"
	"net/http"
)

type getUserSettingResponse struct {
	UserID   int64        `json:"user_id"`
	UIMode   model.UIMode `json:"ui_mode"`
	Timezone string       `json:"timezone"`
}

func GetUserSetting(db *sql.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.AuthenticatedUserID(r.Context())
		if !ok {
			logger.ErrorContext(r.Context(), "authenticated user id is missing from request context")
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		query := `select user_id, ui_mode, timezone from user_setting where user_id = ?`
		row := db.QueryRowContext(r.Context(), query, userID)

		var res getUserSettingResponse
		err := row.Scan(
			&res.UserID,
			&res.UIMode,
			&res.Timezone,
		)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				logger.InfoContext(r.Context(), "user setting not found", "user_id", userID)
				http.Error(w, "user setting not found", http.StatusNotFound)
				return
			}
			logger.ErrorContext(r.Context(), "failed to retrieve user setting", "user_id", userID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(res); err != nil {
			logger.ErrorContext(r.Context(), "failed to encode user setting response", "user_id", userID, "error", err)
		}
	}
}
