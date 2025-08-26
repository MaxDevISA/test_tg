package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"p2pTG-crypto-exchange/internal/model"
)

// NotificationService представляет сервис для работы с уведомлениями
// Обрабатывает создание, отправку и управление уведомлениями пользователей
type NotificationService struct {
	telegramToken string                                                 // Токен Telegram бота для отправки сообщений
	httpClient    *http.Client                                           // HTTP клиент для запросов к Telegram Bot API
	templates     map[model.NotificationType]*model.NotificationTemplate // Шаблоны уведомлений
	webAppURL     string                                                 // URL веб-приложения для создания кнопок
}

// NewNotificationService создает новый экземпляр сервиса уведомлений
func NewNotificationService(telegramToken, webAppURL string) *NotificationService {
	log.Println("[INFO] Инициализация сервиса уведомлений")

	service := &NotificationService{
		telegramToken: telegramToken,
		httpClient:    &http.Client{Timeout: 10 * time.Second}, // HTTP клиент с таймаутом 10 сек
		templates:     make(map[model.NotificationType]*model.NotificationTemplate),
		webAppURL:     webAppURL,
	}

	// Инициализируем шаблоны уведомлений
	service.initTemplates()

	return service
}

// initTemplates инициализирует шаблоны уведомлений для разных типов событий
func (ns *NotificationService) initTemplates() {
	log.Println("[INFO] Инициализация шаблонов уведомлений")

	// Шаблон для нового отклика на заявку
	ns.templates[model.NotificationTypeNewResponse] = &model.NotificationTemplate{
		Type:        model.NotificationTypeNewResponse,
		Title:       "🔔 Новый отклик на вашу заявку",
		Message:     "На вашу заявку %s %s %s по курсу %.2f %s откликнулся пользователь %s.\n\n💬 Сообщение: \"%s\"\n\n📊 Объем сделки: %.8f %s (%.2f %s)",
		Description: "Уведомление автору заявки о новом отклике",
	}

	// Шаблон для принятого отклика
	ns.templates[model.NotificationTypeResponseAccepted] = &model.NotificationTemplate{
		Type:        model.NotificationTypeResponseAccepted,
		Title:       "✅ Ваш отклик принят!",
		Message:     "Отличные новости! Автор заявки %s принял ваш отклик на %s %s %s.\n\n💰 Сумма сделки: %.8f %s (%.2f %s)\n\n🚀 Необходимо перейти в приложение и начать процесс сделки.",
		Description: "Уведомление участнику о принятии его отклика",
	}

	// Шаблон для отклоненного отклика
	ns.templates[model.NotificationTypeResponseRejected] = &model.NotificationTemplate{
		Type:        model.NotificationTypeResponseRejected,
		Title:       "❌ Отклик отклонен",
		Message:     "К сожалению, автор заявки %s отклонил ваш отклик на %s %s %s.\n\n📝 Не расстраивайтесь, попробуйте откликнуться на другие заявки!",
		Description: "Уведомление участнику об отклонении его отклика",
	}

	// Шаблон для созданной сделки
	ns.templates[model.NotificationTypeDealCreated] = &model.NotificationTemplate{
		Type:        model.NotificationTypeDealCreated,
		Title:       "🤝 Сделка создана",
		Message:     "Создана новая сделка #%d между вами и %s.\n\n📋 Детали:\n• %s %s %s\n• Объем: %.8f %s\n• Курс: %.2f %s\n• Сумма: %.2f %s\n\n⏰ У вас есть время для завершения сделки. Переходите в приложение для продолжения.",
		Description: "Уведомление участникам о создании сделки",
	}

	// Шаблон для подтверждения сделки
	ns.templates[model.NotificationTypeDealConfirmed] = &model.NotificationTemplate{
		Type:        model.NotificationTypeDealConfirmed,
		Title:       "✔️ Сделка подтверждена",
		Message:     "%s подтвердил свою часть сделки #%d.\n\n📋 Статус:\n• %s ✅ Подтверждено\n• %s ⏳ Ожидается подтверждение\n\n💡 Проверьте детали сделки и подтвердите получение/отправку платежа.",
		Description: "Уведомление о подтверждении сделки одной стороной",
	}

	// Шаблон для завершенной сделки
	ns.templates[model.NotificationTypeDealCompleted] = &model.NotificationTemplate{
		Type:        model.NotificationTypeDealCompleted,
		Title:       "🎉 Сделка успешно завершена!",
		Message:     "Поздравляем! Сделка #%d успешно завершена.\n\n📊 Итоги:\n• Объем: %.8f %s\n• Сумма: %.2f %s\n• Участники: %s и %s\n\n⭐ Не забудьте оставить отзыв о сделке для повышения рейтинга!",
		Description: "Уведомление об успешном завершении сделки",
	}

	// Шаблон для системных сообщений
	ns.templates[model.NotificationTypeSystemMessage] = &model.NotificationTemplate{
		Type:        model.NotificationTypeSystemMessage,
		Title:       "🔧 Системное уведомление",
		Message:     "%s",
		Description: "Системные сообщения от администрации",
	}

	log.Printf("[INFO] Загружено шаблонов уведомлений: %d", len(ns.templates))
}

