package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/go-sql-driver/mysql"
)

type createLanguageRequest struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	NativeName  string `json:"native_name"`
	Description string `json:"description"`
}
type createLanguageResponse struct {
	ID          int64  `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	NativeName  string `json:"native_name"`
	Description string `json:"description"`
}

func (r createLanguageRequest) Validate() error {
	if r.Code == "" {
		return errors.New("code is required")
	}
	if utf8.RuneCountInString(r.Code) > 20 {
		return errors.New("code must be 20 characters or less")
	}

	if r.Name == "" {
		return errors.New("name is required")
	}
	if utf8.RuneCountInString(r.Name) > 255 {
		return errors.New("name must be 255 characters or less")
	}

	if r.NativeName == "" {
		return errors.New("native name is required")
	}
	if utf8.RuneCountInString(r.NativeName) > 255 {
		return errors.New("native name must be 255 characters or less")
	}

	if r.Description == "" {
		return errors.New("description is required")
	}
	if utf8.RuneCountInString(r.Description) > 255 {
		return errors.New("description must be 255 characters or less")
	}

	return nil
}

func (r *createLanguageRequest) Normalize() {
	r.Code = strings.TrimSpace(r.Code)
	r.Name = strings.TrimSpace(r.Name)
	r.NativeName = strings.TrimSpace(r.NativeName)
	r.Description = strings.TrimSpace(r.Description)
}

func CreateLanguage(db *sql.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
		var req createLanguageRequest


		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.InfoContext(r.Context(), "failed to decode create language request", "error", err)
			http.Error(w, "request must be valid JSON", http.StatusBadRequest)
			return
		}

		req.Normalize()

		if err := req.Validate(); err != nil {
			logger.InfoContext(r.Context(), "failed to validate request", "error", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		query := `insert into language (code, name, native_name, description) values (?, ?, ?, ?)`
		result, err := db.ExecContext(r.Context(), query, req.Code, req.Name, req.NativeName, req.Description)
		if err != nil {
			var mysqlErr *mysql.MySQLError

			if errors.As(err, &mysqlErr) {
				switch mysqlErr.Number {
				case 1062:
					logger.InfoContext(r.Context(), "create language conflict", "code", req.Code)
					http.Error(w, "language code already exists", http.StatusConflict)
					return
				}
			}
			logger.ErrorContext(r.Context(), "failed to create language", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		id, err := result.LastInsertId()
		if err != nil {
			logger.ErrorContext(r.Context(), "failed to read created language id", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		res := createLanguageResponse{
			ID:          id,
			Code:        req.Code,
			Name:        req.Name,
			NativeName:  req.NativeName,
			Description: req.Description,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(res); err != nil {
			logger.ErrorContext(r.Context(), "failed to encode create language response", "error", err)
		}
	}
}
