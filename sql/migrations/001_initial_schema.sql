-- Миграция для создания начальной схемы базы данных P2P криптобиржи
-- Версия: 001
-- Описание: Создание всех основных таблиц для пользователей, заявок, сделок, отзывов и системных настроек

-- Включаем расширения PostgreSQL (если используется PostgreSQL)
-- CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- =====================================================
-- ТАБЛИЦА ПОЛЬЗОВАТЕЛЕЙ
-- =====================================================
-- Основная таблица для хранения информации о пользователях
-- Содержит данные авторизации через Telegram и профильную информацию
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,                          -- Уникальный идентификатор пользователя
    telegram_id BIGINT NOT NULL UNIQUE,                -- ID пользователя в Telegram (уникальный)
    telegram_user_id VARCHAR(255),                     -- Username в Telegram (@username)
    first_name VARCHAR(255) NOT NULL,                  -- Имя из Telegram профиля
    last_name VARCHAR(255),                            -- Фамилия из Telegram профиля (может быть пустой)
    username VARCHAR(255),                             -- Username из Telegram профиля (может быть пустым)
    photo_url TEXT,                                    -- URL фото профиля из Telegram
    is_bot BOOLEAN NOT NULL DEFAULT FALSE,             -- Флаг бота (должен быть false для пользователей)
    language_code VARCHAR(10) DEFAULT 'ru',           -- Код языка пользователя (по умолчанию русский)
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),      -- Дата и время создания аккаунта
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),      -- Дата и время последнего обновления
    is_active BOOLEAN NOT NULL DEFAULT TRUE,          -- Активен ли пользователь (для блокировки)
    rating DECIMAL(3,2) NOT NULL DEFAULT 0.00,        -- Средний рейтинг пользователя (0.00-5.00)
    total_deals INTEGER NOT NULL DEFAULT 0,           -- Общее количество завершенных сделок
    successful_deals INTEGER NOT NULL DEFAULT 0,      -- Количество успешных сделок
    chat_member BOOLEAN NOT NULL DEFAULT FALSE        -- Является ли членом закрытого чата
);

-- Индексы для таблицы пользователей для быстрого поиска
CREATE INDEX idx_users_telegram_id ON users(telegram_id);     -- Поиск по Telegram ID
CREATE INDEX idx_users_username ON users(username);           -- Поиск по username
CREATE INDEX idx_users_rating ON users(rating DESC);          -- Сортировка по рейтингу
CREATE INDEX idx_users_active ON users(is_active, chat_member); -- Фильтр активных пользователей чата

-- =====================================================
-- ТАБЛИЦА ПРОФИЛЕЙ ПОЛЬЗОВАТЕЛЕЙ
-- =====================================================
-- Расширенная информация о пользователях
CREATE TABLE user_profiles (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE, -- Связь с пользователем
    bio TEXT,                                      -- Краткая биография пользователя
    location VARCHAR(255),                         -- Местоположение пользователя
    phone_number VARCHAR(20),                      -- Номер телефона (необязательно)
    email VARCHAR(255),                            -- Email адрес (необязательно)
    is_verified BOOLEAN NOT NULL DEFAULT FALSE,   -- Верифицирован ли пользователь
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),  -- Дата создания профиля
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),  -- Дата последнего обновления
    PRIMARY KEY (user_id)                         -- Первичный ключ по ID пользователя
);

