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

func (ur *createUserRequest) Normalize() {
	ur.Name = strings.TrimSpace(ur.Name)
	ur.DisplayName = strings.TrimSpace(ur.DisplayName)
	ur.Handle = strings.TrimSpace(ur.Handle)
	ur.Email = strings.TrimSpace(ur.Email)
	ur.Bio = strings.TrimSpace(ur.Bio)
	ur.Birthday = strings.TrimSpace(ur.Birthday)
}

func (ur createUserRequest) Validate() error {
	if ur.Name == "" {
		return errors.New("name is required")
	}
	if utf8.RuneCountInString(ur.Name) > 20 {
		return errors.New("name must be 20 characters or less")
	}

	if ur.DisplayName == "" {
		return errors.New("display name is required")
	}
	if utf8.RuneCountInString(ur.DisplayName) > 20 {
		return errors.New("display name must be 20 characters or less")
	}

	if ur.Handle == "" {
		return errors.New("handle is required")
	}
	if utf8.RuneCountInString(ur.Handle) > 20 {
		return errors.New("handle must be 20 characters or less")
	}
	if strings.ContainsAny(ur.Handle, "/?#:@=&") {
		return errors.New("handle contains invalid characters")
	}

	if ur.Email == "" {
		return errors.New("email is required")
	}
	if utf8.RuneCountInString(ur.Email) > 255 {
		return errors.New("email must be 255 characters or less")
	}
	address, err := mail.ParseAddress(ur.Email)
	if err != nil || address.Address != ur.Email {
		return errors.New("email has an invalid format")
	}

	if ur.Password == "" {
		return errors.New("password is required")
	}
	if len([]byte(ur.Password)) > 72 || len([]byte(ur.Password)) < 8 {
		return errors.New("password must be between 8 and 72 bytes")
	}

	if utf8.RuneCountInString(ur.Bio) > 255 {
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
			logger.InfoContext(r.Context(), "failed to decode body", "error", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		req.Normalize()
		if err := req.Validate(); err != nil {
			logger.InfoContext(r.Context(), err.Error(), "error", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		birthday, err := parseBirthday(req.Birthday)
		if err != nil {
			logger.InfoContext(r.Context(), "invalid birthday format", "error", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		hashedPassword, err := password2hash(req.Password)
		if err != nil {
			logger.ErrorContext(r.Context(), "failed to hash password", "error", err)
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
					http.Error(w, "email already exists", http.StatusConflict)
				case strings.Contains(mysqlErr.Message, "uq_users_handle"):
					http.Error(w, "handle already exists", http.StatusConflict)
				default:
					http.Error(w, "user already exists", http.StatusConflict)
				}
				return
			}
			logger.ErrorContext(r.Context(), "failed to execute insert query", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		id, err := result.LastInsertId()
		if err != nil {
			logger.ErrorContext(r.Context(), "failed to get last insert id", "error", err)
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
		_ = json.NewEncoder(w).Encode(response)
	}
}
