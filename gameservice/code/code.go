package code

import "google.golang.org/grpc/status"

var (
	SessionNotFound     = status.Error(300, "session not found")
	TableNotFound       = status.Error(301, "table not found")
	PlayerAlreadyJoined = status.Error(302, "player already joined table")
	NotEnoughPlayers    = status.Error(303, "not enough players to start game")
)
