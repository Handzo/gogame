package postgres

import (
	"context"

	// "github.com/Handzo/gogame/gameservice/code"

	"github.com/Handzo/gogame/common/log"
	basemodel "github.com/Handzo/gogame/common/model"
	"github.com/Handzo/gogame/gameservice/repository/model"
	"github.com/go-pg/pg/v9"
	"github.com/go-pg/pg/v9/orm"
	"github.com/go-redis/redis"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

type pgGameRepository struct {
	DB     *pg.DB
	redis  *redis.Client
	tracer opentracing.Tracer
	logger log.Factory
}

func New(redis *redis.Client, tracer opentracing.Tracer, logger log.Factory) *pgGameRepository {
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
		&model.Unit{},
		&model.Table{},
		&model.Participant{},
		&model.Round{},
		&model.Deal{},
		&model.DealOrder{},
	}

	if err := basemodel.Sync(DB, tables, false); err != nil {
		panic(err)
	}

	if err := (&model.Unit{}).Populate(DB, false); err != nil {
		panic(err)
	}

	return &pgGameRepository{
		DB:     DB,
		redis:  redis,
		tracer: tracer,
		logger: logger,
	}
}

func (r *pgGameRepository) Select(ctx context.Context, model interface{}, columns ...string) error {
	err := r.DB.ModelContext(ctx, model).
		Column(columns...).
		WherePK().
		Select()
	if err != nil {
		r.logger.For(ctx).Error(err)
	}
	return err
}

func (r *pgGameRepository) Insert(ctx context.Context, model interface{}) error {
	_, err := r.DB.ModelContext(ctx, model).Insert()
	if err != nil {
		r.logger.For(ctx).Error(err)
	}
	return err
}

func (r *pgGameRepository) Update(ctx context.Context, model interface{}, columns ...string) error {
	query := r.DB.ModelContext(ctx, model).WherePK()
	if len(columns) != 0 {
		query = query.Column(columns...)
	}

	_, err := query.Update()
	if err != nil {
		r.logger.For(ctx).Error(err)
	}
	return err
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
		r.logger.For(ctx).Error(err)
		return false, err
	}

	return created, nil
}

func (r *pgGameRepository) CreateSession(ctx context.Context, session *model.Session) error {
	_, err := r.DB.ModelContext(ctx, session).Insert()
	if err != nil {
		r.logger.For(ctx).Error(err)
	}
	return err
}

func (r *pgGameRepository) GetOpenTables(ctx context.Context) ([]*model.Table, error) {
	tables := []*model.Table{}
	err := r.DB.ModelContext(ctx, &tables).
		Where(`end_time IS NULL`).
		Select()

	if err != nil {
		r.logger.For(ctx).Error(err)
	}

	return tables, err
}

func (r *pgGameRepository) GetOpenedSessionForRemote(ctx context.Context, remote string) (*model.Session, error) {
	session := &model.Session{}
	err := r.DB.ModelContext(ctx, session).
		Where(`remote = ?`, remote).
		Where(`closed_at IS NULL`).
		Select()
	if err != nil {
		if err != pg.ErrNoRows {
			r.logger.For(ctx).Error(err)
			return nil, err
		}

		// no session has been found
		return nil, nil
	}

	return session, nil
}

func (r *pgGameRepository) CreateTable(ctx context.Context, creatorId string, unitType string, bet uint32) (*model.Table, error) {
	logger := r.logger.For(ctx)

	unit := &model.Unit{}
	err := r.DB.ModelContext(ctx, unit).
		Where(`unit_type = ?`, unitType).
		Column(`id`, `unit_type`).
		Select()
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	logger.Info("Inserting new table",
		log.String("unit_type", unitType),
		log.Int64("bet", int64(bet)),
		log.String("creator_id", creatorId),
	)

	table := &model.Table{
		Bet:       bet,
		UnitId:    unit.Id,
		CreatorId: creatorId,
	}

	_, err = r.DB.ModelContext(ctx, table).Insert()
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	models := make([]interface{}, 4)
	for i := 0; i < 4; i++ {
		models[i] = &model.Participant{
			TableId: table.Id,
			Order:   i + 1,
			State:   model.FREE,
		}
	}

	logger.Info("Creating 4 participants")

	_, err = r.DB.ModelContext(ctx, models...).Insert()
	if err != nil {
		logger.Error(err)
	}
	return table, err
}

func (r *pgGameRepository) FindTable(ctx context.Context, tableId string) (*model.Table, error) {
	table := &model.Table{}
	err := r.DB.ModelContext(ctx, table).
		Relation(`Unit`).
		Relation(`Participants`).
		Relation(`Participants.Player`).
		Where(`"table"."id" = ?`, tableId).
		Select()
	if err != nil {
		if err != pg.ErrNoRows {
			r.logger.For(ctx).Error(err)
			return nil, err
		}

		// no table has been found
		return nil, nil
	}

	return table, nil
}

