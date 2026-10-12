package config

import (
	"errors"
	"os"
	"strings"
)

const MaxCreateCommentRequestBodyBytes = 1 << 20
const MaxCreateFollowRequestBodyBytes = 1 << 20
const MaxCreateLanguageRequestBodyBytes = 1 << 20
const MaxCreateQuestionnaireLikeRequestBodyBytes = 1 << 20
const MaxCreateQuestionnaireRequestBodyBytes = 1 << 20
const MaxCreateUserRequestBodyBytes = 1 << 20
const MaxCreateVoteRequestBodyBytes = 1 << 20

const MaxPatchQuestionnaireStatusRequestBodyBytes = 1 << 20
const MaxPatchUserProfileRequestBodyBytes = 1 << 20
const MaxPatchUserSettingRequestBodyBytes = 1 << 20

const MaxUpsertUserNotificationSettingRequestBodyBytes = 1 << 20
const MaxUpsertUserSettingRequestBodyBytes = 1 << 20

const MaxLoginRequestBodyBytes = 1 << 20

type Config struct {
	AppEnv       string
	DBDSN        string
	APIAddress   string
	PprofAddress string
}

func Load() (Config, error) {
	appEnv := os.Getenv("APP_ENV")
	dbDSN := os.Getenv("DB_DSN")
	apiAddress := os.Getenv("API_ADDRESS")
	pprofAddress := os.Getenv("PPROF_ADDRESS")

	if strings.TrimSpace(dbDSN) == "" {
		return Config{}, errors.New("DB_DSN is required")
	}
	if strings.TrimSpace(apiAddress) == "" {
		return Config{}, errors.New("API_ADDRESS is required")
	}
	if strings.TrimSpace(pprofAddress) == "" {
		return Config{}, errors.New("PPROF_ADDRESS is required")
	}

	cfg := Config{
		AppEnv:       appEnv,
		DBDSN:        dbDSN,
		APIAddress:   apiAddress,
		PprofAddress: pprofAddress,
	}
	return cfg, nil
}
