package handler

import (
	"database/sql"
	"encoding/json"
	"jev/model"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type questionnaireListItem struct {
	ID               int64                     `json:"id"`
	CreatedBy        int64                     `json:"created_by"`
	Title            string                    `json:"title"`
	Description      string                    `json:"description"`
	Status           model.QuestionnaireStatus `json:"status"`
	Deadline         time.Time                 `json:"deadline"`
	Type             model.QuestionType        `json:"type"`
	MaxChoices       int                       `json:"max_choices"`
	LanguageCode     model.LanguageCode        `json:"language_code"`
	LanguageDetected bool                      `json:"language_detected"`
	Visibility       model.ResultVisibility    `json:"visibility"`
	CreatedAt        time.Time                 `json:"created_at"`
	UpdatedAt        *time.Time                `json:"updated_at"`
}

type questionnaireListChoice struct {
	ID              int64  `json:"id"`
	QuestionnaireID int64  `json:"questionnaire_id"`
	Title           string `json:"title"`
	DisplayOrder    int    `json:"display_order"`
}

type getQuestionnaireListResponse struct {
	Questionnaire questionnaireListItem     `json:"questionnaire"`
	Choices       []questionnaireListChoice `json:"choices"`
}

func GetQuestionnaires(db *sql.DB, logger *slog.Logger) http.HandlerFunc {
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

		query := `
		select id, created_by, title, description, status, deadline, type, max_choices, language_code, language_detected, visibility, created_at, updated_at
		from questionnaire
		where status = 'published'
		order by created_at desc, id desc
		limit ? offset ?
		`

		rows, err := db.QueryContext(r.Context(), query, limit, offset)
		if err != nil {
			logger.ErrorContext(r.Context(), "failed to retrieve questionnaires", "limit", limit, "offset", offset, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		defer func() {
			if cerr := rows.Close(); cerr != nil {
				logger.ErrorContext(r.Context(), "failed to close questionnaire rows", "limit", limit, "offset", offset, "error", cerr)
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
				logger.ErrorContext(r.Context(), "failed to scan questionnaire", "limit", limit, "offset", offset, "error", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			questionnaires = append(questionnaires, questionnaire)
			questionnaireIDs = append(questionnaireIDs, questionnaire.ID)
		}
		if err := rows.Err(); err != nil {
			logger.ErrorContext(r.Context(), "failed while iterating over questionnaires", "limit", limit, "offset", offset, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		if len(questionnaireIDs) == 0 {
			responses := make([]getQuestionnaireListResponse, 0)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(responses); err != nil {
				logger.ErrorContext(r.Context(), "failed to encode empty questionnaire list response", "limit", limit, "offset", offset, "error", err)
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
			logger.ErrorContext(r.Context(), "failed to retrieve questionnaire choices", "questionnaire_ids", questionnaireIDs, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		defer func() {
			if cerr := choiceRows.Close(); cerr != nil {
				logger.ErrorContext(r.Context(), "failed to close questionnaire choice rows", "questionnaire_ids", questionnaireIDs, "error", cerr)
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
				logger.ErrorContext(r.Context(), "failed to scan questionnaire choice", "questionnaire_ids", questionnaireIDs, "error", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			choicesByQuestionnaireID[choice.QuestionnaireID] = append(choicesByQuestionnaireID[choice.QuestionnaireID], choice)
		}
		if err := choiceRows.Err(); err != nil {
			logger.ErrorContext(r.Context(), "failed while iterating over questionnaire choices", "questionnaire_ids", questionnaireIDs, "error", err)
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
			logger.ErrorContext(r.Context(), "failed to encode questionnaire list response", "limit", limit, "offset", offset, "error", err)
		}
	}
}
