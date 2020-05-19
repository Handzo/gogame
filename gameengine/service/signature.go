package service

import (
	"github.com/Handzo/gogame/gameengine/service/deck"
)

var (
	belkaFaces = []deck.Face{deck.ACE, deck.SEVEN, deck.EIGHT, deck.NINE, deck.TEN, deck.JACK, deck.QUEEN, deck.KING}
)

const (
	PLAYER_0 int = iota
	PLAYER_1
	PLAYER_2
	PLAYER_3
	TRUMP
	TURN
	TABLE
	CPLAYER
	DEALER
	TEAM_1_ROUND_SCORES
	TEAM_2_ROUND_SCORES
	TEAM_1_CARDS
	TEAM_2_CARDS
	TEAM_1_TOTAL
	TEAM_2_TOTAL
	SIG_LENGTH
)
