package bot

import (
	"context"
	"fmt"

	maxbot "github.com/max-messenger/max-bot-api-client-go"
)

// InlineKeyboardButton represents a button in an inline keyboard
type InlineKeyboardButton struct {
	Text string `json:"text"`
	Data string `json:"callback_data,omitempty"`
	URL  string `json:"url,omitempty"`
}

// SendMessageWithKeyboard sends a message with inline keyboard buttons
func (b *BotClient) SendMessageWithKeyboard(chatID int64, text string, buttons [][]InlineKeyboardButton) error {
	// Create a new message
	message := maxbot.NewMessage()
	message.SetChat(chatID)
	message.SetText(text)

	// Convert our button format to the format expected by the Max Messenger API
	var keyboard [][]maxbot.InlineKeyboardButton
	for _, row := range buttons {
		var keyboardRow []maxbot.InlineKeyboardButton
		for _, btn := range row {
			button := maxbot.InlineKeyboardButton{
				Text: btn.Text,
			}
			if btn.Data != "" {
				button.CallbackData = &btn.Data
			}
			if btn.URL != "" {
				button.URL = &btn.URL
			}
			keyboardRow = append(keyboardRow, button)
		}
		keyboard = append(keyboard, keyboardRow)
	}

	// Set the keyboard for the message
	message.SetReplyMarkup(maxbot.InlineKeyboardMarkup{
		InlineKeyboard: keyboard,
	})

	err := b.MaxBot.Messages.Send(context.Background(), message)
	if err != nil {
		return fmt.Errorf("failed to send message with keyboard: %w", err)
	}

	return nil
}