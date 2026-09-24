package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
)

type createUserRequest struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Handle      string `json:"handle"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	Bio         string `json:"bio"`
	AvatarURL   string `json:"avatar_url"`
	Birthday    string `json:"birthday"`
}

type createUserResponse struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	DisplayName string     `json:"display_name"`
	Handle      string     `json:"handle"`
	Email       string     `json:"email"`
	Bio         string     `json:"bio"`
	AvatarURL   string     `json:"avatar_url"`
	Birthday    *time.Time `json:"birthday"`
}

func (r *createUserRequest) Normalize() {
	r.Name = strings.TrimSpace(r.Name)
	r.DisplayName = strings.TrimSpace(r.DisplayName)
	r.Handle = strings.TrimSpace(r.Handle)
	r.Email = strings.TrimSpace(r.Email)
	r.Bio = strings.TrimSpace(r.Bio)
	r.Birthday = strings.TrimSpace(r.Birthday)
}

func (r createUserRequest) Validate() error {
	if r.Name == "" {
		return errors.New("name is required")
	}
	if utf8.RuneCountInString(r.Name) > 20 {
		return errors.New("name must be 20 characters or less")
	}

	if r.DisplayName == "" {
		return errors.New("display name is required")
	}
	if utf8.RuneCountInString(r.DisplayName) > 20 {
		return errors.New("display name must be 20 characters or less")
	}

	if r.Handle == "" {
		return errors.New("handle is required")
	}
	if utf8.RuneCountInString(r.Handle) > 20 {
		return errors.New("handle must be 20 characters or less")
	}
	if strings.ContainsAny(r.Handle, "/?#:@=&") {
		return errors.New("handle contains invalid characters")
	}

	if r.Email == "" {
		return errors.New("email is required")
	}
	if utf8.RuneCountInString(r.Email) > 255 {
		return errors.New("email must be 255 characters or less")
	}
	address, err := mail.ParseAddress(r.Email)
	if err != nil || address.Address != r.Email {
		return errors.New("email has an invalid format")
	}

	if r.Password == "" {
		return errors.New("password is required")
	}
	if len([]byte(r.Password)) > 72 || len([]byte(r.Password)) < 8 {
		return errors.New("password must be between 8 and 72 bytes")
	}

	if utf8.RuneCountInString(r.Bio) > 255 {
		return errors.New("bio must be 255 characters or less")
	}

	return nil
}

func parseBirthday(birthday string) (*time.Time, error) {
	if birthday == "" {
		return nil, nil
	}

	parsed, err := time.Parse("2006-01-02", birthday)
	if err != nil {
		return nil, err
	}

	return &parsed, nil
}

func password2hash(password string) (string, error) {
	hashedByte, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	hashedString := string(hashedByte)
	return hashedString, nil
}

func CreateUser(db *sql.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createUserRequest
		r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.InfoContext(r.Context(), "failed to decode create user request", "error", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		req.Normalize()
		if err := req.Validate(); err != nil {
			logger.InfoContext(r.Context(), "create user request validation failed", "error", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		birthday, err := parseBirthday(req.Birthday)
		if err != nil {
			logger.InfoContext(r.Context(), "create user birthday validation failed", "error", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		hashedPassword, err := password2hash(req.Password)
		if err != nil {
			logger.ErrorContext(r.Context(), "failed to hash password for user creation", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		queryInsert := `insert into users (
			name,
			display_name,
			handle,
			email,
			hashed_password,
			bio,
			avatar_url,
			birthday
		) values (?, ?, ?, ?, ?, ?, ?, ?)
		`
		result, err := db.ExecContext(r.Context(), queryInsert, req.Name, req.DisplayName, req.Handle, req.Email, hashedPassword, req.Bio, req.AvatarURL, birthday)
		if err != nil {
			var mysqlErr *mysql.MySQLError

			if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
				switch {
				case strings.Contains(mysqlErr.Message, "uq_users_email"):
					logger.InfoContext(r.Context(), "create user conflict", "field", "email")
					http.Error(w, "email already exists", http.StatusConflict)
				case strings.Contains(mysqlErr.Message, "uq_users_handle"):
					logger.InfoContext(r.Context(), "create user conflict", "field", "handle")
					http.Error(w, "handle already exists", http.StatusConflict)
				default:
					logger.InfoContext(r.Context(), "create user conflict")
					http.Error(w, "user already exists", http.StatusConflict)
				}
				return
			}
			logger.ErrorContext(r.Context(), "failed to create user", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		id, err := result.LastInsertId()
		if err != nil {
			logger.ErrorContext(r.Context(), "failed to read created user id", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		response := createUserResponse{
			ID:          id,
			Name:        req.Name,
			DisplayName: req.DisplayName,
			Handle:      req.Handle,
			Email:       req.Email,
			Bio:         req.Bio,
			AvatarURL:   req.AvatarURL,
			Birthday:    birthday,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			logger.ErrorContext(r.Context(), "failed to encode create user response", "error", err)
		}
	}
}
