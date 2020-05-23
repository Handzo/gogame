package postgres

import (
	"context"
	"fmt"

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

func (r *pgGameRepository) Select(ctx context.Context, model interface{}, columns ...string) error {
	return r.DB.ModelContext(ctx, model).
		Column(columns...).
		WherePK().
		Select()
}

func (r *pgGameRepository) Insert(ctx context.Context, model interface{}) error {
	_, err := r.DB.ModelContext(ctx, model).Insert()
	return err
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

func (r *pgGameRepository) CreateTable(ctx context.Context, unitType string, bet uint32) (*model.Table, error) {
	logger := r.logger.For(ctx)

	unit := &model.Unit{}
	err := r.DB.ModelContext(ctx, unit).
		Where(`unit_type = ?`, unitType).
		Column(`id`, `unit_type`).
		Select()
	if err != nil {
		return nil, err
	}

	logger.Info("Inserting new table", log.String("unit_type", unitType), log.Int64("bet", int64(bet)))

	table := &model.Table{
		Bet:    bet,
		UnitId: unit.Id,
	}

	_, err = r.DB.ModelContext(ctx, table).Insert()
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	models := make([]interface{}, 4)
	for i := 0; i < 4; i++ {
		models[i] = &model.Participant{
			Order:   i + 1,
			TableId: table.Id,
		}
	}

	logger.Info("Creating 4 participants")

	_, err = r.DB.ModelContext(ctx, models...).Insert()
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
			return nil, err
		}

		// no table has been found
		return nil, nil
	}

	return table, nil
}

func (r *pgGameRepository) FindPlayersTable(ctx context.Context, playerId string) (*model.Table, error) {
	participant := &model.Participant{}

	err := r.DB.ModelContext(ctx, participant).
		Relation(`Table`).
		Where(`"participant"."player_id" = ?`, playerId).
		Where(`"table"."start_time" IS NOT NULL`).
		Where(`"table"."end_time" IS NULL`).
		First()

	if err != nil {
		if err != pg.ErrNoRows {
			return nil, err
		}

		return nil, nil
	}

	return r.FindTable(ctx, participant.TableId)
}

func (r *pgGameRepository) CreateParticipants(ctx context.Context, participants ...*model.Participant) error {
	m := make([]interface{}, len(participants))
	for i, p := range participants {
		m[i] = p
	}
	_, err := r.DB.ModelContext(ctx, m...).Insert()
	return err
}

func (r *pgGameRepository) JoinTable(ctx context.Context, tableId string, playerId string) (int, error) {
	ctx, span := r.startSpan(ctx, "param.joinTable")
	if span != nil {
		defer span.Finish()
	}

	key := fmt.Sprintf("table:%s", tableId)

	if err := r.redis.RPush(key, playerId).Err(); err != nil {
		return 0, err
	}

	return 0, nil
}

func (r *pgGameRepository) Update(ctx context.Context, model interface{}, columns ...string) error {
	query := r.DB.ModelContext(ctx, model).WherePK()
	if len(columns) != 0 {
		query = query.Column(columns...)
	}

	_, err := query.Update()
	return err
}

func (r *pgGameRepository) FindCurrentRoundForTable(ctx context.Context, tableId string) (*model.Round, error) {
	round := &model.Round{}
	return round, r.DB.ModelContext(ctx, round).
		Column(`id`, `start_time`, `end_time`, `signature`, `table_id`).
		Where(`table_id = ?`, tableId).
		Where(`start_time IS NOT NULL`).
		Where(`end_time IS NULL`).
		Order(`created_at DESC`).
		First()
}

func (r *pgGameRepository) FindCurrentDealForTable(ctx context.Context, tableId string) (*model.Deal, error) {
	round, err := r.FindCurrentRoundForTable(ctx, tableId)
	if err != nil {
		return nil, err
	}

	deal := &model.Deal{}
	return deal, r.DB.ModelContext(ctx, deal).
		Column(`id`, `start_time`, `end_time`, `signature`, `round_id`).
		Where(`round_id = ?`, round.Id).
		Where(`start_time IS NOT NULL`).
		Where(`end_time IS NULL`).
		Order(`created_at DESC`).
		First()
}

func (r *pgGameRepository) FindCurrentDealOrderForTable(ctx context.Context, tableId string) (*model.DealOrder, error) {
	deal, err := r.FindCurrentDealForTable(ctx, tableId)
	if err != nil {
		return nil, err
	}

	dealOrder := &model.DealOrder{}
	return dealOrder, r.DB.ModelContext(ctx, dealOrder).
		Column(`id`, `start_time`, `end_time`, `signature`, `participant_id`, `deal_id`).
		Where(`deal_id = ?`, deal.Id).
		Where(`start_time IS NOT NULL`).
		Where(`end_time IS NULL`).
		Order(`created_at DESC`).
		First()
}

func (r *pgGameRepository) FindParticipantWithOrder(ctx context.Context, tableId string, order int) (*model.Participant, error) {
	participant := &model.Participant{}
	return participant, r.DB.ModelContext(ctx, participant).
		Relation(`Player`).
		Where(`table_id = ?`, tableId).
		Where(`"participant"."order" = ?`, order).
		Select()
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
