package model

import (
	basemodel "github.com/Handzo/gogame/common/model"
	"github.com/go-pg/pg/v9"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	basemodel.BaseModel
	Username string `pg:",notnull,unique"`
	Password string `pg:",notnull"`
}

func (User) Prepare(*pg.DB, bool) error {
	return nil
}

func (User) Sync(*pg.DB, bool) error {
	return nil
}

func (this *User) HashPassword() error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(this.Password), 10)
	if err != nil {
		return err
	}
	this.Password = string(hashedPassword)
	return nil
}

func (this *User) ValidPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(this.Password), []byte(password))
	return err == nil
}
