package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cortex-x/go-thai-id-card-reader/internal/api"
	"github.com/cortex-x/go-thai-id-card-reader/internal/config"
	"github.com/cortex-x/go-thai-id-card-reader/internal/domain"
	"github.com/cortex-x/go-thai-id-card-reader/internal/infra/logger"
	"github.com/cortex-x/go-thai-id-card-reader/internal/infra/smartcard"
	"github.com/cortex-x/go-thai-id-card-reader/internal/infra/websocket"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Set up file logging
	if err := logger.Setup(&cfg.Log); err != nil {
		log.Fatalf("Failed to setup logger: %v", err)
	}

	// Create WebSocket hub
	hub := websocket.NewHub()

	// Create and start server
	server := api.NewServer(cfg, hub)
	
	// Start server in a goroutine
	go func() {
		if err := server.Start(); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Initialize card reader
	reader, err := smartcard.NewPCSCReader()
	if err != nil {
		logger.Errorf("Warning: Failed to initialize card reader: %v", err)
		// Continue running without card reader functionality
	} else {
		// Set up card event handlers
		reader.OnCardInserted(func(card *domain.ThaiIdCard, err error) {
			if err != nil {
				logger.Errorf("Card read error: %v", err)
				
				// Determine error code based on error message
				var errCode int
				var errMsg string
				
				switch err.Error() {
				case domain.ErrMsgReaderNotFound:
					errCode = domain.ErrCodeReaderNotFound
					errMsg = domain.ErrMsgReaderNotFound
				case domain.ErrMsgCardNotDetected:
					errCode = domain.ErrCodeCardNotDetected
					errMsg = domain.ErrMsgCardNotDetected
				default:
					if err.Error() == domain.ErrMsgUnsupportedCard {
						errCode = domain.ErrCodeUnsupportedCard
						errMsg = domain.ErrMsgUnsupportedCard
					} else {
						errCode = domain.ErrCodeReadFailed
						errMsg = domain.ErrMsgReadFailed
					}
				}
				
				if err := hub.BroadcastMessage("ERROR", domain.ErrorResponse{
					Code:    errCode,
					Message: errMsg,
				}); err != nil {
					logger.Errorf("Failed to broadcast error message: %v", err)
				}
				return
			}
			
			logger.Infof("Card inserted: %s", card.CitizenID)
			if err := hub.BroadcastMessage("CARD_INSERTED", card); err != nil {
				logger.Errorf("Failed to broadcast card inserted message: %v", err)
			}
		})
		
		reader.OnCardRemoved(func() {
			logger.Info("Card removed")
			if err := hub.BroadcastMessage("CARD_REMOVED", nil); err != nil {
				logger.Errorf("Failed to broadcast card removed message: %v", err)
			}
		})
		
		// Start monitoring
		if err := reader.StartMonitoring(); err != nil {
			logger.Errorf("Failed to start card monitoring: %v", err)
		} else {
			logger.Info("Card reader monitoring started")
		}
	}

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Stop card monitoring
	if reader != nil {
		reader.StopMonitoring()
	}

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Info("Server exited")
}