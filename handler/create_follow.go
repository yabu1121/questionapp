package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-sql-driver/mysql"
)

type createFollowRequest struct {
	FollowingID int64 `json:"following_id"`
}

type createFollowResponse struct {
	FollowerID  int64 `json:"follower_id"`
	FollowingID int64 `json:"following_id"`
}

func (r createFollowRequest) Validate() error {
	if r.FollowingID <= 0 {
		return errors.New("following id must be a positive integer")
	}

	return nil
}

func CreateFollow(db *sql.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
		followerID, err := strconv.ParseInt(r.PathValue("follower_id"), 10, 64)
		if err != nil || followerID <= 0 {
			logger.InfoContext(r.Context(), "failed to parse follower id for follow creation", "follower_id", r.PathValue("follower_id"), "error", err)
			http.Error(w, "follower_id must be a positive integer", http.StatusBadRequest)
			return
		}

		var req createFollowRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.InfoContext(r.Context(), "failed to decode create follow request", "follower_id", followerID, "error", err)
			http.Error(w, "request body must be valid JSON", http.StatusBadRequest)
			return
		}

		if err := req.Validate(); err != nil {
			logger.InfoContext(r.Context(), "create follow request validation failed", "follower_id", followerID, "following_id", req.FollowingID, "error", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if followerID == req.FollowingID {
			logger.InfoContext(r.Context(), "user attempted to follow themselves", "user_id", followerID)
			http.Error(w, "users cannot follow themselves", http.StatusBadRequest)
			return
		}

		insertQuery := `insert into follows (follower_id, following_id) values (?, ?)`
		_, err = db.ExecContext(r.Context(), insertQuery, followerID, req.FollowingID)
		if err != nil {
			var mysqlErr *mysql.MySQLError
			if errors.As(err, &mysqlErr) {
				switch mysqlErr.Number {
				case 1062:
					logger.InfoContext(r.Context(), "user is already followed", "follower_id", followerID, "following_id", req.FollowingID)
					http.Error(w, "user is already followed", http.StatusConflict)
					return
				case 1452:
					logger.InfoContext(r.Context(), "follower or following user not found", "follower_id", followerID, "following_id", req.FollowingID)
					http.Error(w, "follower or following user not found", http.StatusNotFound)
					return
				}
			}
			logger.ErrorContext(r.Context(), "failed to create follow", "follower_id", followerID, "following_id", req.FollowingID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		res := createFollowResponse{
			FollowerID:  followerID,
			FollowingID: req.FollowingID,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(res); err != nil {
			logger.ErrorContext(r.Context(), "failed to encode create follow response", "follower_id", followerID, "following_id", req.FollowingID, "error", err)
		}
	}
}
