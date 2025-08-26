package model

import (
	"time"
)

// ResponseStatus определяет статус отклика на заявку
type ResponseStatus string

const (
	ResponseStatusWaiting  ResponseStatus = "waiting"  // Ожидает рассмотрения автором заявки
	ResponseStatusAccepted ResponseStatus = "accepted" // Принят автором заявки
	ResponseStatusRejected ResponseStatus = "rejected" // Отклонен автором заявки
)

// Response представляет отклик пользователя на заявку
// Это промежуточный этап между заявкой и сделкой
type Response struct {
	ID         int64          `json:"id" db:"id"`                   // Уникальный идентификатор отклика
	OrderID    int64          `json:"order_id" db:"order_id"`       // ID заявки на которую откликнулись
	UserID     int64          `json:"user_id" db:"user_id"`         // ID пользователя который откликнулся
	Message    string         `json:"message" db:"message"`         // Сообщение от откликающегося
	Status     ResponseStatus `json:"status" db:"status"`           // Статус отклика
	CreatedAt  time.Time      `json:"created_at" db:"created_at"`   // Время создания отклика
	UpdatedAt  time.Time      `json:"updated_at" db:"updated_at"`   // Время последнего обновления
	ReviewedAt *time.Time     `json:"reviewed_at" db:"reviewed_at"` // Время рассмотрения автором (null если еще не рассмотрен)
	
	// Дополнительные поля для фронтенда (не сохраняются в БД)
	UserName     string `json:"user_name,omitempty"`     // Полное имя откликнувшегося
	Username     string `json:"username,omitempty"`      // Telegram username откликнувшегося
	AuthorName   string `json:"author_name,omitempty"`   // Полное имя автора заявки
	AuthorUsername string `json:"author_username,omitempty"` // Telegram username автора заявки
	OrderType    string `json:"order_type,omitempty"`    // Тип заявки (buy/sell)
	Cryptocurrency string `json:"cryptocurrency,omitempty"` // Криптовалюта
	FiatCurrency   string `json:"fiat_currency,omitempty"`  // Фиатная валюта
	Amount         float64 `json:"amount,omitempty"`        // Объем
	Price          float64 `json:"price,omitempty"`         // Цена
	TotalAmount    float64 `json:"total_amount,omitempty"`  // Общая сумма
}

// CreateResponseRequest содержит данные для создания нового отклика
type CreateResponseRequest struct {
	OrderID int64  `json:"order_id" validate:"required"` // ID заявки (обязательно)
	Message string `json:"message" validate:"max=500"`   // Сообщение (макс 500 символов)
}

// ResponseWithDetails содержит отклик с дополнительной информацией о заявке и пользователе
type ResponseWithDetails struct {
	Response *Response `json:"response"` // Основные данные отклика
	Order    *Order    `json:"order"`    // Информация о заявке
	User     *User     `json:"user"`     // Информация о пользователе который откликнулся
}

// ResponseFilter содержит параметры для фильтрации откликов
type ResponseFilter struct {
	OrderID       *int64          `json:"order_id"`       // Фильтр по заявке
	UserID        *int64          `json:"user_id"`        // Фильтр по пользователю (кто откликнулся)
	AuthorID      *int64          `json:"author_id"`      // Фильтр по автору заявки (для кого отклики)
	Status        *ResponseStatus `json:"status"`         // Фильтр по статусу
	CreatedAfter  *time.Time      `json:"created_after"`  // Созданы после даты
	CreatedBefore *time.Time      `json:"created_before"` // Созданы до даты
	SortBy        string          `json:"sort_by"`        // Сортировка (created_at, updated_at)
	SortOrder     string          `json:"sort_order"`     // Порядок сортировки (asc, desc)
	Limit         int             `json:"limit"`          // Лимит результатов
	Offset        int             `json:"offset"`         // Смещение для пагинации
}
