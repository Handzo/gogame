package service

import (
	"context"
	"time"

	authpb "github.com/Handzo/gogame/authservice/proto"
	"github.com/Handzo/gogame/common/log"
	"github.com/Handzo/gogame/common/tracing"
	pb "github.com/Handzo/gogame/gameservice/proto"
	"github.com/Handzo/gogame/gameservice/repository"
	"github.com/Handzo/gogame/gameservice/repository/model"
	"github.com/Handzo/gogame/gameservice/repository/postgres"
	"github.com/Handzo/gogame/gameservice/service/pubsub"
	"github.com/go-redis/redis"
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-lib/metrics"
)

type gameService struct {
	authsvc authpb.AuthServiceClient
	tracer  opentracing.Tracer
	logger  log.Factory
	repo    repository.GameRepository
	pubsub  *pubsub.PubSub
}

func NewGameService(authsvc authpb.AuthServiceClient, tracer opentracing.Tracer, metricsFactory metrics.Factory, logger log.Factory) pb.GameServiceServer {
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
		repo: postgres.New(
			tracing.New("game-db-pg", metricsFactory, logger),
			logger,
		),
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
	return &pb.CloseSessionResponse{}, nil
}

func (g *gameService) closeSession(ctx context.Context, session *model.Session) error {
	session.ClosedAt = time.Now()
	if err := g.repo.UpdateSessions(ctx, session); err != nil {
		return err
	}

	g.pubsub.Publish(ctx, ctx.Value("remote").(string), &pubsub.CloseEvent{
		Event:     "Closssss",
		SessionId: session.Id,
		PlayerId:  session.PlayerId,
	})
	return nil
}
