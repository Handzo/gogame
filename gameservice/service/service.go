package service

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	authpb "github.com/Handzo/gogame/authservice/proto"
	"github.com/Handzo/gogame/common/log"
	enginepb "github.com/Handzo/gogame/gameengine/proto"
	enginesig "github.com/Handzo/gogame/gameengine/service"
	"github.com/Handzo/gogame/gameservice/code"
	pb "github.com/Handzo/gogame/gameservice/proto"
	"github.com/Handzo/gogame/gameservice/repository"
	"github.com/Handzo/gogame/gameservice/repository/model"
	"github.com/Handzo/gogame/gameservice/service/pubsub"
	"github.com/Handzo/gogame/rmq"
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
	worker    *rmq.Worker
}

func NewGameService(authsvc authpb.AuthServiceClient, enginesvc enginepb.GameEngineClient, repo repository.GameRepository, pubsub *pubsub.PubSub, tracer opentracing.Tracer, metricsFactory metrics.Factory, logger log.Factory) pb.GameServiceServer {
	gamesvc := &gameService{
		authsvc:   authsvc,
		enginesvc: enginesvc,
		tracer:    tracer,
		logger:    logger,
		repo:      repo,
		pubsub:    pubsub,
		worker:    rmq.NewWorker(),
	}
	go gamesvc.startWorker()

	return gamesvc
}

func (g *gameService) startWorker() {
	g.worker.Start()
	for task := range g.worker.Channel() {
		go func() {
			span, ctx, logger := g.logger.StartForWithTracer(context.Background(), g.tracer, task.Callback)
			defer span.Finish()

			logger.Info(task)

			if task.Callback == "StartGame" {
				if err := g.startTable(ctx, task); err != nil {
					logger.Error(err)
				}
			} else if task.Callback == "StartMove" {
				if err := g.startMove(ctx, task); err != nil {
					logger.Error(err)
				}
			}
		}()
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

	table, err := g.repo.FindPlayersTable(ctx, player.Id)
	if err != nil {
		return nil, err
	}

	response := &pb.OpenSessionResponse{
		SessionId: session.Id,
		PlayerId:  session.PlayerId,
	}

	if table != nil {
		tableData := parseSignature(table.Signature)
		tableData.Id = table.Id

		// remove cards of other players
		for _, p := range table.Participants {
			o := p.Order - 1
			tableData.Players[o].Id = p.PlayerId
			if p.PlayerId != player.Id {
				tableData.Players[o].Cards = ""
			}
		}

		response.Table = &tableData
	}

	return response, nil
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
		Event:     pubsub.Event{"CloseSession"},
		SessionId: session.Id,
		PlayerId:  session.PlayerId,
	})
	return nil
}

func (g *gameService) CreateTable(ctx context.Context, req *pb.CreateTableRequest) (*pb.CreateTableResponse, error) {
	g.logger.Bg().Info("create table")
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
	logger := g.logger.For(ctx)

	// find table in pg
	table, err := g.repo.FindTable(ctx, req.TableId)
	if err != nil {
		return nil, err
	}

	if table == nil {
		return nil, code.TableNotFound
	}

	if len(table.Participants) == 0 {
		logger.Error("participants not found for table: " + table.Id)
		return nil, code.InternalError
	}

	playerId := ctx.Value("player_id").(string)

	// hasEmptyPlace := false
	for _, p := range table.Participants {
		if p.PlayerId == playerId {
			// return nil, code.PlayerAlreadyJoined
		}
		if p.PlayerId == "" {
			// hasEmptyPlace = true
		}
	}

	// if !hasEmptyPlace {
	// 	return nil, code.NoEmptyPlaces
	// }

	logger.Info("get player info")

	player := &model.Player{}
	player.Id = playerId

	err = g.repo.Select(ctx, player, "id", "name")
	if err != nil {
		return nil, err
	}

	for _, p := range table.Participants {
		if p.PlayerId == "" {
			p.PlayerId = playerId
			p.Player = player
			err = g.repo.Update(ctx, p, "player_id")
			if err != nil {
				return nil, err
			}
			break
		}
	}

	res := &pb.JoinTableResponse{
		TableId:  table.Id,
		UnitType: string(table.Unit.UnitType),
		Bet:      table.Bet,
	}

	ps := make([]string, 0, 4)

	sort.Slice(table.Participants, func(i, j int) bool {
		return table.Participants[i].Order < table.Participants[j].Order
	})

	for _, p := range table.Participants {
		part := &pb.Participant{
			Id:    p.Id,
			Order: uint32(p.Order),
		}

		if p.Player != nil {
			ps = append(ps, p.Player.Id)
			part.Player = &pb.Player{
				Id:   p.Player.Id,
				Name: p.Player.Name,
			}
		}
		res.Participants = append(res.Participants, part)
	}

	g.pubsub.AddToRoom(ctx, table.Id, playerId)

	if len(ps) == 4 {
		startTask := rmq.NewTask("StartGame", strings.Join(ps, ","), rmq.WithDelay(time.Second), rmq.WithTopic(table.Id))
		g.worker.AddTask(startTask)
	}

	g.logger.For(ctx).Info("Player joined", log.String("player_id", playerId))

	return res, nil
}

