package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"p2pTG-crypto-exchange/internal/model"

	_ "github.com/lib/pq"
)

// Repository представляет слой доступа к данным
// Содержит все методы для работы с базой данных
// Реализует паттерн Repository для изоляции бизнес-логики от деталей БД
type Repository struct {
	db *sql.DB // Соединение с базой данных PostgreSQL
}

// NewRepository создает новый экземпляр репозитория
// Принимает строку подключения к базе данных и возвращает инициализированный репозиторий
// dbURL должен быть в формате: "postgres://user:password@host:port/dbname?sslmode=disable"
func NewRepository(dbURL string) (*Repository, error) {
	// Устанавливаем соединение с базой данных
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть соединение с базой данных: %w", err)
	}

	// Проверяем соединение с базой данных
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("не удалось подключиться к базе данных: %w", err)
	}

	// Настраиваем пул соединений для оптимальной производительности
	db.SetMaxOpenConns(25) // Максимальное количество открытых соединений
	db.SetMaxIdleConns(25) // Максимальное количество неактивных соединений

	log.Println("[INFO] Соединение с PostgreSQL успешно установлено")

	return &Repository{
		db: db,
	}, nil
}

// Close закрывает соединение с базой данных
// Должен вызываться при завершении работы приложения
func (r *Repository) Close() error {
	if r.db != nil {
		log.Println("[INFO] Закрытие соединения с базой данных")
		return r.db.Close()
	}
	return nil
}

// =====================================================
// МЕТОДЫ ДЛЯ РАБОТЫ С ПОЛЬЗОВАТЕЛЯМИ
// =====================================================

// CreateUser создает нового пользователя в базе данных
// Принимает данные пользователя из Telegram авторизации
// Возвращает созданного пользователя с присвоенным ID
func (r *Repository) CreateUser(user *model.User) error {
	// SQL запрос для вставки нового пользователя
	// RETURNING id позволяет получить автогенерируемый ID
	query := `
		INSERT INTO users (
			telegram_id, telegram_user_id, first_name, last_name, 
			username, photo_url, is_bot, language_code, chat_member
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		) RETURNING id, created_at, updated_at`

	// Выполняем запрос и сканируем результат
	err := r.db.QueryRow(
		query,
		user.TelegramID,
		user.TelegramUserID,
		user.FirstName,
		user.LastName,
		user.Username,
		user.PhotoURL,
		user.IsBot,
		user.LanguageCode,
		user.ChatMember,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return fmt.Errorf("не удалось создать пользователя: %w", err)
	}

	log.Printf("[INFO] Создан новый пользователь: ID=%d, TelegramID=%d, Username=%s",
		user.ID, user.TelegramID, user.Username)

	return nil
}

// GetUserByTelegramID находит пользователя по его Telegram ID
// Возвращает пользователя или ошибку, если пользователь не найден
func (r *Repository) GetUserByTelegramID(telegramID int64) (*model.User, error) {
	user := &model.User{}

	// SQL запрос для поиска пользователя по Telegram ID
	query := `
		SELECT id, telegram_id, telegram_user_id, first_name, last_name,
		       username, photo_url, is_bot, language_code, created_at,
		       updated_at, is_active, rating, total_deals, successful_deals, chat_member
		FROM users 
		WHERE telegram_id = $1`

	// Выполняем запрос и сканируем результат в структуру пользователя
	err := r.db.QueryRow(query, telegramID).Scan(
		&user.ID,
		&user.TelegramID,
		&user.TelegramUserID,
		&user.FirstName,
		&user.LastName,
		&user.Username,
		&user.PhotoURL,
		&user.IsBot,
		&user.LanguageCode,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.IsActive,
		&user.Rating,
		&user.TotalDeals,
		&user.SuccessfulDeals,
		&user.ChatMember,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("пользователь с Telegram ID %d не найден", telegramID)
		}
		return nil, fmt.Errorf("не удалось найти пользователя: %w", err)
	}

	return user, nil
}

// UpdateUserChatMembership обновляет статус членства пользователя в чате
// Вызывается когда пользователь присоединяется к чату или покидает его
func (r *Repository) UpdateUserChatMembership(telegramID int64, isMember bool) error {
	// SQL запрос для обновления статуса членства в чате
	query := `
		UPDATE users 
		SET chat_member = $1, updated_at = NOW()
		WHERE telegram_id = $2`

	// Выполняем запрос на обновление
	result, err := r.db.Exec(query, isMember, telegramID)
	if err != nil {
		return fmt.Errorf("не удалось обновить статус членства пользователя: %w", err)
	}

	// Проверяем, был ли обновлен какой-либо ряд
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("не удалось проверить результат обновления: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("пользователь с Telegram ID %d не найден для обновления", telegramID)
	}

	log.Printf("[INFO] Обновлен статус членства пользователя TelegramID=%d: isMember=%t",
		telegramID, isMember)

	return nil
}

// =====================================================
// МЕТОДЫ ДЛЯ РАБОТЫ С ЗАЯВКАМИ
// =====================================================

// CreateOrder создает новую заявку на покупку или продажу
// Принимает заявку от пользователя и сохраняет ее в базе данных
func (r *Repository) CreateOrder(order *model.Order) error {
	// SQL запрос для создания новой заявки
	// Используем JSONB для хранения массива способов оплаты
	query := `
		INSERT INTO orders (
			user_id, type, cryptocurrency, fiat_currency, amount, 
			price, total_amount, min_amount, max_amount, 
			payment_methods, description, expires_at, auto_match
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		) RETURNING id, created_at, updated_at, status, is_active`

	// Сериализуем способы оплаты в JSON
	paymentMethodsJSON, err := json.Marshal(order.PaymentMethods)
	if err != nil {
		return fmt.Errorf("не удалось сериализовать способы оплаты: %w", err)
	}

	// Выполняем запрос и получаем сгенерированные поля
	err = r.db.QueryRow(
		query,
		order.UserID,
		order.Type,
		order.Cryptocurrency,
		order.FiatCurrency,
		order.Amount,
		order.Price,
		order.TotalAmount,
		order.MinAmount,
		order.MaxAmount,
		paymentMethodsJSON, // JSON-сериализованный массив способов оплаты
		order.Description,
		order.ExpiresAt,
		false, // AutoMatch больше не используется в новой логике откликов
	).Scan(
		&order.ID,
		&order.CreatedAt,
		&order.UpdatedAt,
		&order.Status,
		&order.IsActive,
	)

	if err != nil {
		return fmt.Errorf("не удалось создать заявку: %w", err)
	}

	log.Printf("[INFO] Создана новая заявка: ID=%d, Type=%s, Crypto=%s, Amount=%.8f",
		order.ID, order.Type, order.Cryptocurrency, order.Amount)

	return nil
}

