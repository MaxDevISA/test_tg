package model

import (
	"time"
)

// Config представляет конфигурацию приложения
// Содержит все необходимые параметры для работы P2P биржи
type Config struct {
	// Настройки базы данных
	Database DatabaseConfig `json:"database"`

	// Настройки Telegram Bot
	Telegram TelegramConfig `json:"telegram"`

	// Настройки веб-сервера
	Server ServerConfig `json:"server"`

	// Настройки безопасности
	Security SecurityConfig `json:"security"`

	// Бизнес-логика
	Business BusinessConfig `json:"business"`

	// Настройки логирования
	Logging LoggingConfig `json:"logging"`
}

// DatabaseConfig содержит настройки для подключения к базе данных
type DatabaseConfig struct {
	Driver          string        `json:"driver" env:"DB_DRIVER"`                   // Драйвер БД (postgres, mysql, sqlite)
	Host            string        `json:"host" env:"DB_HOST"`                       // Хост базы данных
	Port            int           `json:"port" env:"DB_PORT"`                       // Порт базы данных
	Database        string        `json:"database" env:"DB_NAME"`                   // Имя базы данных
	Username        string        `json:"username" env:"DB_USER"`                   // Пользователь БД
	Password        string        `json:"password" env:"DB_PASSWORD"`               // Пароль БД
	SSLMode         string        `json:"ssl_mode" env:"DB_SSL_MODE"`               // Режим SSL (disable, require)
	MaxOpenConns    int           `json:"max_open_conns" env:"DB_MAX_OPEN"`         // Максимальное количество открытых соединений
	MaxIdleConns    int           `json:"max_idle_conns" env:"DB_MAX_IDLE"`         // Максимальное количество неактивных соединений
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime" env:"DB_CONN_LIFETIME"` // Максимальное время жизни соединения
}

// TelegramConfig содержит настройки для работы с Telegram Bot API
type TelegramConfig struct {
	BotToken     string  `json:"bot_token" env:"TELEGRAM_BOT_TOKEN"`      // Токен Telegram бота
	WebhookURL   string  `json:"webhook_url" env:"TELEGRAM_WEBHOOK_URL"`  // URL для webhook
	WebAppURL    string  `json:"webapp_url" env:"TELEGRAM_WEBAPP_URL"`    // URL веб-приложения
	ChatID       int64   `json:"chat_id" env:"TELEGRAM_CHAT_ID"`          // ID закрытого чата
	AdminUserIDs []int64 `json:"admin_user_ids" env:"TELEGRAM_ADMIN_IDS"` // ID администраторов
	UseWebhook   bool    `json:"use_webhook" env:"TELEGRAM_USE_WEBHOOK"`  // Использовать webhook или polling
	TimeoutSec   int     `json:"timeout_sec" env:"TELEGRAM_TIMEOUT"`      // Таймаут для запросов к API
}

// ServerConfig содержит настройки веб-сервера
type ServerConfig struct {
	Host           string        `json:"host" env:"SERVER_HOST"`                   // Хост сервера (0.0.0.0)
	Port           int           `json:"port" env:"SERVER_PORT"`                   // Порт сервера (8080)
	ReadTimeout    time.Duration `json:"read_timeout" env:"SERVER_READ_TIMEOUT"`   // Таймаут чтения
	WriteTimeout   time.Duration `json:"write_timeout" env:"SERVER_WRITE_TIMEOUT"` // Таймаут записи
	IdleTimeout    time.Duration `json:"idle_timeout" env:"SERVER_IDLE_TIMEOUT"`   // Таймаут простоя
	MaxHeaderBytes int           `json:"max_header_bytes" env:"SERVER_MAX_HEADER"` // Максимальный размер заголовков
	EnableCORS     bool          `json:"enable_cors" env:"SERVER_ENABLE_CORS"`     // Включить CORS
	EnableTLS      bool          `json:"enable_tls" env:"SERVER_ENABLE_TLS"`       // Включить TLS/HTTPS
	CertFile       string        `json:"cert_file" env:"SERVER_CERT_FILE"`         // Путь к сертификату
	KeyFile        string        `json:"key_file" env:"SERVER_KEY_FILE"`           // Путь к приватному ключу
}

// SecurityConfig содержит настройки безопасности
type SecurityConfig struct {
	JWTSecret          string        `json:"jwt_secret" env:"JWT_SECRET"`                     // Секретный ключ для JWT
	JWTExpiration      time.Duration `json:"jwt_expiration" env:"JWT_EXPIRATION"`             // Время жизни JWT токена
	RateLimitRequests  int           `json:"rate_limit_requests" env:"RATE_LIMIT_REQUESTS"`   // Лимит запросов в минуту
	RateLimitDuration  time.Duration `json:"rate_limit_duration" env:"RATE_LIMIT_DURATION"`   // Период для лимита запросов
	EncryptionKey      string        `json:"encryption_key" env:"ENCRYPTION_KEY"`             // Ключ для шифрования данных
	HashSalt           string        `json:"hash_salt" env:"HASH_SALT"`                       // Соль для хеширования
	MaxLoginAttempts   int           `json:"max_login_attempts" env:"MAX_LOGIN_ATTEMPTS"`     // Максимальное количество попыток входа
	LoginAttemptWindow time.Duration `json:"login_attempt_window" env:"LOGIN_ATTEMPT_WINDOW"` // Окно для подсчета попыток входа
	EnableIPWhitelist  bool          `json:"enable_ip_whitelist" env:"ENABLE_IP_WHITELIST"`   // Включить белый список IP
	WhitelistedIPs     []string      `json:"whitelisted_ips" env:"WHITELISTED_IPS"`           // Белый список IP адресов
}

