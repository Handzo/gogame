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
	worker    *WorkManager
}

func NewGameService(
	authsvc authpb.AuthServiceClient,
	enginesvc enginepb.GameEngineClient,
	repo repository.GameRepository,
	pubsub *pubsub.PubSub,
	tracer opentracing.Tracer,
	metricsFactory metrics.Factory,
	logger log.Factory) pb.GameServiceServer {
	gamesvc := &gameService{
		authsvc:   authsvc,
		enginesvc: enginesvc,
		tracer:    tracer,
		logger:    logger,
		repo:      repo,
		pubsub:    pubsub,
		worker:    NewWorkManager(rmq.NewWorker(), tracer, logger),
	}

	gamesvc.worker.Register("StartTable", gamesvc.startTable)     // set start time
	gamesvc.worker.Register("NewRound", gamesvc.newRound)         // generate new signature
	gamesvc.worker.Register("NewDeal", gamesvc.newDeal)           // create new deal
	gamesvc.worker.Register("NewDealOrder", gamesvc.newDealOrder) // send which player's turn to move
	go gamesvc.worker.Start()

	return gamesvc
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

	if table != nil && table.Signature != "" {
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

	hasEmptyPlace := false
	for _, p := range table.Participants {
		if p.PlayerId == playerId {
			return nil, code.PlayerAlreadyJoined
		}
		if p.Player == nil {
			hasEmptyPlace = true
		}
	}

	if !hasEmptyPlace {
		return nil, code.NoEmptyPlaces
	}

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
			logger.Info("set player as participant", log.String("player_id", playerId), log.String("participant_id", p.Id))
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

	// TODO sort participant by id on pg db query
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
		g.worker.AddTask(rmq.NewTask("StartTable", table.Id,
			rmq.WithDelay(time.Second),
			rmq.WithPayload(strings.Join(ps, ",")),
		))
	}

	g.logger.For(ctx).Info("Player joined", log.String("player_id", playerId))

	return res, nil
}

func (g *gameService) MakeMove(ctx context.Context, req *pb.MakeMoveRequest) (*pb.MakeMoveResponse, error) {
	playerId := ctx.Value("player_id")

	table := &model.Table{}
	table.Id = req.TableId
	if err := g.repo.Select(ctx, table, "start_time", "end_time", "signature"); err != nil {
		return nil, err
	}

	// table has not beed started
	if table.StartTime.IsZero() {
		return nil, code.TableNotStarted
	}

	// table already closed
	if !table.EndTime.IsZero() {
		return nil, code.TableClosed
	}

	dealOrder, err := g.repo.FindCurrentDealOrderForTable(ctx, req.TableId)
	if err != nil {
		return nil, err
	}

	participant := &model.Participant{}
	participant.Id = dealOrder.ParticipantId
	if err = g.repo.Select(ctx, participant, "order", "player_id"); err != nil {
		return nil, err
	}

	if participant.PlayerId != playerId {
		return nil, code.OrderError
	}

	res, err := g.enginesvc.Move(ctx, &enginepb.MoveRequest{
		Signature: table.Signature,
		Card:      req.Card,
	})
	if err != nil {
		return nil, err
	}

	table.Signature = res.Signature

	if err = g.repo.Update(ctx, table, "signature"); err != nil {
		return nil, err
	}

	dealOrder.EndTime = time.Now()
	dealOrder.Signature = req.Card
	if err = g.repo.Update(ctx, dealOrder, "signature", "end_time"); err != nil {
		return nil, err
	}

	g.pubsub.Room(table.Id).Publish(ctx, &pubsub.PlayerMoved{
		Event: pubsub.Event{"PlayerMoved"},
		Card:  req.Card,
		Order: participant.Order,
	})

	sigArray := strings.Split(res.Signature, ":")
	if sigArray[enginesig.TABLE] == "" {
		deal, err := g.repo.FindCurrentDealForTable(ctx, table.Id)
		if err != nil {
			return nil, err
		}

		deal.EndTime = time.Now()
		if err = g.repo.Update(ctx, deal, "end_time"); err != nil {
			return nil, err
		}

		g.worker.AddTask(rmq.NewTask("NewDeal", table.Id, rmq.WithDelay(time.Second)))
	} else {
		g.worker.AddTask(rmq.NewTask("NewDealOrder", table.Id, rmq.WithDelay(time.Second)))
	}

	return &pb.MakeMoveResponse{}, nil
}

func (g *gameService) startTable(ctx context.Context, task *rmq.Task) error {
	logger := g.logger.For(ctx)
	logger.Info("Starting new game for table", log.String("table", task.Topic))

	// TODO: check if table already started
	table := &model.Table{}
	table.Id = task.Topic

	if err := g.repo.Select(ctx, table, "start_time"); err != nil {
		return err
	}

	g.logger.Bg().Info(table.StartTime)

	table.StartTime = time.Now()
	// table.Signature = res.Signature
	err := g.repo.Update(ctx, table, "start_time")
	if err != nil {
		return err
	}

	g.pubsub.Room(table.Id).Publish(ctx, &pubsub.TableStartedEvent{
		Event:     pubsub.Event{"TableStarted"},
		TableId:   table.Id,
		StartTime: table.StartTime,
	})

	g.worker.AddTask(rmq.NewTask("NewRound", table.Id, rmq.WithDelay(time.Second), rmq.WithPayload(task.Payload)))

	return nil
}

