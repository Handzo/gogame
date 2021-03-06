package model

import (
	"github.com/go-pg/pg/v9"
	"github.com/go-pg/pg/v9/orm"
)

type Model interface {
	Prepare(*pg.DB, bool) error
	Sync(*pg.DB, bool) error
}

func Sync(db *pg.DB, tables []Model, force bool) error {
	for _, t := range tables {
		if force {
			db.DropTable(t, &orm.DropTableOptions{Cascade: true, IfExists: true})
		}
		if err := t.Prepare(db, force); err != nil {
			return err
		}
		if err := db.CreateTable(t, &orm.CreateTableOptions{IfNotExists: true, FKConstraints: true}); err != nil {
			return err
		}
		if err := t.Sync(db, force); err != nil {
			return err
		}
	}

	return nil
}