// GetOrdersByFilter получает заявки по заданным фильтрам
// Реализует гибкий поиск заявок с пагинацией и сортировкой
func (r *Repository) GetOrdersByFilter(filter *model.OrderFilter) ([]*model.Order, error) {
	// Начальная часть SQL запроса
	query := `
		SELECT id, user_id, type, cryptocurrency, fiat_currency, 
		       amount, price, total_amount, min_amount, max_amount,
		       payment_methods, description, status, created_at,
		       updated_at, expires_at, completed_at, is_active
		FROM orders`

	// Условия WHERE
	whereClauses := []string{}

	// Проверяем активность (по умолчанию только активные, кроме случая включения неактивных)
	if !filter.IncludeInactive {
		whereClauses = append(whereClauses, "is_active = true")
	}

	// Параметры для подстановки в запрос
	args := []interface{}{}
	argCount := 0

	// Динамически добавляем условия фильтрации
	if filter.Type != nil {
		argCount++
		whereClauses = append(whereClauses, fmt.Sprintf("type = $%d", argCount))
		args = append(args, *filter.Type)
	}

	if filter.Cryptocurrency != nil {
		argCount++
		whereClauses = append(whereClauses, fmt.Sprintf("cryptocurrency = $%d", argCount))
		args = append(args, *filter.Cryptocurrency)
	}

	if filter.FiatCurrency != nil {
		argCount++
		whereClauses = append(whereClauses, fmt.Sprintf("fiat_currency = $%d", argCount))
		args = append(args, *filter.FiatCurrency)
	}

	if filter.Status != nil {
		argCount++
		whereClauses = append(whereClauses, fmt.Sprintf("status = $%d", argCount))
		args = append(args, *filter.Status)
	}

	if filter.UserID != nil {
		argCount++
		whereClauses = append(whereClauses, fmt.Sprintf("user_id = $%d", argCount))
		args = append(args, *filter.UserID)
	}

	// Фильтр по дате создания (КРИТИЧЕСКИ ВАЖНО для CleanupService!)
	if filter.CreatedAfter != nil {
		argCount++
		whereClauses = append(whereClauses, fmt.Sprintf("created_at > $%d", argCount))
		args = append(args, *filter.CreatedAfter)
	}

	if filter.CreatedBefore != nil {
		argCount++
		whereClauses = append(whereClauses, fmt.Sprintf("created_at < $%d", argCount))
		args = append(args, *filter.CreatedBefore)
	}

	// Собираем условия WHERE
	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Добавляем сортировку
	sortBy := "created_at"
	if filter.SortBy != "" {
		sortBy = filter.SortBy
	}

	sortOrder := "DESC"
	if filter.SortOrder == "asc" {
		sortOrder = "ASC"
	}

	query += fmt.Sprintf(" ORDER BY %s %s", sortBy, sortOrder)

	// Добавляем лимит и смещение для пагинации
	if filter.Limit > 0 {
		argCount++
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, filter.Limit)
	}

	if filter.Offset > 0 {
		argCount++
		query += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, filter.Offset)
	}

	// Выполняем запрос
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("не удалось выполнить запрос поиска заявок: %w", err)
	}
	defer rows.Close()

	// Сканируем результаты в слайс заявок
	var orders []*model.Order
	for rows.Next() {
		order := &model.Order{}
		var paymentMethodsJSON []byte

		err := rows.Scan(
			&order.ID,
			&order.UserID,
			&order.Type,
			&order.Cryptocurrency,
			&order.FiatCurrency,
			&order.Amount,
			&order.Price,
			&order.TotalAmount,
			&order.MinAmount,
			&order.MaxAmount,
			&paymentMethodsJSON, // JSON из базы данных
			&order.Description,
			&order.Status,
			&order.CreatedAt,
			&order.UpdatedAt,
			&order.ExpiresAt,
			&order.CompletedAt,
			&order.IsActive,
		)
		if err != nil {
			return nil, fmt.Errorf("не удалось сканировать заявку: %w", err)
		}

		// Парсим JSON для способов оплаты
		if err := json.Unmarshal(paymentMethodsJSON, &order.PaymentMethods); err != nil {
			return nil, fmt.Errorf("не удалось парсить способы оплаты: %w", err)
		}

		// Устанавливаем значения по умолчанию для фронтенда
		order.ResponseCount = 0        // Будет вычисляться отдельно если нужно
		order.AcceptedResponseID = nil // Будет устанавливаться отдельно если нужно

		orders = append(orders, order)
	}

	// Проверяем на ошибки во время итерации
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка при итерации по заявкам: %w", err)
	}

	log.Printf("[INFO] Найдено заявок по фильтру: %d", len(orders))
	return orders, nil
}

// UpdateOrderStatus обновляет статус заявки
// Используется для смены статуса заявки (активная, сматчена, завершена, отменена)
func (r *Repository) UpdateOrderStatus(orderID int64, status model.OrderStatus) error {
	// SQL запрос для обновления статуса заявки
	query := `
		UPDATE orders 
		SET status = $1, updated_at = NOW()
		WHERE id = $2`

	// Выполняем обновление
	result, err := r.db.Exec(query, status, orderID)
	if err != nil {
		return fmt.Errorf("не удалось обновить статус заявки: %w", err)
	}

	// Проверяем, что заявка была обновлена
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("не удалось проверить результат обновления заявки: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("заявка с ID %d не найдена для обновления", orderID)
	}

	log.Printf("[INFO] Обновлен статус заявки ID=%d: status=%s", orderID, status)
	return nil
}

// =====================================================
// ВСПОМОГАТЕЛЬНЫЕ МЕТОДЫ
// =====================================================

// =====================================================
// МЕТОДЫ ДЛЯ РАБОТЫ СО СДЕЛКАМИ
// =====================================================

// CreateDeal создает новую сделку между двумя пользователями
// Вызывается когда две заявки успешно сопоставлены
func (r *Repository) CreateDeal(deal *model.Deal) error {
	// SQL запрос для создания новой сделки
	query := `
		INSERT INTO deals (
			buy_order_id, sell_order_id, buyer_id, seller_id,
			cryptocurrency, fiat_currency, amount, price, total_amount,
			payment_method, status
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		) RETURNING id, created_at`

	// Выбираем первый способ оплаты из массива
	paymentMethod := "bank_transfer" // По умолчанию
	if len(deal.PaymentMethods) > 0 {
		paymentMethod = deal.PaymentMethods[0]
	}

	// Выполняем запрос и получаем ID и время создания
	err := r.db.QueryRow(
		query,
		deal.ResponseID,
		deal.OrderID,
		deal.AuthorID,
		deal.CounterpartyID,
		deal.Cryptocurrency,
		deal.FiatCurrency,
		deal.Amount,
		deal.Price,
		deal.TotalAmount,
		paymentMethod, // Используем первый способ оплаты
		deal.Status,
	).Scan(&deal.ID, &deal.CreatedAt)

	if err != nil {
		return fmt.Errorf("не удалось создать сделку: %w", err)
	}

	log.Printf("[INFO] Создана новая сделка: ID=%d, AuthorID=%d, CounterpartyID=%d, Amount=%.8f %s",
		deal.ID, deal.AuthorID, deal.CounterpartyID, deal.Amount, deal.Cryptocurrency)

	return nil
}

