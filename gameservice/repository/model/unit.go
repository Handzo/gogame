package model

import (
	basemodel "github.com/Handzo/gogame/common/model"
	"github.com/go-pg/pg/v9"
)

type UnitType string

var (
	GOLD UnitType = "gold"
)

type Unit struct {
	basemodel.BaseModel
	UnitType UnitType `pg:",unique,notnull,type:unit_type"`
}

func (Unit) Prepare(db *pg.DB, force bool) error {
	return basemodel.CreateEnum(
		db, force, "unit_type",
		string(GOLD),
	)
}

func (Unit) Sync(*pg.DB, bool) error {
	return nil
}

func (Unit) Populate(db *pg.DB, force bool) error {
	gold := &Unit{UnitType: GOLD}

	_, err := db.Model(gold).
		OnConflict(`DO NOTHING`).
		Returning(`id`).
		Insert()

	if err != nil && err == pg.ErrNoRows {
		return nil
	}

	return err
}
