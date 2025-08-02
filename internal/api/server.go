package api

import (
	"context"
	"fmt"
	"log"

	"github.com/cortex-x/go-thai-id-card-reader/internal/config"
	"github.com/cortex-x/go-thai-id-card-reader/internal/infra/websocket"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Server struct {
	echo    *echo.Echo
	config  *config.Config
	hub     *websocket.Hub
	handler *Handler
}

func NewServer(cfg *config.Config, hub *websocket.Hub) *Server {
	e := echo.New()
	e.HideBanner = true
	
	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	handler := NewHandler(hub)

	// Routes
	e.GET("/health", handler.HealthCheck)
	e.GET("/ws", handler.WebSocketHandler)

	return &Server{
		echo:    e,
		config:  cfg,
		hub:     hub,
		handler: handler,
	}
}

func (s *Server) Start() error {
	// Start WebSocket hub
	go s.hub.Run()

	addr := fmt.Sprintf(":%d", s.config.Server.Port)
	log.Printf("Starting WebSocket server on %s", addr)
	
	return s.echo.Start(addr)
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.echo.Shutdown(ctx)
}