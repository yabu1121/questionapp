package model

type UserLanguageSetting struct {
	UserID       int64
	LanguageCode LanguageCode
	IsPrimary    bool
}