-- =====================================================
-- ТАБЛИЦА ЗАЯВОК (ОРДЕРОВ)
-- =====================================================
-- Основная таблица для заявок на покупку и продажу криптовалют
CREATE TABLE orders (
    id BIGSERIAL PRIMARY KEY,                          -- Уникальный идентификатор заявки
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE, -- Создатель заявки
    type VARCHAR(10) NOT NULL CHECK (type IN ('buy', 'sell')), -- Тип заявки (покупка/продажа)
    cryptocurrency VARCHAR(20) NOT NULL,               -- Название криптовалюты (BTC, ETH, USDT)
    fiat_currency VARCHAR(10) NOT NULL DEFAULT 'RUB',  -- Фиатная валюта (RUB, USD, EUR)
    amount DECIMAL(20,8) NOT NULL CHECK (amount > 0),  -- Количество криптовалюты (до 8 знаков после запятой)
    price DECIMAL(15,2) NOT NULL CHECK (price > 0),    -- Цена за единицу криптовалюты
    total_amount DECIMAL(15,2) NOT NULL CHECK (total_amount > 0), -- Общая сумма сделки (amount * price)
    min_amount DECIMAL(15,2) CHECK (min_amount >= 0),  -- Минимальная сумма для сделки (может быть 0)
    max_amount DECIMAL(15,2) CHECK (max_amount >= min_amount), -- Максимальная сумма для сделки
    payment_methods JSONB NOT NULL DEFAULT '[]',       -- Способы оплаты (JSON массив строк)
    description TEXT,                                  -- Дополнительное описание заявки
    status VARCHAR(20) NOT NULL DEFAULT 'active'       -- Статус заявки
        CHECK (status IN ('active', 'matched', 'completed', 'cancelled', 'expired')),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),      -- Дата и время создания заявки
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),      -- Дата и время последнего обновления
    expires_at TIMESTAMP NOT NULL DEFAULT NOW() + INTERVAL '24 hours', -- Дата истечения заявки
    matched_user_id BIGINT REFERENCES users(id),      -- ID пользователя, с которым сматчена заявка
    matched_at TIMESTAMP,                             -- Время сопоставления заявки
    completed_at TIMESTAMP,                           -- Время завершения сделки
    is_active BOOLEAN NOT NULL DEFAULT TRUE,          -- Активна ли заявка (для быстрой фильтрации)
    auto_match BOOLEAN NOT NULL DEFAULT TRUE          -- Автоматическое сопоставление заявок
);

-- Индексы для таблицы заявок для оптимизации поиска и сортировки
CREATE INDEX idx_orders_user_id ON orders(user_id);                    -- Заявки пользователя
CREATE INDEX idx_orders_type_crypto ON orders(type, cryptocurrency);   -- Поиск по типу и криптовалюте
CREATE INDEX idx_orders_status_active ON orders(status, is_active);    -- Фильтр активных заявок
CREATE INDEX idx_orders_price ON orders(price);                        -- Сортировка по цене
CREATE INDEX idx_orders_created_at ON orders(created_at DESC);         -- Сортировка по дате создания
CREATE INDEX idx_orders_expires_at ON orders(expires_at);              -- Поиск истекающих заявок
CREATE INDEX idx_orders_crypto_fiat ON orders(cryptocurrency, fiat_currency); -- Торговые пары

-- =====================================================
-- ТАБЛИЦА СДЕЛОК
-- =====================================================
-- Таблица для завершенных сделок между пользователями
CREATE TABLE deals (
    id BIGSERIAL PRIMARY KEY,                          -- Уникальный идентификатор сделки
    buy_order_id BIGINT NOT NULL REFERENCES orders(id), -- ID заявки на покупку
    sell_order_id BIGINT NOT NULL REFERENCES orders(id), -- ID заявки на продажу
    buyer_id BIGINT NOT NULL REFERENCES users(id),    -- ID покупателя
    seller_id BIGINT NOT NULL REFERENCES users(id),   -- ID продавца
    cryptocurrency VARCHAR(20) NOT NULL,              -- Торгуемая криптовалюта
    fiat_currency VARCHAR(10) NOT NULL,               -- Фиатная валюта
    amount DECIMAL(20,8) NOT NULL CHECK (amount > 0), -- Количество криптовалюты
    price DECIMAL(15,2) NOT NULL CHECK (price > 0),   -- Цена за единицу
    total_amount DECIMAL(15,2) NOT NULL CHECK (total_amount > 0), -- Общая сумма сделки
    payment_method VARCHAR(50) NOT NULL,              -- Используемый способ оплаты
    status VARCHAR(20) NOT NULL DEFAULT 'pending'     -- Статус сделки
        CHECK (status IN ('pending', 'payment_sent', 'completed', 'disputed', 'cancelled')),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),      -- Время создания сделки
    completed_at TIMESTAMP,                           -- Время завершения сделки
    buyer_confirmed BOOLEAN NOT NULL DEFAULT FALSE,   -- Подтвердил ли покупатель получение криптовалюты
    seller_confirmed BOOLEAN NOT NULL DEFAULT FALSE,  -- Подтвердил ли продавец получение оплаты
    payment_proof TEXT,                               -- Доказательство оплаты (ссылка на скриншот)
    notes TEXT                                        -- Заметки по сделке
);

