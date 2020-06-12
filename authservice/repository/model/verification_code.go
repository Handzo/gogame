package model

import (
	"time"

	basemodel "github.com/Handzo/gogame/common/model"
	"github.com/go-pg/pg/v9"
)

type CodeType string

var (
	PASSWORD CodeType = "password"
	EMAIL    CodeType = "email"
)

type VerificationCode struct {
	basemodel.BaseModel
	UserId   string `pg:"user_id,notnull,type:uuid"`
	User     *User
	ExpireAt time.Time `pg:",notnull"`
	Code     string    `pg:",notnull"`
}

func (VerificationCode) Prepare(*pg.DB, bool) error {
	return nil
}

func (VerificationCode) Sync(*pg.DB, bool) error {
	return nil
}
