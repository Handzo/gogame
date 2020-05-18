package code

import "google.golang.org/grpc/status"

var (
	InvalidEventType            = status.Error(100, "invalid event type")
	InvalidRequestPayloadError  = status.Error(101, "invalid request payload")
	InvalidResponsePayloadError = status.Error(102, "invalid response error")
)
