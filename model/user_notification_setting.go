package model

import "errors"

type Channel string

const (
	ChannelSlack Channel = "slack"
	ChannelEmail Channel = "email"
)

func ParseChannel(value string) (Channel, error) {
	parsedChannel := Channel(value)
	switch parsedChannel {
	case ChannelEmail, ChannelSlack:
		return parsedChannel, nil
	default:
		return "", errors.New("invalid channel")
	}
}

type UserNotificationSetting struct {
	ID        int64
	UserID    int64
	Channel   Channel
	IsEnabled bool
}