// GetDealsByUserID получает все сделки пользователя (как покупателя и продавца)
func (r *Repository) GetDealsByUserID(userID int64) ([]*model.Deal, error) {
	// SQL запрос для поиска всех сделок пользователя
	query := `
		SELECT id, buy_order_id, sell_order_id, buyer_id, seller_id,
		       cryptocurrency, fiat_currency, amount, price, total_amount,
		       payment_method, status, created_at, completed_at,
		       author_confirmed, counter_confirmed, author_proof, notes
		FROM deals 
		WHERE buyer_id = $1 OR seller_id = $1
		ORDER BY created_at DESC`

	// Выполняем запрос
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить сделки пользователя: %w", err)
	}
	defer rows.Close()

	// Сканируем результаты
	var deals []*model.Deal
	for rows.Next() {
		deal := &model.Deal{}
		var paymentMethodStr string           // Временная переменная для сканирования payment_method
		var authorProof, notes sql.NullString // Переменные для NULL-значений

		err := rows.Scan(
			&deal.ID,
			&deal.ResponseID,
			&deal.OrderID,
			&deal.AuthorID,
			&deal.CounterpartyID,
			&deal.Cryptocurrency,
			&deal.FiatCurrency,
			&deal.Amount,
			&deal.Price,
			&deal.TotalAmount,
			&paymentMethodStr, // Сканируем в string, не в []string
			&deal.Status,
			&deal.CreatedAt,
			&deal.CompletedAt,
			&deal.AuthorConfirmed,
			&deal.CounterConfirmed,
			&authorProof, // NULL-safe сканирование author_proof
			&notes,       // NULL-safe сканирование notes
		)
		if err != nil {
			return nil, fmt.Errorf("не удалось сканировать сделку: %w", err)
		}

		// Конвертируем payment_method string в []string для совместимости с моделью
		deal.PaymentMethods = []string{paymentMethodStr}

		// Конвертируем NULL-значения в строки
		deal.AuthorProof = authorProof.String // sql.NullString.String возвращает "" если NULL
		deal.Notes = notes.String             // sql.NullString.String возвращает "" если NULL

		deals = append(deals, deal)
	}

	return deals, nil
}

// GetDealByID получает сделку по её ID
func (r *Repository) GetDealByID(dealID int64) (*model.Deal, error) {
	deal := &model.Deal{}

	// SQL запрос для поиска сделки по ID
	query := `
		SELECT id, buy_order_id, sell_order_id, buyer_id, seller_id,
		       cryptocurrency, fiat_currency, amount, price, total_amount,
		       payment_method, status, created_at, completed_at,
		       author_confirmed, counter_confirmed, author_proof, notes
		FROM deals 
		WHERE id = $1`

	// Выполняем запрос и сканируем результат
	var paymentMethodStr string           // Временная переменная для сканирования payment_method
	var authorProof, notes sql.NullString // Переменные для NULL-значений

	err := r.db.QueryRow(query, dealID).Scan(
		&deal.ID,
		&deal.ResponseID,
		&deal.OrderID,
		&deal.AuthorID,
		&deal.CounterpartyID,
		&deal.Cryptocurrency,
		&deal.FiatCurrency,
		&deal.Amount,
		&deal.Price,
		&deal.TotalAmount,
		&paymentMethodStr, // Сканируем в string, не в []string
		&deal.Status,
		&deal.CreatedAt,
		&deal.CompletedAt,
		&deal.AuthorConfirmed,
		&deal.CounterConfirmed,
		&authorProof, // NULL-safe сканирование author_proof
		&notes,       // NULL-safe сканирование notes
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("сделка с ID %d не найдена", dealID)
		}
		return nil, fmt.Errorf("не удалось найти сделку: %w", err)
	}

	// Конвертируем payment_method string в []string для совместимости с моделью
	deal.PaymentMethods = []string{paymentMethodStr}

	// Конвертируем NULL-значения в строки
	deal.AuthorProof = authorProof.String // sql.NullString.String возвращает "" если NULL
	deal.Notes = notes.String             // sql.NullString.String возвращает "" если NULL

	return deal, nil
}

// UpdateDealStatus обновляет статус сделки
func (r *Repository) UpdateDealStatus(dealID int64, status string) error {
	// SQL запрос для обновления статуса сделки
	query := `
		UPDATE deals 
		SET status = $1::varchar, completed_at = CASE 
			WHEN $1::varchar IN ('completed', 'expired', 'cancelled') THEN NOW() 
			ELSE completed_at 
		END
		WHERE id = $2`

	// Выполняем обновление
	result, err := r.db.Exec(query, status, dealID)
	if err != nil {
		return fmt.Errorf("не удалось обновить статус сделки: %w", err)
	}

	// Проверяем, что сделка была обновлена
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("не удалось проверить результат обновления сделки: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("сделка с ID %d не найдена для обновления", dealID)
	}

	log.Printf("[INFO] Обновлен статус сделки ID=%d: status=%s", dealID, status)
	return nil
}

