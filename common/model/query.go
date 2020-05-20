package model

import (
	"fmt"
	"strings"

	"github.com/go-pg/pg/v9"
)

func CreateEnum(db *pg.DB, force bool, name string, columns ...string) error {
	exists := false

	if _, err := db.Query(&exists, "select exists (select true from pg_type where typname = ?);", name); err != nil {
		return err
	}

	if exists && force {
		if _, err := db.Exec(fmt.Sprintf(`drop type %s cascade;`, name)); err != nil {
			return err
		}
		exists = false
	}

	if exists {
		return nil
	}

	cols := make([]string, len(columns))
	for i, col := range columns {
		cols[i] = "'" + col + "'"
	}

	if _, err := db.Exec(fmt.Sprintf(`create type %s as enum (%s)`, name, strings.Join(cols, ","))); err != nil {
		return err
	}

	return nil
}
