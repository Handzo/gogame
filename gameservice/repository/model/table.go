package model

import (
	"time"

	basemodel "github.com/Handzo/gogame/common/model"
	"github.com/go-pg/pg/v9"
)

type Table struct {
	basemodel.BaseModel
	StartTime    time.Time
	EndTime      time.Time
	Signature    string
	UnitId       string `pg:",notnull,type:uuid"`
	Unit         *Unit
	Bet          uint32 `pg:",default:0"`
	Result       string
	Participants []*Participant
	Rounds       []*Round
	CreatorId    string  `pg:",notnull,type:uuid"`
	Creator      *Player `pg:",fk:creator_id"`
}

func (Table) Prepare(*pg.DB, bool) error {
	return nil
}

func (Table) Sync(*pg.DB, bool) error {
	return nil
}

func (t Table) HasEmptyPlaces() bool {
	for _, p := range t.Participants {
		if p.Player == nil {
			return true
		}
	}

	return false
}

func (t Table) IsOpen() bool {
	return !t.StartTime.IsZero() && t.EndTime.IsZero()
}
