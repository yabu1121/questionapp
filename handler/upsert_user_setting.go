package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"jev/model"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

type upsertUserSettingRequest struct {
	UIMode   model.UIMode `json:"ui_mode"`
	Timezone string       `json:"timezone"`
}

type upsertUserSettingResponse struct {
	UserID   int64        `json:"user_id"`
	UIMode   model.UIMode `json:"ui_mode"`
	Timezone string       `json:"timezone"`
}

func (r *upsertUserSettingRequest) Normalize() {
	r.Timezone = strings.TrimSpace(r.Timezone)
	if r.UIMode == "" {
		r.UIMode = model.UIModeLight
	}

	if r.Timezone == "" {
		r.Timezone = "Asia/Tokyo"
	}
}

func (r upsertUserSettingRequest) Validate() error {
	switch r.UIMode {
	case model.UIModeDark, model.UIModeLight, model.UIModeSystem:
	default:
		return errors.New("ui_mode must be light, dark, or system")
	}

	if len(r.Timezone) > 64 {
		return errors.New("timezone must be 64 characters or less")
	}

	if _, err := time.LoadLocation(r.Timezone); err != nil {
		return errors.New("timezone must be a valid IANA timezone")
	}

	return nil
}

func UpsertUserSetting(db *sql.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stringUserID := r.PathValue("user_id")
		userID, err := strconv.ParseInt(stringUserID, 10, 64)
		if err != nil {
			logger.InfoContext(r.Context(), "failed to parse user id for user setting upsert", "user_id", stringUserID, "error", err)
			http.Error(w, "user_id must be a positive integer", http.StatusBadRequest)
			return
		}
		if userID <= 0 {
			logger.InfoContext(r.Context(), "invalid user id for user setting upsert", "user_id", userID)
			http.Error(w, "user_id must be a positive integer", http.StatusBadRequest)
			return
		}

		var req upsertUserSettingRequest
		r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.InfoContext(r.Context(), "failed to decode upsert user setting request", "user_id", userID, "error", err)
			http.Error(w, "request body must be valid JSON", http.StatusBadRequest)
			return
		}

		req.Normalize()
		if err := req.Validate(); err != nil {
			logger.InfoContext(r.Context(), "upsert user setting request validation failed", "user_id", userID, "error", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		query := `insert into user_setting (user_id, ui_mode, timezone) values (?, ?, ?) on duplicate key update ui_mode = values(ui_mode), timezone = values(timezone)`
		result, err := db.ExecContext(r.Context(), query, userID, req.UIMode, req.Timezone)
		if err != nil {
			var mysqlErr *mysql.MySQLError
			if errors.As(err, &mysqlErr) {
				switch mysqlErr.Number {
				case 1452:
					logger.InfoContext(r.Context(), "user not found for user setting upsert", "user_id", userID)
					http.Error(w, "user not found", http.StatusNotFound)
					return
				}
			}
			logger.ErrorContext(r.Context(), "failed to upsert user setting", "user_id", userID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		res := upsertUserSettingResponse{
			UserID:   userID,
			UIMode:   req.UIMode,
			Timezone: req.Timezone,
		}

		rows, err := result.RowsAffected()
		if err != nil {
			logger.ErrorContext(r.Context(), "failed to read affected rows after upserting user setting", "user_id", userID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if rows == 1 {
			w.WriteHeader(http.StatusCreated)
		} else {
			w.WriteHeader(http.StatusOK)
		}

		if err := json.NewEncoder(w).Encode(res); err != nil {
			logger.ErrorContext(r.Context(), "failed to encode upsert user setting response", "user_id", userID, "error", err)
		}
	}
}
