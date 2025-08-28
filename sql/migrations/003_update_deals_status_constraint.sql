-- Миграция для обновления CHECK constraint статусов сделок
-- Версия: 003
-- Описание: Приводим статусы сделок в PostgreSQL в соответствие с Go моделью

-- =====================================================
-- ОБНОВЛЕНИЕ CHECK CONSTRAINT ДЛЯ СТАТУСОВ СДЕЛОК
-- =====================================================

-- Удаляем старый CHECK constraint
ALTER TABLE deals DROP CONSTRAINT IF EXISTS deals_status_check;

-- Добавляем новый CHECK constraint с правильными статусами из Go модели
ALTER TABLE deals ADD CONSTRAINT deals_status_check 
    CHECK (status IN ('in_progress', 'waiting_confirmation', 'completed', 'expired', 'dispute', 'cancelled'));

-- Обновляем DEFAULT значение на 'in_progress' (вместо 'pending')
ALTER TABLE deals ALTER COLUMN status SET DEFAULT 'in_progress';

-- Комментарий к изменению
COMMENT ON CONSTRAINT deals_status_check ON deals IS 'Статусы сделок должны соответствовать Go модели DealStatus';