// ConfirmDeal подтверждает сделку со стороны покупателя или продавца
func (r *Repository) ConfirmDeal(dealID int64, userID int64, isPaymentProof bool, paymentProof string) error {
	// Начинаем транзакцию для атомарного обновления
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("не удалось начать транзакцию: %w", err)
	}
	defer tx.Rollback() // Откатываем транзакцию если что-то пойдет не так

	// Получаем информацию о сделке
	var buyerID, sellerID int64
	var buyerConfirmed, sellerConfirmed bool
	var currentStatus string

	query := `
		SELECT buyer_id, seller_id, author_confirmed, counter_confirmed, status
		FROM deals 
		WHERE id = $1`

	err = tx.QueryRow(query, dealID).Scan(&buyerID, &sellerID, &buyerConfirmed, &sellerConfirmed, &currentStatus)
	if err != nil {
		return fmt.Errorf("не удалось получить информацию о сделке: %w", err)
	}

	// Определяем кто подтверждает: покупатель или продавец
	var updateQuery string
	var newStatus string

	if userID == buyerID {
		// Покупатель подтверждает получение криптовалюты
		updateQuery = `
			UPDATE deals 
			SET author_confirmed = true, updated_at = NOW()
			WHERE id = $1`
		buyerConfirmed = true
	} else if userID == sellerID {
		// Продавец подтверждает получение оплаты и может прикладывать доказательство
		if isPaymentProof {
			updateQuery = `
				UPDATE deals 
				SET counter_confirmed = true, author_proof = $2, updated_at = NOW()
				WHERE id = $1`
		} else {
			updateQuery = `
				UPDATE deals 
				SET counter_confirmed = true, updated_at = NOW()
				WHERE id = $1`
		}
		sellerConfirmed = true
	} else {
		return fmt.Errorf("пользователь не участвует в данной сделке")
	}

	// Выполняем обновление подтверждения
	if isPaymentProof && userID == sellerID {
		_, err = tx.Exec(updateQuery, dealID, paymentProof)
	} else {
		_, err = tx.Exec(updateQuery, dealID)
	}

	if err != nil {
		return fmt.Errorf("не удалось подтвердить сделку: %w", err)
	}

	// Если обе стороны подтвердили, завершаем сделку
	if buyerConfirmed && sellerConfirmed {
		newStatus = "completed"
		completeQuery := `
			UPDATE deals 
			SET status = $1, completed_at = NOW()
			WHERE id = $2`

		_, err = tx.Exec(completeQuery, newStatus, dealID)
		if err != nil {
			return fmt.Errorf("не удалось завершить сделку: %w", err)
		}

		// Обновляем статистику пользователей
		err = r.updateUserDealsStatsTx(tx, buyerID)
		if err != nil {
			return fmt.Errorf("не удалось обновить статистику покупателя: %w", err)
		}

		err = r.updateUserDealsStatsTx(tx, sellerID)
		if err != nil {
			return fmt.Errorf("не удалось обновить статистику продавца: %w", err)
		}

		log.Printf("[INFO] Сделка ID=%d завершена успешно", dealID)
	}

	// Фиксируем транзакцию
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("не удалось зафиксировать транзакцию: %w", err)
	}

	log.Printf("[INFO] Пользователь ID=%d подтвердил сделку ID=%d", userID, dealID)
	return nil
}

// updateUserDealsStatsTx обновляет статистику сделок пользователя в рамках транзакции
func (r *Repository) updateUserDealsStatsTx(tx *sql.Tx, userID int64) error {
	// Обновляем общее количество сделок и успешных сделок
	query := `
		UPDATE users 
		SET 
			total_deals = total_deals + 1,
			successful_deals = successful_deals + 1,
			updated_at = NOW()
		WHERE id = $1`

	_, err := tx.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("не удалось обновить статистику пользователя ID=%d: %w", userID, err)
	}

	return nil
}

// GetMatchingOrders находит заявки, подходящие для автоматического сопоставления
// Ищет заявки противоположного типа с совпадающими параметрами
func (r *Repository) GetMatchingOrders(order *model.Order) ([]*model.Order, error) {
	// Определяем противоположный тип заявки
	var oppositeType model.OrderType
	if order.Type == model.OrderTypeBuy {
		oppositeType = model.OrderTypeSell
	} else {
		oppositeType = model.OrderTypeBuy
	}

	// SQL запрос для поиска подходящих заявок
	query := `
		SELECT id, user_id, type, cryptocurrency, fiat_currency, 
		       amount, price, total_amount, min_amount, max_amount,
		       payment_methods, description, status, created_at,
		       updated_at, expires_at, matched_user_id, matched_at,
		       completed_at, is_active, auto_match
		FROM orders
		WHERE 
			type = $1 AND                          -- Противоположный тип
			cryptocurrency = $2 AND               -- Та же криптовалюта
			fiat_currency = $3 AND                -- Та же фиатная валюта
			status = 'active' AND                 -- Активная заявка
			is_active = true AND                  -- Не отключена
			auto_match = true AND                 -- Разрешено автосопоставление
			user_id != $4                         -- Не наша заявка
			-- Убираем проверку expires_at - таймеры больше не используются
		ORDER BY 
			CASE WHEN $1 = 'sell' THEN price END ASC,     -- Для покупки: сначала дешевые продажи
			CASE WHEN $1 = 'buy' THEN price END DESC,     -- Для продажи: сначала дорогие покупки  
			created_at ASC                                 -- При равной цене: сначала старые заявки
		LIMIT 10`

	// Выполняем запрос
	rows, err := r.db.Query(query, oppositeType, order.Cryptocurrency, order.FiatCurrency, order.UserID)
	if err != nil {
		return nil, fmt.Errorf("не удалось найти подходящие заявки: %w", err)
	}
	defer rows.Close()

	// Сканируем результаты
	var matchingOrders []*model.Order
	for rows.Next() {
		matchingOrder := &model.Order{}
		err := rows.Scan(
			&matchingOrder.ID,
			&matchingOrder.UserID,
			&matchingOrder.Type,
			&matchingOrder.Cryptocurrency,
			&matchingOrder.FiatCurrency,
			&matchingOrder.Amount,
			&matchingOrder.Price,
			&matchingOrder.TotalAmount,
			&matchingOrder.MinAmount,
			&matchingOrder.MaxAmount,
			&matchingOrder.PaymentMethods,
			&matchingOrder.Description,
			&matchingOrder.Status,
			&matchingOrder.CreatedAt,
			&matchingOrder.UpdatedAt,
			&matchingOrder.ExpiresAt,
			&matchingOrder.CompletedAt,
			&matchingOrder.IsActive,
			&matchingOrder.ResponseCount,
			&matchingOrder.AcceptedResponseID,
		)
		if err != nil {
			return nil, fmt.Errorf("не удалось сканировать подходящую заявку: %w", err)
		}
		matchingOrders = append(matchingOrders, matchingOrder)
	}

	log.Printf("[INFO] Найдено подходящих заявок для Order ID=%d: %d", order.ID, len(matchingOrders))
	return matchingOrders, nil
}

// MatchOrders сопоставляет две заявки и обновляет их статус
func (r *Repository) MatchOrders(orderID1, orderID2 int64) error {
	// Начинаем транзакцию
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("не удалось начать транзакцию сопоставления: %w", err)
	}
	defer tx.Rollback()

	// Текущее время для сопоставления
	now := time.Now()

	// Обновляем первую заявку
	query1 := `
		UPDATE orders 
		SET 
			status = 'matched', 
			matched_user_id = (SELECT user_id FROM orders WHERE id = $2),
			matched_at = $3,
			updated_at = NOW()
		WHERE id = $1 AND status = 'active'`

	result1, err := tx.Exec(query1, orderID1, orderID2, now)
	if err != nil {
		return fmt.Errorf("не удалось обновить первую заявку: %w", err)
	}

	// Обновляем вторую заявку
	query2 := `
		UPDATE orders 
		SET 
			status = 'matched', 
			matched_user_id = (SELECT user_id FROM orders WHERE id = $2),
			matched_at = $3,
			updated_at = NOW()
		WHERE id = $1 AND status = 'active'`

	result2, err := tx.Exec(query2, orderID2, orderID1, now)
	if err != nil {
		return fmt.Errorf("не удалось обновить вторую заявку: %w", err)
	}

	// Проверяем что обе заявки были обновлены
	rows1, _ := result1.RowsAffected()
	rows2, _ := result2.RowsAffected()

	if rows1 == 0 || rows2 == 0 {
		return fmt.Errorf("одна из заявок недоступна для сопоставления")
	}

	// Фиксируем транзакцию
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("не удалось зафиксировать сопоставление: %w", err)
	}

	log.Printf("[INFO] Заявки сопоставлены: OrderID1=%d, OrderID2=%d", orderID1, orderID2)
	return nil
}

