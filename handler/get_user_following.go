package handler

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

type getUserFollowingResponse struct {
	FollowerID  int64     `json:"follower_id"`
	FollowingID int64     `json:"following_id"`
	CreatedAt   time.Time `json:"created_at"`
}

func GetUserFollowing(db *sql.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stringUserID := r.PathValue("user_id")
		userID, err := strconv.ParseInt(stringUserID, 10, 64)
		if err != nil || userID <= 0 {
			logger.InfoContext(r.Context(), "invalid user id for following retrieval", "user_id", stringUserID)
			http.Error(w, "user_id must be a positive integer", http.StatusBadRequest)
			return
		}

		query := `select follower_id, following_id, created_at from follows where follower_id = ? order by created_at desc, following_id desc`
		rows, err := db.QueryContext(r.Context(), query, userID)
		if err != nil {
			logger.ErrorContext(r.Context(), "failed to retrieve users followed by user", "user_id", userID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		defer func() {
			if cerr := rows.Close(); cerr != nil {
				logger.ErrorContext(r.Context(), "failed to close user following rows", "user_id", userID, "error", cerr)
			}
		}()

		userFollowing := make([]getUserFollowingResponse, 0)
		for rows.Next() {
			var followedUser getUserFollowingResponse
			err = rows.Scan(
				&followedUser.FollowerID,
				&followedUser.FollowingID,
				&followedUser.CreatedAt,
			)
			if err != nil {
				logger.ErrorContext(r.Context(), "failed to scan followed user", "user_id", userID, "error", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			userFollowing = append(userFollowing, followedUser)
		}
		if err := rows.Err(); err != nil {
			logger.ErrorContext(r.Context(), "failed while iterating over followed users", "user_id", userID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(userFollowing); err != nil {
			logger.ErrorContext(r.Context(), "failed to encode followed users response", "user_id", userID, "error", err)
		}
	}
}
