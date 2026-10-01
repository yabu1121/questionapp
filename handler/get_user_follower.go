package handler

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

type getUserFollowerResponse struct {
	FollowerID  int64     `json:"follower_id"`
	FollowingID int64     `json:"following_id"`
	CreatedAt   time.Time `json:"created_at"`
}

func GetUserFollowers(db *sql.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stringUserID := r.PathValue("user_id")
		userID, err := strconv.ParseInt(stringUserID, 10, 64)
		if err != nil || userID <= 0 {
			logger.InfoContext(r.Context(), "invalid user id for follower retrieval", "user_id", stringUserID)
			http.Error(w, "user_id must be a positive integer", http.StatusBadRequest)
			return
		}

		query := `select follower_id, following_id, created_at from follows where following_id = ? order by created_at desc, follower_id desc`
		rows, err := db.QueryContext(r.Context(), query, userID)
		if err != nil {
			logger.ErrorContext(r.Context(), "failed to retrieve user followers", "user_id", userID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		userFollowers := make([]getUserFollowerResponse, 0)
		for rows.Next() {
			var userFollower getUserFollowerResponse
			err = rows.Scan(
				&userFollower.FollowerID,
				&userFollower.FollowingID,
				&userFollower.CreatedAt,
			)
			if err != nil {
				logger.ErrorContext(r.Context(), "failed to scan user follower", "user_id", userID, "error", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			userFollowers = append(userFollowers, userFollower)
		}
		if err := rows.Err(); err != nil {
			logger.ErrorContext(r.Context(), "failed while iterating over user followers", "user_id", userID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(userFollowers); err != nil {
			logger.ErrorContext(r.Context(), "failed to encode user followers response", "user_id", userID, "error", err)
		}
	}
}
