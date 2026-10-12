package main

import (
	"context"
	"database/sql"
	"jev/handler"
	"jev/middleware"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	_ "net/http/pprof"

	_ "github.com/go-sql-driver/mysql"
)

func GetDSN() string {
	dsn := os.Getenv("DB_DSN")
	return dsn
}

func NewDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func ConnectDB(db *sql.DB, ctx context.Context) error {
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(3 * time.Minute)

	return db.PingContext(ctx)
}

func main() {
	dsn := GetDSN()

	db, err := NewDB(dsn)

	if err != nil {
		log.Fatal("failed to open new db", err)
	}

	defer func() {
		if cerr := db.Close(); cerr != nil {
			log.Printf("failed to close database: %v", cerr)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := ConnectDB(db, ctx); err != nil {
		log.Fatal("failed to connect db", err)
	}

	logger := slog.New(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			AddSource: false,
			Level:     slog.LevelInfo,
		}),
	)

	mux := http.NewServeMux()

	mux.HandleFunc("POST /v1/auth/login", handler.Login(db, logger))

	mux.HandleFunc("GET /health", handler.Health)
	mux.HandleFunc("GET /v1/users/{user_id}", handler.GetUser(db, logger))
	mux.HandleFunc("GET /v1/languages", handler.GetLanguages(db, logger))
	mux.HandleFunc("GET /v1/questionnaires/{questionnaire_id}/comments", handler.GetQuestionnaireComments(db, logger))
	mux.HandleFunc("GET /v1/users/{user_id}/comments", handler.GetUserComments(db, logger))
	mux.HandleFunc("GET /v1/questionnaires/{questionnaire_id}/likes", handler.GetQuestionnaireLikes(db, logger))
	mux.HandleFunc("GET /v1/questionnaires/{questionnaire_id}/likes/count", handler.GetQuestionnaireLikesCount(db, logger))
	mux.HandleFunc("GET /v1/users/{user_id}/following", handler.GetUserFollowing(db, logger))
	mux.HandleFunc("GET /v1/users/{user_id}/followers", handler.GetUserFollowers(db, logger))
	mux.HandleFunc("GET /v1/questionnaires/{questionnaire_id}/choices", handler.GetQuestionnaireChoices(db, logger))
	mux.HandleFunc("GET /v1/questionnaires", handler.GetQuestionnaires(db, logger))
	mux.HandleFunc("GET /v1/users/{user_id}/questionnaires", handler.GetUserQuestionnaires(db, logger))
	mux.HandleFunc("GET /v1/questionnaires/{questionnaire_id}", handler.GetQuestionnaire(db, logger))
	mux.HandleFunc("GET /v1/questionnaires/{questionnaire_id}/results", handler.GetQuestionnaireResults(db, logger))
	mux.HandleFunc("GET /v1/questionnaires/{questionnaire_id}/votes/{user_id}", handler.GetQuestionnaireVote(db, logger))

	mux.HandleFunc("POST /v1/users", handler.CreateUser(db, logger))
	mux.HandleFunc("POST /v1/languages", handler.CreateLanguage(db, logger))

	// -- auth middleware --
	mux.HandleFunc("GET /v1/me", middleware.Authenticate(db, logger, handler.GetMe(db, logger)))
	mux.HandleFunc("GET /v1/me/likes", middleware.Authenticate(db, logger, handler.GetUserLikes(db, logger)))
	mux.HandleFunc("GET /v1/me/setting", middleware.Authenticate(db, logger, handler.GetUserSetting(db, logger)))
	mux.HandleFunc("GET /v1/me/notification-settings", middleware.Authenticate(db, logger, handler.GetUserNotificationSetting(db, logger)))

	mux.HandleFunc("POST /v1/questionnaires/{status}", middleware.Authenticate(db, logger, handler.CreateQuestionnaire(db, logger)))
	mux.HandleFunc("POST /v1/questionnaires/{questionnaire_id}/votes", middleware.Authenticate(db, logger, handler.CreateVote(db, logger)))
	mux.HandleFunc("POST /v1/questionnaires/{questionnaire_id}/comments", middleware.Authenticate(db, logger, handler.CreateComment(db, logger)))
	mux.HandleFunc("POST /v1/questionnaires/{questionnaire_id}/likes", middleware.Authenticate(db, logger, handler.CreateQuestionnaireLike(db, logger)))
	mux.HandleFunc("POST /v1/me/follows", middleware.Authenticate(db, logger, handler.CreateFollow(db, logger)))

	mux.HandleFunc("PUT /v1/me/setting", middleware.Authenticate(db, logger, handler.UpsertUserSetting(db, logger)))
	mux.HandleFunc("PUT /v1/me/notification-settings/{channel}", middleware.Authenticate(db, logger, handler.UpsertUserNotificationSetting(db, logger)))

	mux.HandleFunc("PATCH /v1/me", middleware.Authenticate(db, logger, handler.PatchUserProfile(db, logger)))
	mux.HandleFunc("PATCH /v1/me/setting", middleware.Authenticate(db, logger, handler.PatchUserSetting(db, logger)))
	mux.HandleFunc("PATCH /v1/me/questionnaires/{questionnaire_id}/status", middleware.Authenticate(db, logger, handler.PatchQuestionnaireStatus(db, logger)))

	mux.HandleFunc("DELETE /v1/questionnaires/{questionnaire_id}/likes", middleware.Authenticate(db, logger, handler.DeleteQuestionnaireLike(db, logger)))
	mux.HandleFunc("DELETE /v1/me/questionnaires/{questionnaire_id}", middleware.Authenticate(db, logger, handler.DeleteQuestionnaire(db, logger)))
	mux.HandleFunc("DELETE /v1/me/follows/{following_id}", middleware.Authenticate(db, logger, handler.DeleteFollow(db, logger)))
	mux.HandleFunc("DELETE /v1/me/comments/{comment_id}", middleware.Authenticate(db, logger, handler.DeleteComment(db, logger)))
	mux.HandleFunc("DELETE /v1/me", middleware.Authenticate(db, logger, handler.DeleteUser(db, logger)))

	log.Println("listening on http://localhost:8080")
	server := http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	go func() {
		log.Println("pprof listening on http://127.0.0.1:6060")

		if err := http.ListenAndServe("127.0.0.1:6060", nil); err != nil {
			log.Printf("pprof server stopped: %v", err)
		}
	}()

	if err := server.ListenAndServe(); err != nil {
		log.Fatal("failed to serve API: ", err)
	}
}
