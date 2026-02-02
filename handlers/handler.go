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
	// Map of chatID to slice of messages
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

	// Determine the chat ID to store the message under
	// For group chats, use ChatId directly
	// For private chats, we could use a different identifier if needed
	chatID := msg.Recipient.ChatId

	// Initialize slice if needed
	if _, exists := ms.messages[chatID]; !exists {
		ms.messages[chatID] = make([]schemes.Message, 0)
	}

	ms.messages[chatID] = append(ms.messages[chatID], msg)
}

// GetMessageHistory returns stored messages for a specific chat
func (ms *MessageStorage) GetMessageHistory(chatID int64) []schemes.Message {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()

	result := make([]schemes.Message, len(ms.messages[chatID]))
	copy(result, ms.messages[chatID])
	return result
}

// GetAllChats returns all chat IDs that have stored messages
func (ms *MessageStorage) GetAllChats() []int64 {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()

	chats := make([]int64, 0, len(ms.messages))
	for chatID := range ms.messages {
		chats = append(chats, chatID)
	}
	return chats
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
	AdminUserID   int64 // ID администратора: +79310071775
}

// NewMessageHandler creates a new message handler
func NewMessageHandler(bot *bot.BotClient) *MessageHandler {
	handler := &MessageHandler{
		Bot:          bot,
		MessageStore: NewMessageStorage(),
		AdminUserID:  79310071775, // Установим ID администратора
	}
	return handler
}

// Handle processes incoming messages and responds accordingly
func (h *MessageHandler) Handle(update schemes.UpdateInterface) error {
	// Check if this is a message created update
	if update.GetUpdateType() == schemes.TypeMessageCreated {
		msgUpdate, ok := update.(*schemes.MessageCreatedUpdate)
		if !ok {
			return fmt.Errorf("could not cast update to MessageCreatedUpdate")
		}

		msg := msgUpdate.Message

		log.Printf("Received message from %d: %s", msg.Sender.UserId, msg.Body.Text)

		// Determine if this is a group chat or private chat
		isGroupChat := msg.Recipient.ChatType != schemes.DIALOG

		// Store all messages for future analysis (except bot's own messages)
		// According to requirements, bot should read and store messages from groups where it's a member
		if msg.Sender.UserId != 0 { // Assuming bot's user ID is 0 or we can determine it differently
			h.MessageStore.AddMessage(msg)
		}

		// Check if someone is mentioning the bot in a group chat
		botMentioned := h.isBotMentioned(msg.Body.Text)

		// If it's a group chat and bot is mentioned, respond with "Извините, я пока не умею разговаривать."
		if isGroupChat && botMentioned {
			responseText := "Извините, я пока не умею разговаривать."
			err := h.Bot.SendMessage(msg.Recipient.ChatId, responseText)
			if err != nil {
				return fmt.Errorf("failed to send response: %w", err)
			}
			log.Printf("Sent response to %d: %s", msg.Recipient.ChatId, responseText)
			return nil
		}

		// If it's a private chat with admin, allow special commands
		if !isGroupChat && msg.Sender.UserId == h.AdminUserID {
			return h.handleAdminCommands(msg)
		}

		// If it's a private chat with non-admin user, respond normally
		if !isGroupChat {
			return h.handlePrivateMessage(msg)
		}

		// For group chats (without mentioning bot), just store the message and don't respond
		// The message has already been stored above
		return nil
	}

	return nil
}

// isBotMentioned checks if the bot is mentioned in the message
func (h *MessageHandler) isBotMentioned(text string) bool {
	// Simple implementation - check for @mentions or direct address
	// In a real implementation, we would check the entities in the message
	lowerText := strings.ToLower(text)

	// Check if message starts with bot-related keywords that suggest direct address
	keywords := []string{"@bot", "бот,", "бот ", "бот!", "бот?"}
	for _, keyword := range keywords {
		if strings.Contains(lowerText, keyword) {
			return true
		}
	}

	// Check if the entire message is just a command directed at the bot
	if strings.HasPrefix(lowerText, "/") {
		return true
	}

	return false
}

