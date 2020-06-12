package postgres

import (
	"context"
	"fmt"
	"strconv"
	"time"

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
		&model.VerificationCode{},
	}

	force := true

	if err := basemodel.Sync(DB, tables, force); err != nil {
		panic(err)
	}

	if err := (&model.User{}).Populate(DB, force); err != nil {
		panic(err)
	}

	return &pgAuthRepository{
		DB: DB,
	}
}

func (r *pgAuthRepository) Update(ctx context.Context, model interface{}, columns ...string) error {
	query := r.DB.ModelContext(ctx, model).WherePK()
	if len(columns) != 0 {
		query = query.Column(columns...)
	}

	_, err := query.Update()
	// if err != nil {
	// 	r.logger.For(ctx).Error(err)
	// }
	return err
}

func (r *pgAuthRepository) Select(ctx context.Context, model interface{}, columns ...string) error {
	err := r.DB.ModelContext(ctx, model).
		Column(columns...).
		WherePK().
		Select()
	// if err != nil {
	// 	r.logger.For(ctx).Error(err)
	// }
	return err
}

func (r *pgAuthRepository) Insert(ctx context.Context, model interface{}) error {
	_, err := r.DB.ModelContext(ctx, model).Insert()
	// if err != nil {
	// r.logger.For(ctx).Error(err)
	// }
	return err
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

func (r *pgAuthRepository) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	user := &model.User{}

	err := r.DB.ModelContext(ctx, user).Where(`email = ?`, email).Select()
	if err != nil {
		return nil, code.UserNotFound
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

func (r *pgAuthRepository) CreateVerificationCode(ctx context.Context, user *model.User) (*model.VerificationCode, error) {
	vercode := &model.VerificationCode{}
	err := r.DB.ModelContext(ctx, vercode).
		Order(`expire_at DESC`).
		First()

	if err != nil {
		if err != pg.ErrNoRows {
			return nil, err
		}
	}

	fmt.Println(vercode.Code)
	c, err := strconv.ParseInt("0"+vercode.Code, 10, 32)
	if err != nil {
		return nil, err
	}

	c = (c + 1) % 10000

	vercode.Id = ""
	vercode.UserId = user.Id
	vercode.ExpireAt = time.Now().Add(time.Minute * 30)
	vercode.Code = fmt.Sprintf("%04d", c)

	if _, err = r.DB.ModelContext(ctx, vercode).Insert(); err != nil {
		return nil, err
	}

	return vercode, nil
}

func (r *pgAuthRepository) GetVerificationCode(ctx context.Context, code string) (*model.VerificationCode, error) {
	c := &model.VerificationCode{}
	if err := r.DB.ModelContext(ctx, c).
		Where(`code = ?`, code).
		Where(`expire_at > ?`, time.Now()).
		Select(); err != nil {
		return nil, err
	}

	return c, nil
}
