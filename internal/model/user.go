package model

import (
	"time"
)

// User представляет модель пользователя в системе P2P биржи
// Каждый пользователь авторизуется через Telegram и имеет свой профиль
type User struct {
	ID              int64     `json:"id" db:"id"`                             // Уникальный идентификатор пользователя
	TelegramID      int64     `json:"telegram_id" db:"telegram_id"`           // ID пользователя в Telegram
	TelegramUserID  string    `json:"telegram_user_id" db:"telegram_user_id"` // Username в Telegram (@username)
	FirstName       string    `json:"first_name" db:"first_name"`             // Имя из Telegram профиля
	LastName        string    `json:"last_name" db:"last_name"`               // Фамилия из Telegram профиля
	Username        string    `json:"username" db:"username"`                 // Username из Telegram профиля
	PhotoURL        string    `json:"photo_url" db:"photo_url"`               // URL фото профиля из Telegram
	IsBot           bool      `json:"is_bot" db:"is_bot"`                     // Флаг бота (должен быть false)
	LanguageCode    string    `json:"language_code" db:"language_code"`       // Код языка пользователя
	CreatedAt       time.Time `json:"created_at" db:"created_at"`             // Дата создания аккаунта
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`             // Дата последнего обновления
	IsActive        bool      `json:"is_active" db:"is_active"`               // Активен ли пользователь
	Rating          float32   `json:"rating" db:"rating"`                     // Средний рейтинг пользователя (0-5 звезд)
	TotalDeals      int       `json:"total_deals" db:"total_deals"`           // Общее количество завершенных сделок
	SuccessfulDeals int       `json:"successful_deals" db:"successful_deals"` // Количество успешных сделок
	ChatMember      bool      `json:"chat_member" db:"chat_member"`           // Является ли членом закрытого чата
}

// UserProfile содержит расширенную информацию о пользователе
type UserProfile struct {
	UserID      int64  `json:"user_id" db:"user_id"`           // ID пользователя (внешний ключ)
	Bio         string `json:"bio" db:"bio"`                   // Краткая биография пользователя
	Location    string `json:"location" db:"location"`         // Местоположение пользователя
	PhoneNumber string `json:"phone_number" db:"phone_number"` // Номер телефона (необязательно)
	Email       string `json:"email" db:"email"`               // Email адрес (необязательно)
	IsVerified  bool   `json:"is_verified" db:"is_verified"`   // Верифицирован ли пользователь
}

// TelegramAuthData содержит данные для авторизации через Telegram WebApp
// Эта структура используется для валидации данных от Telegram WebApp
type TelegramAuthData struct {
	ID        int64  `json:"id"`         // Telegram User ID
	FirstName string `json:"first_name"` // Имя пользователя
	LastName  string `json:"last_name"`  // Фамилия пользователя (может быть пустой)
	Username  string `json:"username"`   // Username пользователя (может быть пустым)
	PhotoURL  string `json:"photo_url"`  // URL фото профиля
	AuthDate  int64  `json:"auth_date"`  // Unix timestamp авторизации
	Hash      string `json:"hash"`       // Хеш для валидации данных
}
