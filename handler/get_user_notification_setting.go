package handler

import (
	"database/sql"
	"encoding/json"
	"jev/model"
	"log/slog"
	"net/http"
	"strconv"
)

type getUserNotificationSettingResponse struct {
	UserID    int64         `json:"user_id"`
	Channel   model.Channel `json:"channel"`
	IsEnabled bool          `json:"is_enabled"`
}

func GetUserNotificationSetting(db *sql.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stringUserID := r.PathValue("user_id")
		userID, err := strconv.ParseInt(stringUserID, 10, 64)
		if err != nil || userID <= 0 {
			logger.InfoContext(r.Context(), "invalid user id for user notification setting retrieval", "user_id", stringUserID)
			http.Error(w, "user_id must be a positive integer", http.StatusBadRequest)
			return
		}

		query := `select user_id, channel, is_enabled from user_notification_setting where user_id = ? order by channel asc`
		rows, err := db.QueryContext(r.Context(), query, userID)
		if err != nil {
			logger.ErrorContext(r.Context(), "failed to retrieve user notification settings", "user_id", userID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		defer func() {
			if cerr := rows.Close(); cerr != nil {
				logger.ErrorContext(r.Context(), "failed to close user notification setting rows", "user_id", userID, "error", cerr)
			}
		}()

		userNotificationSettings := make([]getUserNotificationSettingResponse, 0)
		for rows.Next() {
			var userNotificationSetting getUserNotificationSettingResponse
			err = rows.Scan(
				&userNotificationSetting.UserID,
				&userNotificationSetting.Channel,
				&userNotificationSetting.IsEnabled,
			)
			if err != nil {
				logger.ErrorContext(r.Context(), "failed to scan user notification setting", "user_id", userID, "error", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			userNotificationSettings = append(userNotificationSettings, userNotificationSetting)
		}
		if err := rows.Err(); err != nil {
			logger.ErrorContext(r.Context(), "failed while iterating over user notification settings", "user_id", userID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(userNotificationSettings); err != nil {
			logger.ErrorContext(r.Context(), "failed to encode user notification settings response", "user_id", userID, "error", err)
		}
	}
}
