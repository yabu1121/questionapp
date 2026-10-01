package handler

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

type getQuestionnaireLikeResponse struct {
	QuestionnaireID int64     `json:"questionnaire_id"`
	UserID          int64     `json:"user_id"`
	CreatedAt       time.Time `json:"created_at"`
}

func GetQuestionnaireLikes(db *sql.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stringQuestionnaireID := r.PathValue("questionnaire_id")
		questionnaireID, err := strconv.ParseInt(stringQuestionnaireID, 10, 64)
		if err != nil || questionnaireID <= 0 {
			logger.InfoContext(r.Context(), "invalid questionnaire id for like retrieval", "questionnaire_id", stringQuestionnaireID)
			http.Error(w, "questionnaire_id must be a positive integer", http.StatusBadRequest)
			return
		}

		query := `select questionnaire_id, user_id, created_at from questionnaire_like where questionnaire_id = ? order by created_at desc, user_id desc`
		rows, err := db.QueryContext(r.Context(), query, questionnaireID)
		if err != nil {
			logger.ErrorContext(r.Context(), "failed to retrieve questionnaire likes", "questionnaire_id", questionnaireID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		questionnaireLikes := make([]getQuestionnaireLikeResponse, 0)
		for rows.Next() {
			var questionnaireLike getQuestionnaireLikeResponse
			err = rows.Scan(
				&questionnaireLike.QuestionnaireID,
				&questionnaireLike.UserID,
				&questionnaireLike.CreatedAt,
			)
			if err != nil {
				logger.ErrorContext(r.Context(), "failed to scan questionnaire like", "questionnaire_id", questionnaireID, "error", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			questionnaireLikes = append(questionnaireLikes, questionnaireLike)
		}
		if err := rows.Err(); err != nil {
			logger.ErrorContext(r.Context(), "failed while iterating over questionnaire likes", "questionnaire_id", questionnaireID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(questionnaireLikes); err != nil {
			logger.ErrorContext(r.Context(), "failed to encode questionnaire likes response", "questionnaire_id", questionnaireID, "error", err)
		}
	}
}
