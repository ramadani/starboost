package transport

type PublishRequest struct {
	Topic   string      `json:"topic"`
	Message interface{} `json:"message"`
}
