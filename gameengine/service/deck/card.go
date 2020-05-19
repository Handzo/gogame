package deck

import (
	"fmt"
	"strconv"
)

// Suit represents the suit of the card (spade, heart, diamond, club)
type Suit int

// Face represents the face of the card (ace, two...queen, king)
type Face int

// Constants for Suit ♠♥♦♣
const (
	CLUB Suit = iota
	SPADE
	HEART
	DIAMOND
)

// Constants for Face
const (
	TWO Face = iota
	THREE
	FOUR
	FIVE
	SIX
	SEVEN
	EIGHT
	NINE
	TEN
	JACK
	QUEEN
	KING
	ACE
)

// Global Variables representing the default suits and faces in a deck of cards
var (
	SUITS = []Suit{CLUB, DIAMOND, HEART, SPADE}
	FACES = []Face{ACE, TWO, THREE, FOUR, FIVE, SIX, SEVEN, EIGHT, NINE, TEN, JACK, QUEEN, KING}
)

func (s Suit) String() string {
	switch s {
	case CLUB:
		return "♣"
	case DIAMOND:
		return "♦"
	case HEART:
		return "♥"
	case SPADE:
		return "♠"
	}
	return ""
}

func (f Face) String() string {
	switch f {
	case ACE:
		return "A"
	case TWO:
		return "2"
	case THREE:
		return "3"
	case FOUR:
		return "4"
	case FIVE:
		return "5"
	case SIX:
		return "6"
	case SEVEN:
		return "7"
	case EIGHT:
		return "8"
	case NINE:
		return "9"
	case TEN:
		return "T"
	case JACK:
		return "J"
	case QUEEN:
		return "Q"
	case KING:
		return "K"
	}
	return ""
}

// Card represents a playing card with a Face and a Suit
type Card int

func (c Card) String() string {
	return fmt.Sprintf("%s%s", c.Face(), c.Suit())
}

// Face is a utility function to get the face of a card
func (c Card) Face() Face {
	return Face(c / 4)
}

// Suit is a utility function to get the suit of a card
func (c Card) Suit() Suit {
	return Suit(c % 4)
}

// NewCard creates a new card with a face and suit
func NewCard(face Face, suit Suit) Card {
	return Card(int(face)*4 + int(suit))
}

// GetSignature is the hex representation of the Face and Suit of the card
func (c *Card) GetSignature() string {
	return fmt.Sprintf("%x%x", int(c.Face()), int(c.Suit()))
}

// GetCard returns Card from card's hex representation
func GetCard(card string) Card {
	return NewCard(GetFace(string(card[0])), GetSuit(string(card[1])))
}

// GetFace returns Face from the hex representation
func GetFace(sig string) Face {
	face, _ := strconv.ParseInt(sig, 16, 8)
	return Face(face)
}

// GetSuit returns Suit from the hex representation
func GetSuit(sig string) Suit {
	face, _ := strconv.ParseInt(sig, 16, 8)
	return Suit(face)
}
