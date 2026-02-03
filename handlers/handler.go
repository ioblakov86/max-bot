package handlers

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"max-bot/bot"
	schemes "github.com/max-messenger/max-bot-api-client-go/schemes"
)

// MessageStorage stores messages for analysis
type MessageStorage struct {
	messages map[int64][]schemes.Message
	mutex    sync.RWMutex
}

// NewMessageStorage creates a new message storage
func NewMessageStorage() *MessageStorage {
	return &MessageStorage{
		messages: make(map[int64][]schemes.Message),
	}
}

// AddMessage adds a message to storage
func (ms *MessageStorage) AddMessage(msg schemes.Message) {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	chatID := msg.Recipient.ChatId
	ms.messages[chatID] = append(ms.messages[chatID], msg)
}

// GetMessageHistory returns stored messages for a specific chat
func (ms *MessageStorage) GetMessageHistory(chatID int64) []schemes.Message {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()

	history := ms.messages[chatID]
	result := make([]schemes.Message, len(history))
	copy(result, history)
	return result
}

// GetMessageHistoryForLastN returns the last N messages for a specific chat
func (ms *MessageStorage) GetMessageHistoryForLastN(chatID int64, n int) []schemes.Message {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()

	allMessages := ms.messages[chatID]

	if len(allMessages) <= n {
		result := make([]schemes.Message, len(allMessages))
		copy(result, allMessages)
		return result
	}

	startIndex := len(allMessages) - n
	result := make([]schemes.Message, n)
	copy(result, allMessages[startIndex:])
	return result
}

// MessageHandler handles incoming messages
type MessageHandler struct {
	Bot           *bot.BotClient
	MessageStore  *MessageStorage
	AdminUserID   int64 // Admin user ID: +79310071775
}

// NewMessageHandler creates a new message handler
func NewMessageHandler(bot *bot.BotClient, adminUserID int64) *MessageHandler {
	return &MessageHandler{
		Bot:          bot,
		MessageStore: NewMessageStorage(),
		AdminUserID:  adminUserID,
	}
}

// Handle processes incoming messages and responds accordingly
func (h *MessageHandler) Handle(update schemes.UpdateInterface) error {
	if update.GetUpdateType() != schemes.TypeMessageCreated {
		return nil
	}

	msgUpdate, ok := update.(*schemes.MessageCreatedUpdate)
	if !ok {
		return fmt.Errorf("could not cast update to MessageCreatedUpdate")
	}

	msg := msgUpdate.Message
	log.Printf("Received message from %d: %s", msg.Sender.UserId, msg.Body.Text)

	// Store all messages except bot's own messages
	if msg.Sender.UserId != 0 {
		h.MessageStore.AddMessage(msg)
	}

	isGroupChat := msg.Recipient.ChatType != schemes.DIALOG
	botMentioned := h.isBotMentioned(msg.Body.Text)

	// Handle group chat mentions
	if isGroupChat && botMentioned {
		return h.sendSimpleResponse(msg.Recipient.ChatId, "Извините, я пока не умею разговаривать.")
	}

	// Handle private messages
	if !isGroupChat {
		return h.handlePrivateMessage(msg)
	}

	return nil
}

// isBotMentioned checks if the bot is mentioned in the message
func (h *MessageHandler) isBotMentioned(text string) bool {
	lowerText := strings.ToLower(text)

	// Check for bot mentions
	keywords := []string{"@bot", "бот,", "бот ", "бот!", "бот?"}
	for _, keyword := range keywords {
		if strings.Contains(lowerText, keyword) {
			return true
		}
	}

	// Check for commands
	return strings.HasPrefix(lowerText, "/")
}

// handlePrivateMessage handles private messages
func (h *MessageHandler) handlePrivateMessage(msg schemes.Message) error {
	// Admin commands
	if msg.Sender.UserId == h.AdminUserID {
		return h.handleAdminCommands(msg)
	}

	// Regular user commands
	return h.handleRegularCommands(msg)
}