func (g *gameService) newRound(ctx context.Context, task *rmq.Task) error {
	logger := g.logger.For(ctx)
	logger.Info("Starting new round for table", log.String("table", task.Topic))

	// TODO: get players id from db
	table := &model.Table{}
	table.Id = task.Topic

	if err := g.repo.Select(ctx, table, "start_time", "end_time", "signature"); err != nil {
		return err
	}

	// table has not beed started
	if table.StartTime.IsZero() {
		return code.TableNotStarted
	}

	// table already closed
	if !table.EndTime.IsZero() {
		return code.TableClosed
	}

	logger.Info("Send request to game engine for new round signature")

	res, err := g.enginesvc.NewRound(ctx, &enginepb.NewRoundRequest{
		Signature: table.Signature,
	})
	if err != nil {
		return err
	}

	logger.Info("Saving signature to table")
	table.Signature = res.Signature
	err = g.repo.Update(ctx, table, "signature")
	if err != nil {
		return err
	}

	// create new round
	logger.Info("Creating new round")
	round := &model.Round{
		StartTime: time.Now(),
		Signature: res.Signature,
		TableId:   table.Id,
	}

	if err = g.repo.Insert(ctx, round); err != nil {
		return err
	}

	// parse signature
	tableData := parseSignature(res.Signature)
	tableData.Id = table.Id

	// take copy of players with all cards
	players := tableData.Players

	// substitute players without cards
	tableData.Players = make([]*pb.Player, 4)
	for i, p := range strings.Split(task.Payload, ",") {
		tableData.Players[i] = &pb.Player{
			Id:         p,
			Order:      uint32(i + 1),
			CardsCount: uint32(len(players[i].Cards) / 2),
		}
	}

	logger.Info("Sending cards to players")
	ev := pubsub.NewRoundEvent{
		Event: pubsub.Event{"NewRound"},
		Table: tableData,
	}

	nocards := tableData.Players

	for i, player := range players {
		send := ev
		send.Table.Players = copyPlayers(nocards)
		send.Table.Players[i].Cards = player.Cards
		go g.pubsub.ToPlayer(ctx, send.Table.Players[i].Id, send)
	}

	g.worker.AddTask(rmq.NewTask("NewDeal", table.Id, rmq.WithDelay(time.Second)))
	return nil
}

func (g *gameService) newDeal(ctx context.Context, task *rmq.Task) error {
	logger := g.logger.For(ctx)
	logger.Info("Creating new deal for table", log.String("table", task.Topic))

	table := &model.Table{}
	table.Id = task.Topic
	if err := g.repo.Select(ctx, table, "start_time", "end_time", "signature"); err != nil {
		return err
	}

	// table has not beed started
	if table.StartTime.IsZero() {
		return code.TableNotStarted
	}

	// table already closed
	if !table.EndTime.IsZero() {
		return code.TableClosed
	}

	logger.Info("Get currrent round")
	round, err := g.repo.FindCurrentRoundForTable(ctx, table.Id)
	if err != nil {
		return err
	}

	// create new deal
	logger.Info("Creating new deal for round", log.String("round", round.Id))
	deal := &model.Deal{
		StartTime: time.Now(),
		Signature: table.Signature,
		RoundId:   round.Id,
	}

	if err = g.repo.Insert(ctx, deal); err != nil {
		return err
	}

	g.worker.AddTask(rmq.NewTask("NewDealOrder", table.Id, rmq.WithDelay(time.Second)))
	return nil
}

func (g *gameService) newDealOrder(ctx context.Context, task *rmq.Task) error {
	logger := g.logger.For(ctx)
	logger.Info("Creating new deal order for table", log.String("table", task.Topic))

	table := &model.Table{}
	table.Id = task.Topic
	if err := g.repo.Select(ctx, table, "start_time", "end_time", "signature"); err != nil {
		return err
	}

	// table has not beed started
	if table.StartTime.IsZero() {
		return code.TableNotStarted
	}

	// table already closed
	if !table.EndTime.IsZero() {
		return code.TableClosed
	}

	logger.Info("Get current deal order")
	deal, err := g.repo.FindCurrentDealForTable(ctx, table.Id)
	if err != nil {
		return err
	}

	sigArray := strings.Split(table.Signature, ":")
	turn, _ := strconv.Atoi(sigArray[enginesig.TURN])
	turn += 1

	logger.Info("Get participant with order", log.Int("order", turn))
	participant, err := g.repo.FindParticipantWithOrder(ctx, table.Id, turn)
	if err != nil {
		return err
	}

	logger.Info("Creating deal order for deal", log.String("deal", deal.Id))
	dealOrder := &model.DealOrder{
		StartTime:     time.Now(),
		DealId:        deal.Id,
		ParticipantId: participant.Id,
	}

	if err = g.repo.Insert(ctx, dealOrder); err != nil {
		return err
	}

	g.pubsub.Room(table.Id).Publish(ctx, &pubsub.NewDealOrderEvent{
		Event:   pubsub.Event{"NewDealOrder"},
		TableId: table.Id,
		Player: pubsub.Player{
			Id:            participant.Player.Id,
			ParticipantId: participant.Id,
			Name:          participant.Player.Name,
			Order:         turn,
		},
	})

	// TODO: set deal order timeout callback

	return nil
}

func copyPlayers(players []*pb.Player) []*pb.Player {
	ps := make([]*pb.Player, len(players))
	for i, p := range players {
		c := *p
		ps[i] = &c
	}

	return ps
}

// func (g *gameService) startMove(ctx context.Context, task *rmq.Task) error {
// 	g.logger.For(ctx).Info(task)
// 	return nil
// }

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
