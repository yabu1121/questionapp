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

type createUserSettingRequest struct {
	UIMode   model.UIMode `json:"ui_mode"`
	Timezone string       `json:"timezone"`
}

type createUserSettingResponse struct {
	UserID   int64        `json:"user_id"`
	UIMode   model.UIMode `json:"ui_mode"`
	Timezone string       `json:"timezone"`
}

func (r *createUserSettingRequest) Normalize() {
	r.Timezone = strings.TrimSpace(r.Timezone)
	if r.UIMode == "" {
		r.UIMode = model.UIModeLight
	}

	if r.Timezone == "" {
		r.Timezone = "Asia/Tokyo"
	}
}

func (r createUserSettingRequest) Validate() error {
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

func CreateUserSetting(db *sql.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stringUserID := r.PathValue("user_id")
		userID, err := strconv.ParseInt(stringUserID, 10, 64)
		if err != nil {
			logger.InfoContext(r.Context(), "failed to parse user id for user setting creation", "user_id", stringUserID, "error", err)
			http.Error(w, "user_id must be a positive integer", http.StatusBadRequest)
			return
		}
		if userID <= 0 {
			logger.InfoContext(r.Context(), "invalid user id for user setting creation", "user_id", userID)
			http.Error(w, "user_id must be a positive integer", http.StatusBadRequest)
			return
		}

		var req createUserSettingRequest
		r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.InfoContext(r.Context(), "failed to decode create user setting request", "user_id", userID, "error", err)
			http.Error(w, "request body must be valid JSON", http.StatusBadRequest)
			return
		}

		req.Normalize()
		if err := req.Validate(); err != nil {
			logger.InfoContext(r.Context(), "create user setting request validation failed", "user_id", userID, "error", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		query := `insert into user_setting (user_id, ui_mode, timezone) values (?, ?, ?)`
		_, err = db.ExecContext(r.Context(), query, userID, req.UIMode, req.Timezone)
		if err != nil {
			var mysqlErr *mysql.MySQLError
			if errors.As(err, &mysqlErr) {
				switch mysqlErr.Number {
				case 1062:
					logger.InfoContext(r.Context(), "create user setting conflict", "user_id", userID)
					http.Error(w, "user setting already exists", http.StatusConflict)
					return
				case 1452:
					logger.InfoContext(r.Context(), "user not found for user setting creation", "user_id", userID)
					http.Error(w, "user not found", http.StatusNotFound)
					return
				}
			}
			logger.ErrorContext(r.Context(), "failed to create user setting", "user_id", userID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		res := createUserSettingResponse{
			UserID:   userID,
			UIMode:   req.UIMode,
			Timezone: req.Timezone,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(res); err != nil {
			logger.ErrorContext(r.Context(), "failed to encode create user setting response", "user_id", userID, "error", err)
		}
	}
}