// =====================================================
// МЕТОДЫ ДЛЯ РАБОТЫ С ОТЗЫВАМИ И РЕЙТИНГАМИ
// =====================================================

// CreateReview создает новый отзыв о пользователе после завершения сделки
func (r *Repository) CreateReview(review *model.Review) error {
	// SQL запрос для создания отзыва
	query := `
		INSERT INTO reviews (
			deal_id, from_user_id, to_user_id, rating, type,
			comment, is_anonymous
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		) RETURNING id, created_at, updated_at`

	// Определяем тип отзыва на основе рейтинга
	var reviewType model.ReviewType
	if review.Rating >= 4 {
		reviewType = model.ReviewTypePositive
	} else if review.Rating == 3 {
		reviewType = model.ReviewTypeNeutral
	} else {
		reviewType = model.ReviewTypeNegative
	}

	// Выполняем запрос
	err := r.db.QueryRow(
		query,
		review.DealID,
		review.FromUserID,
		review.ToUserID,
		review.Rating,
		reviewType,
		review.Comment,
		review.IsAnonymous,
	).Scan(&review.ID, &review.CreatedAt, &review.UpdatedAt)

	if err != nil {
		return fmt.Errorf("не удалось создать отзыв: %w", err)
	}

	log.Printf("[INFO] Создан отзыв: ID=%d, FromUser=%d, ToUser=%d, Rating=%d",
		review.ID, review.FromUserID, review.ToUserID, review.Rating)

	return nil
}

// GetReviewsByUserID получает все отзывы о пользователе
func (r *Repository) GetReviewsByUserID(userID int64, limit, offset int) ([]*model.Review, error) {
	// SQL запрос для получения отзывов о пользователе
	query := `
		SELECT r.id, r.deal_id, r.from_user_id, r.to_user_id, r.rating,
		       r.type, r.comment, r.is_anonymous, r.created_at, r.updated_at,
		       r.is_visible, r.reported_count,
		       CASE 
			   WHEN r.is_anonymous THEN 'Аноним' 
			   ELSE CONCAT(u.first_name, CASE WHEN u.last_name IS NOT NULL THEN ' ' || u.last_name ELSE '' END) 
		       END as from_user_name,
		       CASE WHEN r.is_anonymous THEN '' ELSE u.username END as from_user_username
		FROM reviews r
		LEFT JOIN users u ON u.id = r.from_user_id
		WHERE r.to_user_id = $1 AND r.is_visible = true
		ORDER BY r.created_at DESC
		LIMIT $2 OFFSET $3`

	// Выполняем запрос
	rows, err := r.db.Query(query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить отзывы: %w", err)
	}
	defer rows.Close()

	// Сканируем результаты
	var reviews []*model.Review
	for rows.Next() {
		review := &model.Review{}

		err := rows.Scan(
			&review.ID,
			&review.DealID,
			&review.FromUserID,
			&review.ToUserID,
			&review.Rating,
			&review.Type,
			&review.Comment,
			&review.IsAnonymous,
			&review.CreatedAt,
			&review.UpdatedAt,
			&review.IsVisible,
			&review.ReportedCount,
			&review.FromUserName,
			&review.FromUserUsername,
		)
		if err != nil {
			return nil, fmt.Errorf("не удалось сканировать отзыв: %w", err)
		}

		reviews = append(reviews, review)
	}

	log.Printf("[INFO] Получено отзывов для пользователя ID=%d: %d", userID, len(reviews))
	return reviews, nil
}

// GetUserRating получает рейтинг пользователя
func (r *Repository) GetUserRating(userID int64) (*model.Rating, error) {
	rating := &model.Rating{UserID: userID}

	// SQL запрос для получения рейтинга пользователя
	query := `
		SELECT average_rating, total_reviews, positive_reviews, neutral_reviews,
		       negative_reviews, five_stars, four_stars, three_stars, two_stars,
		       one_star, updated_at
		FROM ratings
		WHERE user_id = $1`

	err := r.db.QueryRow(query, userID).Scan(
		&rating.AverageRating,
		&rating.TotalReviews,
		&rating.PositiveReviews,
		&rating.NeutralReviews,
		&rating.NegativeReviews,
		&rating.FiveStars,
		&rating.FourStars,
		&rating.ThreeStars,
		&rating.TwoStars,
		&rating.OneStar,
		&rating.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// Если рейтинга еще нет, создаем пустой
			rating.AverageRating = 0.0
			rating.TotalReviews = 0
			return rating, nil
		}
		return nil, fmt.Errorf("не удалось получить рейтинг пользователя: %w", err)
	}

	return rating, nil
}

// CheckCanReview проверяет может ли пользователь оставить отзыв для сделки
func (r *Repository) CheckCanReview(dealID, fromUserID, toUserID int64) (bool, error) {
	// Проверяем что сделка завершена и пользователь участвовал в ней
	var dealStatus string
	var buyerID, sellerID int64

	dealQuery := `
		SELECT status, buyer_id, seller_id
		FROM deals
		WHERE id = $1`

	err := r.db.QueryRow(dealQuery, dealID).Scan(&dealStatus, &buyerID, &sellerID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, fmt.Errorf("сделка не найдена")
		}
		return false, fmt.Errorf("не удалось проверить сделку: %w", err)
	}

	// Проверяем что сделка завершена
	if dealStatus != "completed" {
		return false, fmt.Errorf("отзыв можно оставить только для завершенных сделок")
	}

	// Проверяем что пользователь участвовал в сделке
	if fromUserID != buyerID && fromUserID != sellerID {
		return false, fmt.Errorf("вы не участвовали в данной сделке")
	}

	// Проверяем что получатель отзыва тоже участвовал в сделке
	if toUserID != buyerID && toUserID != sellerID {
		return false, fmt.Errorf("получатель отзыва не участвовал в сделке")
	}

	// Проверяем что пользователь не пытается оставить отзыв самому себе
	if fromUserID == toUserID {
		return false, fmt.Errorf("нельзя оставлять отзыв самому себе")
	}

	// Проверяем что отзыв еще не был оставлен
	var existingReviewID int64
	reviewQuery := `
		SELECT id FROM reviews
		WHERE deal_id = $1 AND from_user_id = $2
		LIMIT 1`

	err = r.db.QueryRow(reviewQuery, dealID, fromUserID).Scan(&existingReviewID)
	if err == nil {
		return false, fmt.Errorf("вы уже оставили отзыв для этой сделки")
	} else if err != sql.ErrNoRows {
		return false, fmt.Errorf("не удалось проверить существующие отзывы: %w", err)
	}

	return true, nil
}

