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
	"github.com/cortex-x/go-thai-id-card-reader/internal/infra/smartcard"
	"github.com/cortex-x/go-thai-id-card-reader/internal/infra/websocket"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Set up logging
	if cfg.Log.Level == "debug" {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	} else {
		log.SetFlags(log.LstdFlags)
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
		log.Printf("Warning: Failed to initialize card reader: %v", err)
		// Continue running without card reader functionality
	} else {
		// Set up card event handlers
		reader.OnCardInserted(func(card *domain.ThaiIdCard, err error) {
			if err != nil {
				log.Printf("Card read error: %v", err)
				
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
					log.Printf("Failed to broadcast error message: %v", err)
				}
				return
			}
			
			log.Printf("Card inserted: %s", card.CitizenID)
			if err := hub.BroadcastMessage("CARD_INSERTED", card); err != nil {
				log.Printf("Failed to broadcast card inserted message: %v", err)
			}
		})
		
		reader.OnCardRemoved(func() {
			log.Println("Card removed")
			if err := hub.BroadcastMessage("CARD_REMOVED", nil); err != nil {
				log.Printf("Failed to broadcast card removed message: %v", err)
			}
		})
		
		// Start monitoring
		if err := reader.StartMonitoring(); err != nil {
			log.Printf("Failed to start card monitoring: %v", err)
		} else {
			log.Println("Card reader monitoring started")
		}
	}

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

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

	log.Println("Server exited")
}