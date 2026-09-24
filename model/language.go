package model

type LanguageCode string

type Language struct {
	ID          int64
	Code        LanguageCode
	Name        string
	NativeName  string
	Description string
}
