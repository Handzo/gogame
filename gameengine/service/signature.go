package service

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Handzo/gogame/gameengine/code"
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

type signature struct {
	PlayerCards []string
	Trump       string
	Turn        int
	TableCards  string
	ClubPlayer  int
	Dealer      int
	Team1Scores int
	Team2Scores int
	Team1Cards  string
	Team2Cards  string
	Team1Total  int
	Team2Total  int
}

func Parse(sigStr string) (*signature, error) {
	sigArray := strings.Split(sigStr, ":")
	if len(sigArray) != SIG_LENGTH {
		return nil, code.InvalidSignature
	}

	// invalid player cards
	if len(sigArray[PLAYER_0])%2 != 0 ||
		len(sigArray[PLAYER_1])%2 != 0 ||
		len(sigArray[PLAYER_2])%2 != 0 ||
		len(sigArray[PLAYER_3])%2 != 0 ||
		len(sigArray[TEAM_1_CARDS])%4 != 0 ||
		len(sigArray[TEAM_2_CARDS])%4 != 0 {
		return nil, code.InvalidSignature
	}

	sig := &signature{
		PlayerCards: sigArray[:4],
		Trump:       sigArray[TRUMP],
		TableCards:  sigArray[TABLE],
		Team1Cards:  sigArray[TEAM_1_CARDS],
		Team2Cards:  sigArray[TEAM_2_CARDS],
	}

	for _, s := range []int{TURN, CPLAYER, DEALER, TEAM_1_ROUND_SCORES, TEAM_2_ROUND_SCORES, TEAM_1_TOTAL, TEAM_2_TOTAL} {
		if sigArray[s] == "" {
			continue
		}
		v, err := strconv.Atoi(sigArray[s])
		if err != nil {
			fmt.Println(s)
			return nil, code.InvalidSignature
		}

		switch s {
		case TURN:
			sig.Turn = v
		case CPLAYER:
			sig.ClubPlayer = v
		case DEALER:
			sig.Dealer = v
		case TEAM_1_ROUND_SCORES:
			sig.Team1Scores = v
		case TEAM_2_ROUND_SCORES:
			sig.Team2Scores = v
		case TEAM_1_TOTAL:
			sig.Team1Total = v
		case TEAM_2_TOTAL:
			sig.Team2Total = v
		}
	}

	return sig, nil
}

func (s *signature) TableEmpty() bool {
	return len(s.TableCards) == 0
}

func (s *signature) IsRoundFinished() bool {
	return s.PlayerCards[0] == "" &&
		s.PlayerCards[1] == "" &&
		s.PlayerCards[2] == "" &&
		s.PlayerCards[3] == ""
}

func (s *signature) IsGameFinished() bool {
	return s.Team1Total >= 12 || s.Team2Total >= 12
}
