package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"jev/model"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-sql-driver/mysql"
)

type upsertUserNotificationSettingRequest struct {
	IsEnabled *bool `json:"is_enabled"`
}

type upsertUserNotificationSettingResponse struct {
	UserID    int64         `json:"user_id"`
	Channel   model.Channel `json:"channel"`
	IsEnabled bool          `json:"is_enabled"`
}

func (r upsertUserNotificationSettingRequest) Validate() error {
	if r.IsEnabled == nil {
		return errors.New("is_enabled is required")
	}

	return nil
}

func UpsertUserNotificationSetting(db *sql.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)

		stringID := r.PathValue("user_id")
		userID, err := strconv.ParseInt(stringID, 10, 64)
		if err != nil {
			logger.InfoContext(r.Context(), "failed to parse user id for user notification setting upsert", "user_id", stringID, "error", err)
			http.Error(w, "user_id must be a positive integer", http.StatusBadRequest)
			return
		}
		if userID <= 0 {
			logger.InfoContext(r.Context(), "invalid user id for user notification setting upsert", "user_id", userID)
			http.Error(w, "user_id must be a positive integer", http.StatusBadRequest)
			return
		}

		channel := model.Channel(r.PathValue("channel"))
		switch channel {
		case model.ChannelEmail, model.ChannelSlack:
		default:
			logger.InfoContext(r.Context(), "invalid channel for user notification setting upsert", "user_id", userID, "channel", channel)
			http.Error(w, "channel must be slack or email", http.StatusBadRequest)
			return
		}

		var req upsertUserNotificationSettingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.InfoContext(r.Context(), "failed to decode upsert user notification setting request", "user_id", userID, "channel", channel, "error", err)
			http.Error(w, "request body must be valid JSON", http.StatusBadRequest)
			return
		}
		if err := req.Validate(); err != nil {
			logger.InfoContext(r.Context(), "upsert user notification setting request validation failed", "user_id", userID, "channel", channel, "error", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		query := `insert into user_notification_setting (user_id, channel, is_enabled) values (?, ?, ?) on duplicate key update is_enabled = values(is_enabled)`
		result, err := db.ExecContext(r.Context(), query, userID, channel, *req.IsEnabled)
		if err != nil {
			var mysqlErr *mysql.MySQLError
			if errors.As(err, &mysqlErr) && mysqlErr.Number == 1452 {
				logger.InfoContext(r.Context(), "user not found for user notification setting upsert", "user_id", userID, "channel", channel)
				http.Error(w, "user not found", http.StatusNotFound)
				return
			}
			logger.ErrorContext(r.Context(), "failed to upsert user notification setting", "user_id", userID, "channel", channel, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		rows, err := result.RowsAffected()
		if err != nil {
			logger.ErrorContext(r.Context(), "failed to read affected rows after upserting user notification setting", "user_id", userID, "channel", channel, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		res := upsertUserNotificationSettingResponse{
			UserID:    userID,
			Channel:   channel,
			IsEnabled: *req.IsEnabled,
		}

		w.Header().Set("Content-Type", "application/json")
		if rows == 1 {
			w.WriteHeader(http.StatusCreated)
		} else {
			w.WriteHeader(http.StatusOK)
		}

		if err := json.NewEncoder(w).Encode(res); err != nil {
			logger.ErrorContext(r.Context(), "failed to encode upsert user notification setting response", "user_id", userID, "channel", channel, "error", err)
		}
	}
}
