package pubsub

import (
	"time"

	pb "github.com/Handzo/gogame/gameservice/proto"
)

type TableStartedEvent struct {
	Event
	TableId   string    `json:"table_id"`
	StartTime time.Time `json:"start_time"`
}

type NewRoundEvent struct {
	Event
	Table pb.Table `json:"table"`
}

type NewDealOrderEvent struct {
	Event
	TableId string `json:"table_id"`
	Player  Player `json:"player"`
}

type PlayerMoved struct {
	Event
	Card  string `json:"card"`
	Order int    `json:"order"`
}

type Table struct {
	Id         string   `json:"id"`
	Trump      string   `json:"trump"`
	Turn       int      `json:"turn"`
	TableCards string   `json:"table_cards"`
	ClubPlayer int      `json:"club_player"`
	Dealer     int      `json:"dealer"`
	Team1Score int      `json:"team_1_score"`
	Team2Score int      `json:"team_2_score"`
	Team1Total int      `json:"team_1_total"`
	Team2Total int      `json:"team_2_total"`
	Players    []Player `json:"players"`
}

type Player struct {
	Id            string `json:"id"`
	ParticipantId string `json:"participant_id"`
	Name          string `json:"name"`
	Order         int    `json:"order"`
	Cards         string `json:"cards"`
	CardsCount    int    `json:"cards_count"`
}
