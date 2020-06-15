package model

import (
	"github.com/go-pg/pg/v9"
)

type ProductToGood struct {
	Good      *Good
	Product   *Product
	GoodId    string `pg:",notnull,type:uuid"`
	ProductId string `pg:",notnull,type:uuid"`
}

func (ProductToGood) Prepare(*pg.DB, bool) error {
	return nil
}

func (ProductToGood) Sync(*pg.DB, bool) error {
	return nil
}
