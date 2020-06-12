package model

import (
	basemodel "github.com/Handzo/gogame/common/model"
	"github.com/go-pg/pg/v9"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	basemodel.BaseModel
	Email    string `pg:",notnull,unique"`
	Username string `pg:",notnull,unique"`
	Password string `pg:",notnull"`
}

func (User) Prepare(*pg.DB, bool) error {
	return nil
}

func (User) Sync(*pg.DB, bool) error {
	return nil
}

func (User) Populate(db *pg.DB, force bool) error {
	users := []*User{
		&User{Email: "handzo@test.ru", Username: "Handzo", Password: "testpass"},
		&User{Email: "h1@test.ru", Username: "h1", Password: "123"},
		&User{Email: "h2@test.ru", Username: "h2", Password: "123"},
		&User{Email: "h3@test.ru", Username: "h3", Password: "123"},
	}

	for _, user := range users {
		if err := user.HashPassword(); err != nil {
			return err
		}

		if _, err := db.Model(user).
			OnConflict(`DO NOTHING`).
			Insert(user); err != nil {
			return err
		}
	}

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
