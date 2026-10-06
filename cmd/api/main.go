package main

import (
	"context"
	"database/sql"
	"jev/handler"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

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
	mux.HandleFunc("GET /health", handler.Health)
	mux.HandleFunc("GET /v1/users/{user_id}", handler.GetUser(db, logger))
	mux.HandleFunc("GET /v1/languages", handler.GetLanguages(db, logger))
	mux.HandleFunc("GET /v1/users/{user_id}/setting", handler.GetUserSetting(db, logger))
	mux.HandleFunc("GET /v1/users/{user_id}/notification-settings", handler.GetUserNotificationSetting(db, logger))
	mux.HandleFunc("GET /v1/questionnaires/{questionnaire_id}/comments", handler.GetQuestionnaireComments(db, logger))
	mux.HandleFunc("GET /v1/users/{user_id}/comments", handler.GetUserComments(db, logger))
	mux.HandleFunc("GET /v1/questionnaires/{questionnaire_id}/likes", handler.GetQuestionnaireLikes(db, logger))
	mux.HandleFunc("GET /v1/questionnaires/{questionnaire_id}/likes/count", handler.GetQuestionnaireLikesCount(db, logger))
	mux.HandleFunc("GET /v1/users/{user_id}/likes", handler.GetUserLikes(db, logger))
	mux.HandleFunc("GET /v1/users/{user_id}/following", handler.GetUserFollowing(db, logger))
	mux.HandleFunc("GET /v1/users/{user_id}/followers", handler.GetUserFollowers(db, logger))
	mux.HandleFunc("GET /v1/questionnaires/{questionnaire_id}/choices", handler.GetQuestionnaireChoices(db, logger))
	mux.HandleFunc("GET /v1/questionnaires", handler.GetQuestionnaires(db, logger))
	mux.HandleFunc("GET /v1/users/{user_id}/questionnaires", handler.GetUserQuestionnaires(db, logger))
	mux.HandleFunc("GET /v1/questionnaires/{questionnaire_id}", handler.GetQuestionnaire(db, logger))
	mux.HandleFunc("GET /v1/questionnaires/{questionnaire_id}/results", handler.GetQuestionnaireResults(db, logger))
	mux.HandleFunc("GET /v1/questionnaires/{questionnaire_id}/votes/{user_id}", handler.GetQuestionnaireVote(db, logger))

	mux.HandleFunc("PUT /v1/users/{user_id}/setting", handler.UpsertUserSetting(db, logger))
	mux.HandleFunc("PUT /v1/users/{user_id}/notification-settings/{channel}", handler.UpsertUserNotificationSetting(db, logger))

	mux.HandleFunc("PATCH /v1/users/{user_id}", handler.PatchUserProfile(db, logger))
	mux.HandleFunc("PATCH /v1/users/{user_id}/setting", handler.PatchUserSetting(db, logger))
	mux.HandleFunc("PATCH /v1/users/{user_id}/questionnaires/{questionnaire_id}/status", handler.PatchQuestionnaireStatus(db, logger))

	mux.HandleFunc("POST /v1/users", handler.CreateUser(db, logger))
	mux.HandleFunc("POST /v1/languages", handler.CreateLanguage(db, logger))
	mux.HandleFunc("POST /v1/questionnaires/{status}", handler.CreateQuestionnaire(db, logger))
	mux.HandleFunc("POST /v1/questionnaires/{questionnaire_id}/votes", handler.CreateVote(db, logger))
	mux.HandleFunc("POST /v1/questionnaires/{questionnaire_id}/comments", handler.CreateComment(db, logger))
	mux.HandleFunc("POST /v1/questionnaires/{questionnaire_id}/likes", handler.CreateQuestionnaireLike(db, logger))
	mux.HandleFunc("POST /v1/users/{follower_id}/follows", handler.CreateFollow(db, logger))

	mux.HandleFunc("DELETE /v1/questionnaires/{questionnaire_id}/likes/{user_id}", handler.DeleteQuestionnaireLike(db, logger))
	mux.HandleFunc("DELETE /v1/users/{user_id}/questionnaires/{questionnaire_id}", handler.DeleteQuestionnaire(db, logger))
	mux.HandleFunc("DELETE /v1/users/{follower_id}/follows/{following_id}", handler.DeleteFollow(db, logger))
	mux.HandleFunc("DELETE /v1/users/{user_id}/comments/{comment_id}", handler.DeleteComment(db, logger))
	mux.HandleFunc("DELETE /v1/users/{user_id}", handler.DeleteUser(db, logger))

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
	if err := server.ListenAndServe(); err != nil {
		log.Fatal("failed to serve API: ", err)
	}
}
