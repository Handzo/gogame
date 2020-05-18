package model

import (
	"context"
	"time"

	"github.com/go-pg/pg/v9"
)

type Model interface {
	BeforeUpdate(context.Context) (context.Context, error)
	Sync(*pg.DB) error
}

type BaseModel struct {
	Id        string    `pg:",pk,type:uuid,default:gen_random_uuid()"`
	CreatedAt time.Time `pg:",notnull,default:now()"`
	UpdatedAt time.Time `pg:",notnull,default:now()"`
	DeletedAt *time.Time
}

// func (m *BaseModel) BeforeInsert(c context.Context, db orm.DB) error {
// 	now := time.Now()
// 	if m.CreatedAt.IsZero() {
// 		m.CreatedAt = now
// 	}
// 	if m.UpdatedAt.IsZero() {
// 		m.UpdatedAt = now
// 	}
// 	return nil
// }

func (m *BaseModel) BeforeUpdate(ctx context.Context) (context.Context, error) {
	m.UpdatedAt = time.Now()
	return ctx, nil
}

func (m *BaseModel) Sync(db *pg.DB) error {
	return nil
}
