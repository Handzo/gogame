package model

import (
	basemodel "github.com/Handzo/gogame/common/model"
	"github.com/go-pg/pg/v9"
)

type Good struct {
	basemodel.BaseModel
	GoodItemId string `pg:",notnull,type:uuid"`
	GoodItem   *GoodItem
	Amount     uint32
}

func (Good) Prepare(*pg.DB, bool) error {
	return nil
}

func (Good) Sync(*pg.DB, bool) error {
	return nil
}