-- Индексы для таблицы сделок
CREATE INDEX idx_deals_buyer_id ON deals(buyer_id);                   -- Сделки покупателя
CREATE INDEX idx_deals_seller_id ON deals(seller_id);                 -- Сделки продавца
CREATE INDEX idx_deals_status ON deals(status);                       -- Фильтр по статусу
CREATE INDEX idx_deals_created_at ON deals(created_at DESC);          -- Сортировка по дате
CREATE INDEX idx_deals_crypto ON deals(cryptocurrency, fiat_currency); -- Торговые пары

-- =====================================================
-- ТАБЛИЦА ОТЗЫВОВ
-- =====================================================
-- Отзывы пользователей друг о друге после совершения сделок
CREATE TABLE reviews (
    id BIGSERIAL PRIMARY KEY,                          -- Уникальный идентификатор отзыва
    deal_id BIGINT NOT NULL REFERENCES deals(id),     -- ID сделки, по которой оставлен отзыв
    from_user_id BIGINT NOT NULL REFERENCES users(id), -- Пользователь, оставивший отзыв
    to_user_id BIGINT NOT NULL REFERENCES users(id),  -- Пользователь, которому оставлен отзыв
    rating INTEGER NOT NULL CHECK (rating BETWEEN 1 AND 5), -- Рейтинг от 1 до 5 звезд
    type VARCHAR(10) NOT NULL DEFAULT 'neutral'        -- Тип отзыва
        CHECK (type IN ('positive', 'neutral', 'negative')),
    comment TEXT,                                      -- Текст отзыва
    is_anonymous BOOLEAN NOT NULL DEFAULT FALSE,       -- Анонимный ли отзыв
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),      -- Дата создания отзыва
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),      -- Дата последнего обновления
    is_visible BOOLEAN NOT NULL DEFAULT TRUE,         -- Видимый ли отзыв
    reported_count INTEGER NOT NULL DEFAULT 0,        -- Количество жалоб на отзыв
    
    -- Ограничение: один отзыв на сделку от каждого пользователя
    UNIQUE(deal_id, from_user_id)
);

-- Индексы для таблицы отзывов
CREATE INDEX idx_reviews_to_user_id ON reviews(to_user_id, is_visible); -- Отзывы о пользователе
CREATE INDEX idx_reviews_from_user_id ON reviews(from_user_id);          -- Отзывы от пользователя
CREATE INDEX idx_reviews_rating ON reviews(rating);                     -- Фильтр по рейтингу
CREATE INDEX idx_reviews_created_at ON reviews(created_at DESC);        -- Сортировка по дате

