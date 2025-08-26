package model

import (
	"time"
)

// NotificationType определяет тип уведомления
type NotificationType string

const (
	// Уведомления по откликам
	NotificationTypeNewResponse      NotificationType = "new_response"      // Новый отклик на заявку
	NotificationTypeResponseAccepted NotificationType = "response_accepted" // Отклик принят
	NotificationTypeResponseRejected NotificationType = "response_rejected" // Отклик отклонен

	// Уведомления по сделкам
	NotificationTypeDealCreated   NotificationType = "deal_created"   // Сделка создана
	NotificationTypeDealConfirmed NotificationType = "deal_confirmed" // Сделка подтверждена одной стороной
	NotificationTypeDealCompleted NotificationType = "deal_completed" // Сделка полностью завершена
	NotificationTypeDealExpiring  NotificationType = "deal_expiring"  // Сделка скоро истекает
	NotificationTypeDealCancelled NotificationType = "deal_cancelled" // Сделка отменена

	// Системные уведомления
	NotificationTypeSystemMessage NotificationType = "system_message" // Системные сообщения
)

// NotificationStatus определяет статус уведомления
type NotificationStatus string

const (
	NotificationStatusPending NotificationStatus = "pending" // Ожидает отправки
	NotificationStatusSent    NotificationStatus = "sent"    // Отправлено успешно
	NotificationStatusFailed  NotificationStatus = "failed"  // Ошибка отправки
)

// Notification представляет уведомление пользователю
type Notification struct {
	ID          int64                  `json:"id" db:"id"`                     // Уникальный идентификатор уведомления
	UserID      int64                  `json:"user_id" db:"user_id"`           // ID пользователя для отправки
	TelegramID  int64                  `json:"telegram_id" db:"telegram_id"`   // Telegram ID пользователя
	Type        NotificationType       `json:"type" db:"type"`                 // Тип уведомления
	Status      NotificationStatus     `json:"status" db:"status"`             // Статус уведомления
	Title       string                 `json:"title" db:"title"`               // Заголовок уведомления
	Message     string                 `json:"message" db:"message"`           // Текст сообщения
	Data        map[string]interface{} `json:"data" db:"data"`                 // Дополнительные данные (JSON)
	OrderID     *int64                 `json:"order_id" db:"order_id"`         // ID связанной заявки
	ResponseID  *int64                 `json:"response_id" db:"response_id"`   // ID связанного отклика
	DealID      *int64                 `json:"deal_id" db:"deal_id"`           // ID связанной сделки
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`     // Дата создания
	SentAt      *time.Time             `json:"sent_at" db:"sent_at"`           // Дата отправки
	FailedAt    *time.Time             `json:"failed_at" db:"failed_at"`       // Дата неудачной попытки
	RetryCount  int                    `json:"retry_count" db:"retry_count"`   // Количество повторных попыток
	ErrorReason string                 `json:"error_reason" db:"error_reason"` // Причина ошибки отправки
}

// NotificationTemplate содержит шаблон для генерации уведомлений
type NotificationTemplate struct {
	Type        NotificationType `json:"type"`        // Тип уведомления
	Title       string           `json:"title"`       // Шаблон заголовка
	Message     string           `json:"message"`     // Шаблон сообщения
	Description string           `json:"description"` // Описание шаблона
}

// TelegramMessage представляет структуру для отправки сообщения в Telegram
type TelegramMessage struct {
	ChatID                int64                   `json:"chat_id"`                            // ID чата получателя
	Text                  string                  `json:"text"`                               // Текст сообщения
	ParseMode             string                  `json:"parse_mode,omitempty"`               // Режим парсинга (HTML, Markdown)
	DisableWebPagePreview bool                    `json:"disable_web_page_preview,omitempty"` // Отключить превью ссылок
	DisableNotification   bool                    `json:"disable_notification,omitempty"`     // Тихое уведомление
	ReplyMarkup           *TelegramInlineKeyboard `json:"reply_markup,omitempty"`             // Inline клавиатура
}

// TelegramInlineKeyboard представляет inline клавиатуру
type TelegramInlineKeyboard struct {
	InlineKeyboard [][]TelegramInlineKeyboardButton `json:"inline_keyboard"` // Ряды кнопок
}

// TelegramInlineKeyboardButton представляет кнопку inline клавиатуры
type TelegramInlineKeyboardButton struct {
	Text         string              `json:"text"`                    // Текст кнопки
	URL          string              `json:"url,omitempty"`           // Ссылка для перехода (открывается в браузере)
	WebApp       *TelegramWebAppInfo `json:"web_app,omitempty"`       // WebApp для открытия в Telegram
	CallbackData string              `json:"callback_data,omitempty"` // Данные для callback
}

// TelegramWebAppInfo представляет информацию о веб-приложении Telegram
type TelegramWebAppInfo struct {
	URL string `json:"url"` // URL веб-приложения для открытия в Telegram
}

// NotificationQueue представляет очередь уведомлений для отправки
type NotificationQueue struct {
	Notifications []*Notification `json:"notifications"` // Список уведомлений
	ProcessedAt   time.Time       `json:"processed_at"`  // Время последней обработки
}

// CreateNotificationRequest содержит данные для создания уведомления
type CreateNotificationRequest struct {
	UserID     int64                  `json:"user_id" validate:"required"` // ID пользователя
	Type       NotificationType       `json:"type" validate:"required"`    // Тип уведомления
	Title      string                 `json:"title" validate:"required"`   // Заголовок
	Message    string                 `json:"message" validate:"required"` // Сообщение
	Data       map[string]interface{} `json:"data,omitempty"`              // Дополнительные данные
	OrderID    *int64                 `json:"order_id,omitempty"`          // ID заявки
	ResponseID *int64                 `json:"response_id,omitempty"`       // ID отклика
	DealID     *int64                 `json:"deal_id,omitempty"`           // ID сделки
}
