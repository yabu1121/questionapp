package handler

import (
	"database/sql"
	"log/slog"
	"net/http"
	"strconv"
)

func DeleteFollow(db *sql.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stringFollowerID := r.PathValue("follower_id")
		stringFollowingID := r.PathValue("following_id")

		followerID, err := strconv.ParseInt(stringFollowerID, 10, 64)
		if err != nil || followerID <= 0 {
			logger.InfoContext(r.Context(), "invalid follower id for follow deletion", "follower_id", stringFollowerID)
			http.Error(w, "follower_id must be a positive integer", http.StatusBadRequest)
			return
		}
		followingID, err := strconv.ParseInt(stringFollowingID, 10, 64)
		if err != nil || followingID <= 0 {
			logger.InfoContext(r.Context(), "invalid following id for follow deletion", "following_id", stringFollowingID)
			http.Error(w, "following_id must be a positive integer", http.StatusBadRequest)
			return
		}

		query := `delete from follows where follower_id = ? and following_id = ?`

		result, err := db.ExecContext(r.Context(), query, followerID, followingID)
		if err != nil {
			logger.ErrorContext(r.Context(), "failed to delete follow", "follower_id", followerID, "following_id", followingID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		count, err := result.RowsAffected()
		if err != nil {
			logger.ErrorContext(r.Context(), "failed to read affected rows after deleting follow", "follower_id", followerID, "following_id", followingID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		if count == 0 {
			logger.InfoContext(r.Context(), "follow not found for deletion", "follower_id", followerID, "following_id", followingID)
			http.Error(w, "follow not found", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
