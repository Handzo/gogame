package model

import (
	basemodel "github.com/Handzo/gogame/common/model"
	"github.com/go-pg/pg/v9"
)

type ParticipantState string

var (
	FREE       ParticipantState = "free"
	BUSY       ParticipantState = "busy"
	DISCONNECT ParticipantState = "disconnect"
)

type Participant struct {
	basemodel.BaseModel
	TableId  string `pg:",notnull,type:uuid"`
	Table    *Table
	PlayerId string `pg:",type:uuid"`
	Player   *Player
	Order    int              `pg:",notnull,use_zero"`
	State    ParticipantState `pg:",notnull,type:participant_state"`
}

func (Participant) Prepare(db *pg.DB, force bool) error {
	return basemodel.CreateEnum(
		db, force, "participant_state",
		string(FREE),
		string(BUSY),
		string(DISCONNECT),
	)
}

func (Participant) Sync(*pg.DB, bool) error {
	return nil
}
