package model

import (
	basemodel "github.com/Handzo/gogame/common/model"
	"github.com/go-pg/pg/v9"
)

type Player struct {
	basemodel.BaseModel
	UserId    string `pg:",notnull,type:uuid"`
	Nickname  string `pg:",unique,notnull"`
	Level     uint32 `pg:",notnull,default:1"`
	Exp       uint64 `pg:",notnull,default:0"`
	Nuts      uint64 `pg:",notnull,default:0"`
	Gold      uint64 `pg:",notnull,default:0"`
	Avatar    string
	ProfileId string `pg:",type:uuid"`
	Profile   *Profile
	Sessions  []*Session `pg:"fk:player_id"`
}

func (Player) Prepare(*pg.DB, bool) error {
	return nil
}

func (Player) Sync(*pg.DB, bool) error {
	return nil
}
