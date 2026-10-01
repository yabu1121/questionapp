package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"jev/model"
	"log/slog"
	"net/http"
	"strconv"
)

type getUserSettingResponse struct {
	UserID   int64        `json:"user_id"`
	UIMode   model.UIMode `json:"ui_mode"`
	Timezone string       `json:"timezone"`
}

func GetUserSetting(db *sql.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stringUserID := r.PathValue("user_id")
		userID, err := strconv.ParseInt(stringUserID, 10, 64)
		if err != nil {
			logger.InfoContext(r.Context(), "failed to parse user id for user setting retrieval", "user_id", stringUserID, "error", err)
			http.Error(w, "user_id must be a positive integer", http.StatusBadRequest)
			return
		}
		if userID <= 0 {
			logger.InfoContext(r.Context(), "invalid user id for user setting retrieval", "user_id", userID)
			http.Error(w, "user_id must be a positive integer", http.StatusBadRequest)
			return
		}

		query := `select user_id, ui_mode, timezone from user_setting where user_id = ?`
		row := db.QueryRowContext(r.Context(), query, userID)

		var res getUserSettingResponse
		err = row.Scan(
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
