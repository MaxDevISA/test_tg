-- Миграция для обновления CHECK constraint статусов заявок
-- Версия: 004  
-- Описание: Приводим статусы заявок в PostgreSQL в соответствие с Go моделью

-- =====================================================
-- ОБНОВЛЕНИЕ CHECK CONSTRAINT ДЛЯ СТАТУСОВ ЗАЯВОК
-- =====================================================

-- Удаляем старый CHECK constraint
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_status_check;

-- Добавляем новый CHECK constraint с правильными статусами из Go модели
ALTER TABLE orders ADD CONSTRAINT orders_status_check 
    CHECK (status IN ('active', 'cancelled', 'completed', 'expired', 'in_deal'));

-- Комментарий к изменению
COMMENT ON CONSTRAINT orders_status_check ON orders IS 'Статусы заявок должны соответствовать Go модели OrderStatus';
