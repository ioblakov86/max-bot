package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"max-bot/bot"
	"max-bot/handlers"

	"github.com/joho/godotenv"
	schemes "github.com/max-messenger/max-bot-api-client-go/schemes"
)

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Get bot token from environment variable
	token := os.Getenv("MAX_BOT_TOKEN")
	if token == "" {
		log.Fatal("MAX_BOT_TOKEN environment variable is required")
	}

	// Get admin user ID from environment variable
	adminUserID := int64(0)
	if adminUserIDStr := os.Getenv("ADMIN_USER_ID"); adminUserIDStr != "" {
		var err error
		adminUserID, err = strconv.ParseInt(adminUserIDStr, 10, 64)
		if err != nil {
			log.Printf("Failed to parse ADMIN_USER_ID: %v, using default value", err)
			adminUserID = 0
		}
	} else {
		log.Println("ADMIN_USER_ID not set, using default value")
	}

	// Create bot client
	client := bot.NewBotClient(token)

	// Create message handler
	handler := handlers.NewMessageHandler(client, adminUserID)

	// Create context with cancellation for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Received shutdown signal...")
		cancel()
	}()

	log.Println("Max Bot is starting...")

	// Get updates channel
	updates, err := client.GetUpdates(ctx)
	if err != nil {
		log.Fatalf("Failed to get updates: %v", err)
	}

	// Main loop to process updates
	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down Max Bot...")
			return
		case update := <-updates:
			if update != nil {
				// Handle the update asynchronously
				go func(u schemes.UpdateInterface) {
					if err := handler.Handle(u); err != nil {
						log.Printf("Error handling update: %v", err)
					}
				}(update)
			}
		}
	}
}