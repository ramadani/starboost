package transport

import (
	"encoding/json"
	"fmt"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/labstack/echo/v4"
	"net/http"
)

type Handler struct {
	Publisher message.Publisher
}

func (h *Handler) Ping(c echo.Context) error {
	return c.String(http.StatusOK, "pong")
}

func (h *Handler) Publish(c echo.Context) error {
	req := new(PublishRequest)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	var payload []byte

	switch val := req.Message.(type) {
	case string:
		payload = []byte(val)
	default:
		b, _err := json.Marshal(val)
		if _err != nil {
			return fmt.Errorf("generate payload error: %q", _err)
		}
		payload = b
	}

	id := watermill.NewUUID()
	msg := message.NewMessage(id, payload)

	if err := h.Publisher.Publish(req.Topic, msg); err != nil {
		return fmt.Errorf("publishing message error: %q", err)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"id": id,
	})
}
