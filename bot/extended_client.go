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

// ExtendedMessage represents the message structure for API requests
type ExtendedMessage struct {
	Chat        ChatInfo           `json:"chat"`
	Text        string             `json:"text"`
	ParseMode   string             `json:"parse_mode,omitempty"`
	ReplyMarkup InlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

// ChatInfo represents chat information for API requests
type ChatInfo struct {
	ID int64 `json:"id"`
}

// SendMessageWithParseMode sends a message with explicit parse mode using direct API call
func (b *BotClient) SendMessageWithParseMode(chatID int64, text string, parseMode string) error {
	// Create the message payload
	message := ExtendedMessage{
		Chat:      ChatInfo{ID: chatID},
		Text:      text,
		ParseMode: parseMode,
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

	// Create the API endpoint URL - using typical messenger API pattern
	// Note: This URL should be verified against Max Messenger API documentation
	url := fmt.Sprintf("https://api.max.ru/bot%s/messages/send", token)

	// Create HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

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

// SendMessageWithKeyboard sends a message with inline keyboard buttons
func (b *BotClient) SendMessageWithKeyboard(chatID int64, text string, buttons [][]InlineKeyboardButton) error {
	message := ExtendedMessage{
		Chat: ChatInfo{ID: chatID},
		Text: text,
		ReplyMarkup: InlineKeyboardMarkup{
			InlineKeyboard: buttons,
		},
	}

	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	token := os.Getenv("MAX_BOT_TOKEN")
	if token == "" {
		return fmt.Errorf("MAX_BOT_TOKEN environment variable is required")
	}

	url := fmt.Sprintf("https://api.max.ru/bot%s/messages/send", token)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	return nil
}