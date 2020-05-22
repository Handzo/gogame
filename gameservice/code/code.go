package code

import "google.golang.org/grpc/status"

var (
	SessionNotFound     = status.Error(300, "session not found")
	TableNotFound       = status.Error(301, "table not found")
	PlayerAlreadyJoined = status.Error(302, "player already joined table")
	NotEnoughPlayers    = status.Error(303, "not enough players to start game")
	InternalError       = status.Error(304, "table not found")
	NoEmptyPlaces       = status.Error(305, "no empty places at the table")
	BindAdressError     = status.Error(306, "bind adress error")
)
