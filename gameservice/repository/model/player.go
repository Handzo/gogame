package model

import (
	basemodel "github.com/Handzo/gogame/common/model"
	"github.com/go-pg/pg/v9"
)

type Player struct {
	basemodel.BaseModel
	UserId       string `pg:",notnull,type:uuid"`
	Nickname     string `pg:",unique,notnull"`
	Nuts         uint64 `pg:",notnull,default:0"`
	Gold         uint64 `pg:",notnull,default:0"`
	Avatar       string
	PlayerInfoId string `pg:",type:uuid"`
	PlayerInfo   *PlayerInfo
	Sessions     []*Session `pg:"fk:player_id"`
}

func (Player) Prepare(*pg.DB, bool) error {
	return nil
}

func (Player) Sync(*pg.DB, bool) error {
	return nil
}
