package handlers

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"max-bot/ai"
	"max-bot/bot"
	"max-bot/joomla"
	"max-bot/storage"
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
	Bot              *bot.BotClient
	MessageStore     *MessageStorage
	AdminUserID      int64 // Admin user ID: +79310071775
	AdminChatID      int64 // Admin's chat ID for notifications (from environment variable)
	TrackedChatID    int64 // ID of the chat to track for AI analysis
	AIAnalyzer       *ai.OpenRouterConfig
	CallbackStorage  *storage.CallbackStorage // Persistent storage for callback results
	JoomlaClient     *joomla.JoomlaClient     // Client for Joomla integration
}

// NewMessageHandler creates a new message handler
func NewMessageHandler(bot *bot.BotClient, adminUserID int64) *MessageHandler {
	// Load tracked chat ID from environment
	trackedChatID := int64(0)
	if trackedChatIDStr := os.Getenv("CHAT_ID"); trackedChatIDStr != "" {
		id, err := strconv.ParseInt(trackedChatIDStr, 10, 64)
		if err != nil {
			log.Printf("Failed to parse CHAT_ID: %v, using default value", err)
		} else {
			trackedChatID = id
		}
	}

	// Load admin chat ID from environment
	adminChatID := int64(0)
	if adminChatIDStr := os.Getenv("ADMIN_CHAT_ID"); adminChatIDStr != "" {
		id, err := strconv.ParseInt(adminChatIDStr, 10, 64)
		if err != nil {
			log.Printf("Failed to parse ADMIN_CHAT_ID: %v, using default value", err)
		} else {
			adminChatID = id
		}
	}

	// Load the prompt from prompt.txt file using multiple possible paths
	var promptBytes []byte
	var err error

	// Try different possible paths where prompt.txt might be located
	pathsToTry := []string{
		"../prompt.txt",      // From handlers directory when running from main
		"../../prompt.txt",   // From handlers directory when running from deeper path
		"./prompt.txt",       // Current directory
		"prompt.txt",         // Project root when running from project root
	}

	for _, path := range pathsToTry {
		promptBytes, err = os.ReadFile(path)
		if err == nil {
			log.Printf("Successfully loaded prompt.txt from: %s", path)
			break
		}
	}

	if err != nil {
		log.Printf("Failed to read prompt.txt from any path: %v, using empty prompt", err)
		promptBytes = []byte("")
	}

	prompt := string(promptBytes)

	// Initialize callback storage
	callbackStorage, err := storage.NewCallbackStorage("callbacks.json")
	if err != nil {
		log.Printf("Warning: Failed to initialize callback storage: %v", err)
	}

	// Initialize Joomla client
	joomlaClient := joomla.NewJoomlaClient()

	return &MessageHandler{
		Bot:             bot,
		MessageStore:    NewMessageStorage(),
		AdminUserID:     adminUserID,
		AdminChatID:     adminChatID,
		TrackedChatID:   trackedChatID,
		AIAnalyzer:      ai.NewOpenRouterConfig(prompt),
		CallbackStorage: callbackStorage,
		JoomlaClient:    joomlaClient,
	}
}

// Handle processes incoming messages and responds accordingly
func (h *MessageHandler) Handle(update schemes.UpdateInterface) error {
	// Handle callback queries from inline keyboard buttons
	if update.GetUpdateType() == schemes.TypeMessageCallback {
		return h.handleCallbackQuery(update)
	}

	// Handle regular messages
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

		// Perform AI analysis if this is the tracked chat
		if msg.Recipient.ChatId == h.TrackedChatID {
			go h.analyzeMessageWithAI(msg)
		}
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

		// If this is the admin and the admin chat ID wasn't set via environment variable, update it for future notifications
		if msg.Sender.UserId == h.AdminUserID && h.AdminChatID == 0 {
			h.AdminChatID = msg.Recipient.ChatId
			log.Printf("Updated admin chat ID to %d", h.AdminChatID)
		}

		return h.handlePrivateMessage(msg)
	}

	log.Printf("Message from group/channel %d processed, no response needed", msg.Recipient.ChatId)
	// For group chats, just store the message and don't respond (unless mentioned, handled above)
	return nil
}

// isEmptyMessage checks if the message is empty or contains only whitespace
func (h *MessageHandler) isEmptyMessage(text string) bool {
	trimmed := strings.TrimSpace(text)
	return len(trimmed) == 0
}

