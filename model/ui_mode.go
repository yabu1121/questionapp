package model

type UIMode string

const (
	UIModeDark   UIMode = "dark"
	UIModeLight  UIMode = "light"
	UIModeSystem UIMode = "system"
)

type UserSetting struct {
	UserID   int64
	UIMode   UIMode
	Timezone string
}
