package model

import (
	"time"

	basemodel "github.com/Handzo/gogame/common/model"
	"github.com/go-pg/pg/v9"
)

type Round struct {
	basemodel.BaseModel
	StartTime time.Time `pg:",notnull,default:now()"`
	EndTime   time.Time
	Signature string
	TableId   string `pg:",notnull,type:uuid"`
	Table     *Table
	Deals     []*Deal
}

func (Round) Prepare(*pg.DB, bool) error {
	return nil
}

func (Round) Sync(*pg.DB, bool) error {
	return nil
}
