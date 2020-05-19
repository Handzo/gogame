package code

import (
	"google.golang.org/grpc/status"
)

var (
	InvalidSignature = status.Error(500, "invalid game signature")
	CardNotFound     = status.Error(501, "card not found")
	InvalidMove      = status.Error(502, "invalid move")
)
