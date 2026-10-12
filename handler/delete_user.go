package handler

import (
	"database/sql"
	"jev/middleware"
	"log/slog"
	"net/http"
)

func DeleteUser(db *sql.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.AuthenticatedUserID(r.Context())
		if !ok {
			logger.ErrorContext(r.Context(), "authenticated user id is missing from request context")
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		query := `delete from users where id = ?`

		result, err := db.ExecContext(r.Context(), query, userID)
		if err != nil {
			logger.ErrorContext(r.Context(), "failed to delete user", "user_id", userID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		count, err := result.RowsAffected()
		if err != nil {
			logger.ErrorContext(r.Context(), "failed to read affected rows after deleting user", "user_id", userID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		if count == 0 {
			logger.InfoContext(r.Context(), "user not found for deletion", "user_id", userID)
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