// ReportReview создает жалобу на отзыв
func (r *Repository) ReportReview(report *model.ReviewReport) error {
	// SQL запрос для создания жалобы
	query := `
		INSERT INTO review_reports (
			review_id, user_id, reason, comment
		) VALUES (
			$1, $2, $3, $4
		) RETURNING id, created_at`

	err := r.db.QueryRow(
		query,
		report.ReviewID,
		report.UserID,
		report.Reason,
		report.Comment,
	).Scan(&report.ID, &report.CreatedAt)

	if err != nil {
		return fmt.Errorf("не удалось создать жалобу на отзыв: %w", err)
	}

	// Увеличиваем счетчик жалоб на отзыв
	updateQuery := `
		UPDATE reviews
		SET reported_count = reported_count + 1
		WHERE id = $1`

	_, err = r.db.Exec(updateQuery, report.ReviewID)
	if err != nil {
		log.Printf("[WARN] Не удалось обновить счетчик жалоб для отзыва ID=%d: %v", report.ReviewID, err)
	}

	log.Printf("[INFO] Создана жалоба на отзыв: ReportID=%d, ReviewID=%d, UserID=%d",
		report.ID, report.ReviewID, report.UserID)

	return nil
}

// GetUserReviewStats получает статистику отзывов пользователя для профиля
func (r *Repository) GetUserReviewStats(userID int64) (*model.ReviewStats, error) {
	stats := &model.ReviewStats{UserID: userID}

	// Получаем основную статистику рейтинга
	rating, err := r.GetUserRating(userID)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить рейтинг: %w", err)
	}

	stats.AverageRating = rating.AverageRating
	stats.TotalReviews = rating.TotalReviews

	// Вычисляем процент положительных отзывов
	if rating.TotalReviews > 0 {
		stats.PositivePercent = float32(rating.PositiveReviews) / float32(rating.TotalReviews) * 100
	}

	// Получаем последние отзывы (до 5)
	recentReviews, err := r.GetReviewsByUserID(userID, 5, 0)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить последние отзывы: %w", err)
	}

	// Конвертируем указатели в значения для совместимости типов
	convertedReviews := make([]model.Review, len(recentReviews))
	for i, review := range recentReviews {
		convertedReviews[i] = *review
	}
	stats.RecentReviews = convertedReviews

	// Создаем распределение по звездам
	stats.RatingDistribution = map[int]int{
		1: rating.OneStar,
		2: rating.TwoStars,
		3: rating.ThreeStars,
		4: rating.FourStars,
		5: rating.FiveStars,
	}

	return stats, nil
}

// HealthCheck проверяет доступность базы данных
// Используется для мониторинга состояния соединения
func (r *Repository) HealthCheck() error {
	// Простой запрос для проверки соединения
	var result int
	err := r.db.QueryRow("SELECT 1").Scan(&result)
	if err != nil {
		return fmt.Errorf("база данных недоступна: %w", err)
	}
	return nil
}

