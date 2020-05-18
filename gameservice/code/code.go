package code

import "google.golang.org/grpc/status"

var (
	SessionNotFound = status.Error(300, "session not found")
)
