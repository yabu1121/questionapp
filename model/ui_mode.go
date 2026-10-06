package model

import "errors"

type UIMode string

const (
	UIModeDark   UIMode = "dark"
	UIModeLight  UIMode = "light"
	UIModeSystem UIMode = "system"
)

func ParseUIMode(value string) (UIMode, error) {
	uiMode := UIMode(value)
	switch uiMode {
	case UIModeDark, UIModeLight, UIModeSystem:
		return uiMode, nil
	default:
		return "", errors.New("invalid UI mode")
	}
}

type UserSetting struct {
	UserID   int64
	UIMode   UIMode
	Timezone string
}
