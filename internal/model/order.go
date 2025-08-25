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
	OrderStatusActive    OrderStatus = "active"    // Активная заявка, ищет контрагента
	OrderStatusMatched   OrderStatus = "matched"   // Заявка найдена, но сделка не завершена
	OrderStatusCompleted OrderStatus = "completed" // Сделка успешно завершена
	OrderStatusCancelled OrderStatus = "cancelled" // Заявка отменена пользователем
	OrderStatusExpired   OrderStatus = "expired"   // Заявка истекла по времени
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
	ID             int64       `json:"id" db:"id"`                           // Уникальный идентификатор заявки
	UserID         int64       `json:"user_id" db:"user_id"`                 // ID создателя заявки
	Type           OrderType   `json:"type" db:"type"`                       // Тип заявки (buy/sell)
	Cryptocurrency string      `json:"cryptocurrency" db:"cryptocurrency"`   // Название криптовалюты (BTC, ETH, USDT и т.д.)
	FiatCurrency   string      `json:"fiat_currency" db:"fiat_currency"`     // Фиатная валюта (RUB, USD, EUR)
	Amount         float64     `json:"amount" db:"amount"`                   // Количество криптовалюты
	Price          float64     `json:"price" db:"price"`                     // Цена за единицу криптовалюты
	TotalAmount    float64     `json:"total_amount" db:"total_amount"`       // Общая сумма сделки (amount * price)
	MinAmount      float64     `json:"min_amount" db:"min_amount"`           // Минимальная сумма для сделки
	MaxAmount      float64     `json:"max_amount" db:"max_amount"`           // Максимальная сумма для сделки
	PaymentMethods []string    `json:"payment_methods" db:"payment_methods"` // Способы оплаты (JSON array)
	Description    string      `json:"description" db:"description"`         // Дополнительное описание заявки
	Status         OrderStatus `json:"status" db:"status"`                   // Статус заявки
	CreatedAt      time.Time   `json:"created_at" db:"created_at"`           // Дата создания заявки
	UpdatedAt      time.Time   `json:"updated_at" db:"updated_at"`           // Дата последнего обновления
	ExpiresAt      time.Time   `json:"expires_at" db:"expires_at"`           // Дата истечения заявки
	MatchedUserID  *int64      `json:"matched_user_id" db:"matched_user_id"` // ID пользователя, с которым сматчена заявка
	MatchedAt      *time.Time  `json:"matched_at" db:"matched_at"`           // Время сопоставления заявки
	CompletedAt    *time.Time  `json:"completed_at" db:"completed_at"`       // Время завершения сделки
	IsActive       bool        `json:"is_active" db:"is_active"`             // Активна ли заявка
	AutoMatch      bool        `json:"auto_match" db:"auto_match"`           // Автоматическое сопоставление
}

// Deal представляет завершенную сделку между двумя пользователями
// Создается когда две заявки успешно сопоставляются и завершаются
type Deal struct {
	ID              int64         `json:"id" db:"id"`                             // Уникальный идентификатор сделки
	BuyOrderID      int64         `json:"buy_order_id" db:"buy_order_id"`         // ID заявки на покупку
	SellOrderID     int64         `json:"sell_order_id" db:"sell_order_id"`       // ID заявки на продажу
	BuyerID         int64         `json:"buyer_id" db:"buyer_id"`                 // ID покупателя
	SellerID        int64         `json:"seller_id" db:"seller_id"`               // ID продавца
	Cryptocurrency  string        `json:"cryptocurrency" db:"cryptocurrency"`     // Торгуемая криптовалюта
	FiatCurrency    string        `json:"fiat_currency" db:"fiat_currency"`       // Фиатная валюта
	Amount          float64       `json:"amount" db:"amount"`                     // Количество криптовалюты
	Price           float64       `json:"price" db:"price"`                       // Цена за единицу
	TotalAmount     float64       `json:"total_amount" db:"total_amount"`         // Общая сумма сделки
	PaymentMethod   PaymentMethod `json:"payment_method" db:"payment_method"`     // Используемый способ оплаты
	Status          string        `json:"status" db:"status"`                     // Статус сделки
	CreatedAt       time.Time     `json:"created_at" db:"created_at"`             // Время создания сделки
	CompletedAt     *time.Time    `json:"completed_at" db:"completed_at"`         // Время завершения сделки
	BuyerConfirmed  bool          `json:"buyer_confirmed" db:"buyer_confirmed"`   // Подтвердил ли покупатель
	SellerConfirmed bool          `json:"seller_confirmed" db:"seller_confirmed"` // Подтвердил ли продавец
	PaymentProof    string        `json:"payment_proof" db:"payment_proof"`       // Доказательство оплаты (ссылка на скриншот)
	Notes           string        `json:"notes" db:"notes"`                       // Заметки по сделке
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
