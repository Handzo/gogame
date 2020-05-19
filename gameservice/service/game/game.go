package game

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Handzo/gogame/gameservice/service/game/deck"
	"google.golang.org/grpc/status"
)

var (
	belkaFaces = []deck.Face{deck.ACE, deck.SEVEN, deck.EIGHT, deck.NINE, deck.TEN, deck.JACK, deck.QUEEN, deck.KING}
)

var (
	InvalidSignature = status.Error(500, "invalid game signature")
	CardNotFound     = status.Error(501, "card not found")
	InvalidMove      = status.Error(502, "invalid move")
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

type GameEngine struct {
	players []*deck.Deck
	next    int
}

func NewGameEngine() *GameEngine {
	return &GameEngine{
		players: make([]*deck.Deck, 4),
	}
}

func (g *GameEngine) StartNewGame() string {
	sigArray := make([]string, SIG_LENGTH)

	sigArray[TRUMP] = "0"
	sigArray[CPLAYER] = ""
	sigArray[DEALER] = "2"
	sigArray[TEAM_1_TOTAL] = "0"
	sigArray[TEAM_2_TOTAL] = "0"

	sig := strings.Join(sigArray, ":")
	sig, err := g.NewRound(sig)
	if err != nil {
		panic(err)
	}
	return sig
}

func (g *GameEngine) NewRound(sig string) (string, error) {
	sigArray := strings.Split(sig, ":")

	// define dealer
	dealer, err := strconv.Atoi(sigArray[DEALER])
	if err != nil {
		return "", InvalidSignature
	}

	dealer++

	sigArray[TURN] = strconv.Itoa(int((dealer + 1) % 4))

	h1 := &deck.Deck{}
	h2 := &deck.Deck{}
	h3 := &deck.Deck{}
	h4 := &deck.Deck{}

	newDeck := deck.New(deck.Faces(belkaFaces...), deck.Unshuffled)
	newDeck.Deal(8, h1, h2, h3, h4)

	sigArray[PLAYER_0] = h1.GetSignature()
	sigArray[PLAYER_1] = h2.GetSignature()
	sigArray[PLAYER_2] = h3.GetSignature()
	sigArray[PLAYER_3] = h4.GetSignature()
	sigArray[TABLE] = ""
	sigArray[TEAM_1_ROUND_SCORES] = "0"
	sigArray[TEAM_2_ROUND_SCORES] = "0"
	sigArray[TEAM_1_CARDS] = ""
	sigArray[TEAM_2_CARDS] = ""

	if sigArray[CPLAYER] != "" {
		cJack := deck.NewCard(deck.JACK, deck.CLUB)
		for i, h := range []*deck.Deck{h1, h2, h3, h4} {
			if h.HasCard(cJack) {
				sigArray[TRUMP] = strconv.Itoa(i)
				break
			}
		}
	}

	return strings.Join(sigArray, ":"), nil
}

func (g *GameEngine) Move(sig, cardSig string) (string, error) {
	sigArray := strings.Split(sig, ":")

	if len(sigArray) < SIG_LENGTH {
		return "", InvalidSignature
	}

	turn, err := strconv.Atoi(sigArray[TURN])
	if err != nil {
		return "", InvalidSignature
	}

	hand := deck.New(deck.Unshuffled, deck.FromSignature(sigArray[turn]))
	card := deck.GetCard(cardSig)

	table := deck.New(deck.Unshuffled, deck.FromSignature(sigArray[TABLE]))
	trump := deck.GetSuit(sigArray[TRUMP])

	if len(sigArray[CPLAYER]) == 0 && card.Face() == deck.JACK && card.Suit() == deck.CLUB {
		sigArray[CPLAYER] = strconv.Itoa(turn)
	}

	if !g.ValidateMove(table, hand, card, trump) {
		return "", InvalidMove
	}

	sigArray[turn] = hand.GetSignature()

	if table.NumberOfCards() == 3 {
		// calculate scores
		table.Cards = append(table.Cards, card)

		scores, sci := g.CalculateScores(table, trump)

		// clear table
		sigArray[TABLE] = ""
		turn = (turn + sci + 1) % 4
		sigArray[TURN] = strconv.Itoa(turn)

		fmt.Println(scores, sci)

		t1, err := strconv.Atoi(sigArray[TEAM_1_ROUND_SCORES])
		if err != nil {
			return "", InvalidSignature
		}

		t2, err := strconv.Atoi(sigArray[TEAM_2_ROUND_SCORES])
		if err != nil {
			return "", InvalidSignature
		}

		if turn == 0 || turn == 2 {
			t1 += scores
			sigArray[TEAM_1_ROUND_SCORES] = strconv.Itoa(t1)
			sigArray[TEAM_1_CARDS] += table.GetSignature()
		} else {
			t2 += scores
			sigArray[TEAM_2_ROUND_SCORES] = strconv.Itoa(t2)
			sigArray[TEAM_2_CARDS] += table.GetSignature()
		}

		if t1+t2 == 120 {
			sigArray[TEAM_1_ROUND_SCORES] = "0"
			sigArray[TEAM_2_ROUND_SCORES] = "0"
			// round ends
			if t1 > t2 { // team 1 won
				win := 1
				if t2 < 30 {
					win++
				}
				if trump == deck.HEART || trump == deck.DIAMOND {
					win++
				}

				total, err := strconv.Atoi(sigArray[TEAM_1_TOTAL])
				if err != nil {
					return "", InvalidSignature
				}

				sigArray[TEAM_1_TOTAL] = strconv.Itoa(total + win)
			} else { // team 2 won
				win := 1
				if t2 < 30 {
					win++
				}
				if trump == deck.CLUB || trump == deck.SPADE {
					win++
				}

				total, err := strconv.Atoi(sigArray[TEAM_2_TOTAL])
				if err != nil {
					return "", InvalidSignature
				}

				sigArray[TEAM_2_TOTAL] = strconv.Itoa(total + win)
			}
		}
	} else {
		sigArray[TABLE] += cardSig
		sigArray[TURN] = strconv.Itoa(int((turn + 1) % 4))
	}
	// g.players
	return strings.Join(sigArray, ":"), nil
}

func (g *GameEngine) CalculateScores(table *deck.Deck, trump deck.Suit) (int, int) {
	scores := 0
	sci := 0
	card := table.Cards[0]
	for i, c := range table.Cards {
		if !stronger(card, c, trump) {
			// fmt.Println(c, "greater than", card)
			card = c
			sci = i
		}
		scores += getScore(c)
	}
	return scores, sci
}

func stronger(left, right deck.Card, trump deck.Suit) bool {
	if left.Face() == deck.JACK {
		// left card is JACK, true for right card is JACK and has weaker SUIT
		return right.Face() != deck.JACK || left.Suit() < right.Suit()
	} else if left.Suit() == trump {
		// left card is a TRUMP but not a JACK
		// false if right card is a JACK
		// true if right card is not a TRUMP or has weaker FACE
		if right.Face() == deck.JACK {
			return false
		} else {
			return right.Suit() != trump || left.Face() > right.Face()
		}
	} else {
		// left card is general card
		// false if right card is a TRUMP or JACK
		// false if right card has distinct SUIT
		// true if left card has stronger FACE
		if right.Suit() == trump || right.Face() == deck.JACK {
			return false
		} else if right.Suit() != left.Suit() {
			return false
		} else {
			return left.Face() > right.Face()
		}
	}
	fmt.Println(left, right)
	panic("bug")
}

func getScore(card deck.Card) int {
	switch card.Face() {
	case deck.ACE:
		return 11
	case deck.TEN:
		return 10
	case deck.JACK:
		return 2
	case deck.QUEEN:
		return 3
	case deck.KING:
		return 4
	}
	return 0
}

func (g *GameEngine) ValidateMove(table *deck.Deck, hand *deck.Deck, card deck.Card, trump deck.Suit) bool {
	// remove card from hand
	if !hand.Remove(card) {
		return false
	}

	// first card on table
	if table.NumberOfCards() == 0 {
		return true
	}

	firstCard := table.Cards[0]

	// first card is trump or jack
	if firstCard.Suit() == trump || firstCard.Face() == deck.JACK {
		// check if card to move is trump or jack
		if card.Suit() == trump || card.Face() == deck.JACK {
			return true
		}

		// check if player has no trump or jack
		for _, c := range hand.Cards {
			if c.Suit() == trump || c.Face() == deck.JACK {
				return false
			}
		}
		return true
	}

	// same suit
	if firstCard.Suit() == card.Suit() && card.Face() != deck.JACK {
		return true
	}

	// player has no such suit in his hand
	for _, c := range hand.Cards {
		if c.Suit() == firstCard.Suit() && c.Face() != deck.JACK {
			return false
		}
	}

	return true
}

func print(sig string) {
	if len(sig) == 0 {
		return
	}
	fmt.Println(sig)
	sigArray := strings.Split(sig, ":")
	// fmt.Println(sigArray[PLAYER_1], deck.New(deck.FromSignature(sigArray[PLAYER_1])))
	fmt.Println("PLAYER_0", deck.New(deck.Unshuffled, deck.FromSignature(sigArray[PLAYER_0])))
	fmt.Println("PLAYER_1", deck.New(deck.Unshuffled, deck.FromSignature(sigArray[PLAYER_1])))
	fmt.Println("PLAYER_2", deck.New(deck.Unshuffled, deck.FromSignature(sigArray[PLAYER_2])))
	fmt.Println("PLAYER_3", deck.New(deck.Unshuffled, deck.FromSignature(sigArray[PLAYER_3])))
	fmt.Println("TRUMP", deck.GetSuit(sigArray[TRUMP]).String())
	fmt.Println("TURN", sigArray[TURN])

	fmt.Println("TABLE", deck.New(deck.Unshuffled, deck.FromSignature(sigArray[TABLE])))
	fmt.Println("CPLAYER", sigArray[CPLAYER])
	fmt.Println("TEAM_1_ROUND_SCORES", sigArray[TEAM_1_ROUND_SCORES])
	fmt.Println("TEAM_2_ROUND_SCORES", sigArray[TEAM_2_ROUND_SCORES])
	fmt.Println("TEAM_1_CARDS", deck.New(deck.Unshuffled, deck.FromSignature(sigArray[TEAM_1_CARDS])))
	fmt.Println("TEAM_2_CARDS", deck.New(deck.Unshuffled, deck.FromSignature(sigArray[TEAM_2_CARDS])))
	fmt.Println("TEAM_1_TOTAL", sigArray[TEAM_1_TOTAL])
	fmt.Println("TEAM_2_TOTAL", sigArray[TEAM_2_TOTAL])
	fmt.Println("---------------------------")
}
