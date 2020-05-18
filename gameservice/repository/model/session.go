package model

import (
	"time"

	basemodel "github.com/Handzo/gogame/common/model"
)

type Session struct {
	basemodel.BaseModel
	Remote   string `pg:",notnull"`
	ClosedAt time.Time
	PlayerId string `pg:",type:uuid"`
	Player   *Player
}
