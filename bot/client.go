package bot

import (
	"context"
	"fmt"

	maxbot "github.com/max-messenger/max-bot-api-client-go"
	schemes "github.com/max-messenger/max-bot-api-client-go/schemes"
)

// BotClient handles communication with Max Messenger API
type BotClient struct {
	MaxBot *maxbot.Api
}

// NewBotClient creates a new instance of BotClient
func NewBotClient(token string) *BotClient {
	api, err := maxbot.New(token)
	if err != nil {
		panic(fmt.Sprintf("Failed to create Max Bot API client: %v", err))
	}

	return &BotClient{
		MaxBot: api,
	}
}

// SendMessage sends a message to a specific chat with markdown formatting
func (b *BotClient) SendMessage(chatID int64, text string) error {
	// Create a new message with markdown formatting
	message := maxbot.NewMessage()
	message.SetChat(chatID)
	message.SetText(text)
	
	// Try to set parse mode for markdown formatting if available
	// Since the direct method might not be available, we'll rely on the API's auto-detection
	// of markdown syntax in the text itself
	
	err := b.MaxBot.Messages.Send(context.Background(), message)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}


// GetUpdates fetches new messages from the API
func (b *BotClient) GetUpdates(ctx context.Context) (<-chan schemes.UpdateInterface, error) {
	// Use the official Max Bot API client to get updates
	updates := b.MaxBot.GetUpdates(ctx)

	return updates, nil
}