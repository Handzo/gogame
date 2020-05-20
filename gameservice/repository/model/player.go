package model

import (
	basemodel "github.com/Handzo/gogame/common/model"
	"github.com/go-pg/pg/v9"
)

type Player struct {
	basemodel.BaseModel
	UserId   string     `pg:",notnull,type:uuid"`
	Name     string     `pg:",unique,notnull"`
	Balance  uint64     `pg:",notnulldefault:0"`
	Sessions []*Session `pg:"fk:player_id"`
}

func (Player) Prepare(*pg.DB, bool) error {
	return nil
}

func (Player) Sync(*pg.DB, bool) error {
	return nil
}
