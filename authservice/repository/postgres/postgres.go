package postgres

import (
	"context"

	"github.com/Handzo/gogame/authservice/code"
	"github.com/Handzo/gogame/authservice/repository/model"
	"github.com/Handzo/gogame/common/log"
	basemodel "github.com/Handzo/gogame/common/model"
	"github.com/go-pg/pg/v9"
	"github.com/opentracing/opentracing-go"
)

type pgAuthRepository struct {
	DB *pg.DB
}

func New(tracer opentracing.Tracer, logger log.Factory) *pgAuthRepository {
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
		&model.User{},
	}

	if err := basemodel.Sync(DB, tables, false); err != nil {
		panic(err)
	}

	return &pgAuthRepository{
		DB: DB,
	}
}

func (r *pgAuthRepository) CreateUser(ctx context.Context, user *model.User) error {
	result, err := r.DB.ModelContext(ctx, user).
		OnConflict(`DO NOTHING`).
		Insert(user)

	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return code.UserAlreadyExists
	}

	return nil
}

func (r *pgAuthRepository) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	user := &model.User{}

	err := r.DB.ModelContext(ctx, user).
		Where(`username = ?`, username).
		Select()

	if err != nil {
		if err == pg.ErrNoRows {
			return nil, code.UserNotFound
		} else {
			return nil, err
		}
	}

	return user, nil
}

func (r *pgAuthRepository) GetUserById(ctx context.Context, userId string) (*model.User, error) {
	user := &model.User{}

	err := r.DB.ModelContext(ctx, user).Where(`id = ?`, userId).Select()
	if err != nil {
		return nil, code.UserNotFound
	}

	return user, nil
}
