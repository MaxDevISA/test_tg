-- Миграция для добавления таблицы откликов (responses)
-- Версия: 002
-- Описание: Создание таблицы responses для хранения откликов пользователей на заявки

-- =====================================================
-- ТАБЛИЦА ОТКЛИКОВ
-- =====================================================
-- Отклики пользователей на заявки других пользователей
CREATE TABLE IF NOT EXISTS responses (
    id BIGSERIAL PRIMARY KEY,                          -- Уникальный идентификатор отклика
    order_id BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE, -- ID заявки на которую откликнулись
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,    -- ID пользователя который откликнулся
    message TEXT NOT NULL DEFAULT '',                  -- Сообщение от откликающегося
    status VARCHAR(20) NOT NULL DEFAULT 'waiting'      -- Статус отклика
        CHECK (status IN ('waiting', 'accepted', 'rejected')),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),      -- Время создания отклика
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),      -- Время последнего обновления
    reviewed_at TIMESTAMP,                            -- Время рассмотрения автором (null если еще не рассмотрен)
    
    -- Ограничение: один отклик от пользователя на заявку
    UNIQUE(order_id, user_id)
);

-- Индексы для таблицы откликов для быстрого поиска
CREATE INDEX idx_responses_order_id ON responses(order_id);       -- Отклики на заявку
CREATE INDEX idx_responses_user_id ON responses(user_id);         -- Отклики пользователя
CREATE INDEX idx_responses_status ON responses(status);           -- Фильтр по статусу
CREATE INDEX idx_responses_created_at ON responses(created_at DESC); -- Сортировка по времени

-- Триггер для автоматического обновления updated_at при изменении записи
CREATE TRIGGER update_responses_updated_at 
    BEFORE UPDATE ON responses 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Комментарий к таблице
COMMENT ON TABLE responses IS 'Отклики пользователей на заявки других пользователей';
