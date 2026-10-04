package handler

import (
	"database/sql"
	"log/slog"
	"net/http"
	"strconv"
)

func DeleteQuestionnaire(db *sql.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stringQuestionnaireID := r.PathValue("questionnaire_id")
		stringUserID := r.PathValue("user_id")

		questionnaireID, err := strconv.ParseInt(stringQuestionnaireID, 10, 64)
		if err != nil || questionnaireID <= 0 {
			logger.InfoContext(r.Context(), "invalid questionnaire id for questionnaire deletion", "questionnaire_id", stringQuestionnaireID)
			http.Error(w, "questionnaire_id must be a positive integer", http.StatusBadRequest)
			return
		}

		userID, err := strconv.ParseInt(stringUserID, 10, 64)
		if err != nil || userID <= 0 {
			logger.InfoContext(r.Context(), "invalid user id for questionnaire deletion", "user_id", stringUserID)
			http.Error(w, "user_id must be a positive integer", http.StatusBadRequest)
			return
		}

		query := `delete from questionnaire where id = ? and created_by = ?`

		result, err := db.ExecContext(r.Context(), query, questionnaireID, userID)
		if err != nil {
			logger.ErrorContext(r.Context(), "failed to delete questionnaire", "questionnaire_id", questionnaireID, "user_id", userID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		count, err := result.RowsAffected()
		if err != nil {
			logger.ErrorContext(r.Context(), "failed to read affected rows after deleting questionnaire", "questionnaire_id", questionnaireID, "user_id", userID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		if count == 0 {
			logger.InfoContext(r.Context(), "questionnaire not found for deletion", "questionnaire_id", questionnaireID, "user_id", userID)
			http.Error(w, "questionnaire not found", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
