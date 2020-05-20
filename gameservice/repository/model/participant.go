package model

import (
	basemodel "github.com/Handzo/gogame/common/model"
	"github.com/go-pg/pg/v9"
)

type Participant struct {
	basemodel.BaseModel
	TableId      string `pg:",notnull,type:uuid"`
	Table        *Table
	PlayerId     string `pg:",notnull,type:uuid"`
	Player       *Player
	InitialOrder int `pg:",notnull,use_zero"`
}

func (Participant) Prepare(*pg.DB, bool) error {
	return nil
}

func (Participant) Sync(*pg.DB, bool) error {
	return nil
}
