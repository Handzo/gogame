package service

import (
	"context"
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

const (
	START_GAME   string = "START_GAME"
	FINISH_GAME  string = "FINISH_GAME"
	START_ROUND  string = "START_ROUND"
	FINISH_ROUND string = "FINISH_ROUND"
	START_DEAL   string = "START_DEAL"
	FINISH_DEAL  string = "FINISH_DEAL"
	NEXT_MOVE    string = "NEXT_MOVE"
)

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

	gamesvc.worker.Register(START_GAME, gamesvc.startGame)     // set start time
	gamesvc.worker.Register(FINISH_GAME, gamesvc.finishGame)   // set start time
	gamesvc.worker.Register(START_ROUND, gamesvc.startRound)   // generate new signature
	gamesvc.worker.Register(FINISH_ROUND, gamesvc.finishRound) // generate new signature
	gamesvc.worker.Register(START_DEAL, gamesvc.startDeal)     // create new deal
	gamesvc.worker.Register(FINISH_DEAL, gamesvc.finishDeal)   // close current deal, start new deal/round or close table
	gamesvc.worker.Register(NEXT_MOVE, gamesvc.nextMove)       // send which player's turn to move
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

	table, err := g.repo.FindTableWithPlayer(ctx, player.Id)
	if err != nil {
		return nil, err
	}

	response := &pb.OpenSessionResponse{
		SessionId: session.Id,
		PlayerId:  session.PlayerId,
	}

	// if player at the table
	if table != nil && table.Signature != "" {
		sig, err := enginesig.Parse(table.Signature)
		if err != nil {
			return nil, err
		}

		tableData := &pb.Table{
			Id:           table.Id,
			Trump:        sig.Trump,
			Turn:         uint32(sig.Turn + 1),
			TableCards:   sig.TableCards,
			ClubPlayer:   uint32(sig.ClubPlayer + 1),
			Dealer:       uint32(sig.Dealer + 1),
			Team_1Score:  uint32(sig.Team1Scores),
			Team_2Score:  uint32(sig.Team2Scores),
			Team_1Total:  uint32(sig.Team1Total),
			Team_2Total:  uint32(sig.Team2Total),
			Participants: make([]*pb.Participant, 4),
		}

		// remove cards of other players
		for _, p := range table.Participants {
			o := p.Order - 1
			tableData.Participants[o] = &pb.Participant{
				Id:         p.Id,
				Order:      uint32(p.Order),
				CardsCount: uint32(len(sig.PlayerCards[o]) / 2),
			}
			if p.PlayerId != "" {
				tableData.Participants[o].Player = &pb.Player{
					Id:   p.Player.Id,
					Name: p.Player.Name,
				}
				if p.PlayerId == player.Id {
					tableData.Participants[o].Cards = sig.PlayerCards[o]
				}
			}
		}

		g.pubsub.AddToRoom(ctx, table.Id, session.PlayerId)

		response.Table = tableData
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
	if err := g.repo.Update(ctx, session, "closed_at"); err != nil {
		return err
	}

	// TODO: remove from room, publish 'player leaved' after
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
	table, err := g.repo.CreateTable(ctx, ctx.Value("player_id").(string), req.UnitType, req.Bet)
	if err != nil {
		return nil, err
	}

	return &pb.CreateTableResponse{
		TableId: table.Id,
		Bet:     table.Bet,
	}, nil
}

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
		logger.Error("participants not found for table", log.String("table", table.Id))
		return nil, code.InternalError
	}

	if !table.HasEmptyPlaces() {
		return nil, code.NoEmptyPlaces
	}

	playerId := ctx.Value("player_id").(string)
	for _, p := range table.Participants {
		if p.PlayerId == playerId {
			return nil, code.PlayerAlreadyJoined
		}
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

	tableData := &pb.Table{
		Id:           table.Id,
		Participants: make([]*pb.Participant, 4),
	}

	pcount := 0
	for _, p := range table.Participants {
		o := p.Order - 1
		tableData.Participants[o] = &pb.Participant{
			Id:    p.Id,
			Order: uint32(p.Order),
		}
		if p.PlayerId != "" {
			pcount++
			tableData.Participants[o].Player = &pb.Player{
				Id:   p.Player.Id,
				Name: p.Player.Name,
			}
		}
	}

	if table.Signature != "" {
		sig, err := enginesig.Parse(table.Signature)
		if err != nil {
			return nil, err
		}

		tableData.Trump = sig.Trump
		tableData.Turn = uint32(sig.Turn + 1)
		tableData.TableCards = sig.TableCards
		tableData.ClubPlayer = uint32(sig.ClubPlayer + 1)
		tableData.Dealer = uint32(sig.Dealer + 1)
		tableData.Team_1Score = uint32(sig.Team1Scores)
		tableData.Team_2Score = uint32(sig.Team2Scores)
		tableData.Team_1Total = uint32(sig.Team1Total)
		tableData.Team_2Total = uint32(sig.Team2Total)

		for _, p := range tableData.Participants {
			p.CardsCount = uint32(len(sig.PlayerCards[p.Order-1]) / 2)
			if p.Player.Id != "" {
				p.Cards = sig.PlayerCards[p.Order-1]
			}
		}
	}

	res := &pb.JoinTableResponse{
		Table: tableData,
	}

	g.pubsub.AddToRoom(ctx, table.Id, playerId)

	if pcount == 4 {
		g.worker.AddTask(rmq.NewTask(START_GAME, table.Id,
			rmq.WithDelay(time.Second),
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

	signature, err := enginesig.Parse(res.Signature)
	if err != nil {
		return nil, err
	}

	if signature.TableEmpty() {
		g.worker.AddTask(rmq.NewTask(FINISH_DEAL, table.Id, rmq.WithDelay(time.Second)))
	} else {
		g.worker.AddTask(rmq.NewTask(NEXT_MOVE, table.Id, rmq.WithDelay(time.Second)))
	}

	return &pb.MakeMoveResponse{}, nil
}

func (g *gameService) startGame(ctx context.Context, task *rmq.Task) error {
	logger := g.logger.For(ctx)
	logger.Info("Starting new game for table", log.String("table", task.Topic))

	// TODO: check if table already started
	table := &model.Table{}
	table.Id = task.Topic

	if err := g.repo.Select(ctx, table, "start_time"); err != nil {
		return err
	}

	if !table.StartTime.IsZero() {
		return code.TableAlreadyStarted
	}

	table.StartTime = time.Now()
	err := g.repo.Update(ctx, table, "start_time")
	if err != nil {
		return err
	}

	// TOOD: maybe set table data, participants
	g.pubsub.Room(table.Id).Publish(ctx, &pubsub.GameStarted{
		Event: pubsub.Event{"GameStarted"},
		Table: pubsub.Table{
			Id: table.Id,
		},
		StartTime: table.StartTime,
	})

	g.worker.AddTask(rmq.NewTask(START_ROUND, table.Id, rmq.WithDelay(time.Second)))

	return nil
}

func (g *gameService) startRound(ctx context.Context, task *rmq.Task) error {
	logger := g.logger.For(ctx)
	logger.Info("Starting new round for table", log.String("table", task.Topic))

	table, err := g.repo.FindTable(ctx, task.Topic)
	if err != nil {
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
	sig, err := enginesig.Parse(res.Signature)
	if err != nil {
		return err
	}

	tableData := pubsub.Table{
		Id:           table.Id,
		Trump:        sig.Trump,
		ClubPlayer:   sig.ClubPlayer + 1,
		Dealer:       sig.Dealer + 1,
		Team1Score:   sig.Team1Scores,
		Team2Score:   sig.Team2Scores,
		Team1Total:   sig.Team1Total,
		Team2Total:   sig.Team2Total,
		Participants: make([]pubsub.Participant, 4),
	}

	table, err = g.repo.FindTable(ctx, table.Id)
	if err != nil {
		return err
	}

	for i, p := range table.Participants {
		tableData.Participants[i] = pubsub.Participant{
			Id:         p.Id,
			Order:      p.Order,
			CardsCount: len(sig.PlayerCards[p.Order-1]) / 2,
		}
		if p.PlayerId != "" {
			tableData.Participants[i].Player = pubsub.Player{
				Id:   p.Player.Id,
				Name: p.Player.Name,
			}
		}
	}

	logger.Info("Sending cards to players")
	ev := pubsub.RoundStarted{
		Event: pubsub.Event{"RoundStarted"},
		Table: tableData,
	}

	nocards := tableData.Participants

	for _, participant := range table.Participants {
		send := ev
		send.Table.Participants = copyParticipants(nocards)
		send.Table.Participants[participant.Order-1].Cards = sig.PlayerCards[participant.Order-1]
		go g.pubsub.ToPlayer(ctx, participant.PlayerId, send)
	}

	g.worker.AddTask(rmq.NewTask(START_DEAL, table.Id, rmq.WithDelay(time.Second)))
	return nil
}

func (g *gameService) startDeal(ctx context.Context, task *rmq.Task) error {
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

	g.worker.AddTask(rmq.NewTask(NEXT_MOVE, table.Id, rmq.WithDelay(time.Second)))
	return nil
}

func (g *gameService) nextMove(ctx context.Context, task *rmq.Task) error {
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

	signature, err := enginesig.Parse(table.Signature)
	if err != nil {
		return err
	}
	signature.Turn += 1

	logger.Info("Get participant with order", log.Int("order", signature.Turn))
	participant, err := g.repo.FindParticipantWithOrder(ctx, table.Id, signature.Turn)
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

	g.pubsub.Room(table.Id).Publish(ctx, &pubsub.WaitForMove{
		Event:   pubsub.Event{"WaitForMove"},
		TableId: table.Id,
		Participant: pubsub.Participant{
			Id:    participant.Id,
			Order: signature.Turn,
		},
	})

	// TODO: set deal order timeout callback

	return nil
}

func (g *gameService) finishDeal(ctx context.Context, task *rmq.Task) error {
	logger := g.logger.For(ctx)
	logger.Info("Creating new deal order for table", log.String("table", task.Topic))

	table := &model.Table{}
	table.Id = task.Topic
	if err := g.repo.Select(ctx, table, "start_time", "end_time", "signature"); err != nil {
		return err
	}

	deal, err := g.repo.FindCurrentDealForTable(ctx, table.Id)
	if err != nil {
		return err
	}

	deal.EndTime = time.Now()
	if err = g.repo.Update(ctx, deal, "end_time"); err != nil {
		return err
	}

	sig, err := enginesig.Parse(table.Signature)
	if err != nil {
		return err
	}

	g.pubsub.Room(table.Id).Publish(ctx, &pubsub.DealFinished{
		Event: pubsub.Event{"DealFinished"},
		Table: pubsub.Table{
			Id:         table.Id,
			Turn:       sig.Turn + 1,
			Team1Score: sig.Team1Scores,
			Team2Score: sig.Team2Scores,
		},
	})

	if sig.IsRoundFinished() {
		g.worker.AddTask(rmq.NewTask(FINISH_ROUND, table.Id, rmq.WithDelay(time.Second)))
	} else {
		// new deal
		g.worker.AddTask(rmq.NewTask(START_DEAL, table.Id, rmq.WithDelay(time.Second)))
	}

	return nil
}

func (g *gameService) finishRound(ctx context.Context, task *rmq.Task) error {
	table := &model.Table{}
	table.Id = task.Topic
	if err := g.repo.Select(ctx, table, "end_time", "start_time", "signature"); err != nil {
		return err
	}

	if table.StartTime.IsZero() {
		return code.TableNotStarted
	}

	if !table.EndTime.IsZero() {
		return code.TableClosed
	}

	round, err := g.repo.FindCurrentRoundForTable(ctx, table.Id)
	if err != nil {
		return err
	}

	if !round.EndTime.IsZero() {
		return code.RoundClosedError
	}

	round.EndTime = time.Now()
	if err := g.repo.Update(ctx, round, "end_time"); err != nil {
		return err
	}

	sig, err := enginesig.Parse(table.Signature)
	if err != nil {
		return err
	}

	g.pubsub.Room(task.Topic).Publish(ctx, &pubsub.RoundFinished{
		Event: pubsub.Event{"RoundFinished"},
		Table: pubsub.Table{
			Id:         table.Id,
			Team1Total: sig.Team1Total,
			Team2Total: sig.Team2Total,
		},
	})

	// push total scores
	if sig.IsGameFinished() {
		g.worker.AddTask(rmq.NewTask(FINISH_GAME, table.Id, rmq.WithDelay(time.Second)))
	} else {
		g.worker.AddTask(rmq.NewTask(START_ROUND, table.Id, rmq.WithDelay(time.Second)))
	}

	return nil
}

func (g *gameService) finishGame(ctx context.Context, task *rmq.Task) error {
	table := &model.Table{}
	table.Id = task.Topic
	if err := g.repo.Select(ctx, table, "end_time"); err != nil {
		return err
	}

	if !table.EndTime.IsZero() {
		return code.TableClosed
	}

	// close table
	table.EndTime = time.Now()
	if err := g.repo.Update(ctx, table, "end_time"); err != nil {
		return err
	}

	g.pubsub.Room(table.Id).Publish(ctx, &pubsub.GameFinished{
		Event:   pubsub.Event{"GameFinished"},
		EndTime: table.EndTime,
	})

	return nil
}

func copyParticipants(participants []pubsub.Participant) []pubsub.Participant {
	ps := make([]pubsub.Participant, len(participants))
	for i, p := range participants {
		ps[i] = p
	}

	return ps
}
