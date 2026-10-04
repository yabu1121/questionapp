package handler

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

type getQuestionnaireVoteResponse struct {
	VoteID          int64                     `json:"vote_id"`
	QuestionnaireID int64                     `json:"questionnaire_id"`
	UserID          int64                     `json:"user_id"`
	CreatedAt       time.Time                 `json:"created_at"`
	Choices         []questionnaireVoteChoice `json:"choices"`
}

type questionnaireVoteChoice struct {
	ChoiceID int64  `json:"choice_id"`
	Title    string `json:"title"`
}

func GetQuestionnaireVote(db *sql.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stringQuestionnaireID := r.PathValue("questionnaire_id")
		stringUserID := r.PathValue("user_id")

		questionnaireID, err := strconv.ParseInt(stringQuestionnaireID, 10, 64)
		if err != nil || questionnaireID <= 0 {
			logger.InfoContext(r.Context(), "invalid questionnaire id for questionnaire vote retrieval", "questionnaire_id", stringQuestionnaireID)
			http.Error(w, "questionnaire_id must be a positive integer", http.StatusBadRequest)
			return
		}

		userID, err := strconv.ParseInt(stringUserID, 10, 64)
		if err != nil || userID <= 0 {
			logger.InfoContext(r.Context(), "invalid user id for questionnaire vote retrieval", "user_id", stringUserID)
			http.Error(w, "user_id must be a positive integer", http.StatusBadRequest)
			return
		}

		query := `
		select vote.id, vote.questionnaire_id, vote.user_id, vote.created_at, choice.id, choice.title
		from answer_item
		join vote on answer_item.vote_id = vote.id
		join choice on answer_item.choice_id = choice.id
		where vote.questionnaire_id = ? and vote.user_id = ?
		order by choice.display_order, choice.id;
		`
		rows, err := db.QueryContext(r.Context(), query, questionnaireID, userID)
		if err != nil {
			logger.ErrorContext(r.Context(), "failed to retrieve questionnaire vote", "questionnaire_id", questionnaireID, "user_id", userID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		response := getQuestionnaireVoteResponse{
			Choices: make([]questionnaireVoteChoice, 0),
		}

		found := false
		for rows.Next() {
			var vote getQuestionnaireVoteResponse
			var choice questionnaireVoteChoice
			err = rows.Scan(
				&vote.VoteID,
				&vote.QuestionnaireID,
				&vote.UserID,
				&vote.CreatedAt,
				&choice.ChoiceID,
				&choice.Title,
			)
			if err != nil {
				logger.ErrorContext(r.Context(), "failed to scan questionnaire vote", "questionnaire_id", questionnaireID, "user_id", userID, "error", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			if !found {
				response.VoteID = vote.VoteID
				response.QuestionnaireID = vote.QuestionnaireID
				response.UserID = vote.UserID
				response.CreatedAt = vote.CreatedAt
				found = true
			}

			response.Choices = append(response.Choices, choice)
		}

		if err := rows.Err(); err != nil {
			logger.ErrorContext(r.Context(), "failed while iterating over questionnaire vote", "questionnaire_id", questionnaireID, "user_id", userID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		if !found {
			logger.InfoContext(r.Context(), "questionnaire vote not found", "questionnaire_id", questionnaireID, "user_id", userID)
			http.Error(w, "questionnaire vote not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(response); err != nil {
			logger.ErrorContext(r.Context(), "failed to encode questionnaire vote response", "questionnaire_id", questionnaireID, "user_id", userID, "error", err)
		}
	}
}
