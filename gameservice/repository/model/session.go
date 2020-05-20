package model

import (
	"time"

	basemodel "github.com/Handzo/gogame/common/model"
	"github.com/go-pg/pg/v9"
)

type Session struct {
	basemodel.BaseModel
	Remote   string `pg:",notnull"`
	ClosedAt time.Time
	PlayerId string `pg:",type:uuid"`
	Player   *Player
}

func (Session) Prepare(*pg.DB, bool) error {
	return nil
}

func (Session) Sync(*pg.DB, bool) error {
	return nil
}
