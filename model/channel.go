package model

type Channel string

const (
	ChannelSlack Channel = "slack"
	ChannelEmail Channel = "email"
)

type UserNotificationSetting struct {
	ID        int64
	UserID    int64
	Channel   Channel
	IsEnabled bool
}