// BusinessConfig содержит бизнес-настройки P2P биржи
type BusinessConfig struct {
	// Настройки заявок
	OrderExpirationHours      int      `json:"order_expiration_hours" env:"ORDER_EXPIRATION_HOURS"`         // Срок действия заявки в часах
	MaxActiveOrdersPerUser    int      `json:"max_active_orders_per_user" env:"MAX_ACTIVE_ORDERS_USER"`     // Максимум активных заявок на пользователя
	MinOrderAmount            float64  `json:"min_order_amount" env:"MIN_ORDER_AMOUNT"`                     // Минимальная сумма заявки
	MaxOrderAmount            float64  `json:"max_order_amount" env:"MAX_ORDER_AMOUNT"`                     // Максимальная сумма заявки
	SupportedCryptocurrencies []string `json:"supported_cryptocurrencies" env:"SUPPORTED_CRYPTOCURRENCIES"` // Поддерживаемые криптовалюты
	SupportedFiatCurrencies   []string `json:"supported_fiat_currencies" env:"SUPPORTED_FIAT_CURRENCIES"`   // Поддерживаемые фиатные валюты
	SupportedPaymentMethods   []string `json:"supported_payment_methods" env:"SUPPORTED_PAYMENT_METHODS"`   // Поддерживаемые способы оплаты

	// Настройки сделок
	DealConfirmationTimeoutMinutes int  `json:"deal_confirmation_timeout_minutes" env:"DEAL_CONFIRMATION_TIMEOUT"` // Таймаут подтверждения сделки в минутах
	RequireBothConfirmations       bool `json:"require_both_confirmations" env:"REQUIRE_BOTH_CONFIRMATIONS"`       // Требовать подтверждение от обеих сторон
	EnableAutoMatch                bool `json:"enable_auto_match" env:"ENABLE_AUTO_MATCH"`                         // Включить автоматическое сопоставление заявок

	// Настройки рейтинга
	MinRatingForNewUsers     float32 `json:"min_rating_for_new_users" env:"MIN_RATING_NEW_USERS"`         // Минимальный рейтинг для новых пользователей
	RequireMinRatingForDeals bool    `json:"require_min_rating_for_deals" env:"REQUIRE_MIN_RATING_DEALS"` // Требовать минимальный рейтинг для сделок

	// Настройки комиссии
	EnableCommission    bool    `json:"enable_commission" env:"ENABLE_COMMISSION"`         // Включить комиссию
	CommissionPercent   float64 `json:"commission_percent" env:"COMMISSION_PERCENT"`       // Процент комиссии
	MinCommissionAmount float64 `json:"min_commission_amount" env:"MIN_COMMISSION_AMOUNT"` // Минимальная комиссия
}

// LoggingConfig содержит настройки логирования
type LoggingConfig struct {
	Level       string `json:"level" env:"LOG_LEVEL"`               // Уровень логирования (debug, info, warn, error)
	Format      string `json:"format" env:"LOG_FORMAT"`             // Формат логов (text, json)
	OutputPath  string `json:"output_path" env:"LOG_OUTPUT"`        // Путь для записи логов (stdout, stderr, /path/to/file)
	MaxSize     int    `json:"max_size" env:"LOG_MAX_SIZE"`         // Максимальный размер файла лога в MB
	MaxBackups  int    `json:"max_backups" env:"LOG_MAX_BACKUPS"`   // Количество резервных копий логов
	MaxAge      int    `json:"max_age" env:"LOG_MAX_AGE"`           // Максимальный возраст логов в днях
	Compress    bool   `json:"compress" env:"LOG_COMPRESS"`         // Сжимать старые логи
	EnableStack bool   `json:"enable_stack" env:"LOG_ENABLE_STACK"` // Включить стек трейсы для ошибок
}

// SystemSettings представляет системные настройки, которые можно изменять через админ-панель
type SystemSettings struct {
	ID                  int64     `json:"id" db:"id"`                                     // Уникальный идентификатор настройки
	MaintenanceMode     bool      `json:"maintenance_mode" db:"maintenance_mode"`         // Режим технического обслуживания
	RegistrationEnabled bool      `json:"registration_enabled" db:"registration_enabled"` // Включена ли регистрация
	TradingEnabled      bool      `json:"trading_enabled" db:"trading_enabled"`           // Включена ли торговля
	MinUserRating       float32   `json:"min_user_rating" db:"min_user_rating"`           // Минимальный рейтинг для торговли
	MaxDailyDeals       int       `json:"max_daily_deals" db:"max_daily_deals"`           // Максимальное количество сделок в день
	AnnouncementText    string    `json:"announcement_text" db:"announcement_text"`       // Текст объявления
	AnnouncementActive  bool      `json:"announcement_active" db:"announcement_active"`   // Активно ли объявление
	LastUpdatedAt       time.Time `json:"last_updated_at" db:"last_updated_at"`           // Дата последнего обновления настроек
	UpdatedByAdminID    int64     `json:"updated_by_admin_id" db:"updated_by_admin_id"`   // ID админа, обновившего настройки
}
