// Example usage of Max Bot
package main

import (
	"fmt"
	"log"
	"time"

	"max-bot/bot"
	"max-bot/handlers"
)

// Example of how to use the bot in a programmatic way
func exampleUsage() {
	// Get bot token from environment variable
	token := "YOUR_MAX_BOT_TOKEN_HERE" // Replace with actual token
	if token == "YOUR_MAX_BOT_TOKEN_HERE" {
		log.Println("Note: You need to set your actual bot token")
		return
	}

	// Create bot client
	client := bot.NewBotClient(token)
	
	// Create message handler
	handler := handlers.NewMessageHandler(client)

	// Simulate receiving a message
	simulatedMessage := bot.Message{
		ID:        "1",
		UserID:    "user123",
		ChatID:    "chat123",
		Text:      "hello",
		Timestamp: time.Now().Unix(),
	}

	fmt.Printf("Simulating message: %s\n", simulatedMessage.Text)
	
	// Handle the message
	err := handler.Handle(simulatedMessage)
	if err != nil {
		log.Printf("Error handling simulated message: %v", err)
	}
}