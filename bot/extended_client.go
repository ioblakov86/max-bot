package bot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

// Attachment represents an attachment in Max Messenger API
type Attachment struct {
	Type    string          `json:"type"`
	Payload KeyboardPayload `json:"payload"`
}

// KeyboardPayload represents the payload for inline keyboard
type KeyboardPayload struct {
	Buttons [][]KeyboardButton `json:"buttons"`
}

// KeyboardButton represents a button in Max Messenger API
type KeyboardButton struct {
	Type    string `json:"type"` // "callback" or "link"
	Text    string `json:"text"`
	Payload string `json:"payload,omitempty"` // for callback buttons
	URL     string `json:"url,omitempty"`     // for link buttons
}

// MessageWithKeyboard represents the message structure for API requests with keyboard
type MessageWithKeyboard struct {
	Text        string       `json:"text"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

// SendMessageWithKeyboard sends a message with inline keyboard buttons using the correct Max Messenger API format
func (b *BotClient) SendMessageWithKeyboard(chatID int64, text string, buttons [][]InlineKeyboardButton) (string, error) {
	// Convert our button format to Max Messenger API format
	var keyboardButtons [][]KeyboardButton
	for _, row := range buttons {
		var keyboardRow []KeyboardButton
		for _, btn := range row {
			button := KeyboardButton{
				Text: btn.Text,
			}
			if btn.Data != "" {
				button.Type = "callback"
				button.Payload = btn.Data
			}
			if btn.URL != "" {
				button.Type = "link"
				button.URL = btn.URL
			}
			keyboardRow = append(keyboardRow, button)
		}
		keyboardButtons = append(keyboardButtons, keyboardRow)
	}

	// Create the message payload with keyboard in attachments
	message := MessageWithKeyboard{
		Text: text,
		Attachments: []Attachment{
			{
				Type: "inline_keyboard",
				Payload: KeyboardPayload{
					Buttons: keyboardButtons,
				},
			},
		},
	}

	// Convert to JSON
	payload, err := json.Marshal(message)
	if err != nil {
		return "", fmt.Errorf("failed to marshal message: %w", err)
	}

	// Get the bot token from environment
	token := os.Getenv("MAX_BOT_TOKEN")
	if token == "" {
		return "", fmt.Errorf("MAX_BOT_TOKEN environment variable is required")
	}

	// Create the API endpoint URL with chat_id as query parameter
	baseURL := "https://platform-api.max.ru/messages"
	params := url.Values{}
	params.Set("chat_id", fmt.Sprintf("%d", chatID))
	fullURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	// Create HTTP request
	req, err := http.NewRequest("POST", fullURL, bytes.NewBuffer(payload))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers according to Max Messenger API documentation
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Check if the request was successful
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	// Parse response to get message ID
	// API returns: {"message": {"body": {"mid": "mid.xxx", ...}, ...}, ...}
	var result map[string]interface{}
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Try to get message ID from message.body.mid
	if message, ok := result["message"].(map[string]interface{}); ok {
		if body, ok := message["body"].(map[string]interface{}); ok {
			if id, ok := body["mid"].(string); ok {
				return id, nil
			}
		}
	}

	return "", fmt.Errorf("message ID not found in response: %s", string(responseBody))
}