// ConfirmDealWithRole обновляет статус сделки с указанием роли пользователя (PostgreSQL)
func (r *Repository) ConfirmDealWithRole(dealID int64, userID int64, isAuthor bool, paymentProof string) error {
	log.Printf("[INFO] Подтверждение сделки ID=%d пользователем ID=%d (isAuthor=%v)", dealID, userID, isAuthor)

	query := `
		UPDATE deals 
		SET 
			author_confirmed = CASE WHEN $2 = true THEN true ELSE author_confirmed END,
			author_proof = CASE WHEN $2 = true THEN $3 ELSE author_proof END,
			counter_confirmed = CASE WHEN $2 = false THEN true ELSE counter_confirmed END,
			counter_proof = CASE WHEN $2 = false THEN $3 ELSE counter_proof END,
			status = CASE 
				WHEN (
					($2 = true AND counter_confirmed = true) OR 
					($2 = false AND author_confirmed = true)
				) THEN 'completed' 
				ELSE status 
			END,
			completed_at = CASE 
				WHEN (
					($2 = true AND counter_confirmed = true) OR 
					($2 = false AND author_confirmed = true)
				) THEN NOW() 
				ELSE completed_at 
			END
		WHERE id = $1`

	result, err := r.db.Exec(query, dealID, isAuthor, paymentProof)
	if err != nil {
		return fmt.Errorf("не удалось подтвердить сделку: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("не удалось получить количество обновленных записей: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("сделка с ID=%d не найдена", dealID)
	}

	// Проверяем завершилась ли сделка (обновляем счетчики только при завершении)
	var completedStatus string
	checkQuery := `SELECT status FROM deals WHERE id = $1`
	err = r.db.QueryRow(checkQuery, dealID).Scan(&completedStatus)
	if err != nil {
		log.Printf("[WARN] Не удалось проверить статус сделки ID=%d: %v", dealID, err)
	} else if completedStatus == "completed" {
		// Получаем участников сделки для обновления их счетчиков
		var authorID, counterpartyID int64
		participantsQuery := `SELECT author_id, counterparty_id FROM deals WHERE id = $1`
		err = r.db.QueryRow(participantsQuery, dealID).Scan(&authorID, &counterpartyID)
		if err != nil {
			log.Printf("[WARN] Не удалось получить участников сделки ID=%d: %v", dealID, err)
		} else {
			// Начинаем транзакцию для обновления счетчиков
			tx, err := r.db.Begin()
			if err != nil {
				log.Printf("[WARN] Не удалось начать транзакцию для обновления счетчиков: %v", err)
			} else {
				// Обновляем счетчики обеих сторон
				updateStatsQuery := `
					UPDATE users 
					SET 
						total_deals = total_deals + 1,
						successful_deals = successful_deals + 1,
						updated_at = NOW()
					WHERE id = $1`

				// Обновляем автора
				_, err = tx.Exec(updateStatsQuery, authorID)
				if err != nil {
					tx.Rollback()
					log.Printf("[WARN] Не удалось обновить счетчик автора ID=%d: %v", authorID, err)
				} else {
					// Обновляем контрагента
					_, err = tx.Exec(updateStatsQuery, counterpartyID)
					if err != nil {
						tx.Rollback()
						log.Printf("[WARN] Не удалось обновить счетчик контрагента ID=%d: %v", counterpartyID, err)
					} else {
						// Фиксируем транзакцию
						err = tx.Commit()
						if err != nil {
							log.Printf("[WARN] Не удалось зафиксировать обновление счетчиков: %v", err)
						} else {
							log.Printf("[INFO] Обновлены счетчики сделок для пользователей ID=%d и ID=%d", authorID, counterpartyID)
						}
					}
				}
			}
		}
	}

	log.Printf("[INFO] Сделка ID=%d подтверждена пользователем ID=%d как %s", dealID, userID,
		map[bool]string{true: "автор", false: "контрагент"}[isAuthor])

	return nil
}

// GetExpiredDeals получает активные сделки старше указанного времени (PostgreSQL)
func (r *Repository) GetExpiredDeals(cutoffTime time.Time) ([]*model.Deal, error) {
	log.Printf("[INFO] Поиск сделок созданных до %v", cutoffTime)

	query := `
		SELECT id, buy_order_id, sell_order_id, buyer_id, seller_id,
		       cryptocurrency, fiat_currency, amount, price, total_amount,
		       payment_method, status, created_at, completed_at,
		       author_confirmed, counter_confirmed, author_proof, notes
		FROM deals 
		WHERE (status = $1 OR status = $2) 
		AND created_at < $3
		ORDER BY created_at ASC`

	rows, err := r.db.Query(query,
		model.DealStatusInProgress,
		model.DealStatusWaitingConfirmation,
		cutoffTime)
	if err != nil {
		return nil, fmt.Errorf("не удалось найти устаревшие сделки: %w", err)
	}
	defer rows.Close()

	var deals []*model.Deal
	for rows.Next() {
		deal := &model.Deal{}
		var paymentMethodStr string           // Временная переменная для сканирования payment_method
		var authorProof, notes sql.NullString // Переменные для NULL-значений

		err := rows.Scan(
			&deal.ID,
			&deal.ResponseID,     // buy_order_id
			&deal.OrderID,        // sell_order_id
			&deal.AuthorID,       // buyer_id
			&deal.CounterpartyID, // seller_id
			&deal.Cryptocurrency,
			&deal.FiatCurrency,
			&deal.Amount,
			&deal.Price,
			&deal.TotalAmount,
			&paymentMethodStr, // payment_method как строка
			&deal.Status,
			&deal.CreatedAt,
			&deal.CompletedAt,
			&deal.AuthorConfirmed,
			&deal.CounterConfirmed,
			&authorProof, // NULL-safe сканирование
			&notes,       // NULL-safe сканирование
		)
		if err != nil {
			log.Printf("[ERROR] Ошибка сканирования сделки: %v", err)
			continue
		}

		// Конвертируем payment_method string в []string для совместимости с моделью
		deal.PaymentMethods = []string{paymentMethodStr}

		// Конвертируем NULL-значения в строки
		deal.AuthorProof = authorProof.String // sql.NullString.String возвращает "" если NULL
		deal.Notes = notes.String             // sql.NullString.String возвращает "" если NULL

		deals = append(deals, deal)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка при чтении сделок: %w", err)
	}

	log.Printf("[INFO] Найдено %d устаревших сделок", len(deals))
	return deals, nil
}

// GetUserByID получает пользователя по его внутреннему ID (PostgreSQL)
func (r *Repository) GetUserByID(userID int64) (*model.User, error) {
	log.Printf("[INFO] Получение пользователя по ID=%d", userID)

	query := `
		SELECT id, telegram_id, telegram_user_id, first_name, last_name, 
		       username, photo_url, is_bot, language_code, created_at, 
		       updated_at, is_active, rating, total_deals, successful_deals, chat_member
		FROM users 
		WHERE id = $1`

	user := &model.User{}
	err := r.db.QueryRow(query, userID).Scan(
		&user.ID, &user.TelegramID, &user.TelegramUserID, &user.FirstName, &user.LastName,
		&user.Username, &user.PhotoURL, &user.IsBot, &user.LanguageCode, &user.CreatedAt,
		&user.UpdatedAt, &user.IsActive, &user.Rating, &user.TotalDeals, &user.SuccessfulDeals, &user.ChatMember,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("пользователь с ID=%d не найден", userID)
		}
		return nil, fmt.Errorf("не удалось получить пользователя: %w", err)
	}

	log.Printf("[INFO] Пользователь найден: ID=%d, TelegramID=%d", user.ID, user.TelegramID)
	return user, nil
}

// GetOrderByID получает заявку по ID (PostgreSQL)
func (r *Repository) GetOrderByID(orderID int64) (*model.Order, error) {
	log.Printf("[INFO] Получение заявки по ID=%d", orderID)

	query := `
		SELECT id, user_id, type, cryptocurrency, fiat_currency, amount, price, total_amount,
		       min_amount, max_amount, payment_methods, description, status, created_at, updated_at,
		       expires_at, completed_at, is_active
		FROM orders 
		WHERE id = $1`

	order := &model.Order{}
	var paymentMethodsJSON []byte
	err := r.db.QueryRow(query, orderID).Scan(
		&order.ID, &order.UserID, &order.Type, &order.Cryptocurrency, &order.FiatCurrency,
		&order.Amount, &order.Price, &order.TotalAmount, &order.MinAmount, &order.MaxAmount,
		&paymentMethodsJSON, &order.Description, &order.Status, &order.CreatedAt, &order.UpdatedAt,
		&order.ExpiresAt, &order.CompletedAt, &order.IsActive,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("заявка с ID=%d не найдена", orderID)
		}
		return nil, fmt.Errorf("не удалось получить заявку: %w", err)
	}

	// Парсим JSON для способов оплаты
	if err := json.Unmarshal(paymentMethodsJSON, &order.PaymentMethods); err != nil {
		return nil, fmt.Errorf("не удалось парсить способы оплаты: %w", err)
	}

	log.Printf("[INFO] Заявка найдена: ID=%d, Type=%s, Amount=%.2f", order.ID, order.Type, order.Amount)
	return order, nil
}

