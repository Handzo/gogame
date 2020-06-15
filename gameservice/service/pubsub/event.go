package pubsub

type Event struct {
	Event   string      `json:"event"`
	Payload interface{} `json:"payload"`
}
