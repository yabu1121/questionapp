package handler

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
)

type getQuestionnaireResultsResponse struct {
	QuestionnaireID int64          `json:"questionnaire_id"`
	TotalVotes      int            `json:"total_votes"`
	Choices         []ChoiceResult `json:"choices"`
}

type ChoiceResult struct {
	ChoiceID       int64  `json:"choice_id"`
	Title          string `json:"title"`
	DisplayOrder   int    `json:"display_order"`
	SelectionCount int    `json:"selection_count"`
}

func GetQuestionnaireResults(db *sql.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stringQuestionnaireID := r.PathValue("questionnaire_id")

		questionnaireID, err := strconv.ParseInt(stringQuestionnaireID, 10, 64)
		if err != nil || questionnaireID <= 0 {
			logger.InfoContext(r.Context(), "invalid questionnaire id for questionnaire results retrieval", "questionnaire_id", stringQuestionnaireID)
			http.Error(w, "questionnaire_id must be a positive integer", http.StatusBadRequest)
			return
		}

		totalVotesQuery := `
		select count(*)
		from vote
		where questionnaire_id = ?
		`

		var totalVotes int
		if err := db.QueryRowContext(r.Context(), totalVotesQuery, questionnaireID).Scan(&totalVotes); err != nil {
			logger.ErrorContext(r.Context(), "failed to retrieve questionnaire total votes", "questionnaire_id", questionnaireID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		query := `
		select
			choice.id,
			choice.title,
			choice.display_order,
			count(answer_item.id)
		from choice
		left join answer_item on choice.id = answer_item.choice_id
		where choice.questionnaire_id = ?
		group by choice.id, choice.title, choice.display_order
		order by choice.display_order asc
		`

		rows, err := db.QueryContext(r.Context(), query, questionnaireID)
		if err != nil {
			logger.ErrorContext(r.Context(), "failed to retrieve questionnaire choice results", "questionnaire_id", questionnaireID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		defer func() {
			if cerr := rows.Close(); cerr != nil {
				logger.ErrorContext(r.Context(), "failed to close questionnaire result rows", "questionnaire_id", questionnaireID, "error", cerr)
			}
		}()

		results := make([]ChoiceResult, 0)
		for rows.Next() {
			var result ChoiceResult
			err := rows.Scan(
				&result.ChoiceID,
				&result.Title,
				&result.DisplayOrder,
				&result.SelectionCount,
			)

			if err != nil {
				logger.ErrorContext(r.Context(), "failed to scan questionnaire choice result", "questionnaire_id", questionnaireID, "error", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			results = append(results, result)
		}
		if err := rows.Err(); err != nil {
			logger.ErrorContext(r.Context(), "failed while iterating over questionnaire results", "questionnaire_id", questionnaireID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		response := getQuestionnaireResultsResponse{
			QuestionnaireID: questionnaireID,
			TotalVotes:      totalVotes,
			Choices:         results,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			logger.ErrorContext(r.Context(), "failed to encode questionnaire results response", "questionnaire_id", questionnaireID, "error", err)
		}
	}
}