// UpdateOrder обновляет заявку в базе данных (PostgreSQL)
func (r *Repository) UpdateOrder(order *model.Order) error {
	log.Printf("[INFO] Обновление заявки ID=%d", order.ID)

	// Сериализуем способы оплаты в JSON
	paymentMethodsJSON, err := json.Marshal(order.PaymentMethods)
	if err != nil {
		return fmt.Errorf("не удалось сериализовать способы оплаты: %w", err)
	}

	query := `
		UPDATE orders 
		SET type = $2, cryptocurrency = $3, fiat_currency = $4, amount = $5, price = $6, 
		    total_amount = $7, min_amount = $8, max_amount = $9, payment_methods = $10, 
		    description = $11, updated_at = NOW()
		WHERE id = $1`

	result, err := r.db.Exec(query,
		order.ID, order.Type, order.Cryptocurrency, order.FiatCurrency, order.Amount,
		order.Price, order.TotalAmount, order.MinAmount, order.MaxAmount,
		paymentMethodsJSON, order.Description,
	)
	if err != nil {
		return fmt.Errorf("не удалось обновить заявку: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("не удалось получить количество обновленных строк: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("заявка с ID=%d не найдена", order.ID)
	}

	log.Printf("[INFO] Заявка ID=%d успешно обновлена", order.ID)
	return nil
}

// =====================================================
// МЕТОДЫ ДЛЯ РАБОТЫ С ОТКЛИКАМИ (RESPONSES)
// =====================================================

// CreateResponse создает новый отклик в базе данных (PostgreSQL)
func (r *Repository) CreateResponse(response *model.Response) error {
	log.Printf("[INFO] Создание отклика на заявку ID=%d от пользователя ID=%d", response.OrderID, response.UserID)

	query := `
		INSERT INTO responses (order_id, user_id, message, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id, created_at, updated_at`

	err := r.db.QueryRow(query, response.OrderID, response.UserID, response.Message, string(response.Status)).
		Scan(&response.ID, &response.CreatedAt, &response.UpdatedAt)
	if err != nil {
		return fmt.Errorf("не удалось создать отклик: %w", err)
	}

	log.Printf("[INFO] Отклик создан с ID=%d", response.ID)
	return nil
}

// UpdateResponseStatus обновляет статус отклика (PostgreSQL)
func (r *Repository) UpdateResponseStatus(responseID int64, status model.ResponseStatus) error {
	log.Printf("[INFO] Обновление статуса отклика ID=%d на %s", responseID, status)

	query := `
		UPDATE responses 
		SET status = $2, updated_at = NOW(), reviewed_at = CASE WHEN $3 != 'waiting' THEN NOW() ELSE reviewed_at END
		WHERE id = $1`

	result, err := r.db.Exec(query, responseID, string(status), string(status))
	if err != nil {
		return fmt.Errorf("не удалось обновить статус отклика: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("не удалось получить количество обновленных строк: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("отклик с ID=%d не найден", responseID)
	}

	log.Printf("[INFO] Статус отклика ID=%d обновлен на %s", responseID, status)
	return nil
}

// GetResponsesByFilter получает отклики по фильтру (PostgreSQL)
func (r *Repository) GetResponsesByFilter(filter *model.ResponseFilter) ([]*model.Response, error) {
	log.Printf("[INFO] Получение откликов по фильтру: %+v", filter)

	// Базовый запрос
	query := `
		SELECT r.id, r.order_id, r.user_id, r.message, r.status, r.created_at, r.updated_at, r.reviewed_at,
		       u.first_name || COALESCE(' ' || u.last_name, '') as user_name, u.username,
		       o.type, o.cryptocurrency, o.fiat_currency, o.amount, o.price, o.total_amount,
		       author.first_name || COALESCE(' ' || author.last_name, '') as author_name, author.username as author_username
		FROM responses r
		JOIN users u ON r.user_id = u.id
		JOIN orders o ON r.order_id = o.id
		JOIN users author ON o.user_id = author.id
		WHERE 1=1`

	args := []interface{}{}
	argIndex := 1

	// Добавляем условия фильтрации
	if filter.OrderID != nil {
		query += fmt.Sprintf(" AND r.order_id = $%d", argIndex)
		args = append(args, *filter.OrderID)
		argIndex++
	}

	if filter.UserID != nil {
		query += fmt.Sprintf(" AND r.user_id = $%d", argIndex)
		args = append(args, *filter.UserID)
		argIndex++
	}

	if filter.AuthorID != nil {
		query += fmt.Sprintf(" AND o.user_id = $%d", argIndex)
		args = append(args, *filter.AuthorID)
		argIndex++
	}

	if filter.Status != nil {
		query += fmt.Sprintf(" AND r.status = $%d", argIndex)
		args = append(args, string(*filter.Status))
		argIndex++
	}

	// Сортировка
	if filter.SortBy != "" {
		sortBy := "r.created_at" // по умолчанию
		switch filter.SortBy {
		case "created_at":
			sortBy = "r.created_at"
		case "updated_at":
			sortBy = "r.updated_at"
		}

		sortOrder := "DESC" // по умолчанию
		if filter.SortOrder == "asc" {
			sortOrder = "ASC"
		}

		query += fmt.Sprintf(" ORDER BY %s %s", sortBy, sortOrder)
	} else {
		query += " ORDER BY r.created_at DESC"
	}

	// Лимит и оффсет
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filter.Limit)
		argIndex++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filter.Offset)
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("не удалось выполнить запрос откликов: %w", err)
	}
	defer rows.Close()

	var responses []*model.Response
	for rows.Next() {
		response := &model.Response{}
		err := rows.Scan(
			&response.ID, &response.OrderID, &response.UserID, &response.Message, &response.Status,
			&response.CreatedAt, &response.UpdatedAt, &response.ReviewedAt,
			&response.UserName, &response.Username,
			&response.OrderType, &response.Cryptocurrency, &response.FiatCurrency,
			&response.Amount, &response.Price, &response.TotalAmount,
			&response.AuthorName, &response.AuthorUsername,
		)
		if err != nil {
			return nil, fmt.Errorf("не удалось прочитать отклик: %w", err)
		}
		responses = append(responses, response)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка при чтении откликов: %w", err)
	}

	log.Printf("[INFO] Найдено откликов: %d", len(responses))
	return responses, nil
}

// GetResponsesForOrder получает все отклики для заявки (PostgreSQL)
func (r *Repository) GetResponsesForOrder(orderID int64) ([]*model.Response, error) {
	filter := &model.ResponseFilter{
		OrderID:   &orderID,
		SortBy:    "created_at",
		SortOrder: "asc",
	}
	return r.GetResponsesByFilter(filter)
}

// GetResponsesFromUser получает все отклики пользователя (PostgreSQL)
func (r *Repository) GetResponsesFromUser(userID int64) ([]*model.Response, error) {
	filter := &model.ResponseFilter{
		UserID:    &userID,
		SortBy:    "created_at",
		SortOrder: "desc",
	}
	return r.GetResponsesByFilter(filter)
}

// GetResponsesForAuthor получает отклики на заявки автора (PostgreSQL)
func (r *Repository) GetResponsesForAuthor(authorID int64) ([]*model.Response, error) {
	filter := &model.ResponseFilter{
		AuthorID:  &authorID,
		SortBy:    "created_at",
		SortOrder: "desc",
	}
	return r.GetResponsesByFilter(filter)
}