func (g *gameService) startTable(ctx context.Context, task *rmq.Task) error {
	// TODO: check if table already started
	table := &model.Table{}
	table.Id = task.Topic

	if err := g.repo.Select(ctx, table, "start_time"); err != nil {
		return err
	}

	// res, err := g.enginesvc.StartNewGame(ctx, &enginepb.StartNewGameRequest{})
	// if err != nil {
	// 	return err
	// }

	// // parse signature
	// tableData := parseSignature(res.Signature)
	// tableData.Id = table.Id

	// // take copy of players with all cards
	// players := tableData.Players

	// // substitute players without cards
	// tableData.Players = make([]*pb.Player, 4)
	// for i, p := range strings.Split(task.Payload, ",") {
	// 	tableData.Players[i] = &pb.Player{
	// 		Id:         p,
	// 		Order:      uint32(i + 1),
	// 		CardsCount: uint32(len(players[i].Cards) / 2),
	// 	}
	// }

	// ev := pubsub.StartGameEvent{
	// 	Event: pubsub.Event{"GameStarted"},
	// 	Table: tableData,
	// }

	// for i, player := range players {
	// 	send := ev
	// 	send.Table.Players = append(make([]*pb.Player, 0), ev.Table.Players...)
	// 	send.Table.Players[i].Cards = player.Cards
	// 	go g.pubsub.ToPlayer(ctx, send.Table.Players[i].Id, send)
	// }

	table.StartTime = time.Now()
	// table.Signature = res.Signature
	err = g.repo.Update(ctx, table, "start_time")
	if err != nil {
		return err
	}

	g.worker.AddTask(rmq.NewTask("StartMove",
		fmt.Sprintf("%d", tableData.Turn),
		rmq.WithDelay(time.Second*2),
		rmq.WithTopic(table.Id),
	))

	return nil
}

func (g *gameService) startMove(ctx context.Context, task *rmq.Task) error {
	g.logger.For(ctx).Info(task)
	return nil
}

func parseSignature(signature string) pb.Table {
	data := strings.Split(signature, ":")
	turn, _ := strconv.Atoi(data[enginesig.TURN])
	cplayer, _ := strconv.Atoi(data[enginesig.CPLAYER])
	dealer, _ := strconv.Atoi(data[enginesig.DEALER])
	team1Score, _ := strconv.Atoi(data[enginesig.TEAM_1_ROUND_SCORES])
	team2Score, _ := strconv.Atoi(data[enginesig.TEAM_2_ROUND_SCORES])
	team1Total, _ := strconv.Atoi(data[enginesig.TEAM_1_TOTAL])
	team2Total, _ := strconv.Atoi(data[enginesig.TEAM_2_TOTAL])

	table := pb.Table{
		Trump:       data[enginesig.TRUMP],
		Turn:        uint32(turn + 1),
		TableCards:  data[enginesig.TABLE],
		ClubPlayer:  uint32(cplayer + 1),
		Dealer:      uint32(dealer + 1),
		Team_1Score: uint32(team1Score),
		Team_2Score: uint32(team2Score),
		Team_1Total: uint32(team1Total),
		Team_2Total: uint32(team2Total),
		Players:     make([]*pb.Player, 4),
	}

	for i, _ := range table.Players {
		table.Players[i] = &pb.Player{
			CardsCount: uint32(len(data[i]) / 2),
			Cards:      data[i],
		}
	}

	fmt.Println(table.Players)

	return table
}
