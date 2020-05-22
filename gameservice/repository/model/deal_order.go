package model

import (
	"time"

	basemodel "github.com/Handzo/gogame/common/model"
	"github.com/go-pg/pg/v9"
)

type DealOrder struct {
	basemodel.BaseModel
	StartTime     time.Time
	EndTime       time.Time
	Signature     string
	ParticipantId string `pg:",notnull,type:uuid"`
	Participant   *Participant
	DealId        string `pg:",notnull,type:uuid"`
	Deal          *Deal
}

func (DealOrder) Prepare(*pg.DB, bool) error {
	return nil
}

func (DealOrder) Sync(*pg.DB, bool) error {
	return nil
}
