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

/*
Команды бота:

Для всех пользователей:
- /help - получить список доступных команд
- /echo [text] - повторить за пользователем текст

Для администратора:
- /help - получить список всех команд (включая администраторские)
- /echo [text] - повторить за пользователем текст
- /last - получить последние сохраненные сообщения из текущего чата
- /stat - получить статистику сохраненных сообщений с группировкой по чатам
*/

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
	log.Printf("Received message - UserID: %d, ChatID: %d, ChatType: %v, Text: %s",
		msg.Sender.UserId, msg.Recipient.ChatId, msg.Recipient.ChatType, msg.Body.Text)

	// Store all messages from all chats except bot's own messages
	if msg.Sender.UserId != 0 {
		h.MessageStore.AddMessage(msg)
		log.Printf("Stored message in chat %d, total messages in this chat: %d",
			msg.Recipient.ChatId, len(h.MessageStore.GetMessageHistory(msg.Recipient.ChatId)))
	} else {
		log.Printf("Ignoring message from bot itself")
	}

	isGroupChat := msg.Recipient.ChatType != schemes.DIALOG
	log.Printf("Chat type check - Is group/channel: %v, ChatType: %v", isGroupChat, msg.Recipient.ChatType)

	botMentioned := h.isBotMentioned(msg.Body.Text)
	log.Printf("Bot mention check - Text: '%s', Mentioned: %v", msg.Body.Text, botMentioned)

	// Handle group chat mentions
	if isGroupChat && botMentioned {
		log.Printf("Responding to bot mention in chat %d", msg.Recipient.ChatId)
		return h.sendSimpleResponse(msg.Recipient.ChatId, "Извините, я пока не умею разговаривать.")
	}

	// Handle private messages
	if !isGroupChat {
		log.Printf("Processing private message for user %d", msg.Sender.UserId)
		return h.handlePrivateMessage(msg)
	}

	log.Printf("Message from group/channel %d processed, no response needed", msg.Recipient.ChatId)
	// For group chats, just store the message and don't respond (unless mentioned, handled above)
	return nil
}

// isBotMentioned checks if the bot is mentioned in the message
func (h *MessageHandler) isBotMentioned(text string) bool {
	lowerText := strings.ToLower(text)

	// Check for bot mentions
	keywords := []string{"@bot", "бот,", "бот ", "bot,", "bot "}
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
	text := strings.TrimSpace(msg.Body.Text)

	switch {
	case text == "/help":
		helpText := "Доступные команды администратора:\n- /help: Показать это справку\n- /echo [text]: Повторить за вами текст\n- /last: Последние сохраненные сообщения из текущего чата\n- /stat: Статистика сохраненных сообщений с группировкой по чатам"
		return h.sendSimpleResponse(msg.Recipient.ChatId, helpText)
	case strings.HasPrefix(text, "/echo"):
		responseText := strings.TrimPrefix(text, "/echo")
		responseText = strings.TrimSpace(responseText)
		if responseText == "" {
			responseText = "Эхо... эхо... эхо..."
		} else {
			responseText = "Вы сказали: " + responseText
		}
		return h.sendSimpleResponse(msg.Recipient.ChatId, responseText)
	case text == "/last":
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
	case text == "/stat":
		chats := h.MessageStore.GetAllChats()
		responseText := fmt.Sprintf("Статистика сообщений по чатам (%d):\n", len(chats))
		for _, chatID := range chats {
			history := h.MessageStore.GetMessageHistory(chatID)
			responseText += fmt.Sprintf("- Chat %d: %d сообщений\n", chatID, len(history))
		}
		return h.sendSimpleResponse(msg.Recipient.ChatId, responseText)
	default:
		responseText := fmt.Sprintf("Неизвестная команда. Используйте /help для получения списка команд.")
		return h.sendSimpleResponse(msg.Recipient.ChatId, responseText)
	}
}

// handleRegularCommands handles commands for regular users
func (h *MessageHandler) handleRegularCommands(msg schemes.Message) error {
	text := strings.TrimSpace(msg.Body.Text)

	switch {
	case text == "/help":
		helpText := "Доступные команды:\n- /help: Показать это справку\n- /echo [text]: Повторить за вами текст"
		return h.sendSimpleResponse(msg.Recipient.ChatId, helpText)
	case strings.HasPrefix(text, "/echo"):
		responseText := strings.TrimPrefix(text, "/echo")
		responseText = strings.TrimSpace(responseText)
		if responseText == "" {
			responseText = "Эхо... эхо... эхо..."
		} else {
			responseText = "Вы сказали: " + responseText
		}
		return h.sendSimpleResponse(msg.Recipient.ChatId, responseText)
	default:
		responseText := fmt.Sprintf("Неизвестная команда. Используйте /help для получения списка команд.")
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


