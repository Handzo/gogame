package service

import (
	"context"
	"time"

	authpb "github.com/Handzo/gogame/authservice/proto"
	"github.com/Handzo/gogame/common/log"
	"github.com/Handzo/gogame/common/tracing"
	enginepb "github.com/Handzo/gogame/gameengine/proto"
	"github.com/Handzo/gogame/gameservice/code"
	pb "github.com/Handzo/gogame/gameservice/proto"
	"github.com/Handzo/gogame/gameservice/repository"
	"github.com/Handzo/gogame/gameservice/repository/model"
	"github.com/Handzo/gogame/gameservice/service/pubsub"
	"github.com/go-redis/redis"
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-lib/metrics"
)

type gameService struct {
	authsvc   authpb.AuthServiceClient
	enginesvc enginepb.GameEngineClient
	tracer    opentracing.Tracer
	logger    log.Factory
	repo      repository.GameRepository
	pubsub    *pubsub.PubSub
}

func NewGameService(authsvc authpb.AuthServiceClient, enginesvc enginepb.GameEngineClient, repo repository.GameRepository, tracer opentracing.Tracer, metricsFactory metrics.Factory, logger log.Factory) pb.GameServiceServer {
	var rdb *redis.Client
	{
		rdb = redis.NewClient(&redis.Options{
			Addr:     "localhost:6379",
			Password: "",
			DB:       0,
		})

		_, err := rdb.Ping().Result()
		if err != nil {
			logger.Bg().Fatal(err)
		}
	}

	return &gameService{
		authsvc: authsvc,
		tracer:  tracer,
		logger:  logger,
		repo:    repo,
		pubsub: pubsub.New(
			rdb,
			tracing.New("game-pubsub-redis", metricsFactory, logger),
			logger,
		),
	}
}

func (g *gameService) OpenSession(ctx context.Context, req *pb.OpenSessionRequest) (*pb.OpenSessionResponse, error) {
	logger := g.logger.For(ctx)

	res, err := g.authsvc.Validate(ctx, &authpb.ValidateRequest{Token: req.Token})
	if err != nil {
		return nil, err
	}

	logger.Info("Validate user", log.String("user_id", res.UserId), log.String("username", res.Username))

	if res.UserId == "" {
		panic("invalid user id")
	}

	player := &model.Player{
		UserId: res.UserId,
		Name:   res.Username,
	}

	// select player or create new
	if _, err := g.repo.SelectOrInsertPlayer(ctx, player); err != nil {
		return nil, err
	}

	// close all opened player's sessions
	for _, s := range player.Sessions {
		g.closeSession(ctx, s)
	}

	// creat new session for current remote
	session := &model.Session{
		Remote:   ctx.Value("remote").(string),
		PlayerId: player.Id,
	}

	if err = g.repo.CreateSession(ctx, session); err != nil {
		return nil, err
	}

	return &pb.OpenSessionResponse{
		SessionId: session.Id,
		PlayerId:  session.PlayerId,
	}, nil
}

func (g *gameService) CloseSession(ctx context.Context, req *pb.CloseSessionRequest) (*pb.CloseSessionResponse, error) {
	session, err := g.repo.GetOpenedSessionForRemote(ctx, ctx.Value("remote").(string))

	if err != nil {
		return nil, err
	}

	if session != nil {
		g.closeSession(ctx, session)
	}

	return &pb.CloseSessionResponse{}, nil
}

func (g *gameService) closeSession(ctx context.Context, session *model.Session) error {
	session.ClosedAt = time.Now()
	if err := g.repo.UpdateSessions(ctx, session); err != nil {
		return err
	}

	remote := ctx.Value("remote").(string)

	g.pubsub.Publish(ctx, remote, &pubsub.CloseEvent{
		Event:     "CloseSession",
		SessionId: session.Id,
		PlayerId:  session.PlayerId,
	})
	return nil
}

func (g *gameService) CreateTable(ctx context.Context, req *pb.CreateTableRequest) (*pb.CreateTableResponse, error) {
	table, err := g.repo.CreateTable(ctx, req.UnitType, req.Bet)
	if err != nil {
		return nil, err
	}

	return &pb.CreateTableResponse{
		TableId: table.Id,
		Bet:     table.Bet,
	}, nil
}

var players map[string]struct{} = make(map[string]struct{})

func (g *gameService) JoinTable(ctx context.Context, req *pb.JoinTableRequest) (*pb.JoinTableResponse, error) {
	table, err := g.repo.FindTable(ctx, req.TableId)
	if err != nil {
		return nil, err
	}

	if table == nil {
		return nil, code.TableNotFound
	}

	playerId := ctx.Value("player_id").(string)
	if len(players) < 4 {
		if _, ok := players[playerId]; ok {
			return nil, code.PlayerAlreadyJoined
		}

		players[playerId] = struct{}{}

		if len(players) == 4 {
			pps := make([]string, 0, len(players))
			for k := range players {
				pps = append(pps, k)
			}
			// defer func() {

			if err = g.startTable(ctx, table, pps); err != nil {
				return nil, err
			}
		}
	}

	g.logger.For(ctx).Info("Player joined", log.String("player_id", playerId))

	return &pb.JoinTableResponse{}, nil
}

func (g *gameService) startTable(ctx context.Context, table *model.Table, ps []string) error {
	if len(ps) != 4 {
		return code.NotEnoughPlayers
	}

	participants := make([]*model.Participant, len(ps))
	for o, playerId := range ps {
		participants[o] = &model.Participant{
			TableId:      table.Id,
			PlayerId:     playerId,
			InitialOrder: o,
		}
	}

	err := g.repo.CreateParticipants(ctx, participants...)
	if err != nil {
		return err
	}

	return nil
}