// handleAdminCommands handles special commands for admin user
func (h *MessageHandler) handleAdminCommands(msg schemes.Message) error {
	text := strings.ToLower(strings.TrimSpace(msg.Body.Text))

	switch {
	case contains(text, []string{"привет", "здравствуй", "добрый день", "hello", "hi", "hey"}):
		return h.sendSimpleResponse(msg.Recipient.ChatId, "Привет! Это специальный режим для администратора.")
	case contains(text, []string{"help", "помощь"}):
		return h.sendSimpleResponse(msg.Recipient.ChatId, "Команды администратора: привет, помощь, статистика, история")
	case contains(text, []string{"статистика", "stats"}):
		totalMessages := 0
		for _, chatMessages := range h.MessageStore.messages {
			totalMessages += len(chatMessages)
		}
		return h.sendSimpleResponse(msg.Recipient.ChatId, fmt.Sprintf("Общее количество сообщений в хранилище: %d", totalMessages))
	case contains(text, []string{"история", "history"}):
		history := h.MessageStore.GetMessageHistory(msg.Recipient.ChatId)
		if len(history) == 0 {
			return h.sendSimpleResponse(msg.Recipient.ChatId, "История сообщений в этом чате пуста.")
		}

		count := len(history)
		if count > 5 {
			count = 5
		}

		responseText := fmt.Sprintf("Последние %d сообщений в этом чате:", count)
		for i := len(history) - count; i < len(history); i++ {
			msgTime := time.Unix(history[i].Timestamp, 0)
			responseText += fmt.Sprintf("\n- [%s] %s: %s",
				msgTime.Format("2006-01-02 15:04"),
				history[i].Sender.Name,
				history[i].Body.Text)
		}
		return h.sendSimpleResponse(msg.Recipient.ChatId, responseText)
	default:
		return h.sendSimpleResponse(msg.Recipient.ChatId, "Простите, я не понимаю эту команду. Напишите 'помощь' для получения списка команд.")
	}
}

// handleRegularCommands handles commands for regular users
func (h *MessageHandler) handleRegularCommands(msg schemes.Message) error {
	text := strings.ToLower(strings.TrimSpace(msg.Body.Text))

	switch {
	case contains(text, []string{"привет", "здравствуй", "добрый день", "hello", "hi", "hey"}):
		return h.sendSimpleResponse(msg.Recipient.ChatId, "Привет! Я Max Bot. Чем могу помочь?")
	case contains(text, []string{"help", "помощь", "/help"}):
		helpText := "Доступные команды:\n- привет: Приветственное сообщение\n- помощь или /help: Показать это сообщение\n- время: Текущее время\n- повтори [текст]: Повторить за вами текст\n- /last или последние: Последние сообщения в этом чате"
		return h.sendSimpleResponse(msg.Recipient.ChatId, helpText)
	case contains(text, []string{"time", "время", "час", "времени"}):
		currentTime := time.Unix(msg.Timestamp, 0)
		return h.sendSimpleResponse(msg.Recipient.ChatId, fmt.Sprintf("Текущее время: %s", currentTime.Format("2006-01-02 15:04:05")))
	case contains(text, []string{"echo", "повтори", "скажи"}):
		responseText := extractAfterCommand(text, []string{"echo", "повтори", "скажи"})
		if responseText == "" {
			responseText = "Эхо... эхо... эхо..."
		} else {
			responseText = "Вы сказали: " + responseText
		}
		return h.sendSimpleResponse(msg.Recipient.ChatId, responseText)
	case contains(text, []string{"/last", "последние", "last"}):
		history := h.MessageStore.GetMessageHistoryForLastN(msg.Recipient.ChatId, 5)
		if len(history) == 0 {
			return h.sendSimpleResponse(msg.Recipient.ChatId, "В этом чате нет сохраненных сообщений.")
		}

		responseText := fmt.Sprintf("Последние %d сообщений в этом чате:", len(history))
		for _, msgItem := range history {
			msgTime := time.Unix(msgItem.Timestamp, 0)
			responseText += fmt.Sprintf("\n- [%s] %s: %s",
				msgTime.Format("2006-01-02 15:04"),
				msgItem.Sender.Name,
				msgItem.Body.Text)
		}
		return h.sendSimpleResponse(msg.Recipient.ChatId, responseText)
	default:
		responseText := fmt.Sprintf("Я получил ваше сообщение: \"%s\". Я простой бот и могу отвечать на базовые команды вроде 'привет' или 'помощь'.", msg.Body.Text)
		return h.sendSimpleResponse(msg.Recipient.ChatId, responseText)
	}
}

// sendSimpleResponse is a helper to send a response and log it
func (h *MessageHandler) sendSimpleResponse(chatID int64, text string) error {
	err := h.Bot.SendMessage(chatID, text)
	if err != nil {
		return fmt.Errorf("failed to send response: %w", err)
	}
	log.Printf("Sent response to %d: %s", chatID, text)
	return nil
}

// extractAfterCommand extracts text after specified commands
func extractAfterCommand(text string, commands []string) string {
	lowerText := strings.ToLower(text)

	for _, cmd := range commands {
		if strings.Contains(lowerText, cmd+" ") {
			parts := strings.SplitN(lowerText, cmd+" ", 2)
			if len(parts) > 1 {
				originalParts := strings.SplitN(text, cmd+" ", 2)
				return strings.TrimSpace(originalParts[1])
			}
		}
	}

	return ""
}

// contains checks if any of the substrings exist in the text
func contains(text string, substrings []string) bool {
	lowerText := strings.ToLower(text)
	for _, s := range substrings {
		if strings.Contains(lowerText, s) {
			return true
		}
	}
	return false
}