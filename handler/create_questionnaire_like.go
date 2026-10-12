package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"jev/config"
	"jev/middleware"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-sql-driver/mysql"
)

type createQuestionnaireLikeResponse struct {
	QuestionnaireID int64 `json:"questionnaire_id"`
	UserID          int64 `json:"user_id"`
}

func CreateQuestionnaireLike(db *sql.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, config.MaxCreateQuestionnaireLikeRequestBodyBytes)

		userID, ok := middleware.AuthenticatedUserID(r.Context())
		if !ok {
			logger.ErrorContext(r.Context(), "authenticated user id is missing from request context")
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		questionnaireID, err := strconv.ParseInt(r.PathValue("questionnaire_id"), 10, 64)
		if err != nil || questionnaireID <= 0 {
			logger.InfoContext(r.Context(), "failed to parse questionnaire id for like creation", "questionnaire_id", r.PathValue("questionnaire_id"), "error", err)
			http.Error(w, "questionnaire_id must be a positive integer", http.StatusBadRequest)
			return
		}

		getQuestionnaireQuery := `select id from questionnaire where id = ? and status in ('published', 'closed')`

		var foundQuestionnaireID int64
		err = db.QueryRowContext(r.Context(), getQuestionnaireQuery, questionnaireID).Scan(&foundQuestionnaireID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				logger.InfoContext(r.Context(), "questionnaire not found for like creation", "questionnaire_id", questionnaireID, "user_id", userID)
				http.Error(w, "questionnaire not found", http.StatusNotFound)
				return
			}
			logger.ErrorContext(r.Context(), "failed to read questionnaire for like creation", "questionnaire_id", questionnaireID, "user_id", userID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		insertQuery := `insert into questionnaire_like (questionnaire_id, user_id) values (?, ?)`
		_, err = db.ExecContext(r.Context(), insertQuery, questionnaireID, userID)
		if err != nil {
			var mysqlErr *mysql.MySQLError

			if errors.As(err, &mysqlErr) {
				switch mysqlErr.Number {
				case 1062:
					logger.InfoContext(r.Context(), "questionnaire is already liked by user", "questionnaire_id", questionnaireID, "user_id", userID)
					http.Error(w, "questionnaire is already liked", http.StatusConflict)
					return
				case 1452:
					logger.InfoContext(r.Context(), "user or questionnaire not found for like creation", "questionnaire_id", questionnaireID, "user_id", userID)
					http.Error(w, "user or questionnaire not found", http.StatusNotFound)
					return
				}
			}
			logger.ErrorContext(r.Context(), "failed to create questionnaire like", "questionnaire_id", questionnaireID, "user_id", userID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		res := createQuestionnaireLikeResponse{
			QuestionnaireID: questionnaireID,
			UserID:          userID,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(res); err != nil {
			logger.ErrorContext(r.Context(), "failed to encode create questionnaire like response", "questionnaire_id", questionnaireID, "user_id", userID, "error", err)
		}
	}
}