// handleCallbackQuery handles callback queries from inline keyboard buttons
func (h *MessageHandler) handleCallbackQuery(update schemes.UpdateInterface) error {
	callbackUpdate, ok := update.(*schemes.MessageCallbackUpdate)
	if !ok {
		return fmt.Errorf("could not cast update to MessageCallbackUpdate")
	}

	payload := callbackUpdate.Callback.Payload
	chatID := callbackUpdate.Message.Recipient.ChatId
	userID := callbackUpdate.Callback.User.UserId
	callbackID := callbackUpdate.Callback.CallbackID

	log.Printf("Received callback - UserID: %d, ChatID: %d, CallbackID: %s, Payload: %s", userID, chatID, callbackID, payload)

	// Check if this callback has already been processed
	if h.CallbackStorage != nil && h.CallbackStorage.Exists(callbackID) {
		result, _ := h.CallbackStorage.GetResult(callbackID)

		// Send message informing that this callback was already processed
		var responseText string
		if result.Payload == "accept" {
			responseText = "ℹ️ Этот ответ уже был обработан: ✅ Принято! Обработка правки страницы на сайте..."
		} else {
			responseText = "ℹ️ Этот ответ уже был обработан: ❌ Действие отменено"
		}

		log.Printf("Callback %s already processed with payload '%s', informing user", callbackID, result.Payload)
		return h.sendSimpleResponse(chatID, responseText)
	}

	// Handle callback based on payload
	switch payload {
	case "accept":
		log.Printf("User %d accepted the changes", userID)

		// Извлекаем анализ из текста сообщения
		log.Printf("Extracting analysis from message: %s...", callbackUpdate.Message.Body.Text[:50])
		analysis := h.extractAnalysisFromMessage(callbackUpdate.Message.Body.Text)
		
		if analysis == nil {
			log.Printf("Failed to extract analysis from message")
			h.sendSimpleResponse(chatID, "❌ Ошибка: не удалось извлечь данные из сообщения")
			return nil
		}
		
		log.Printf("Extracted analysis: FullName=%s, Status=%s, AbsenceType=%s", 
			analysis.Employee.FullName, analysis.Status, analysis.AbsenceType)

		// Анализируем изменения ещё раз (быстрый путь)
		log.Printf("Running Joomla analysis...")
		joomlaResponse, err := h.JoomlaClient.Analyze(*analysis)
		if err != nil {
			log.Printf("Joomla analysis error: %v", err)
			h.sendSimpleResponse(chatID, fmt.Sprintf("❌ Ошибка анализа: %v", err))
			return nil
		}

		log.Printf("Joomla analysis result: %d changes, success=%v", len(joomlaResponse.Changes), joomlaResponse.Success)

		if joomlaResponse == nil || (len(joomlaResponse.Changes) == 0 && !joomlaResponse.Success) {
			h.sendSimpleResponse(chatID, "⚠️ Нет изменений для применения")
			return nil
		}

		// Применяем изменения
		log.Printf("Applying %d planned changes", len(joomlaResponse.Changes))
		h.sendSimpleResponse(chatID, "⏳ Начинаю обработку изменений на сайте...")

		response, err := h.JoomlaClient.Apply(*analysis, joomlaResponse.Changes)
		if err != nil {
			log.Printf("Joomla apply error: %v", err)
			h.sendSimpleResponse(chatID, fmt.Sprintf("❌ Ошибка: %v", err))
			return nil
		}

		log.Printf("Joomla apply result: updated=%v, errors=%v", response.UpdatedArticles, response.Errors)

		if response.Success && len(response.UpdatedArticles) > 0 {
			msg := fmt.Sprintf("✅ Успешно обновлено статей: %d\nСтатьи: %v",
				len(response.UpdatedArticles), response.UpdatedArticles)
			log.Printf("Success: %s", msg)
			h.sendSimpleResponse(chatID, msg)
		} else if response.Success {
			log.Printf("Success: no changes needed")
			h.sendSimpleResponse(chatID, "✅ Изменений не потребовалось (статус Продолжение)")
		} else {
			errorsText := strings.Join(response.Errors, "\n")
			if len(response.UpdatedArticles) > 0 {
				msg := fmt.Sprintf("⚠️ Частично выполнено.\nОбновлено статей: %d\nОшибки:\n%s",
					len(response.UpdatedArticles), errorsText)
				h.sendSimpleResponse(chatID, msg)
			} else {
				msg := fmt.Sprintf("❌ Ошибка обновления:\n%s", errorsText)
				h.sendSimpleResponse(chatID, msg)
			}
		}

		// Save the result
		if h.CallbackStorage != nil {
			result := storage.CallbackResult{
				CallbackID:     callbackID,
				UserID:         userID,
				ChatID:         chatID,
				Payload:        payload,
				MessageText:    callbackUpdate.Message.Body.Text,
				ProcessedAt:    time.Now(),
				Analysis:       analysis,
				PlannedChanges: joomlaResponse.Changes,
			}
			if err := h.CallbackStorage.AddResult(result); err != nil {
				log.Printf("Error saving callback result: %v", err)
			}
		}

		return nil

	case "cancel":
		log.Printf("User %d cancelled the action", userID)

		// Save the result to storage
		if h.CallbackStorage != nil {
			result := storage.CallbackResult{
				CallbackID:  callbackID,
				UserID:      userID,
				ChatID:      chatID,
				Payload:     payload,
				MessageText: callbackUpdate.Message.Body.Text,
				ProcessedAt: time.Now(),
			}
			if err := h.CallbackStorage.AddResult(result); err != nil {
				log.Printf("Error saving callback result: %v", err)
			}
		}

		return h.sendSimpleResponse(chatID, "❌ Действие отменено")

	default:
		log.Printf("Unknown callback payload: %s", payload)
		return nil
	}
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
		helpText := "Доступные команды администратора:\n- /help: Показать это справку\n- /echo [text]: Повторить за вами текст\n- /last: Последние сохраненные сообщения из текущего чата\n- /stat: Статистика сохраненных сообщений с группировкой по чатам\n- /cstat: Статистика обработанных кнопок (принято/отменено)"
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
	case text == "/cstat":
		if h.CallbackStorage == nil {
			return h.sendSimpleResponse(msg.Recipient.ChatId, "Хранилище callback не инициализировано.")
		}
		total, accepted, cancelled := h.CallbackStorage.GetStats()
		responseText := fmt.Sprintf("📊 Статистика обработанных кнопок:\n\n"+
			"Всего обработано: %d\n"+
			"✅ Принято: %d\n"+
			"❌ Отменено: %d\n"+
			"🕒 Последнее обновление: %s",
			total, accepted, cancelled, time.Now().Format("02.01.2006 15:04"))
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

// analyzeMessageWithAI performs AI analysis on a message if it's from the tracked chat
func (h *MessageHandler) analyzeMessageWithAI(msg schemes.Message) {
	if h.AIAnalyzer == nil {
		log.Printf("AI analyzer not initialized, skipping analysis for message: %s", msg.Body.Text)
		return
	}

	// Skip empty messages
	if h.isEmptyMessage(msg.Body.Text) {
		log.Printf("Skipping empty message analysis for chat %d", msg.Recipient.ChatId)
		return
	}

	log.Printf("Analyzing message with AI: %s", msg.Body.Text)
	analyses, err := h.AIAnalyzer.AnalyzeMessage(msg.Body.Text)
	if err != nil {
		log.Printf("Error analyzing message with AI: %v", err)
		return
	}

	log.Printf("AI Analysis result: %+v", analyses)

	// Process each analysis result (can be multiple employees)
	for _, analysis := range analyses {
		// Only send to admin if the message is valid (contains absence information)
		if analysis.IsValid {
			// Convert ai.MessageAnalysis to joomla.AnalysisResult
			joomlaAnalysis := convertToJoomlaAnalysis(analysis)
			
			// Analyze changes in Joomla
			var plannedChanges []joomla.Change
			var joomlaError error

			if h.JoomlaClient != nil {
				joomlaResponse, err := h.JoomlaClient.Analyze(joomlaAnalysis)
				if err != nil {
					log.Printf("Error analyzing Joomla changes: %v", err)
					joomlaError = err
				} else if joomlaResponse != nil {
					plannedChanges = joomlaResponse.Changes
					if !joomlaResponse.Success && len(joomlaResponse.Errors) > 0 {
						log.Printf("Joomla analysis errors: %v", joomlaResponse.Errors)
					}
				}
			}

			// Send the analysis result to the admin user's private chat
			if h.AdminUserID != 0 {
				// Use the admin's chat ID from environment variable or previously set value
				adminChatID := h.AdminChatID

				// Format the analysis result in a readable way with markdown and emojis
				formattedMessage := h.formatAnalysisMessage(joomlaAnalysis, plannedChanges, joomlaError)

				// Only send if we have a valid chat ID
				if adminChatID != 0 {
					buttons := [][]bot.InlineKeyboardButton{
						{
							{Text: "Принять", Data: "accept"},
							{Text: "Отмена", Data: "cancel"},
						},
					}
					messageID, err := h.Bot.SendMessageWithKeyboard(adminChatID, formattedMessage, buttons)
					if err != nil {
						log.Printf("Error sending formatted AI analysis to admin's chat %d: %v", adminChatID, err)
					} else {
						log.Printf("Successfully sent formatted AI analysis to admin's chat %d with message ID: %s for employee: %s", adminChatID, messageID, analysis.Employee.FullName)

						// Save the analysis and planned changes to callback storage
						// We use the message ID as the callback key
						if h.CallbackStorage != nil {
							// Store analysis and planned changes for later use when user clicks "accept"
							result := storage.CallbackResult{
								CallbackID:     messageID, // Will be matched with callback_id
								UserID:         0,         // Will be set when user clicks
								ChatID:         adminChatID,
								Payload:        "",        // Will be set when user clicks
								MessageText:    formattedMessage,
								ProcessedAt:    time.Now(),
								Analysis:       &joomlaAnalysis,
								PlannedChanges: plannedChanges,
							}
							// Save immediately so we can retrieve it when user clicks
							if err := h.CallbackStorage.AddResult(result); err != nil {
								log.Printf("Error saving callback result with analysis: %v", err)
							} else {
								log.Printf("Saved analysis for message %s with %d planned changes", messageID, len(plannedChanges))
							}
						}
					}
				} else {
					log.Printf("Cannot send AI analysis to admin: ADMIN_CHAT_ID not set in environment variables")
				}
			}
		}
	}

	// Check if all analyses were invalid
	allInvalid := true
	for _, analysis := range analyses {
		if analysis.IsValid {
			allInvalid = false
			break
		}
	}
	if allInvalid && len(analyses) > 0 {
		log.Printf("Message is not valid for absence tracking, not sending to admin")
	}
}

// formatAnalysisMessage formats the analysis result and planned changes into a readable message
func (h *MessageHandler) formatAnalysisMessage(analysis joomla.AnalysisResult, changes []joomla.Change, joomlaErr error) string {
	var sb strings.Builder
	
	sb.WriteString("📋 НОВАЯ ЗАПИСЬ\n\n")
	sb.WriteString(fmt.Sprintf("🏥 Тип: %s\n", analysis.AbsenceType))
	sb.WriteString(fmt.Sprintf("📊 Статус: %s\n", analysis.Status))
	sb.WriteString(fmt.Sprintf("💼 Должность: %s\n", analysis.Employee.Position))
	sb.WriteString(fmt.Sprintf("👤 ФИО: %s\n", analysis.Employee.FullName))
	sb.WriteString(fmt.Sprintf("📅 Дата начала: %s\n", analysis.Dates.StartDate))
	
	endDate := analysis.Dates.EndDate
	if endDate == "" {
		endDate = "⏳"
	}
	sb.WriteString(fmt.Sprintf("🔚 Дата окончания: %s\n", endDate))
	sb.WriteString(fmt.Sprintf("💬 Оригинальное сообщение:\n> %s\n", analysis.OriginalMessage))
	
	// Add Joomla changes info
	sb.WriteString("\n━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString("🔧 ПЛАНИРУЕМЫЕ ИЗМЕНЕНИЯ:\n\n")
	
	if joomlaErr != nil {
		sb.WriteString(fmt.Sprintf("❌ Ошибка анализа сайта: %v\n", joomlaErr))
	} else if len(changes) == 0 {
		if analysis.Status == "Продолжение" {
			sb.WriteString("ℹ️ Статус \"Продолжение\" - изменения не требуются\n")
		} else {
			sb.WriteString("⚠️ Врач не найден в статьях сайта\n")
		}
	} else {
		for i, change := range changes {
			sb.WriteString(fmt.Sprintf("%d. Статья #%d\n", i+1, change.ArticleID))
			sb.WriteString(fmt.Sprintf("   Врач: %s\n", change.Doctor))
			if change.Action == "add" {
				sb.WriteString("   Действие: ➕ Добавить пометку\n")
			} else if change.Action == "remove" {
				sb.WriteString("   Действие: ➖ Убрать пометку\n")
			}
			sb.WriteString("\n")
		}
	}
	
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString("❓ Внести изменения на сайт?\n")
	sb.WriteString("✅ Да / ❌ Нет")
	
	return sb.String()
}

// saveCallbackWithAnalysis saves callback result with analysis and planned changes
func (h *MessageHandler) saveCallbackWithAnalysis(callbackID string, userID, chatID int64, payload, messageText string, analysis *joomla.AnalysisResult, changes []joomla.Change) {
	if h.CallbackStorage == nil {
		return
	}
	
	result := storage.CallbackResult{
		CallbackID:     callbackID,
		UserID:         userID,
		ChatID:         chatID,
		Payload:        payload,
		MessageText:    messageText,
		ProcessedAt:    time.Now(),
		Analysis:       analysis,
		PlannedChanges: changes,
	}
	
	if err := h.CallbackStorage.AddResult(result); err != nil {
		log.Printf("Error saving callback result with analysis: %v", err)
	}
}

// convertToJoomlaAnalysis converts ai.MessageAnalysis to joomla.AnalysisResult
func convertToJoomlaAnalysis(msgAnalysis ai.MessageAnalysis) joomla.AnalysisResult {
	return joomla.AnalysisResult{
		OriginalMessage: msgAnalysis.OriginalMessage,
		IsValid:         msgAnalysis.IsValid,
		Employee: joomla.Employee{
			Position: msgAnalysis.Employee.Position,
			FullName: msgAnalysis.Employee.FullName,
		},
		AbsenceType: msgAnalysis.AbsenceType,
		Dates: joomla.Dates{
			StartDate: msgAnalysis.Dates.StartDate,
			EndDate:   msgAnalysis.Dates.EndDate,
		},
		Status:     msgAnalysis.Status,
		Substitute: msgAnalysis.Substitute,
	}
}

// extractAnalysisFromMessage извлекает данные анализа из текста сообщения
func (h *MessageHandler) extractAnalysisFromMessage(text string) *joomla.AnalysisResult {
	// Парсим сообщение вида:
	// 📋 НОВАЯ ЗАПИСЬ
	// 🏥 Тип: Больничный
	// 📊 Статус: Начало
	// 💼 Должность: врач- терапевт участковый
	// 👤 ФИО: Кузьменко М.С
	// 📅 Дата начала: 10.03.26
	// 🔚 Дата окончания: ⏳
	
	analysis := &joomla.AnalysisResult{}
	
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		if strings.HasPrefix(line, "🏥 Тип:") {
			analysis.AbsenceType = strings.TrimSpace(strings.TrimPrefix(line, "🏥 Тип:"))
		} else if strings.HasPrefix(line, "📊 Статус:") {
			analysis.Status = strings.TrimSpace(strings.TrimPrefix(line, "📊 Статус:"))
		} else if strings.HasPrefix(line, "💼 Должность:") {
			analysis.Employee.Position = strings.TrimSpace(strings.TrimPrefix(line, "💼 Должность:"))
		} else if strings.HasPrefix(line, "👤 ФИО:") {
			analysis.Employee.FullName = strings.TrimSpace(strings.TrimPrefix(line, "👤 ФИО:"))
		} else if strings.HasPrefix(line, "📅 Дата начала:") {
			analysis.Dates.StartDate = strings.TrimSpace(strings.TrimPrefix(line, "📅 Дата начала:"))
		} else if strings.HasPrefix(line, "🔚 Дата окончания:") {
			endDate := strings.TrimSpace(strings.TrimPrefix(line, "🔚 Дата окончания:"))
			if endDate != "⏳" {
				analysis.Dates.EndDate = endDate
			}
		}
	}
	
	// Проверяем, что данные извлечены
	if analysis.Employee.FullName == "" {
		return nil
	}
	
	analysis.IsValid = true
	return analysis
}
