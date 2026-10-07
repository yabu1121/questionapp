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
)

type patchQuestionnaireStatusRequest struct {
	Status string `json:"status"`
}

func (r *patchQuestionnaireStatusRequest) Normalize() {
	r.Status = strings.TrimSpace(r.Status)
}

func (r patchQuestionnaireStatusRequest) Validate() error {
	if r.Status == "" {
		return errors.New("status must not be empty")
	}
	if len(r.Status) > 20 {
		return errors.New("status must be 20 characters or less")
	}
	return nil
}

func PatchQuestionnaireStatus(db *sql.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, config.MaxPatchQuestionnaireStatusRequestBodyBytes)

		stringQuestionnaireID := r.PathValue("questionnaire_id")
		stringUserID := r.PathValue("user_id")

		questionnaireID, err := strconv.ParseInt(stringQuestionnaireID, 10, 64)
		if err != nil || questionnaireID <= 0 {
			logger.InfoContext(r.Context(), "invalid questionnaire id for status update", "questionnaire_id", stringQuestionnaireID)
			http.Error(w, "questionnaire_id must be a positive integer", http.StatusBadRequest)
			return
		}

		userID, err := strconv.ParseInt(stringUserID, 10, 64)
		if err != nil || userID <= 0 {
			logger.InfoContext(r.Context(), "invalid user id for questionnaire status update", "user_id", stringUserID, "questionnaire_id", questionnaireID)
			http.Error(w, "user_id must be a positive integer", http.StatusBadRequest)
			return
		}

		var req patchQuestionnaireStatusRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.InfoContext(r.Context(), "failed to decode questionnaire status update request", "user_id", userID, "questionnaire_id", questionnaireID, "error", err)
			http.Error(w, "request body must be valid JSON", http.StatusBadRequest)
			return
		}

		req.Normalize()

		if err := req.Validate(); err != nil {
			logger.InfoContext(r.Context(), "questionnaire status update request validation failed", "user_id", userID, "questionnaire_id", questionnaireID, "error", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		query := `
			select status
			from questionnaire
			where id = ? and created_by = ?
			limit 1
		`

		var currentStatus model.QuestionnaireStatus
		if err := db.QueryRowContext(r.Context(), query, questionnaireID, userID).Scan(&currentStatus); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				logger.InfoContext(r.Context(), "questionnaire not found for status update", "user_id", userID, "questionnaire_id", questionnaireID)
				http.Error(w, "questionnaire not found", http.StatusNotFound)
				return
			}
			logger.ErrorContext(r.Context(), "failed to retrieve questionnaire status before update", "user_id", userID, "questionnaire_id", questionnaireID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		nextStatus, err := model.ParseQuestionnaireStatus(req.Status)
		if err != nil {
			logger.InfoContext(r.Context(), "invalid questionnaire status for update", "user_id", userID, "questionnaire_id", questionnaireID, "status", req.Status)
			http.Error(w, "status must be draft, published, or closed", http.StatusBadRequest)
			return
		}

		if !model.CheckStatusIsValid(currentStatus, nextStatus) {
			logger.InfoContext(r.Context(), "invalid questionnaire status transition", "user_id", userID, "questionnaire_id", questionnaireID, "current_status", currentStatus, "next_status", nextStatus)
			http.Error(w, "questionnaire status transition is not allowed", http.StatusConflict)
			return
		}

		updateQuery := `
			update questionnaire
			set
				status = ?
			where id = ? and created_by = ? and status = ?
		`

		result, err := db.ExecContext(r.Context(), updateQuery, nextStatus, questionnaireID, userID, currentStatus)
		if err != nil {
			logger.ErrorContext(r.Context(), "failed to update questionnaire status", "user_id", userID, "questionnaire_id", questionnaireID, "current_status", currentStatus, "next_status", nextStatus, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		count, err := result.RowsAffected()
		if err != nil {
			logger.ErrorContext(r.Context(), "failed to read affected rows after updating questionnaire status", "user_id", userID, "questionnaire_id", questionnaireID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		if count == 0 {
			var exists bool
			existsQuery := `select exists(select 1 from questionnaire where id = ? and created_by = ?)`
			if err := db.QueryRowContext(r.Context(), existsQuery, questionnaireID, userID).Scan(&exists); err != nil {
				logger.ErrorContext(r.Context(), "failed to check questionnaire existence after status update", "user_id", userID, "questionnaire_id", questionnaireID, "error", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			if !exists {
				logger.InfoContext(r.Context(), "questionnaire not found after status update", "user_id", userID, "questionnaire_id", questionnaireID)
				http.Error(w, "questionnaire not found", http.StatusNotFound)
				return
			}
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
