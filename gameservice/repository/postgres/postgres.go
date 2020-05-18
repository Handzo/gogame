package postgres

import (
	"context"

	// "github.com/Handzo/gogame/gameservice/code"

	"github.com/Handzo/gogame/common/log"
	basemodel "github.com/Handzo/gogame/common/model"
	"github.com/Handzo/gogame/gameservice/repository/model"
	"github.com/go-pg/pg/v9"
	"github.com/go-pg/pg/v9/orm"
	"github.com/opentracing/opentracing-go"
)

type pgGameRepository struct {
	DB *pg.DB
}

func New(tracer opentracing.Tracer, logger log.Factory) *pgGameRepository {
	DB := pg.Connect(&pg.Options{
		Addr:     "localhost:5432",
		Database: "handzo",
		User:     "handzo",
		PoolSize: 5,
	})

	var connected bool

	if _, err := DB.QueryOne(&connected, `SELECT true`); err != nil {
		panic(err)
	}

	DB.AddQueryHook(basemodel.NewDBLogger(tracer))

	tables := []basemodel.Model{
		&model.Player{},
		&model.Session{},
	}

	if err := basemodel.Sync(DB, tables, false); err != nil {
		panic(err)
	}

	return &pgGameRepository{
		DB: DB,
	}
}

func (r *pgGameRepository) SelectOrInsertPlayer(ctx context.Context, player *model.Player) (bool, error) {
	// created, err := r.DB.ModelContext(ctx, player).
	// 	Where(`user_id = ?user_id`).
	// 	OnConflict(`DO NOTHING`).
	// 	SelectOrInsert(player)
	created, err := r.DB.ModelContext(ctx, player).
		Where(`user_id = ?user_id`).
		Relation(`Sessions`, func(q *orm.Query) (*orm.Query, error) {
			return q.Where(`closed_at IS NULL`), nil
		}).
		OnConflict(`DO NOTHING`).
		SelectOrInsert(player)

	if err != nil {
		return false, err
	}

	return created, nil
}

func (r *pgGameRepository) CreateSession(ctx context.Context, session *model.Session) error {
	_, err := r.DB.ModelContext(ctx, session).Insert()
	return err
}

func (r *pgGameRepository) GetOpenedSessionForRemote(ctx context.Context, remote string) (*model.Session, error) {
	session := &model.Session{}
	err := r.DB.ModelContext(ctx, session).
		Where(`remote = ?`, remote).
		Where(`closed_at IS NULL`).
		Select()
	if err != nil {
		if err != pg.ErrNoRows {
			return nil, err
		}

		// no session has been found
		return nil, nil
	}

	return session, nil
}

func (r *pgGameRepository) UpdateSessions(ctx context.Context, sessions ...*model.Session) error {
	m := make([]interface{}, len(sessions))
	for i, s := range sessions {
		m[i] = s
	}
	_, err := r.DB.ModelContext(ctx, m...).WherePK().Update()
	return err
}