// CreateNotification создает новое уведомление для пользователя
func (ns *NotificationService) CreateNotification(req *model.CreateNotificationRequest) (*model.Notification, error) {
	log.Printf("[INFO] Создание уведомления типа %s для пользователя ID=%d", req.Type, req.UserID)

	// Валидируем тип уведомления
	template, exists := ns.templates[req.Type]
	if !exists {
		log.Printf("[WARN] Неизвестный тип уведомления: %s", req.Type)
		return nil, fmt.Errorf("неподдерживаемый тип уведомления: %s", req.Type)
	}

	// Создаем уведомление
	notification := &model.Notification{
		UserID:     req.UserID,
		Type:       req.Type,
		Status:     model.NotificationStatusPending,
		Title:      req.Title,
		Message:    req.Message,
		Data:       req.Data,
		OrderID:    req.OrderID,
		ResponseID: req.ResponseID,
		DealID:     req.DealID,
		CreatedAt:  time.Now(),
		RetryCount: 0,
	}

	// Если заголовок или сообщение не переданы, используем шаблон
	if notification.Title == "" {
		notification.Title = template.Title
	}
	if notification.Message == "" {
		notification.Message = template.Message
	}

	log.Printf("[INFO] Уведомление создано: Type=%s, Title=%s", notification.Type, notification.Title)
	return notification, nil
}

// SendNotification отправляет уведомление в Telegram
func (ns *NotificationService) SendNotification(notification *model.Notification, userTelegramID int64) error {
	log.Printf("[INFO] Отправка уведомления ID=%d пользователю TelegramID=%d",
		notification.ID, userTelegramID)

	// Создаем сообщение для Telegram
	message := &model.TelegramMessage{
		ChatID:                userTelegramID,
		Text:                  ns.formatNotificationMessage(notification),
		ParseMode:             "HTML", // Используем HTML разметку для форматирования
		DisableWebPagePreview: true,   // Отключаем превью ссылок
		DisableNotification:   false,  // Включаем звуковое уведомление
	}

	// Добавляем inline клавиатуру с кнопками
	message.ReplyMarkup = ns.createInlineKeyboard(notification)

	// Отправляем сообщение через Telegram Bot API
	if err := ns.sendTelegramMessage(message); err != nil {
		log.Printf("[ERROR] Не удалось отправить уведомление: %v", err)
		return fmt.Errorf("ошибка отправки уведомления: %w", err)
	}

	log.Printf("[INFO] Уведомление успешно отправлено пользователю TelegramID=%d", userTelegramID)
	return nil
}

// formatNotificationMessage форматирует текст уведомления для Telegram
func (ns *NotificationService) formatNotificationMessage(notification *model.Notification) string {
	var builder strings.Builder

	// Добавляем заголовок с форматированием
	builder.WriteString(fmt.Sprintf("<b>%s</b>\n\n", notification.Title))

	// Добавляем основное сообщение
	builder.WriteString(notification.Message)

	// Добавляем временную метку
	builder.WriteString(fmt.Sprintf("\n\n<i>📅 %s</i>",
		notification.CreatedAt.Format("02.01.2006 15:04")))

	return builder.String()
}

