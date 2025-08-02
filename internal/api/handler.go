package api

import (
	"log"
	"net/http"

	"github.com/cortex-x/go-thai-id-card-reader/internal/infra/websocket"
	gorilla "github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

type Handler struct {
	hub      *websocket.Hub
	upgrader gorilla.Upgrader
}

func NewHandler(hub *websocket.Hub) *Handler {
	return &Handler{
		hub: hub,
		upgrader: gorilla.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow connections from any origin
				return true
			},
		},
	}
}

func (h *Handler) WebSocketHandler(c echo.Context) error {
	conn, err := h.upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return err
	}

	client := h.hub.RegisterClient(conn)

	// Start goroutines for reading and writing
	go client.WritePump()
	go client.ReadPump()

	return nil
}

func (h *Handler) HealthCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status": "healthy",
		"service": "Thai ID Card Reader",
	})
}