-- =====================================================
-- ТАБЛИЦА РЕЙТИНГОВ
-- =====================================================
-- Агрегированные рейтинги пользователей (обновляются триггерами)
CREATE TABLE ratings (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE, -- ID пользователя
    average_rating DECIMAL(3,2) NOT NULL DEFAULT 0.00,    -- Средний рейтинг (0.00-5.00)
    total_reviews INTEGER NOT NULL DEFAULT 0,             -- Общее количество отзывов
    positive_reviews INTEGER NOT NULL DEFAULT 0,          -- Количество положительных отзывов
    neutral_reviews INTEGER NOT NULL DEFAULT 0,           -- Количество нейтральных отзывов
    negative_reviews INTEGER NOT NULL DEFAULT 0,          -- Количество отрицательных отзывов
    five_stars INTEGER NOT NULL DEFAULT 0,                -- Количество 5-звездочных оценок
    four_stars INTEGER NOT NULL DEFAULT 0,                -- Количество 4-звездочных оценок
    three_stars INTEGER NOT NULL DEFAULT 0,               -- Количество 3-звездочных оценок
    two_stars INTEGER NOT NULL DEFAULT 0,                 -- Количество 2-звездочных оценок
    one_star INTEGER NOT NULL DEFAULT 0,                  -- Количество 1-звездочных оценок
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),          -- Дата последнего обновления рейтинга
    PRIMARY KEY (user_id)
);

-- =====================================================
-- ТАБЛИЦА ЖАЛОБ НА ОТЗЫВЫ
-- =====================================================
-- Жалобы пользователей на неподходящие отзывы
CREATE TABLE review_reports (
    id BIGSERIAL PRIMARY KEY,                          -- Уникальный идентификатор жалобы
    review_id BIGINT NOT NULL REFERENCES reviews(id) ON DELETE CASCADE, -- Отзыв, на который жалуются
    user_id BIGINT NOT NULL REFERENCES users(id),     -- Пользователь, подавший жалобу
    reason VARCHAR(100) NOT NULL,                      -- Причина жалобы
    comment TEXT,                                      -- Дополнительный комментарий
    status VARCHAR(20) NOT NULL DEFAULT 'pending'      -- Статус рассмотрения жалобы
        CHECK (status IN ('pending', 'reviewed', 'resolved', 'rejected')),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),      -- Дата подачи жалобы
    resolved_at TIMESTAMP,                            -- Дата рассмотрения жалобы
    
    -- Ограничение: одна жалоба на отзыв от каждого пользователя
    UNIQUE(review_id, user_id)
);

-- Индексы для жалоб на отзывы
CREATE INDEX idx_review_reports_status ON review_reports(status);         -- Фильтр по статусу
CREATE INDEX idx_review_reports_created_at ON review_reports(created_at DESC); -- Сортировка по дате

-- =====================================================
-- ТАБЛИЦА СИСТЕМНЫХ НАСТРОЕК
-- =====================================================
-- Настройки системы, которые можно изменять через админ-панель
CREATE TABLE system_settings (
    id INTEGER PRIMARY KEY DEFAULT 1,                 -- Всегда один ряд настроек
    maintenance_mode BOOLEAN NOT NULL DEFAULT FALSE,  -- Режим технического обслуживания
    registration_enabled BOOLEAN NOT NULL DEFAULT TRUE, -- Включена ли регистрация новых пользователей
    trading_enabled BOOLEAN NOT NULL DEFAULT TRUE,    -- Включена ли торговля
    min_user_rating DECIMAL(3,2) NOT NULL DEFAULT 0.00, -- Минимальный рейтинг для участия в торговле
    max_daily_deals INTEGER NOT NULL DEFAULT 10,      -- Максимальное количество сделок в день на пользователя
    announcement_text TEXT,                           -- Текст системного объявления
    announcement_active BOOLEAN NOT NULL DEFAULT FALSE, -- Активно ли объявление
    last_updated_at TIMESTAMP NOT NULL DEFAULT NOW(), -- Дата последнего обновления настроек
    updated_by_admin_id BIGINT REFERENCES users(id),  -- ID админа, обновившего настройки
    
    -- Ограничение: только одна запись в таблице
    CHECK (id = 1)
);

-- Вставляем начальные настройки
INSERT INTO system_settings (id) VALUES (1);

-- =====================================================
-- ТРИГГЕРЫ ДЛЯ АВТОМАТИЧЕСКОГО ОБНОВЛЕНИЯ
-- =====================================================

