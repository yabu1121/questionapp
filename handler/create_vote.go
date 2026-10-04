package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"jev/model"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-sql-driver/mysql"
)

const maxCreateVoteRequestBodyBytes = 1 << 20

type createVoteRequest struct {
	UserID    int64   `json:"user_id"`
	ChoiceIDs []int64 `json:"choice_ids"`
}

type createVoteResponse struct {
	ID              int64   `json:"id"`
	QuestionnaireID int64   `json:"questionnaire_id"`
	UserID          int64   `json:"user_id"`
	ChoiceIDs       []int64 `json:"choice_ids"`
}

func (r createVoteRequest) Validate() error {
	if r.UserID <= 0 {
		return errors.New("user id must be positive integer")
	}

	if len(r.ChoiceIDs) == 0 {
		return errors.New("at least one choice id is required")
	}

	seenChoiceIDs := make(map[int64]struct{}, len(r.ChoiceIDs))
	for _, choiceID := range r.ChoiceIDs {
		if choiceID <= 0 {
			return errors.New("choice ids must be positive integers")
		}
		if _, exists := seenChoiceIDs[choiceID]; exists {
			return errors.New("choice ids must not contain duplicates")
		}
		seenChoiceIDs[choiceID] = struct{}{}
	}

	return nil
}

func validateVoteRules(q model.Questionnaire, choiceIDs []int64) error {
	if q.Status != model.QuestionnaireStatusPublished {
		return errors.New("this questionnaire is not published")
	}
	if time.Now().After(q.Deadline) {
		return errors.New("this questionnaire is over deadline")
	}
	if q.Type == model.QuestionTypeSingle {
		if len(choiceIDs) != 1 {
			return errors.New("single type choice is must be 1")
		}
	} else {
		if len(choiceIDs) > q.MaxChoices {
			return errors.New("choice id count is over max choice count")
		}
	}
	return nil
}

func CreateVote(db *sql.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		questionnaireID, err := strconv.ParseInt(r.PathValue("questionnaire_id"), 10, 64)
		if err != nil || questionnaireID <= 0 {
			logger.InfoContext(r.Context(), "failed to parse questionnaire id for vote creation", "questionnaire_id", r.PathValue("questionnaire_id"), "error", err)
			http.Error(w, "questionnaire_id must be a positive integer", http.StatusBadRequest)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxCreateVoteRequestBodyBytes)

		var req createVoteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.InfoContext(r.Context(), "failed to decode create vote request", "error", err)
			http.Error(w, "request body must be valid JSON", http.StatusBadRequest)
			return
		}

		if err := req.Validate(); err != nil {
			logger.InfoContext(r.Context(), "create vote request validation failed", "questionnaire_id", questionnaireID, "user_id", req.UserID, "error", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		tx, err := db.BeginTx(r.Context(), nil)
		if err != nil {
			logger.ErrorContext(r.Context(), "failed to begin create vote transaction", "questionnaire_id", questionnaireID, "user_id", req.UserID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		defer func() {
			if rerr := tx.Rollback(); rerr != nil && !errors.Is(rerr, sql.ErrTxDone) {
				logger.ErrorContext(r.Context(), "failed to roll back create vote transaction", "questionnaire_id", questionnaireID, "user_id", req.UserID, "error", rerr)
			}
		}()

		questionnaireQuery := `select
			id,
			created_by,
			title,
			description,
			status,
			deadline,
			type,
			max_choices,
			language_code,
			language_detected,
			visibility,
			created_at,
			updated_at
		from questionnaire
		where id = ?`
		row := tx.QueryRowContext(r.Context(), questionnaireQuery, questionnaireID)

		var questionnaire model.Questionnaire
		err = row.Scan(
			&questionnaire.ID,
			&questionnaire.CreatedBy,
			&questionnaire.Title,
			&questionnaire.Description,
			&questionnaire.Status,
			&questionnaire.Deadline,
			&questionnaire.Type,
			&questionnaire.MaxChoices,
			&questionnaire.LanguageCode,
			&questionnaire.LanguageDetected,
			&questionnaire.Visibility,
			&questionnaire.CreatedAt,
			&questionnaire.UpdatedAt,
		)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				logger.InfoContext(r.Context(), "questionnaire not found for vote creation", "questionnaire_id", questionnaireID)
				http.Error(w, "questionnaire not found", http.StatusNotFound)
				return
			}
			logger.ErrorContext(r.Context(), "failed to read questionnaire for vote creation", "questionnaire_id", questionnaireID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		if err := validateVoteRules(questionnaire, req.ChoiceIDs); err != nil {
			logger.InfoContext(r.Context(), "questionnaire is not votable", "questionnaire_id", questionnaireID, "error", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		choiceQuery := `select id from choice where id = ? and questionnaire_id = ?`
		for _, choiceID := range req.ChoiceIDs {
			var foundChoiceID int64
			err := tx.QueryRowContext(r.Context(), choiceQuery, choiceID, questionnaireID).Scan(&foundChoiceID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					logger.InfoContext(r.Context(), "choice does not belong to questionnaire", "questionnaire_id", questionnaireID, "choice_id", choiceID)
					http.Error(w, "choice does not belong to questionnaire", http.StatusBadRequest)
					return
				}
				logger.ErrorContext(r.Context(), "failed to read choice for vote creation", "questionnaire_id", questionnaireID, "choice_id", choiceID, "error", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		}

		voteQuery := `insert into vote (questionnaire_id, user_id) value (?, ?)`
		voteResult, err := tx.ExecContext(r.Context(), voteQuery, questionnaireID, req.UserID)
		if err != nil {
			var mysqlErr *mysql.MySQLError
			if errors.As(err, &mysqlErr) {
				switch mysqlErr.Number {
				case 1062:
					logger.InfoContext(r.Context(), "user has already voted", "questionnaire_id", questionnaireID, "user_id", req.UserID)
					http.Error(w, "user has already voted", http.StatusConflict)
					return
				case 1452:
					logger.InfoContext(r.Context(), "user not found for vote creation", "user_id", req.UserID)
					http.Error(w, "user not found", http.StatusNotFound)
					return
				}
			}
			logger.ErrorContext(r.Context(), "failed to create vote", "questionnaire_id", questionnaireID, "user_id", req.UserID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		voteID, err := voteResult.LastInsertId()
		if err != nil {
			logger.ErrorContext(r.Context(), "failed to read created vote id", "questionnaire_id", questionnaireID, "user_id", req.UserID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		answerItemQuery := `insert into answer_item (vote_id, choice_id) values(?, ?)`
		for _, choiceID := range req.ChoiceIDs {
			_, err := tx.ExecContext(r.Context(), answerItemQuery, voteID, choiceID)
			if err != nil {
				logger.ErrorContext(r.Context(), "failed to create answer item", "vote_id", voteID, "choice_id", choiceID, "error", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		}

		if err := tx.Commit(); err != nil {
			logger.ErrorContext(r.Context(), "failed to commit create vote transaction", "questionnaire_id", questionnaireID, "user_id", req.UserID, "vote_id", voteID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		res := createVoteResponse{
			ID:              voteID,
			QuestionnaireID: questionnaireID,
			UserID:          req.UserID,
			ChoiceIDs:       req.ChoiceIDs,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(res); err != nil {
			logger.ErrorContext(r.Context(), "failed to encode create vote response", "vote_id", voteID, "error", err)
		}
	}
}
