package pubsub

type CloseEvent struct {
	Event     string `json:"event"`
	SessionId string `json:"session_id"`
	PlayerId  string `json:"player_id"`
}
