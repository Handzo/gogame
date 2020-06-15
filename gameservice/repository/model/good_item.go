package model

import (
	basemodel "github.com/Handzo/gogame/common/model"
	"github.com/go-pg/pg/v9"
)

type GoodItem struct {
	basemodel.BaseModel
	Title       string `pg:",notnull"`
	Description string
}

func (GoodItem) Prepare(*pg.DB, bool) error {
	return nil
}

func (GoodItem) Sync(*pg.DB, bool) error {
	return nil
}
