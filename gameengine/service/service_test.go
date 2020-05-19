package service

import (
	"fmt"
	"testing"

	"github.com/Handzo/gogame/gameengine/service/deck"
)

func TestNewDeck(t *testing.T) {
	ge := NewGameEngine()
	sig := ge.StartNewGame()

	print(sig)

	cards := []string{
		"c2", "52", "a2", "72",
		"c3", "53", "a3", "73",
		"80", "90", "a0", "70",
		"51", "a1", "71", "c1",
		"82", "50", "62", "b2",
		"91", "60", "b0", "c0",
		"92", "61", "b3", "83",
		"93", "63", "b1", "81",
	}

	var err error
	for i, c := range cards {
		fmt.Println("try", deck.GetCard(c))
		sig, err = ge.Move(sig, c)
		if err != nil {
			fmt.Println(err)
			break
		}
		print(sig)
		fmt.Println("--", i, "--")
	}

	sig, err = ge.NewRound(sig)
	if err != nil {
		panic(err)
	}
	print(sig)
}
