package model

import (
	basemodel "github.com/Handzo/gogame/common/model"
	"github.com/go-pg/pg/v9"
)

type Gender string

var (
	MALE   UnitType = "male"
	FEMALE UnitType = "female"
)

type PlayerInfo struct {
	basemodel.BaseModel
	Name     string
	Age      uint
	Gender   Gender
	Country  string
	Language string
}

func (PlayerInfo) Prepare(db *pg.DB, force bool) error {
	return basemodel.CreateEnum(
		db, force, "gender",
		string(MALE),
		string(FEMALE),
	)
}

func (PlayerInfo) Sync(*pg.DB, bool) error {
	return nil
}
