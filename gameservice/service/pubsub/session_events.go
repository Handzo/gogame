package pubsub

type CloseEvent struct {
	Event
	SessionId string `json:"session_id"`
	PlayerId  string `json:"player_id"`
}