-- Функция для обновления поля updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();  -- Устанавливаем текущее время в поле updated_at
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Триггеры для автоматического обновления updated_at при изменении записей
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_user_profiles_updated_at BEFORE UPDATE ON user_profiles 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_orders_updated_at BEFORE UPDATE ON orders 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_reviews_updated_at BEFORE UPDATE ON reviews 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Функция для обновления рейтинга пользователя при добавлении/изменении отзыва
CREATE OR REPLACE FUNCTION update_user_rating()
RETURNS TRIGGER AS $$
BEGIN
    -- Обновляем статистику рейтинга для пользователя, которому оставлен отзыв
    INSERT INTO ratings (
        user_id, 
        average_rating, 
        total_reviews,
        positive_reviews,
        neutral_reviews, 
        negative_reviews,
        five_stars,
        four_stars,
        three_stars,
        two_stars,
        one_star,
        updated_at
    ) 
    SELECT 
        NEW.to_user_id,
        ROUND(AVG(rating::decimal), 2) as average_rating,
        COUNT(*) as total_reviews,
        COUNT(*) FILTER (WHERE type = 'positive') as positive_reviews,
        COUNT(*) FILTER (WHERE type = 'neutral') as neutral_reviews,
        COUNT(*) FILTER (WHERE type = 'negative') as negative_reviews,
        COUNT(*) FILTER (WHERE rating = 5) as five_stars,
        COUNT(*) FILTER (WHERE rating = 4) as four_stars,
        COUNT(*) FILTER (WHERE rating = 3) as three_stars,
        COUNT(*) FILTER (WHERE rating = 2) as two_stars,
        COUNT(*) FILTER (WHERE rating = 1) as one_star,
        NOW()
    FROM reviews 
    WHERE to_user_id = NEW.to_user_id AND is_visible = TRUE
    ON CONFLICT (user_id) DO UPDATE SET
        average_rating = EXCLUDED.average_rating,
        total_reviews = EXCLUDED.total_reviews,
        positive_reviews = EXCLUDED.positive_reviews,
        neutral_reviews = EXCLUDED.neutral_reviews,
        negative_reviews = EXCLUDED.negative_reviews,
        five_stars = EXCLUDED.five_stars,
        four_stars = EXCLUDED.four_stars,
        three_stars = EXCLUDED.three_stars,
        two_stars = EXCLUDED.two_stars,
        one_star = EXCLUDED.one_star,
        updated_at = NOW();
    
    -- Обновляем рейтинг в основной таблице пользователей
    UPDATE users 
    SET rating = (
        SELECT COALESCE(average_rating, 0.00) 
        FROM ratings 
        WHERE user_id = NEW.to_user_id
    )
    WHERE id = NEW.to_user_id;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Триггер для обновления рейтинга при добавлении отзыва
CREATE TRIGGER update_rating_after_review_insert 
    AFTER INSERT ON reviews 
    FOR EACH ROW EXECUTE FUNCTION update_user_rating();

-- Триггер для обновления рейтинга при изменении отзыва
CREATE TRIGGER update_rating_after_review_update 
    AFTER UPDATE ON reviews 
    FOR EACH ROW EXECUTE FUNCTION update_user_rating();

-- =====================================================
-- КОММЕНТАРИИ К ТАБЛИЦАМ
-- =====================================================

COMMENT ON TABLE users IS 'Основная таблица пользователей с данными авторизации через Telegram';
COMMENT ON TABLE user_profiles IS 'Расширенная информация профиля пользователя';
COMMENT ON TABLE orders IS 'Заявки на покупку и продажу криптовалют';
COMMENT ON TABLE deals IS 'Завершенные сделки между пользователями';
COMMENT ON TABLE reviews IS 'Отзывы пользователей друг о друге';
COMMENT ON TABLE ratings IS 'Агрегированные рейтинги пользователей';
COMMENT ON TABLE review_reports IS 'Жалобы на неподходящие отзывы';
COMMENT ON TABLE system_settings IS 'Системные настройки приложения';

-- =====================================================
-- ЗАВЕРШЕНИЕ МИГРАЦИИ
-- =====================================================
