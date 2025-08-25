package repository

import (
	"database/sql"
	"fmt"
	"log"
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

	// Выполняем запрос и получаем сгенерированные поля
	err := r.db.QueryRow(
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
		order.PaymentMethods, // Gorilla/mux автоматически преобразует []string в JSONB
		order.Description,
		order.ExpiresAt,
		order.AutoMatch,
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
		       updated_at, expires_at, matched_user_id, matched_at,
		       completed_at, is_active, auto_match
		FROM orders
		WHERE is_active = true`

	// Параметры для подстановки в запрос
	args := []interface{}{}
	argCount := 0

	// Динамически добавляем условия фильтрации
	if filter.Type != nil {
		argCount++
		query += fmt.Sprintf(" AND type = $%d", argCount)
		args = append(args, *filter.Type)
	}

	if filter.Cryptocurrency != nil {
		argCount++
		query += fmt.Sprintf(" AND cryptocurrency = $%d", argCount)
		args = append(args, *filter.Cryptocurrency)
	}

	if filter.FiatCurrency != nil {
		argCount++
		query += fmt.Sprintf(" AND fiat_currency = $%d", argCount)
		args = append(args, *filter.FiatCurrency)
	}

	if filter.Status != nil {
		argCount++
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, *filter.Status)
	}

	if filter.UserID != nil {
		argCount++
		query += fmt.Sprintf(" AND user_id = $%d", argCount)
		args = append(args, *filter.UserID)
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
			&order.PaymentMethods,
			&order.Description,
			&order.Status,
			&order.CreatedAt,
			&order.UpdatedAt,
			&order.ExpiresAt,
			&order.MatchedUserID,
			&order.MatchedAt,
			&order.CompletedAt,
			&order.IsActive,
			&order.AutoMatch,
		)
		if err != nil {
			return nil, fmt.Errorf("не удалось сканировать заявку: %w", err)
		}
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

	// Выполняем запрос и получаем ID и время создания
	err := r.db.QueryRow(
		query,
		deal.BuyOrderID,
		deal.SellOrderID,
		deal.BuyerID,
		deal.SellerID,
		deal.Cryptocurrency,
		deal.FiatCurrency,
		deal.Amount,
		deal.Price,
		deal.TotalAmount,
		deal.PaymentMethod,
		deal.Status,
	).Scan(&deal.ID, &deal.CreatedAt)

	if err != nil {
		return fmt.Errorf("не удалось создать сделку: %w", err)
	}

	log.Printf("[INFO] Создана новая сделка: ID=%d, BuyerID=%d, SellerID=%d, Amount=%.8f %s",
		deal.ID, deal.BuyerID, deal.SellerID, deal.Amount, deal.Cryptocurrency)

	return nil
}

// GetDealsByUserID получает все сделки пользователя (как покупателя и продавца)
func (r *Repository) GetDealsByUserID(userID int64) ([]*model.Deal, error) {
	// SQL запрос для поиска всех сделок пользователя
	query := `
		SELECT id, buy_order_id, sell_order_id, buyer_id, seller_id,
		       cryptocurrency, fiat_currency, amount, price, total_amount,
		       payment_method, status, created_at, completed_at,
		       buyer_confirmed, seller_confirmed, payment_proof, notes
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
		err := rows.Scan(
			&deal.ID,
			&deal.BuyOrderID,
			&deal.SellOrderID,
			&deal.BuyerID,
			&deal.SellerID,
			&deal.Cryptocurrency,
			&deal.FiatCurrency,
			&deal.Amount,
			&deal.Price,
			&deal.TotalAmount,
			&deal.PaymentMethod,
			&deal.Status,
			&deal.CreatedAt,
			&deal.CompletedAt,
			&deal.BuyerConfirmed,
			&deal.SellerConfirmed,
			&deal.PaymentProof,
			&deal.Notes,
		)
		if err != nil {
			return nil, fmt.Errorf("не удалось сканировать сделку: %w", err)
		}
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
		       buyer_confirmed, seller_confirmed, payment_proof, notes
		FROM deals 
		WHERE id = $1`

	// Выполняем запрос и сканируем результат
	err := r.db.QueryRow(query, dealID).Scan(
		&deal.ID,
		&deal.BuyOrderID,
		&deal.SellOrderID,
		&deal.BuyerID,
		&deal.SellerID,
		&deal.Cryptocurrency,
		&deal.FiatCurrency,
		&deal.Amount,
		&deal.Price,
		&deal.TotalAmount,
		&deal.PaymentMethod,
		&deal.Status,
		&deal.CreatedAt,
		&deal.CompletedAt,
		&deal.BuyerConfirmed,
		&deal.SellerConfirmed,
		&deal.PaymentProof,
		&deal.Notes,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("сделка с ID %d не найдена", dealID)
		}
		return nil, fmt.Errorf("не удалось найти сделку: %w", err)
	}

	return deal, nil
}

// UpdateDealStatus обновляет статус сделки
func (r *Repository) UpdateDealStatus(dealID int64, status string) error {
	// SQL запрос для обновления статуса сделки
	query := `
		UPDATE deals 
		SET status = $1, updated_at = NOW()
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
		SELECT buyer_id, seller_id, buyer_confirmed, seller_confirmed, status
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
			SET buyer_confirmed = true, updated_at = NOW()
			WHERE id = $1`
		buyerConfirmed = true
	} else if userID == sellerID {
		// Продавец подтверждает получение оплаты и может прикладывать доказательство
		if isPaymentProof {
			updateQuery = `
				UPDATE deals 
				SET seller_confirmed = true, payment_proof = $2, updated_at = NOW()
				WHERE id = $1`
		} else {
			updateQuery = `
				UPDATE deals 
				SET seller_confirmed = true, updated_at = NOW()
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
			user_id != $4 AND                     -- Не наша заявка
			expires_at > NOW()                    -- Не истекла
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
			&matchingOrder.MatchedUserID,
			&matchingOrder.MatchedAt,
			&matchingOrder.CompletedAt,
			&matchingOrder.IsActive,
			&matchingOrder.AutoMatch,
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
		       CASE WHEN r.is_anonymous THEN 'Аноним' ELSE u.first_name END as reviewer_name
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
		var reviewerName string

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
			&reviewerName,
		)
		if err != nil {
			return nil, fmt.Errorf("не удалось сканировать отзыв: %w", err)
		}

		// Дополняем отзыв информацией об авторе (если не анонимный)
		if !review.IsAnonymous {
			// Можно добавить дополнительную информацию об авторе отзыва
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
