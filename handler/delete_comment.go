package handler

import (
	"database/sql"
	"log/slog"
	"net/http"
	"strconv"
)

func DeleteComment(db *sql.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stringUserID := r.PathValue("user_id")
		stringCommentID := r.PathValue("comment_id")

		userID, err := strconv.ParseInt(stringUserID, 10, 64)
		if err != nil || userID <= 0 {
			logger.InfoContext(r.Context(), "invalid user id for comment deletion", "user_id", stringUserID)
			http.Error(w, "user_id must be a positive integer", http.StatusBadRequest)
			return
		}

		commentID, err := strconv.ParseInt(stringCommentID, 10, 64)
		if err != nil || commentID <= 0 {
			logger.InfoContext(r.Context(), "invalid comment id for comment deletion", "comment_id", stringCommentID)
			http.Error(w, "comment_id must be a positive integer", http.StatusBadRequest)
			return
		}
		query := `delete from comments where id = ? and user_id = ?`

		result, err := db.ExecContext(r.Context(), query, commentID, userID)
		if err != nil {
			logger.ErrorContext(r.Context(), "failed to delete comment", "comment_id", commentID, "user_id", userID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		count, err := result.RowsAffected()
		if err != nil {
			logger.ErrorContext(r.Context(), "failed to read affected rows after deleting comment", "comment_id", commentID, "user_id", userID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		if count == 0 {
			logger.InfoContext(r.Context(), "comment not found for deletion", "comment_id", commentID, "user_id", userID)
			http.Error(w, "comment not found", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
