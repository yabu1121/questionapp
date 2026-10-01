package handler

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
)

type getQuestionnaireLikesCountResponse struct {
	QuestionnaireID int64 `json:"questionnaire_id"`
	LikeCount       int64 `json:"like_count"`
}

func GetQuestionnaireLikesCount(db *sql.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stringQuestionnaireID := r.PathValue("questionnaire_id")
		questionnaireID, err := strconv.ParseInt(stringQuestionnaireID, 10, 64)
		if err != nil || questionnaireID <= 0 {
			logger.InfoContext(r.Context(), "invalid questionnaire id for like count retrieval", "questionnaire_id", stringQuestionnaireID)
			http.Error(w, "questionnaire_id must be a positive integer", http.StatusBadRequest)
			return
		}

		query := `select count(*) from questionnaire_like where questionnaire_id = ?`
		row := db.QueryRowContext(r.Context(), query, questionnaireID)

		res := getQuestionnaireLikesCountResponse{
			QuestionnaireID: questionnaireID,
		}

		if err := row.Scan(&res.LikeCount); err != nil {
			logger.ErrorContext(r.Context(), "failed to scan questionnaire like count", "questionnaire_id", questionnaireID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		if err := row.Err(); err != nil {
			logger.ErrorContext(r.Context(), "failed to retrieve questionnaire like count", "questionnaire_id", questionnaireID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(res); err != nil {
			logger.ErrorContext(r.Context(), "failed to encode questionnaire like count response", "questionnaire_id", questionnaireID, "error", err)
		}
	}
}