// handleAdminCommands handles special commands for admin user
func (h *MessageHandler) handleAdminCommands(msg schemes.Message) error {
	text := strings.ToLower(strings.TrimSpace(msg.Body.Text))

	var responseText string

	switch {
	case contains(text, []string{"привет", "здравствуй", "добрый день", "hello", "hi", "hey"}):
		responseText = "Привет! Это специальный режим для администратора."
	case contains(text, []string{"help", "помощь"}):
		responseText = "Команды администратора: привет, помощь, статистика, история"
	case contains(text, []string{"статистика", "stats"}):
		chats := h.MessageStore.GetAllChats()
		totalMessages := 0
		for _, chatID := range chats {
			history := h.MessageStore.GetMessageHistory(chatID)
			totalMessages += len(history)
		}
		responseText = fmt.Sprintf("Общее количество сообщений в хранилище: %d в %d чатах", totalMessages, len(chats))
	case contains(text, []string{"история", "history"}):
		// Get history for the current chat
		history := h.MessageStore.GetMessageHistory(msg.Recipient.ChatId)
		if len(history) == 0 {
			responseText = "История сообщений в этом чате пуста."
		} else {
			count := len(history)
			if count > 5 {
				count = 5 // Show only last 5 messages
			}
			responseText = fmt.Sprintf("Последние %d сообщений в этом чате:", count)
			for i := len(history) - count; i < len(history); i++ {
				msgTime := time.Unix(history[i].Timestamp, 0)
				responseText += fmt.Sprintf("\n- [%s] %s: %s",
					msgTime.Format("2006-01-02 15:04"),
					history[i].Sender.Name,
					history[i].Body.Text)
			}
		}
	default:
		responseText = "Простите, я не понимаю эту команду. Напишите 'помощь' для получения списка команд."
	}

	err := h.Bot.SendMessage(msg.Recipient.ChatId, responseText)
	if err != nil {
		return fmt.Errorf("failed to send response: %w", err)
	}

	log.Printf("Sent admin response to %d: %s", msg.Recipient.ChatId, responseText)
	return nil
}

// handlePrivateMessage handles regular private messages
func (h *MessageHandler) handlePrivateMessage(msg schemes.Message) error {
	text := strings.ToLower(strings.TrimSpace(msg.Body.Text))

	var responseText string

	switch {
	case contains(text, []string{"привет", "здравствуй", "добрый день", "hello", "hi", "hey"}):
		responseText = "Привет! Я Max Bot. Чем могу помочь?"
	case contains(text, []string{"help", "помощь", "/help"}):
		responseText = "Доступные команды:\n- привет: Приветственное сообщение\n- помощь или /help: Показать это сообщение\n- время: Текущее время\n- повтори [текст]: Повторить за вами текст\n- /last или последние: Последние сообщения в этом чате"
	case contains(text, []string{"time", "время", "час", "времени"}):
		currentTime := time.Unix(msg.Timestamp, 0)
		responseText = fmt.Sprintf("Текущее время: %s", currentTime.Format("2006-01-02 15:04:05"))
	case contains(text, []string{"echo", "повтори", "скажи"}):
		// Extract the part after the command
		responseText = extractAfterCommand(text, []string{"echo", "повтори", "скажи"})
		if responseText == "" {
			responseText = "Эхо... эхо... эхо..."
		} else {
			responseText = "Вы сказали: " + responseText
		}
	case contains(text, []string{"/last", "последние", "last"}):
		// Get last messages from the current chat
		history := h.MessageStore.GetMessageHistoryForLastN(msg.Recipient.ChatId, 5)
		if len(history) == 0 {
			responseText = "В этом чате нет сохраненных сообщений."
		} else {
			responseText = fmt.Sprintf("Последние %d сообщений в этом чате:", len(history))
			for _, msgItem := range history {
				msgTime := time.Unix(msgItem.Timestamp, 0)
				responseText += fmt.Sprintf("\n- [%s] %s: %s",
					msgTime.Format("2006-01-02 15:04"),
					msgItem.Sender.Name,
					msgItem.Body.Text)
			}
		}
	default:
		responseText = "Я получил ваше сообщение: \"" + msg.Body.Text + "\". Я простой бот и могу отвечать на базовые команды вроде 'привет' или 'помощь'."
	}

	err := h.Bot.SendMessage(msg.Recipient.ChatId, responseText)
	if err != nil {
		return fmt.Errorf("failed to send response: %w", err)
	}

	log.Printf("Sent response to %d: %s", msg.Recipient.ChatId, responseText)
	return nil
}

// extractAfterCommand extracts text after specified commands
func extractAfterCommand(text string, commands []string) string {
	lowerText := strings.ToLower(text)

	for _, cmd := range commands {
		// Look for the command followed by a space and the text to repeat
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