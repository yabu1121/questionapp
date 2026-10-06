package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-sql-driver/mysql"
)

type createQuestionnaireLikeRequest struct {
	UserID int64 `json:"user_id"`
}

type createQuestionnaireLikeResponse struct {
	QuestionnaireID int64 `json:"questionnaire_id"`
	UserID          int64 `json:"user_id"`
}

func (r createQuestionnaireLikeRequest) Validate() error {
	if r.UserID <= 0 {
		return errors.New("user id must be a positive integer")
	}
	return nil
}

func CreateQuestionnaireLike(db *sql.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
		var req createQuestionnaireLikeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.InfoContext(r.Context(), "failed to decode create questionnaire like", "error", err)
			http.Error(w, "request body must be valid JSON", http.StatusBadRequest)
			return
		}

		if err := req.Validate(); err != nil {
			logger.InfoContext(r.Context(), "create questionnaire like request validation failed", "error", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
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
				logger.InfoContext(r.Context(), "questionnaire not found for like creation", "questionnaire_id", questionnaireID, "user_id", req.UserID)
				http.Error(w, "questionnaire not found", http.StatusNotFound)
				return
			}
			logger.ErrorContext(r.Context(), "failed to read questionnaire for like creation", "questionnaire_id", questionnaireID, "user_id", req.UserID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		insertQuery := `insert into questionnaire_like (questionnaire_id, user_id) values (?, ?)`
		_, err = db.ExecContext(r.Context(), insertQuery, questionnaireID, req.UserID)
		if err != nil {
			var mysqlErr *mysql.MySQLError

			if errors.As(err, &mysqlErr) {
				switch mysqlErr.Number {
				case 1062:
					logger.InfoContext(r.Context(), "questionnaire is already liked by user", "questionnaire_id", questionnaireID, "user_id", req.UserID)
					http.Error(w, "questionnaire is already liked", http.StatusConflict)
					return
				case 1452:
					logger.InfoContext(r.Context(), "user or questionnaire not found for like creation", "questionnaire_id", questionnaireID, "user_id", req.UserID)
					http.Error(w, "user or questionnaire not found", http.StatusNotFound)
					return
				}
			}
			logger.ErrorContext(r.Context(), "failed to create questionnaire like", "questionnaire_id", questionnaireID, "user_id", req.UserID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		res := createQuestionnaireLikeResponse{
			QuestionnaireID: questionnaireID,
			UserID:          req.UserID,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(res); err != nil {
			logger.ErrorContext(r.Context(), "failed to encode create questionnaire like response", "questionnaire_id", questionnaireID, "user_id", req.UserID, "error", err)
		}
	}
}