// createInlineKeyboard создает inline клавиатуру для уведомления
func (ns *NotificationService) createInlineKeyboard(notification *model.Notification) *model.TelegramInlineKeyboard {
	var buttons [][]model.TelegramInlineKeyboardButton

	switch notification.Type {
	case model.NotificationTypeNewResponse:
		// Кнопки для автора заявки: "Посмотреть отклики", "Перейти в приложение"
		buttons = [][]model.TelegramInlineKeyboardButton{
			{
				{Text: "📋 Посмотреть отклики", WebApp: &model.TelegramWebAppInfo{URL: fmt.Sprintf("%s/#responses", ns.webAppURL)}},
			},
			{
				{Text: "🚀 Открыть приложение", WebApp: &model.TelegramWebAppInfo{URL: ns.webAppURL}},
			},
		}

	case model.NotificationTypeResponseAccepted, model.NotificationTypeDealCreated:
		// Кнопки для участника: "Перейти к сделке", "Мои сделки"
		buttons = [][]model.TelegramInlineKeyboardButton{
			{
				{Text: "🤝 Перейти к сделке", WebApp: &model.TelegramWebAppInfo{URL: fmt.Sprintf("%s/#my-orders", ns.webAppURL)}},
			},
			{
				{Text: "📊 Мои сделки", WebApp: &model.TelegramWebAppInfo{URL: fmt.Sprintf("%s/#my-orders", ns.webAppURL)}},
			},
		}

	case model.NotificationTypeResponseRejected:
		// Кнопки для отклоненного участника: "Найти другие заявки"
		buttons = [][]model.TelegramInlineKeyboardButton{
			{
				{Text: "🔍 Найти другие заявки", WebApp: &model.TelegramWebAppInfo{URL: fmt.Sprintf("%s/#orders", ns.webAppURL)}},
			},
			{
				{Text: "🚀 Открыть приложение", WebApp: &model.TelegramWebAppInfo{URL: ns.webAppURL}},
			},
		}

	case model.NotificationTypeDealConfirmed, model.NotificationTypeDealCompleted:
		// Кнопки для сделки: "Перейти к сделке", "Оставить отзыв"
		buttons = [][]model.TelegramInlineKeyboardButton{
			{
				{Text: "🤝 Перейти к сделке", WebApp: &model.TelegramWebAppInfo{URL: fmt.Sprintf("%s/#my-orders", ns.webAppURL)}},
			},
		}

		// Добавляем кнопку отзыва только для завершенных сделок
		if notification.Type == model.NotificationTypeDealCompleted {
			buttons = append(buttons, []model.TelegramInlineKeyboardButton{
				{Text: "⭐ Оставить отзыв", WebApp: &model.TelegramWebAppInfo{URL: fmt.Sprintf("%s/#profile", ns.webAppURL)}},
			})
		}

	default:
		// Универсальная кнопка для всех остальных типов
		buttons = [][]model.TelegramInlineKeyboardButton{
			{
				{Text: "🚀 Открыть приложение", WebApp: &model.TelegramWebAppInfo{URL: ns.webAppURL}},
			},
		}
	}

	return &model.TelegramInlineKeyboard{
		InlineKeyboard: buttons,
	}
}

// sendTelegramMessage отправляет сообщение через Telegram Bot API
func (ns *NotificationService) sendTelegramMessage(message *model.TelegramMessage) error {
	// URL для отправки сообщения через Telegram Bot API
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", ns.telegramToken)

	// Сериализуем сообщение в JSON
	messageData, err := json.Marshal(message)
	if err != nil {
		log.Printf("[ERROR] Ошибка сериализации сообщения Telegram: %v", err)
		return fmt.Errorf("ошибка подготовки сообщения: %w", err)
	}

	// Создаем HTTP запрос
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(messageData))
	if err != nil {
		log.Printf("[ERROR] Ошибка создания HTTP запроса: %v", err)
		return fmt.Errorf("ошибка создания запроса: %w", err)
	}

	// Устанавливаем заголовки
	req.Header.Set("Content-Type", "application/json")

	// Выполняем запрос
	resp, err := ns.httpClient.Do(req)
	if err != nil {
		log.Printf("[ERROR] Ошибка выполнения HTTP запроса к Telegram API: %v", err)
		return fmt.Errorf("ошибка отправки запроса: %w", err)
	}
	defer resp.Body.Close()

	// Проверяем код ответа
	if resp.StatusCode != http.StatusOK {
		log.Printf("[ERROR] Telegram API вернул код ошибки: %d", resp.StatusCode)

		// Читаем тело ответа для диагностики
		var errorResponse map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&errorResponse); err == nil {
			log.Printf("[ERROR] Ответ Telegram API: %+v", errorResponse)
		}

		return fmt.Errorf("Telegram API вернул ошибку: код %d", resp.StatusCode)
	}

	// Парсим успешный ответ
	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Printf("[WARN] Не удалось распарсить ответ Telegram API: %v", err)
		// Не считаем это критической ошибкой, сообщение могло быть отправлено
	}

	// Проверяем поле "ok" в ответе
	if ok, exists := response["ok"]; exists {
		if okBool, isBool := ok.(bool); isBool && !okBool {
			log.Printf("[ERROR] Telegram API вернул ok=false: %+v", response)
			return fmt.Errorf("Telegram API не смог обработать сообщение")
		}
	}

	log.Printf("[DEBUG] Сообщение отправлено в Telegram: ChatID=%d, Length=%d",
		message.ChatID, len(message.Text))

	return nil
}

