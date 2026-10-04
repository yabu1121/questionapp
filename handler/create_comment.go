package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
)

type createCommentRequest struct {
	UserID          int64  `json:"user_id"`
	Content         string `json:"content"`
	ParentCommentID *int64 `json:"parent_comment_id"`
}

type createCommentResponse struct {
	ID              int64  `json:"id"`
	UserID          int64  `json:"user_id"`
	Content         string `json:"content"`
	ParentCommentID *int64 `json:"parent_comment_id"`
}

func (r createCommentRequest) Validate() error {
	if r.UserID <= 0 {
		return errors.New("user id must be positive integer")
	}

	if r.Content == "" {
		return errors.New("content is required")
	}
	if len(r.Content) > 255 {
		return errors.New("content must be 255 characters or less")
	}
	if r.ParentCommentID != nil && *r.ParentCommentID <= 0 {
		return errors.New("parent comment id must be a positive integer")
	}

	return nil
}

func (r *createCommentRequest) Normalize() {
	r.Content = strings.TrimSpace(r.Content)
}

func CreateComment(db *sql.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
		var req createCommentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.InfoContext(r.Context(), "failed to decode create comment request", "error", err)
			http.Error(w, "invalide request body", http.StatusBadRequest)
			return
		}

		req.Normalize()

		if err := req.Validate(); err != nil {
			logger.InfoContext(r.Context(), "invalid json request body", "error", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		questionnaireID, err := strconv.ParseInt(r.PathValue("questionnaire_id"), 10, 64)
		if err != nil || questionnaireID <= 0 {
			logger.InfoContext(r.Context(), "failed to cast questionnaire id in create comment", "error", err)
			http.Error(w, "invalid questionnaire id", http.StatusBadRequest)
			return
		}

		tx, err := db.BeginTx(r.Context(), nil)
		if err != nil {
			logger.ErrorContext(r.Context(), "failed to begin create comment transaction", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		getQuestionnaireQuery := `select id from questionnaire where id = ?`
		var foundQuestionnaireID int64

		err = tx.QueryRowContext(r.Context(), getQuestionnaireQuery, questionnaireID).Scan(&foundQuestionnaireID)

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				logger.InfoContext(r.Context(), "questionnaire not found for comment creation", "questionnaire_id", questionnaireID)
				http.Error(w, "questionnaire not found", http.StatusNotFound)
				return
			}

			logger.ErrorContext(r.Context(), "failed to read questionnaire for comment creation", "questionnaire_id", questionnaireID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		if req.ParentCommentID != nil {
			getParentCommentQuery := `select questionnaire_id from comments where id = ?`
			var parentQuestionnaireID int64

			err := tx.QueryRowContext(r.Context(), getParentCommentQuery, *req.ParentCommentID).Scan(&parentQuestionnaireID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					logger.InfoContext(r.Context(), "parent comment not found", "parent_comment_id", *req.ParentCommentID)
					http.Error(w, "parent comment not found", http.StatusNotFound)
					return
				}

				logger.ErrorContext(r.Context(), "failed to read parent comment", "parent_comment_id", *req.ParentCommentID, "error", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			if parentQuestionnaireID != questionnaireID {
				logger.InfoContext(r.Context(), "parent comment belongs to another questionnaire", "questionnaire_id", questionnaireID, "parent_comment_id", *req.ParentCommentID, "parent_questionnaire_id", parentQuestionnaireID)
				http.Error(w, "parent comment does not belong to questionnaire", http.StatusBadRequest)
				return
			}
		}

		insertCommentQuery := `insert into comments (questionnaire_id, user_id, content, parent_comment_id) values (?, ?, ?, ?)`
		createCommentResult, err := tx.ExecContext(r.Context(), insertCommentQuery, questionnaireID, req.UserID, req.Content, req.ParentCommentID)
		if err != nil {
			logger.ErrorContext(r.Context(), "failed to exec insert create comment", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		commentID, err := createCommentResult.LastInsertId()
		if err != nil {
			logger.ErrorContext(r.Context(), "failed to get last insert created comment id", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		if err := tx.Commit(); err != nil {
			logger.ErrorContext(r.Context(), "failed to commit create comment transaction", "questionnaire_id", questionnaireID, "user_id", req.UserID, "comment_id", commentID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		res := createCommentResponse{
			ID:              commentID,
			UserID:          req.UserID,
			Content:         req.Content,
			ParentCommentID: req.ParentCommentID,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(res); err != nil {
			logger.ErrorContext(r.Context(), "failed to encode created comment", "comment_id", commentID, "error", err)
		}
	}
}
