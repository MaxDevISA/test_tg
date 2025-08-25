package model

import (
	"time"
)

// ReviewType определяет тип отзыва
type ReviewType string

const (
	ReviewTypePositive ReviewType = "positive" // Положительный отзыв
	ReviewTypeNeutral  ReviewType = "neutral"  // Нейтральный отзыв
	ReviewTypeNegative ReviewType = "negative" // Отрицательный отзыв
)

// Review представляет отзыв одного пользователя о другом после совершения сделки
// Отзывы влияют на репутацию и рейтинг пользователей
type Review struct {
	ID            int64      `json:"id" db:"id"`                         // Уникальный идентификатор отзыва
	DealID        int64      `json:"deal_id" db:"deal_id"`               // ID сделки, по которой оставлен отзыв
	FromUserID    int64      `json:"from_user_id" db:"from_user_id"`     // ID пользователя, оставившего отзыв
	ToUserID      int64      `json:"to_user_id" db:"to_user_id"`         // ID пользователя, которому оставлен отзыв
	Rating        int        `json:"rating" db:"rating"`                 // Рейтинг от 1 до 5 звезд
	Type          ReviewType `json:"type" db:"type"`                     // Тип отзыва (positive/neutral/negative)
	Comment       string     `json:"comment" db:"comment"`               // Текст отзыва
	IsAnonymous   bool       `json:"is_anonymous" db:"is_anonymous"`     // Анонимный ли отзыв
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`         // Дата создания отзыва
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`         // Дата последнего обновления
	IsVisible     bool       `json:"is_visible" db:"is_visible"`         // Видимый ли отзыв (может быть скрыт админом)
	ReportedCount int        `json:"reported_count" db:"reported_count"` // Количество жалоб на отзыв
}

// Rating представляет агрегированный рейтинг пользователя
// Обновляется автоматически при добавлении новых отзывов
type Rating struct {
	UserID          int64     `json:"user_id" db:"user_id"`                   // ID пользователя
	AverageRating   float32   `json:"average_rating" db:"average_rating"`     // Средний рейтинг (0-5)
	TotalReviews    int       `json:"total_reviews" db:"total_reviews"`       // Общее количество отзывов
	PositiveReviews int       `json:"positive_reviews" db:"positive_reviews"` // Количество положительных отзывов
	NeutralReviews  int       `json:"neutral_reviews" db:"neutral_reviews"`   // Количество нейтральных отзывов
	NegativeReviews int       `json:"negative_reviews" db:"negative_reviews"` // Количество отрицательных отзывов
	FiveStars       int       `json:"five_stars" db:"five_stars"`             // Количество 5-звездочных оценок
	FourStars       int       `json:"four_stars" db:"four_stars"`             // Количество 4-звездочных оценок
	ThreeStars      int       `json:"three_stars" db:"three_stars"`           // Количество 3-звездочных оценок
	TwoStars        int       `json:"two_stars" db:"two_stars"`               // Количество 2-звездочных оценок
	OneStar         int       `json:"one_star" db:"one_star"`                 // Количество 1-звездочных оценок
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`             // Дата последнего обновления рейтинга
}

// ReviewReport представляет жалобу на отзыв
// Используется для модерации неподходящих отзывов
type ReviewReport struct {
	ID         int64      `json:"id" db:"id"`                   // Уникальный идентификатор жалобы
	ReviewID   int64      `json:"review_id" db:"review_id"`     // ID отзыва, на который жалуются
	UserID     int64      `json:"user_id" db:"user_id"`         // ID пользователя, подавшего жалобу
	Reason     string     `json:"reason" db:"reason"`           // Причина жалобы
	Comment    string     `json:"comment" db:"comment"`         // Дополнительный комментарий к жалобе
	Status     string     `json:"status" db:"status"`           // Статус рассмотрения жалобы
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`   // Дата подачи жалобы
	ResolvedAt *time.Time `json:"resolved_at" db:"resolved_at"` // Дата рассмотрения жалобы
}

// ReviewStats содержит статистику отзывов пользователя для отображения в профиле
type ReviewStats struct {
	UserID             int64       `json:"user_id"`             // ID пользователя
	AverageRating      float32     `json:"average_rating"`      // Средний рейтинг
	TotalReviews       int         `json:"total_reviews"`       // Общее количество отзывов
	PositivePercent    float32     `json:"positive_percent"`    // Процент положительных отзывов
	RecentReviews      []Review    `json:"recent_reviews"`      // Последние отзывы
	RatingDistribution map[int]int `json:"rating_distribution"` // Распределение оценок по звездам
}

// CreateReviewRequest содержит данные для создания нового отзыва
type CreateReviewRequest struct {
	DealID      int64  `json:"deal_id" validate:"required"`            // ID сделки (обязательно)
	ToUserID    int64  `json:"to_user_id" validate:"required"`         // ID получателя отзыва (обязательно)
	Rating      int    `json:"rating" validate:"required,min=1,max=5"` // Рейтинг от 1 до 5
	Comment     string `json:"comment" validate:"max=500"`             // Комментарий (макс 500 символов)
	IsAnonymous bool   `json:"is_anonymous"`                           // Анонимный отзыв
}
