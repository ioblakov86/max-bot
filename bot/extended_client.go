package bot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// InlineKeyboardButton represents a button in an inline keyboard
type InlineKeyboardButton struct {
	Text string `json:"text"`
	Data string `json:"callback_data,omitempty"`
	URL  string `json:"url,omitempty"`
}

// InlineKeyboardMarkup represents an inline keyboard
type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

// MessageWithKeyboard represents the message structure for API requests with keyboard
type MessageWithKeyboard struct {
	ChatID      int64               `json:"chat_id"`
	Text        string              `json:"text"`
	ParseMode   string              `json:"parse_mode,omitempty"`
	ReplyMarkup *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

// SendMessageWithKeyboard sends a message with inline keyboard buttons using the correct Max Messenger API
func (b *BotClient) SendMessageWithKeyboard(chatID int64, text string, buttons [][]InlineKeyboardButton) error {
	// Create the message payload with keyboard
	var markup *InlineKeyboardMarkup
	if len(buttons) > 0 {
		markup = &InlineKeyboardMarkup{
			InlineKeyboard: buttons,
		}
	}

	message := MessageWithKeyboard{
		ChatID:      chatID,
		Text:        text,
		ReplyMarkup: markup,
	}

	// Convert to JSON
	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Get the bot token from environment
	token := os.Getenv("MAX_BOT_TOKEN")
	if token == "" {
		return fmt.Errorf("MAX_BOT_TOKEN environment variable is required")
	}

	// Create the API endpoint URL based on Max Messenger API documentation
	// Using the correct API endpoint: platform-api.max.ru
	url := fmt.Sprintf("https://platform-api.max.ru/messages")

	// Create HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers according to Max Messenger API documentation
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Check if the request was successful
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	return nil
}