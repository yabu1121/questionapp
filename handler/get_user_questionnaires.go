package handler

import (
	"database/sql"
	"encoding/json"
	"jev/model"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
)

func GetUserQuestionnaires(db *sql.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stringLimit := r.URL.Query().Get("limit")
		stringOffset := r.URL.Query().Get("offset")

		if stringLimit == "" {
			stringLimit = "20"
		}

		if stringOffset == "" {
			stringOffset = "0"
		}

		limit, err := strconv.Atoi(stringLimit)
		if err != nil {
			logger.InfoContext(r.Context(), "failed to parse questionnaire list limit", "limit", stringLimit, "error", err)
			http.Error(w, "limit must be an integer", http.StatusBadRequest)
			return
		}
		if limit < 1 || limit > 100 {
			logger.InfoContext(r.Context(), "invalid questionnaire list limit", "limit", limit)
			http.Error(w, "limit must be between 1 and 100", http.StatusBadRequest)
			return
		}

		offset, err := strconv.Atoi(stringOffset)
		if err != nil {
			logger.InfoContext(r.Context(), "failed to parse questionnaire list offset", "offset", stringOffset, "error", err)
			http.Error(w, "offset must be an integer", http.StatusBadRequest)
			return
		}
		if offset < 0 {
			logger.InfoContext(r.Context(), "invalid questionnaire list offset", "offset", offset)
			http.Error(w, "offset must be zero or greater", http.StatusBadRequest)
			return
		}

		stringUserID := r.PathValue("user_id")
		userID, err := strconv.ParseInt(stringUserID, 10, 64)
		if err != nil || userID <= 0 {
			logger.InfoContext(r.Context(), "invalid user id for questionnaire list retrieval", "user_id", stringUserID)
			http.Error(w, "user_id must be a positive integer", http.StatusBadRequest)
			return
		}

		stringStatus := r.URL.Query().Get("status")

		if stringStatus == "" {
			stringStatus = "published"
		}

		status, err := model.ParseQuestionnaireStatus(stringStatus)
		if err != nil {
			logger.InfoContext(r.Context(), "invalid status for user questionnaire list", "user_id", userID, "status", stringStatus, "error", err)
			http.Error(w, "status must be draft, published, or closed", http.StatusBadRequest)
			return
		}

		query := `
		select id, created_by, title, description, status, deadline, type, max_choices, language_code, language_detected, visibility, created_at, updated_at
		from questionnaire
		where status = ? and created_by = ?
		order by created_at desc, id desc
		limit ? offset ?
		`

		rows, err := db.QueryContext(r.Context(), query, status, userID, limit, offset)
		if err != nil {
			logger.ErrorContext(r.Context(), "failed to retrieve user questionnaires", "user_id", userID, "limit", limit, "offset", offset, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		defer func() {
			if cerr := rows.Close(); cerr != nil {
				logger.ErrorContext(r.Context(), "failed to close user questionnaire rows", "user_id", userID, "error", cerr)
			}
		}()

		questionnaires := make([]questionnaireListItem, 0)
		questionnaireIDs := make([]int64, 0)
		for rows.Next() {
			var questionnaire questionnaireListItem
			err = rows.Scan(
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
				logger.ErrorContext(r.Context(), "failed to scan user questionnaire", "user_id", userID, "limit", limit, "offset", offset, "error", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			questionnaires = append(questionnaires, questionnaire)
			questionnaireIDs = append(questionnaireIDs, questionnaire.ID)
		}
		if err := rows.Err(); err != nil {
			logger.ErrorContext(r.Context(), "failed while iterating over user questionnaires", "user_id", userID, "limit", limit, "offset", offset, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		if len(questionnaireIDs) == 0 {
			responses := make([]getQuestionnaireListResponse, 0)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(responses); err != nil {
				logger.ErrorContext(r.Context(), "failed to encode empty user questionnaire list response", "user_id", userID, "limit", limit, "offset", offset, "error", err)
			}
			return
		}

		placeholders := make([]string, len(questionnaireIDs))
		args := make([]any, len(questionnaireIDs))
		for i, questionnaireID := range questionnaireIDs {
			placeholders[i] = "?"
			args[i] = questionnaireID
		}
		placeholderStr := strings.Join(placeholders, ",")

		choiceQuery := `
		select id, questionnaire_id, title, display_order
		from choice
		where questionnaire_id in (` + placeholderStr + `)
		order by questionnaire_id, display_order, id
		`

		choiceRows, err := db.QueryContext(r.Context(), choiceQuery, args...)
		if err != nil {
			logger.ErrorContext(r.Context(), "failed to retrieve choices for user questionnaires", "user_id", userID, "questionnaire_ids", questionnaireIDs, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		defer func() {
			if cerr := choiceRows.Close(); cerr != nil {
				logger.ErrorContext(r.Context(), "failed to close user questionnaire choice rows", "user_id", userID, "questionnaire_ids", questionnaireIDs, "error", cerr)
			}
		}()

		choicesByQuestionnaireID := make(map[int64][]questionnaireListChoice)
		for choiceRows.Next() {
			var choice questionnaireListChoice
			err = choiceRows.Scan(
				&choice.ID,
				&choice.QuestionnaireID,
				&choice.Title,
				&choice.DisplayOrder,
			)
			if err != nil {
				logger.ErrorContext(r.Context(), "failed to scan choice for user questionnaire", "user_id", userID, "questionnaire_ids", questionnaireIDs, "error", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			choicesByQuestionnaireID[choice.QuestionnaireID] = append(choicesByQuestionnaireID[choice.QuestionnaireID], choice)
		}
		if err := choiceRows.Err(); err != nil {
			logger.ErrorContext(r.Context(), "failed while iterating over choices for user questionnaires", "user_id", userID, "questionnaire_ids", questionnaireIDs, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		responses := make([]getQuestionnaireListResponse, 0, len(questionnaires))
		for _, questionnaire := range questionnaires {
			choices := choicesByQuestionnaireID[questionnaire.ID]
			if choices == nil {
				choices = make([]questionnaireListChoice, 0)
			}
			response := getQuestionnaireListResponse{
				Questionnaire: questionnaire,
				Choices:       choices,
			}
			responses = append(responses, response)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(responses); err != nil {
			logger.ErrorContext(r.Context(), "failed to encode user questionnaire list response", "user_id", userID, "limit", limit, "offset", offset, "error", err)
		}
	}
}
