package model

import (
	basemodel "github.com/Handzo/gogame/common/model"
	"github.com/go-pg/pg/v9"
)

type Gender string

var (
	MALE   Gender = "male"
	FEMALE Gender = "female"
)

type Profile struct {
	basemodel.BaseModel
	FirstName string
	LastName  string
	Age       uint32
	Gender    Gender
	Country   string
	Language  string
}

func (Profile) Prepare(db *pg.DB, force bool) error {
	return basemodel.CreateEnum(
		db, force, "gender",
		string(MALE),
		string(FEMALE),
	)
}

func (Profile) Sync(*pg.DB, bool) error {
	return nil
}
