package model

import (
	"time"

	basemodel "github.com/Handzo/gogame/common/model"
	"github.com/go-pg/pg/v9"
)

type Deal struct {
	basemodel.BaseModel
	StartTime  time.Time
	EndTime    time.Time
	Signature  string
	RoundId    string `pg:",notnull,type:uuid"`
	Round      *Round
	DealOrders []*DealOrder
}

func (Deal) Prepare(*pg.DB, bool) error {
	return nil
}

func (Deal) Sync(*pg.DB, bool) error {
	return nil
}