func (r *pgGameRepository) TableReadyCount(ctx context.Context, tableId string) (int, error) {
	p := &model.Participant{}
	count, err := r.DB.ModelContext(ctx, p).
		Where(`table_id = ?`, tableId).
		Where(`state = ?`, model.READY).
		Count()

	if err != nil {
		r.logger.For(ctx).Error(err)
		return count, err
	}

	return count, err
}

func (r *pgGameRepository) FindTableWithPlayer(ctx context.Context, playerId string) (*model.Table, error) {
	participant := &model.Participant{}

	err := r.DB.ModelContext(ctx, participant).
		Relation(`Table`).
		Where(`"participant"."player_id" = ?`, playerId).
		Where(`"table"."start_time" IS NOT NULL`).
		Where(`"table"."end_time" IS NULL`).
		First()

	if err != nil {
		if err != pg.ErrNoRows {
			r.logger.For(ctx).Error(err)
			return nil, err
		}

		return nil, nil
	}

	return r.FindTable(ctx, participant.TableId)
}

func (r *pgGameRepository) GetParticipantsForPlayer(ctx context.Context, playerId string) ([]*model.Participant, error) {
	participants := []*model.Participant{}
	err := r.DB.ModelContext(ctx, &participants).
		Relation(`Table`).
		Where(`"participant"."player_id" = ?`, playerId).
		Where(`"table"."end_time" IS NULL`).
		Select()
	if err != nil {
		r.logger.For(ctx).Error(err)
	}
	return participants, err
}

func (r *pgGameRepository) FindCurrentRoundForTable(ctx context.Context, tableId string) (*model.Round, error) {
	round := &model.Round{}
	err := r.DB.ModelContext(ctx, round).
		Column(`id`, `start_time`, `end_time`, `signature`, `table_id`).
		Where(`table_id = ?`, tableId).
		Where(`start_time IS NOT NULL`).
		Where(`end_time IS NULL`).
		Order(`created_at DESC`).
		First()

	if err != nil {
		r.logger.For(ctx).Error(err)
	}

	return round, err
}

func (r *pgGameRepository) FindCurrentDealForTable(ctx context.Context, tableId string) (*model.Deal, error) {
	round, err := r.FindCurrentRoundForTable(ctx, tableId)
	if err != nil {
		r.logger.For(ctx).Error(err)
		return nil, err
	}

	deal := &model.Deal{}
	err = r.DB.ModelContext(ctx, deal).
		Column(`id`, `start_time`, `end_time`, `signature`, `round_id`).
		Where(`round_id = ?`, round.Id).
		Where(`start_time IS NOT NULL`).
		Where(`end_time IS NULL`).
		Order(`created_at DESC`).
		First()
	if err != nil {
		r.logger.For(ctx).Error(err)
	}
	return deal, err
}

func (r *pgGameRepository) FindCurrentDealOrderForTable(ctx context.Context, tableId string) (*model.DealOrder, error) {
	deal, err := r.FindCurrentDealForTable(ctx, tableId)
	if err != nil {
		r.logger.For(ctx).Error(err)
		return nil, err
	}

	dealOrder := &model.DealOrder{}
	err = r.DB.ModelContext(ctx, dealOrder).
		Column(`id`, `start_time`, `end_time`, `signature`, `participant_id`, `deal_id`).
		Where(`deal_id = ?`, deal.Id).
		Where(`start_time IS NOT NULL`).
		Where(`end_time IS NULL`).
		Order(`created_at DESC`).
		First()

	if err != nil {
		r.logger.For(ctx).Error(err)
	}

	return dealOrder, err
}

func (r *pgGameRepository) FindParticipantWithOrder(ctx context.Context, tableId string, order int) (*model.Participant, error) {
	participant := &model.Participant{}
	err := r.DB.ModelContext(ctx, participant).
		Relation(`Player`).
		Where(`table_id = ?`, tableId).
		Where(`"participant"."order" = ?`, order).
		Select()

	if err != nil {
		r.logger.For(ctx).Error(err)
	}

	return participant, err
}

func (r *pgGameRepository) startSpan(ctx context.Context, operationName string) (context.Context, opentracing.Span) {
	span := opentracing.SpanFromContext(ctx)
	if span != nil {
		span = r.tracer.StartSpan(operationName, opentracing.ChildOf(span.Context()))
		ext.SpanKindRPCClient.Set(span)
		ctx = opentracing.ContextWithSpan(ctx, span)
	}

	return ctx, span
}
