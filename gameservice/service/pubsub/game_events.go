package pubsub

import (
	"time"
)

type GameStarted struct {
	Event
	Table     Table     `json:"table_id"`
	StartTime time.Time `json:"start_time"`
}

type GameFinished struct {
	Event
	EndTime time.Time `json:"end_time"`
}

type RoundStarted struct {
	Event
	Table Table `json:"table"`
}

type RoundFinished struct {
	Event
	Table Table `json:"table"`
}

type DealStarted struct {
}

type DealFinished struct {
	Event
	Table Table `json:"table"`
}

type WaitForMove struct {
	Event
	TableId     string      `json:"table_id"`
	Participant Participant `json:"participant"`
}

type PlayerMoved struct {
	Event
	Card  string `json:"card"`
	Order int    `json:"order"`
}

type PlayerJoined struct {
	Event  string `json:"event"`
	Player Player `json:"player"`
}

type PlayerLeaved struct {
	Event         string `json:"event"`
	TableId       string `json:"table_id"`
	ParticiapntId string `json:"participant_id"`
	PlayerId      string `json:"player_id"`
}

type ParticipantStateChanged struct {
	Event       string      `json:"event"`
	Participant Participant `json:"participant"`
}

type Table struct {
	Id           string        `json:"id"`
	Trump        string        `json:"trump"`
	Turn         int           `json:"turn"`
	TableCards   string        `json:"table_cards"`
	ClubPlayer   int           `json:"club_player"`
	Dealer       int           `json:"dealer"`
	Team1Score   int           `json:"team_1_score"`
	Team2Score   int           `json:"team_2_score"`
	Team1Total   int           `json:"team_1_total"`
	Team2Total   int           `json:"team_2_total"`
	Participants []Participant `json:"participants"`
}

type Participant struct {
	Id         string `json:"id"`
	Order      int    `json:"order"`
	State      string `json:"state"`
	Player     Player `json:"player"`
	Cards      string `json:"cards"`
	CardsCount int    `json:"cards_count"`
}

type Player struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}