// safeDerefInt64 безопасно разыменовывает указатель на int64
func (ns *NotificationService) safeDerefInt64(ptr *int64) int64 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// GetNotificationTemplates возвращает все доступные шаблоны уведомлений
func (ns *NotificationService) GetNotificationTemplates() map[model.NotificationType]*model.NotificationTemplate {
	return ns.templates
}

// FormatResponseNotification форматирует уведомление о новом отклике
func (ns *NotificationService) FormatResponseNotification(order *model.Order, response *model.Response, responderName string) (string, string) {
	template := ns.templates[model.NotificationTypeNewResponse]

	title := template.Title
	message := fmt.Sprintf(template.Message,
		strings.ToUpper(string(order.Type)), // BUY/SELL
		order.Cryptocurrency,                // BTC
		order.FiatCurrency,                  // RUB
		order.Price,                         // 2850000.00
		order.FiatCurrency,                  // RUB
		responderName,                       // Иван Петров
		response.Message,                    // Сообщение от откликнувшегося
		order.Amount,                        // 0.01000000
		order.Cryptocurrency,                // BTC
		order.TotalAmount,                   // 28500.00
		order.FiatCurrency,                  // RUB
	)

	return title, message
}

// FormatAcceptedResponseNotification форматирует уведомление о принятом отклике
func (ns *NotificationService) FormatAcceptedResponseNotification(order *model.Order, authorName string) (string, string) {
	template := ns.templates[model.NotificationTypeResponseAccepted]

	title := template.Title
	message := fmt.Sprintf(template.Message,
		authorName,                          // Автор заявки
		strings.ToUpper(string(order.Type)), // BUY/SELL
		order.Cryptocurrency,                // BTC
		order.FiatCurrency,                  // RUB
		order.Amount,                        // 0.01000000
		order.Cryptocurrency,                // BTC
		order.TotalAmount,                   // 28500.00
		order.FiatCurrency,                  // RUB
	)

	return title, message
}

// FormatRejectedResponseNotification форматирует уведомление об отклоненном отклике
func (ns *NotificationService) FormatRejectedResponseNotification(order *model.Order, authorName string) (string, string) {
	template := ns.templates[model.NotificationTypeResponseRejected]

	title := template.Title
	message := fmt.Sprintf(template.Message,
		authorName,                          // Автор заявки
		strings.ToUpper(string(order.Type)), // BUY/SELL
		order.Cryptocurrency,                // BTC
		order.FiatCurrency,                  // RUB
	)

	return title, message
}

// FormatDealCreatedNotification форматирует уведомление о созданной сделке
func (ns *NotificationService) FormatDealCreatedNotification(deal *model.Deal, counterpartyName string) (string, string) {
	template := ns.templates[model.NotificationTypeDealCreated]

	title := template.Title
	message := fmt.Sprintf(template.Message,
		deal.ID,                                 // Номер сделки
		counterpartyName,                        // Имя контрагента
		strings.ToUpper(string(deal.OrderType)), // BUY/SELL
		deal.Cryptocurrency,                     // BTC
		deal.FiatCurrency,                       // RUB
		deal.Amount,                             // 0.01000000
		deal.Cryptocurrency,                     // BTC
		deal.Price,                              // 2850000.00
		deal.FiatCurrency,                       // RUB
		deal.TotalAmount,                        // 28500.00
		deal.FiatCurrency,                       // RUB
	)

	return title, message
}

// FormatDealConfirmedNotification форматирует уведомление о подтверждении сделки
func (ns *NotificationService) FormatDealConfirmedNotification(deal *model.Deal, confirmedByName string, waitingForName string) (string, string) {
	template := ns.templates[model.NotificationTypeDealConfirmed]

	title := template.Title
	message := fmt.Sprintf(template.Message,
		confirmedByName, // Кто подтвердил
		deal.ID,         // Номер сделки
		confirmedByName, // Кто подтвердил (повторно)
		waitingForName,  // Кто еще должен подтвердить
	)

	return title, message
}

// FormatDealCompletedNotification форматирует уведомление о завершенной сделке
func (ns *NotificationService) FormatDealCompletedNotification(deal *model.Deal, authorName string, counterpartyName string) (string, string) {
	template := ns.templates[model.NotificationTypeDealCompleted]

	title := template.Title
	message := fmt.Sprintf(template.Message,
		deal.ID,             // Номер сделки
		deal.Amount,         // 0.01000000
		deal.Cryptocurrency, // BTC
		deal.TotalAmount,    // 28500.00
		deal.FiatCurrency,   // RUB
		authorName,          // Имя автора
		counterpartyName,    // Имя контрагента
	)

	return title, message
}
