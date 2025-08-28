package model

import (
	"time"
)

// OrderType определяет тип заявки - покупка или продажа
type OrderType string

const (
	OrderTypeBuy  OrderType = "buy"  // Заявка на покупку криптовалюты
	OrderTypeSell OrderType = "sell" // Заявка на продажу криптовалюты
)

// OrderStatus определяет статус заявки
type OrderStatus string

const (
	OrderStatusActive       OrderStatus = "active"        // Активная заявка, доступна для откликов на рынке
	OrderStatusHasResponses OrderStatus = "has_responses" // На заявку есть отклики, но автор еще не выбрал
	OrderStatusInDeal       OrderStatus = "in_deal"       // Автор принял отклик, заявка убрана с рынка
	OrderStatusCompleted    OrderStatus = "completed"     // Сделка успешно завершена
	OrderStatusCancelled    OrderStatus = "cancelled"     // Заявка отменена пользователем
	OrderStatusExpired      OrderStatus = "expired"       // Заявка истекла по времени
)

// PaymentMethod определяет способы оплаты
type PaymentMethod string

const (
	PaymentMethodBank        PaymentMethod = "bank_transfer" // Банковский перевод
	PaymentMethodSberbank    PaymentMethod = "sberbank"      // Сбербанк
	PaymentMethodTinkoff     PaymentMethod = "tinkoff"       // Тинькофф
	PaymentMethodQIWI        PaymentMethod = "qiwi"          // QIWI кошелек
	PaymentMethodYandexMoney PaymentMethod = "yandex_money"  // ЮMoney
	PaymentMethodCash        PaymentMethod = "cash"          // Наличные
	PaymentMethodOther       PaymentMethod = "other"         // Другие способы
)

// Order представляет заявку на покупку или продажу криптовалюты
// Это основная сущность для P2P торговли
type Order struct {
	ID                 int64       `json:"id" db:"id"`                                     // Уникальный идентификатор заявки
	UserID             int64       `json:"user_id" db:"user_id"`                           // ID создателя заявки
	Type               OrderType   `json:"type" db:"type"`                                 // Тип заявки (buy/sell)
	Cryptocurrency     string      `json:"cryptocurrency" db:"cryptocurrency"`             // Название криптовалюты (BTC, ETH, USDT и т.д.)
	FiatCurrency       string      `json:"fiat_currency" db:"fiat_currency"`               // Фиатная валюта (RUB, USD, EUR)
	Amount             float64     `json:"amount" db:"amount"`                             // Количество криптовалюты
	Price              float64     `json:"price" db:"price"`                               // Цена за единицу криптовалюты
	TotalAmount        float64     `json:"total_amount" db:"total_amount"`                 // Общая сумма сделки (amount * price)
	MinAmount          float64     `json:"min_amount" db:"min_amount"`                     // Минимальная сумма для сделки
	MaxAmount          float64     `json:"max_amount" db:"max_amount"`                     // Максимальная сумма для сделки
	PaymentMethods     []string    `json:"payment_methods" db:"payment_methods"`           // Способы оплаты (JSON array)
	Description        string      `json:"description" db:"description"`                   // Дополнительное описание заявки
	Status             OrderStatus `json:"status" db:"status"`                             // Статус заявки
	CreatedAt          time.Time   `json:"created_at" db:"created_at"`                     // Дата создания заявки
	UpdatedAt          time.Time   `json:"updated_at" db:"updated_at"`                     // Дата последнего обновления
	ExpiresAt          time.Time   `json:"expires_at" db:"expires_at"`                     // Дата истечения заявки
	CompletedAt        *time.Time  `json:"completed_at" db:"completed_at"`                 // Время завершения сделки
	IsActive           bool        `json:"is_active" db:"is_active"`                       // Активна ли заявка
	ResponseCount      int         `json:"response_count" db:"response_count"`             // Количество откликов на заявку
	AcceptedResponseID *int64      `json:"accepted_response_id" db:"accepted_response_id"` // ID принятого отклика (если есть)
	
	// Дополнительные поля для фронтенда (не сохраняются в БД)
	UserName   string `json:"user_name,omitempty"`   // Полное имя пользователя 
	Username   string `json:"username,omitempty"`    // Telegram username
	FirstName  string `json:"first_name,omitempty"`  // Имя пользователя
	LastName   string `json:"last_name,omitempty"`   // Фамилия пользователя
}

// DealStatus определяет статус сделки в новой логике
type DealStatus string

