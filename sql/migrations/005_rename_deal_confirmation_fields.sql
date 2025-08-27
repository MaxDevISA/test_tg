-- Миграция для переименования полей подтверждения сделок
-- Версия: 005
-- Описание: Приводим названия полей в PostgreSQL в соответствие с Go моделью Deal

-- =====================================================
-- ПЕРЕИМЕНОВАНИЕ ПОЛЕЙ ПОДТВЕРЖДЕНИЯ СДЕЛОК
-- =====================================================

-- Переименовываем buyer_confirmed в author_confirmed
ALTER TABLE deals RENAME COLUMN buyer_confirmed TO author_confirmed;

-- Переименовываем seller_confirmed в counter_confirmed  
ALTER TABLE deals RENAME COLUMN seller_confirmed TO counter_confirmed;

-- Переименовываем payment_proof в author_proof (для соответствия модели)
ALTER TABLE deals RENAME COLUMN payment_proof TO author_proof;

-- Добавляем новое поле counter_proof (если его нет)
ALTER TABLE deals ADD COLUMN IF NOT EXISTS counter_proof TEXT;

-- Комментарии к изменениям
COMMENT ON COLUMN deals.author_confirmed IS 'Подтвердил ли автор заявки завершение сделки';
COMMENT ON COLUMN deals.counter_confirmed IS 'Подтвердил ли контрагент завершение сделки';
COMMENT ON COLUMN deals.author_proof IS 'Доказательство от автора заявки';  
COMMENT ON COLUMN deals.counter_proof IS 'Доказательство от контрагента';

-- Обновляем индексы если нужно (опционально)
-- CREATE INDEX IF NOT EXISTS idx_deals_author_confirmed ON deals(author_confirmed);
-- CREATE INDEX IF NOT EXISTS idx_deals_counter_confirmed ON deals(counter_confirmed);
