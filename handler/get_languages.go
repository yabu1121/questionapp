package handler

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
)

type getLanguageResponse struct {
	ID          int64  `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	NativeName  string `json:"native_name"`
	Description string `json:"description"`
}

func GetLanguages(db *sql.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := `select id, code, name, native_name, description from language order by id asc`
		rows, err := db.QueryContext(r.Context(), query)
		if err != nil {
			logger.ErrorContext(r.Context(), "failed to retrieve languages", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		defer func() {
			if cerr := rows.Close(); cerr != nil {
				logger.ErrorContext(r.Context(), "failed to close language rows", "error", cerr)
			}
		}()

		languages := make([]getLanguageResponse, 0)
		for rows.Next() {
			var language getLanguageResponse
			err = rows.Scan(
				&language.ID,
				&language.Code,
				&language.Name,
				&language.NativeName,
				&language.Description,
			)
			if err != nil {
				logger.ErrorContext(r.Context(), "failed to scan language", "error", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			languages = append(languages, language)
		}
		if err := rows.Err(); err != nil {
			logger.ErrorContext(r.Context(), "failed while iterating over languages", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(languages); err != nil {
			logger.ErrorContext(r.Context(), "failed to encode languages response", "error", err)
		}
	}
}