const (
	DealStatusInProgress          DealStatus = "in_progress"          // Сделка в процессе
	DealStatusWaitingConfirmation DealStatus = "waiting_confirmation" // Ожидает подтверждения одной из сторон
	DealStatusCompleted           DealStatus = "completed"            // Сделка завершена успешно
	DealStatusExpired             DealStatus = "expired"              // Время сделки истекло
	DealStatusDispute             DealStatus = "dispute"              // Спор по сделке
	DealStatusCancelled           DealStatus = "cancelled"            // Сделка отменена
)

// Deal представляет активную сделку между двумя пользователями
// Создается когда автор заявки принимает отклик
type Deal struct {
	ID               int64      `json:"id" db:"id"`                               // Уникальный идентификатор сделки
	ResponseID       int64      `json:"response_id" db:"response_id"`             // ID отклика, на основе которого создана сделка
	OrderID          int64      `json:"order_id" db:"order_id"`                   // ID исходной заявки
	AuthorID         int64      `json:"author_id" db:"author_id"`                 // ID автора заявки (продавец или покупатель)
	CounterpartyID   int64      `json:"counterparty_id" db:"counterparty_id"`     // ID контрагента (кто откликнулся)
	Cryptocurrency   string     `json:"cryptocurrency" db:"cryptocurrency"`       // Торгуемая криптовалюта
	FiatCurrency     string     `json:"fiat_currency" db:"fiat_currency"`         // Фиатная валюта
	Amount           float64    `json:"amount" db:"amount"`                       // Количество криптовалюты
	Price            float64    `json:"price" db:"price"`                         // Цена за единицу
	TotalAmount      float64    `json:"total_amount" db:"total_amount"`           // Общая сумма сделки
	PaymentMethods   []string   `json:"payment_methods" db:"payment_methods"`     // Доступные способы оплаты
	OrderType        OrderType  `json:"order_type" db:"order_type"`               // Тип исходной заявки (buy/sell)
	Status           DealStatus `json:"status" db:"status"`                       // Статус сделки
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`               // Время создания сделки
	ExpiresAt        time.Time  `json:"expires_at" db:"expires_at"`               // Время истечения сделки (таймер 1 час)
	CompletedAt      *time.Time `json:"completed_at" db:"completed_at"`           // Время завершения сделки
	AuthorConfirmed  bool       `json:"author_confirmed" db:"author_confirmed"`   // Подтвердил ли автор заявки перевод
	CounterConfirmed bool       `json:"counter_confirmed" db:"counter_confirmed"` // Подтвердил ли контрагент перевод
	AuthorProof      string     `json:"author_proof" db:"author_proof"`           // Доказательство перевода от автора
	CounterProof     string     `json:"counter_proof" db:"counter_proof"`         // Доказательство перевода от контрагента
	Notes            string     `json:"notes" db:"notes"`                         // Заметки по сделке
	DisputeReason    string     `json:"dispute_reason" db:"dispute_reason"`       // Причина спора (если есть)
	
	// Дополнительные поля для фронтенда (не сохраняются в БД)
	AuthorUsername      string `json:"author_username,omitempty"`      // Telegram username автора
	AuthorName          string `json:"author_name,omitempty"`          // Полное имя автора
	CounterpartyUsername string `json:"counterparty_username,omitempty"` // Telegram username контрагента  
	CounterpartyName    string `json:"counterparty_name,omitempty"`    // Полное имя контрагента
}

// OrderFilter содержит параметры для фильтрации заявок
// Используется при поиске подходящих заявок
type OrderFilter struct {
	Type           *OrderType   `json:"type"`            // Тип заявки
	Cryptocurrency *string      `json:"cryptocurrency"`  // Криптовалюта
	FiatCurrency   *string      `json:"fiat_currency"`   // Фиатная валюта
	MinPrice       *float64     `json:"min_price"`       // Минимальная цена
	MaxPrice       *float64     `json:"max_price"`       // Максимальная цена
	MinAmount      *float64     `json:"min_amount"`      // Минимальная сумма
	MaxAmount      *float64     `json:"max_amount"`      // Максимальная сумма
	PaymentMethods []string     `json:"payment_methods"` // Способы оплаты
	Status         *OrderStatus `json:"status"`          // Статус заявки
	UserID         *int64       `json:"user_id"`         // ID пользователя
	CreatedAfter   *time.Time   `json:"created_after"`   // Созданы после даты
	CreatedBefore  *time.Time   `json:"created_before"`  // Созданы до даты
	SortBy         string       `json:"sort_by"`         // Сортировка (price, created_at, amount)
	SortOrder      string       `json:"sort_order"`      // Порядок сортировки (asc, desc)
	Limit          int          `json:"limit"`           // Лимит результатов
	Offset         int          `json:"offset"`          // Смещение для пагинации
}
