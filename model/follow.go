package model

import "time"

type Follow struct {
	FollowerID  int64
	FollowingID int64
	CreatedAt   time.Time
}
