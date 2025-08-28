package repository

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"p2pTG-crypto-exchange/internal/model"
)

// FileRepository представляет файловое хранилище данных в JSON формате
// Реализует тот же интерфейс что и PostgreSQL репозиторий, но использует JSON файлы
type FileRepository struct {
	dataDir string       // Путь к папке с данными
	mutex   sync.RWMutex // Мютекс для потокобезопасности при работе с файлами
}

// NewFileRepository создает новый файловый репозиторий
func NewFileRepository(dataDir string) (*FileRepository, error) {
	// Создаем папку для данных если она не существует
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("не удалось создать папку данных: %w", err)
	}

	repo := &FileRepository{
		dataDir: dataDir,
	}

	// Инициализируем файлы с пустыми данными если они не существуют
	if err := repo.initializeFiles(); err != nil {
		return nil, fmt.Errorf("не удалось инициализировать файлы: %w", err)
	}

	log.Printf("[INFO] Файловый репозиторий инициализирован в папке: %s", dataDir)
	return repo, nil
}

// initializeFiles создает начальные JSON файлы если они не существуют
func (r *FileRepository) initializeFiles() error {
	// Список файлов для инициализации
	files := map[string]interface{}{
		"users.json":          []model.User{},
		"orders.json":         []model.Order{},
		"deals.json":          []model.Deal{},
		"reviews.json":        []model.Review{},
		"ratings.json":        []model.Rating{},
		"review_reports.json": []model.ReviewReport{},
		"counters.json":       map[string]int64{"users": 0, "orders": 0, "deals": 0, "reviews": 0, "reports": 0},
	}

	// Создаем файлы если они не существуют
	for filename, initialData := range files {
		filePath := filepath.Join(r.dataDir, filename)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			if err := r.saveToFile(filename, initialData); err != nil {
				return fmt.Errorf("не удалось создать файл %s: %w", filename, err)
			}
			log.Printf("[INFO] Создан файл данных: %s", filename)
		}
	}

	return nil
}

// saveToFile сохраняет данные в JSON файл
func (r *FileRepository) saveToFile(filename string, data interface{}) error {
	filePath := filepath.Join(r.dataDir, filename)

	// Сериализуем данные в JSON с отступами для читаемости
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("не удалось сериализовать данные: %w", err)
	}

	// Записываем в файл
	if err := ioutil.WriteFile(filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("не удалось записать файл: %w", err)
	}

	return nil
}

// loadFromFile загружает данные из JSON файла
func (r *FileRepository) loadFromFile(filename string, dest interface{}) error {
	filePath := filepath.Join(r.dataDir, filename)

	// Читаем файл
	jsonData, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("не удалось прочитать файл: %w", err)
	}

	// Десериализуем JSON
	if err := json.Unmarshal(jsonData, dest); err != nil {
		return fmt.Errorf("не удалось десериализовать данные: %w", err)
	}

	return nil
}

// generateID генерирует уникальный ID для новой записи
func (r *FileRepository) generateID(entityType string) (int64, error) {
	var counters map[string]int64
	if err := r.loadFromFile("counters.json", &counters); err != nil {
		return 0, fmt.Errorf("не удалось загрузить счетчики: %w", err)
	}

	// Увеличиваем счетчик для данного типа сущности
	counters[entityType]++
	newID := counters[entityType]

	// Сохраняем обновленные счетчики
	if err := r.saveToFile("counters.json", counters); err != nil {
		return 0, fmt.Errorf("не удалось сохранить счетчики: %w", err)
	}

	return newID, nil
}

// Close закрывает файловый репозиторий (заглушка для совместимости)
func (r *FileRepository) Close() error {
	log.Println("[INFO] Файловый репозиторий закрыт")
	return nil
}

// =====================================================
// МЕТОДЫ ДЛЯ РАБОТЫ С ПОЛЬЗОВАТЕЛЯМИ
// =====================================================

// CreateUser создает нового пользователя в JSON файле
func (r *FileRepository) CreateUser(user *model.User) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Загружаем существующих пользователей
	var users []model.User
	if err := r.loadFromFile("users.json", &users); err != nil {
		return fmt.Errorf("не удалось загрузить пользователей: %w", err)
	}

	// Проверяем что пользователь с таким Telegram ID не существует
	for _, existingUser := range users {
		if existingUser.TelegramID == user.TelegramID {
			return fmt.Errorf("пользователь с Telegram ID %d уже существует", user.TelegramID)
		}
	}

	// Генерируем новый ID
	newID, err := r.generateID("users")
	if err != nil {
		return err
	}

	// Заполняем системные поля
	user.ID = newID
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	user.IsActive = true
	user.Rating = 0.0
	user.TotalDeals = 0
	user.SuccessfulDeals = 0

	// Добавляем пользователя к списку
	users = append(users, *user)

	// Сохраняем обновленный список
	if err := r.saveToFile("users.json", users); err != nil {
		return fmt.Errorf("не удалось сохранить пользователей: %w", err)
	}

	log.Printf("[INFO] Создан пользователь в JSON: ID=%d, TelegramID=%d", user.ID, user.TelegramID)

	// Принудительно синхронизируем файловую систему
	time.Sleep(10 * time.Millisecond)

	return nil
}

// GetUserByTelegramID находит пользователя по Telegram ID
func (r *FileRepository) GetUserByTelegramID(telegramID int64) (*model.User, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Загружаем пользователей
	var users []model.User
	if err := r.loadFromFile("users.json", &users); err != nil {
		return nil, fmt.Errorf("не удалось загрузить пользователей: %w", err)
	}

	// Ищем пользователя по Telegram ID
	for _, user := range users {
		if user.TelegramID == telegramID {
			return &user, nil
		}
	}

	return nil, fmt.Errorf("пользователь с Telegram ID %d не найден", telegramID)
}

// GetUserByID находит пользователя по внутреннему ID
func (r *FileRepository) GetUserByID(userID int64) (*model.User, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Загружаем пользователей
	var users []model.User
	if err := r.loadFromFile("users.json", &users); err != nil {
		return nil, fmt.Errorf("не удалось загрузить пользователей: %w", err)
	}

	// Ищем пользователя по ID
	for _, user := range users {
		if user.ID == userID {
			return &user, nil
		}
	}

	return nil, fmt.Errorf("пользователь с ID %d не найден", userID)
}

// UpdateUserChatMembership обновляет статус членства в чате
func (r *FileRepository) UpdateUserChatMembership(telegramID int64, isMember bool) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Загружаем пользователей
	var users []model.User
	if err := r.loadFromFile("users.json", &users); err != nil {
		return fmt.Errorf("не удалось загрузить пользователей: %w", err)
	}

	// Находим и обновляем пользователя
	found := false
	for i, user := range users {
		if user.TelegramID == telegramID {
			users[i].ChatMember = isMember
			users[i].UpdatedAt = time.Now()
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("пользователь с Telegram ID %d не найден", telegramID)
	}

	// Сохраняем изменения
	if err := r.saveToFile("users.json", users); err != nil {
		return fmt.Errorf("не удалось сохранить пользователей: %w", err)
	}

	log.Printf("[INFO] Обновлен статус членства пользователя TelegramID=%d: %t", telegramID, isMember)
	return nil
}

// =====================================================
// МЕТОДЫ ДЛЯ РАБОТЫ С ЗАЯВКАМИ
// =====================================================

// CreateOrder создает новую заявку в JSON файле
func (r *FileRepository) CreateOrder(order *model.Order) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Загружаем существующие заявки
	var orders []model.Order
	if err := r.loadFromFile("orders.json", &orders); err != nil {
		return fmt.Errorf("не удалось загрузить заявки: %w", err)
	}

	// Генерируем новый ID
	newID, err := r.generateID("orders")
	if err != nil {
		return err
	}

	// Заполняем системные поля
	order.ID = newID
	order.CreatedAt = time.Now()
	order.UpdatedAt = time.Now()
	order.Status = model.OrderStatusActive
	order.IsActive = true

	// Добавляем заявку к списку
	orders = append(orders, *order)

	// Сохраняем обновленный список
	if err := r.saveToFile("orders.json", orders); err != nil {
		return fmt.Errorf("не удалось сохранить заявки: %w", err)
	}

	log.Printf("[INFO] Создана заявка в JSON: ID=%d, Type=%s, Amount=%.8f",
		order.ID, order.Type, order.Amount)
	return nil
}

// GetOrdersByFilter получает заявки по фильтрам с пагинацией
func (r *FileRepository) GetOrdersByFilter(filter *model.OrderFilter) ([]*model.Order, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Загружаем все заявки
	var orders []model.Order
	if err := r.loadFromFile("orders.json", &orders); err != nil {
		return nil, fmt.Errorf("не удалось загрузить заявки: %w", err)
	}

	// Фильтруем заявки
	var filtered []*model.Order
	for _, order := range orders {
		if r.matchesFilter(&order, filter) {
			orderCopy := order // Создаем копию для избежания проблем с указателями
			filtered = append(filtered, &orderCopy)
		}
	}

	// Сортируем результаты
	r.sortOrders(filtered, filter)

	// Применяем пагинацию
	start := filter.Offset
	if start > len(filtered) {
		start = len(filtered)
	}

	end := start + filter.Limit
	if filter.Limit == 0 || end > len(filtered) {
		end = len(filtered)
	}

	result := filtered[start:end]
	log.Printf("[INFO] Найдено заявок по фильтру: %d (показано %d)", len(filtered), len(result))

	return result, nil
}

// matchesFilter проверяет соответствует ли заявка фильтру
func (r *FileRepository) matchesFilter(order *model.Order, filter *model.OrderFilter) bool {
	// Проверяем активность (только активные заявки по умолчанию)
	if !order.IsActive {
		return false
	}

	// Фильтр по типу
	if filter.Type != nil && order.Type != *filter.Type {
		return false
	}

	// Фильтр по криптовалюте
	if filter.Cryptocurrency != nil && order.Cryptocurrency != *filter.Cryptocurrency {
		return false
	}

	// Фильтр по фиатной валюте
	if filter.FiatCurrency != nil && order.FiatCurrency != *filter.FiatCurrency {
		return false
	}

	// Фильтр по статусу
	if filter.Status != nil && order.Status != *filter.Status {
		return false
	}

	// Фильтр по пользователю
	if filter.UserID != nil && order.UserID != *filter.UserID {
		return false
	}

	// Фильтр по дате создания
	if filter.CreatedAfter != nil && order.CreatedAt.Before(*filter.CreatedAfter) {
		return false
	}

	if filter.CreatedBefore != nil && order.CreatedAt.After(*filter.CreatedBefore) {
		return false
	}

	return true
}

// sortOrders сортирует массив заявок согласно параметрам
func (r *FileRepository) sortOrders(orders []*model.Order, filter *model.OrderFilter) {
	sortBy := filter.SortBy
	if sortBy == "" {
		sortBy = "created_at" // По умолчанию сортируем по дате создания
	}

	ascending := filter.SortOrder == "asc"

	sort.Slice(orders, func(i, j int) bool {
		var less bool
		switch sortBy {
		case "price":
			less = orders[i].Price < orders[j].Price
		case "amount":
			less = orders[i].Amount < orders[j].Amount
		case "created_at":
			less = orders[i].CreatedAt.Before(orders[j].CreatedAt)
		default:
			less = orders[i].CreatedAt.Before(orders[j].CreatedAt)
		}

		if ascending {
			return less
		}
		return !less
	})
}

// UpdateOrderStatus обновляет статус заявки
func (r *FileRepository) UpdateOrderStatus(orderID int64, status model.OrderStatus) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Загружаем заявки
	var orders []model.Order
	if err := r.loadFromFile("orders.json", &orders); err != nil {
		return fmt.Errorf("не удалось загрузить заявки: %w", err)
	}

	// Находим и обновляем заявку
	found := false
	for i, order := range orders {
		if order.ID == orderID {
			orders[i].Status = status
			orders[i].UpdatedAt = time.Now()

			// Если заявка отменена или завершена, делаем ее неактивной
			if status == model.OrderStatusCancelled || status == model.OrderStatusCompleted {
				orders[i].IsActive = false
			}

			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("заявка с ID %d не найдена", orderID)
	}

	// Сохраняем изменения
	if err := r.saveToFile("orders.json", orders); err != nil {
		return fmt.Errorf("не удалось сохранить заявки: %w", err)
	}

	log.Printf("[INFO] Обновлен статус заявки ID=%d: %s", orderID, status)
	return nil
}

// HealthCheck проверяет доступность файлового хранилища
func (r *FileRepository) HealthCheck() error {
	// Проверяем доступность папки данных
	if _, err := os.Stat(r.dataDir); os.IsNotExist(err) {
		return fmt.Errorf("папка данных недоступна: %s", r.dataDir)
	}

	// Проверяем возможность записи в папку
	testFile := filepath.Join(r.dataDir, ".health_check")
	if err := ioutil.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("невозможно записать в папку данных: %w", err)
	}

	// Удаляем тестовый файл
	os.Remove(testFile)

	return nil
}

// =====================================================
// АВТОМАТИЧЕСКОЕ СОПОСТАВЛЕНИЕ ЗАЯВОК
// =====================================================

// GetMatchingOrders ищет заявки противоположного типа для автосопоставления
func (r *FileRepository) GetMatchingOrders(order *model.Order) ([]*model.Order, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	log.Printf("[INFO] Поиск подходящих заявок для Order ID=%d, Type=%s", order.ID, order.Type)

	// Загружаем все заявки
	var orders []model.Order
	if err := r.loadFromFile("orders.json", &orders); err != nil {
		return nil, fmt.Errorf("не удалось загрузить заявки для сопоставления: %w", err)
	}

	// Определяем противоположный тип заявки
	var oppositeType model.OrderType
	if order.Type == model.OrderTypeBuy {
		oppositeType = model.OrderTypeSell
	} else {
		oppositeType = model.OrderTypeBuy
	}

	var matchingOrders []*model.Order
	currentTime := time.Now()

	// Фильтруем подходящие заявки
	for _, candidateOrder := range orders {
		// Проверяем все критерии для сопоставления
		if candidateOrder.Type == oppositeType && // Противоположный тип
			candidateOrder.Cryptocurrency == order.Cryptocurrency && // Та же криптовалюта
			candidateOrder.FiatCurrency == order.FiatCurrency && // Та же фиатная валюта
			candidateOrder.Status == model.OrderStatusActive && // Активная заявка
			candidateOrder.IsActive && // Не отключена
			candidateOrder.AutoMatch && // Разрешено автосопоставление
			candidateOrder.UserID != order.UserID && // Не наша заявка
			candidateOrder.ExpiresAt.After(currentTime) { // Не истекла

			// Проверяем совместимость цен
			if r.isPriceCompatible(order, &candidateOrder) {
				orderCopy := candidateOrder
				matchingOrders = append(matchingOrders, &orderCopy)
			}
		}
	}

	// Сортируем по лучшим ценам
	r.sortMatchingOrders(matchingOrders, order.Type)

	// Ограничиваем результат (максимум 10 заявок)
	if len(matchingOrders) > 10 {
		matchingOrders = matchingOrders[:10]
	}

	log.Printf("[INFO] Найдено подходящих заявок для Order ID=%d: %d", order.ID, len(matchingOrders))
	return matchingOrders, nil
}

// isPriceCompatible проверяет совместимость цен для автосопоставления
func (r *FileRepository) isPriceCompatible(order1, order2 *model.Order) bool {
	// Для покупки: цена покупки должна быть >= цены продажи
	// Для продажи: цена продажи должна быть <= цены покупки
	if order1.Type == model.OrderTypeBuy {
		return order1.Price >= order2.Price // Готовы купить по цене >= цены продажи
	} else {
		return order1.Price <= order2.Price // Готовы продать по цене <= цены покупки
	}
}

// sortMatchingOrders сортирует подходящие заявки по оптимальным ценам
func (r *FileRepository) sortMatchingOrders(orders []*model.Order, orderType model.OrderType) {
	sort.Slice(orders, func(i, j int) bool {
		// Для заявки покупки: сначала самые дешевые продажи
		if orderType == model.OrderTypeBuy {
			if orders[i].Price != orders[j].Price {
				return orders[i].Price < orders[j].Price
			}
		} else {
			// Для заявки продажи: сначала самые дорогие покупки
			if orders[i].Price != orders[j].Price {
				return orders[i].Price > orders[j].Price
			}
		}

		// При равной цене: сначала более старые заявки (по времени создания)
		return orders[i].CreatedAt.Before(orders[j].CreatedAt)
	})
}

// MatchOrders создает сделку между двумя сопоставленными заявками
func (r *FileRepository) MatchOrders(orderID1, orderID2 int64) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	log.Printf("[INFO] Сопоставление заявок: Order1 ID=%d, Order2 ID=%d", orderID1, orderID2)

	// Загружаем все заявки
	var orders []model.Order
	if err := r.loadFromFile("orders.json", &orders); err != nil {
		return fmt.Errorf("не удалось загрузить заявки для сопоставления: %w", err)
	}

	// Находим обе заявки
	var order1, order2 *model.Order
	for i := range orders {
		if orders[i].ID == orderID1 {
			order1 = &orders[i]
		} else if orders[i].ID == orderID2 {
			order2 = &orders[i]
		}
	}

	if order1 == nil || order2 == nil {
		return fmt.Errorf("не удалось найти одну или обе заявки для сопоставления")
	}

	// Проверяем что заявки можно сопоставить
	if order1.Type == order2.Type {
		return fmt.Errorf("нельзя сопоставить заявки одинакового типа")
	}

	if order1.Cryptocurrency != order2.Cryptocurrency || order1.FiatCurrency != order2.FiatCurrency {
		return fmt.Errorf("заявки имеют разные валюты")
	}

	// Обновляем статусы заявок на "matched" (сопоставлено)
	now := time.Now()

	// Обновляем первую заявку
	for i := range orders {
		if orders[i].ID == orderID1 {
			orders[i].Status = model.OrderStatusMatched
			orders[i].MatchedUserID = &order2.UserID
			orders[i].MatchedAt = &now
			orders[i].UpdatedAt = now
			break
		}
	}

	// Обновляем вторую заявку
	for i := range orders {
		if orders[i].ID == orderID2 {
			orders[i].Status = model.OrderStatusMatched
			orders[i].MatchedUserID = &order1.UserID
			orders[i].MatchedAt = &now
			orders[i].UpdatedAt = now
			break
		}
	}

	// Сохраняем обновленные заявки
	if err := r.saveToFile("orders.json", orders); err != nil {
		return fmt.Errorf("не удалось сохранить сопоставленные заявки: %w", err)
	}

	log.Printf("[INFO] Заявки успешно сопоставлены: Order1 ID=%d, Order2 ID=%d", orderID1, orderID2)
	return nil
}

// =====================================================
// УПРАВЛЕНИЕ СДЕЛКАМИ
// =====================================================

// CreateDeal создает новую сделку между пользователями
func (r *FileRepository) CreateDeal(deal *model.Deal) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	log.Printf("[INFO] Создание сделки: Buyer ID=%d, Seller ID=%d, Amount=%.8f %s",
		deal.BuyerID, deal.SellerID, deal.Amount, deal.Cryptocurrency)

	// Загружаем существующие сделки
	var deals []model.Deal
	if err := r.loadFromFile("deals.json", &deals); err != nil {
		return fmt.Errorf("не удалось загрузить сделки: %w", err)
	}

	// Генерируем новый ID для сделки
	dealID, err := r.generateID("deals")
	if err != nil {
		return fmt.Errorf("не удалось сгенерировать ID для сделки: %w", err)
	}

	// Устанавливаем ID и временные метки
	deal.ID = dealID
	deal.Status = "pending" // Статус "ожидает подтверждения"
	deal.CreatedAt = time.Now()

	// Добавляем сделку в список
	deals = append(deals, *deal)

	// Сохраняем обновленный список сделок
	if err := r.saveToFile("deals.json", deals); err != nil {
		return fmt.Errorf("не удалось сохранить сделки: %w", err)
	}

	log.Printf("[INFO] Сделка успешно создана: ID=%d", deal.ID)
	return nil
}

// GetDealsByUserID получает все сделки пользователя (как покупателя и продавца)
func (r *FileRepository) GetDealsByUserID(userID int64) ([]*model.Deal, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	log.Printf("[INFO] Получение сделок для пользователя ID=%d", userID)

	// Загружаем все сделки
	var deals []model.Deal
	if err := r.loadFromFile("deals.json", &deals); err != nil {
		return nil, fmt.Errorf("не удалось загрузить сделки: %w", err)
	}

	// Фильтруем сделки пользователя
	var userDeals []*model.Deal
	for _, deal := range deals {
		// Пользователь участвует в сделке если он покупатель или продавец
		if deal.BuyerID == userID || deal.SellerID == userID {
			dealCopy := deal
			userDeals = append(userDeals, &dealCopy)
		}
	}

	// Сортируем по дате создания (новые сначала)
	sort.Slice(userDeals, func(i, j int) bool {
		return userDeals[i].CreatedAt.After(userDeals[j].CreatedAt)
	})

	log.Printf("[INFO] Найдено сделок для пользователя ID=%d: %d", userID, len(userDeals))
	return userDeals, nil
}

// GetDealByID получает сделку по её ID
func (r *FileRepository) GetDealByID(dealID int64) (*model.Deal, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	log.Printf("[INFO] Получение сделки по ID=%d", dealID)

	// Загружаем все сделки
	var deals []model.Deal
	if err := r.loadFromFile("deals.json", &deals); err != nil {
		return nil, fmt.Errorf("не удалось загрузить сделки: %w", err)
	}

	// Ищем сделку с нужным ID
	for _, deal := range deals {
		if deal.ID == dealID {
			log.Printf("[INFO] Сделка найдена: ID=%d, Status=%s", deal.ID, deal.Status)
			return &deal, nil
		}
	}

	log.Printf("[WARN] Сделка с ID=%d не найдена", dealID)
	return nil, fmt.Errorf("сделка с ID=%d не найдена", dealID)
}

// ConfirmDeal - заглушка для совместимости
func (r *FileRepository) ConfirmDeal(dealID int64, userID int64, isPaymentProof bool, paymentProof string) error {
	log.Printf("[WARN] ConfirmDeal не реализован для файлового хранилища")
	return nil
}

// =====================================================
// УПРАВЛЕНИЕ ОТЗЫВАМИ И РЕЙТИНГАМИ
// =====================================================

// CreateReview создает новый отзыв и обновляет рейтинг пользователя
func (r *FileRepository) CreateReview(review *model.Review) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	log.Printf("[INFO] Создание отзыва от пользователя ID=%d к пользователю ID=%d",
		review.FromUserID, review.ToUserID)

	// Загружаем существующие отзывы
	var reviews []model.Review
	if err := r.loadFromFile("reviews.json", &reviews); err != nil {
		return fmt.Errorf("не удалось загрузить отзывы: %w", err)
	}

	// Генерируем новый ID для отзыва
	reviewID, err := r.generateID("reviews")
	if err != nil {
		return fmt.Errorf("не удалось сгенерировать ID для отзыва: %w", err)
	}

	// Устанавливаем поля отзыва
	review.ID = reviewID
	review.CreatedAt = time.Now()
	review.UpdatedAt = time.Now()
	review.IsVisible = true
	review.ReportedCount = 0

	// Определяем тип отзыва по рейтингу
	if review.Rating >= 4 {
		review.Type = model.ReviewTypePositive
	} else if review.Rating == 3 {
		review.Type = model.ReviewTypeNeutral
	} else {
		review.Type = model.ReviewTypeNegative
	}

	// Добавляем отзыв в список
	reviews = append(reviews, *review)

	// Сохраняем отзывы
	if err := r.saveToFile("reviews.json", reviews); err != nil {
		return fmt.Errorf("не удалось сохранить отзывы: %w", err)
	}

	// Обновляем рейтинг пользователя
	if err := r.updateUserRating(review.ToUserID); err != nil {
		log.Printf("[WARN] Не удалось обновить рейтинг пользователя ID=%d: %v", review.ToUserID, err)
	}

	log.Printf("[INFO] Отзыв успешно создан: ID=%d, рейтинг=%d", review.ID, review.Rating)
	return nil
}

// GetReviewsByUserID получает отзывы о пользователе с пагинацией
func (r *FileRepository) GetReviewsByUserID(userID int64, limit, offset int) ([]*model.Review, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	log.Printf("[INFO] Получение отзывов для пользователя ID=%d (limit=%d, offset=%d)", userID, limit, offset)

	// Загружаем все отзывы
	var allReviews []model.Review
	if err := r.loadFromFile("reviews.json", &allReviews); err != nil {
		return nil, fmt.Errorf("не удалось загрузить отзывы: %w", err)
	}

	// Фильтруем отзывы для данного пользователя (видимые отзывы)
	var userReviews []*model.Review
	for _, review := range allReviews {
		if review.ToUserID == userID && review.IsVisible {
			reviewCopy := review
			userReviews = append(userReviews, &reviewCopy)
		}
	}

	// Сортируем по дате создания (новые сначала)
	sort.Slice(userReviews, func(i, j int) bool {
		return userReviews[i].CreatedAt.After(userReviews[j].CreatedAt)
	})

	// Применяем пагинацию
	totalReviews := len(userReviews)
	if offset >= totalReviews {
		return []*model.Review{}, nil
	}

	end := offset + limit
	if end > totalReviews {
		end = totalReviews
	}

	paginatedReviews := userReviews[offset:end]

	log.Printf("[INFO] Найдено отзывов для пользователя ID=%d: %d (показано %d)", userID, totalReviews, len(paginatedReviews))
	return paginatedReviews, nil
}

// GetUserRating получает агрегированный рейтинг пользователя
func (r *FileRepository) GetUserRating(userID int64) (*model.Rating, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	log.Printf("[INFO] Получение рейтинга пользователя ID=%d", userID)

	// Загружаем рейтинги
	var ratings []model.Rating
	if err := r.loadFromFile("ratings.json", &ratings); err != nil {
		return nil, fmt.Errorf("не удалось загрузить рейтинги: %w", err)
	}

	// Ищем рейтинг пользователя
	for _, rating := range ratings {
		if rating.UserID == userID {
			log.Printf("[INFO] Найден рейтинг пользователя ID=%d: %.2f (%d отзывов)",
				userID, rating.AverageRating, rating.TotalReviews)
			return &rating, nil
		}
	}

	// Если рейтинг не найден, создаем пустой
	emptyRating := &model.Rating{
		UserID:          userID,
		AverageRating:   0.0,
		TotalReviews:    0,
		PositiveReviews: 0,
		NeutralReviews:  0,
		NegativeReviews: 0,
		FiveStars:       0,
		FourStars:       0,
		ThreeStars:      0,
		TwoStars:        0,
		OneStar:         0,
		UpdatedAt:       time.Now(),
	}

	log.Printf("[INFO] Рейтинг пользователя ID=%d не найден, возвращен пустой", userID)
	return emptyRating, nil
}

// CheckCanReview проверяет можно ли пользователю оставить отзыв о другом пользователе по сделке
func (r *FileRepository) CheckCanReview(dealID, fromUserID, toUserID int64) (bool, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	log.Printf("[INFO] Проверка возможности оставить отзыв: Deal ID=%d, From=%d, To=%d",
		dealID, fromUserID, toUserID)

	// 1. Проверяем что сделка существует и завершена
	deal, err := r.getDealByIDInternal(dealID)
	if err != nil {
		log.Printf("[WARN] Сделка не найдена: %v", err)
		return false, fmt.Errorf("сделка не найдена")
	}

	// 2. Проверяем что пользователь участвовал в сделке
	if deal.BuyerID != fromUserID && deal.SellerID != fromUserID {
		log.Printf("[WARN] Пользователь ID=%d не участвовал в сделке ID=%d", fromUserID, dealID)
		return false, fmt.Errorf("вы не участвовали в данной сделке")
	}

	// 3. Проверяем что отзыв оставляется корректному участнику
	if deal.BuyerID == fromUserID && deal.SellerID != toUserID {
		return false, fmt.Errorf("неверный получатель отзыва")
	}
	if deal.SellerID == fromUserID && deal.BuyerID != toUserID {
		return false, fmt.Errorf("неверный получатель отзыва")
	}

	// 4. Проверяем что сделка завершена
	if deal.Status != "completed" {
		log.Printf("[WARN] Сделка ID=%d не завершена (статус: %s)", dealID, deal.Status)
		return false, fmt.Errorf("отзыв можно оставить только по завершенным сделкам")
	}

	// 5. Проверяем что отзыв еще не был оставлен
	var reviews []model.Review
	if err := r.loadFromFile("reviews.json", &reviews); err != nil {
		return false, fmt.Errorf("не удалось загрузить отзывы: %w", err)
	}

	for _, review := range reviews {
		if review.DealID == dealID && review.FromUserID == fromUserID && review.ToUserID == toUserID {
			log.Printf("[WARN] Отзыв уже оставлен по сделке ID=%d от пользователя ID=%d", dealID, fromUserID)
			return false, fmt.Errorf("отзыв по данной сделке уже оставлен")
		}
	}

	log.Printf("[INFO] Отзыв можно оставить: Deal ID=%d, From=%d, To=%d", dealID, fromUserID, toUserID)
	return true, nil
}

// ReportReview создает жалобу на отзыв
func (r *FileRepository) ReportReview(report *model.ReviewReport) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	log.Printf("[INFO] Создание жалобы на отзыв ID=%d от пользователя ID=%d", report.ReviewID, report.UserID)

	// Загружаем жалобы
	var reports []model.ReviewReport
	if err := r.loadFromFile("review_reports.json", &reports); err != nil {
		return fmt.Errorf("не удалось загрузить жалобы: %w", err)
	}

	// Генерируем ID для жалобы
	reportID, err := r.generateID("reports")
	if err != nil {
		return fmt.Errorf("не удалось сгенерировать ID для жалобы: %w", err)
	}

	// Устанавливаем поля жалобы
	report.ID = reportID
	report.Status = "pending"
	report.CreatedAt = time.Now()

	// Добавляем жалобу
	reports = append(reports, *report)

	// Сохраняем жалобы
	if err := r.saveToFile("review_reports.json", reports); err != nil {
		return fmt.Errorf("не удалось сохранить жалобы: %w", err)
	}

	log.Printf("[INFO] Жалоба успешно создана: ID=%d", report.ID)
	return nil
}

// GetUserReviewStats получает детальную статистику отзывов пользователя
func (r *FileRepository) GetUserReviewStats(userID int64) (*model.ReviewStats, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	log.Printf("[INFO] Получение статистики отзывов пользователя ID=%d", userID)

	// Получаем рейтинг пользователя
	rating, err := r.GetUserRating(userID)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить рейтинг: %w", err)
	}

	// Получаем последние отзывы (максимум 5)
	recentReviews, err := r.GetReviewsByUserID(userID, 5, 0)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить отзывы: %w", err)
	}

	// Конвертируем указатели в значения для ReviewStats
	var recentReviewsValues []model.Review
	for _, review := range recentReviews {
		recentReviewsValues = append(recentReviewsValues, *review)
	}

	// Создаем распределение по звездам
	ratingDistribution := map[int]int{
		1: rating.OneStar,
		2: rating.TwoStars,
		3: rating.ThreeStars,
		4: rating.FourStars,
		5: rating.FiveStars,
	}

	// Вычисляем процент положительных отзывов
	positivePercent := float32(0.0)
	if rating.TotalReviews > 0 {
		positivePercent = float32(rating.PositiveReviews) / float32(rating.TotalReviews) * 100
	}

	stats := &model.ReviewStats{
		UserID:             userID,
		AverageRating:      rating.AverageRating,
		TotalReviews:       rating.TotalReviews,
		PositivePercent:    positivePercent,
		RecentReviews:      recentReviewsValues,
		RatingDistribution: ratingDistribution,
	}

	log.Printf("[INFO] Статистика отзывов пользователя ID=%d: %.2f рейтинг, %d отзывов",
		userID, stats.AverageRating, stats.TotalReviews)
	return stats, nil
}

// getDealByIDInternal внутренний метод получения сделки (без блокировки мьютекса)
func (r *FileRepository) getDealByIDInternal(dealID int64) (*model.Deal, error) {
	var deals []model.Deal
	if err := r.loadFromFile("deals.json", &deals); err != nil {
		return nil, fmt.Errorf("не удалось загрузить сделки: %w", err)
	}

	for _, deal := range deals {
		if deal.ID == dealID {
			return &deal, nil
		}
	}

	return nil, fmt.Errorf("сделка с ID=%d не найдена", dealID)
}

// updateUserRating пересчитывает рейтинг пользователя на основе всех его отзывов
func (r *FileRepository) updateUserRating(userID int64) error {
	log.Printf("[INFO] Обновление рейтинга пользователя ID=%d", userID)

	// Загружаем все отзывы пользователя
	var allReviews []model.Review
	if err := r.loadFromFile("reviews.json", &allReviews); err != nil {
		return fmt.Errorf("не удалось загрузить отзывы: %w", err)
	}

	// Фильтруем отзывы для данного пользователя
	var userReviews []model.Review
	for _, review := range allReviews {
		if review.ToUserID == userID && review.IsVisible {
			userReviews = append(userReviews, review)
		}
	}

	// Если отзывов нет, удаляем рейтинг из файла
	if len(userReviews) == 0 {
		return r.removeUserRating(userID)
	}

	// Вычисляем статистику
	var totalRating int
	rating := model.Rating{
		UserID:    userID,
		UpdatedAt: time.Now(),
	}

	for _, review := range userReviews {
		totalRating += review.Rating

		// Подсчитываем по типам
		switch review.Type {
		case model.ReviewTypePositive:
			rating.PositiveReviews++
		case model.ReviewTypeNeutral:
			rating.NeutralReviews++
		case model.ReviewTypeNegative:
			rating.NegativeReviews++
		}

		// Подсчитываем по звездам
		switch review.Rating {
		case 1:
			rating.OneStar++
		case 2:
			rating.TwoStars++
		case 3:
			rating.ThreeStars++
		case 4:
			rating.FourStars++
		case 5:
			rating.FiveStars++
		}
	}

	rating.TotalReviews = len(userReviews)
	rating.AverageRating = float32(totalRating) / float32(len(userReviews))

	// Сохраняем рейтинг
	return r.saveUserRating(&rating)
}

// removeUserRating удаляет рейтинг пользователя
func (r *FileRepository) removeUserRating(userID int64) error {
	var ratings []model.Rating
	if err := r.loadFromFile("ratings.json", &ratings); err != nil {
		return fmt.Errorf("не удалось загрузить рейтинги: %w", err)
	}

	// Фильтруем рейтинги, удаляя целевого пользователя
	var filteredRatings []model.Rating
	for _, rating := range ratings {
		if rating.UserID != userID {
			filteredRatings = append(filteredRatings, rating)
		}
	}

	return r.saveToFile("ratings.json", filteredRatings)
}

// saveUserRating сохраняет или обновляет рейтинг пользователя
func (r *FileRepository) saveUserRating(newRating *model.Rating) error {
	var ratings []model.Rating
	if err := r.loadFromFile("ratings.json", &ratings); err != nil {
		return fmt.Errorf("не удалось загрузить рейтинги: %w", err)
	}

	// Ищем существующий рейтинг
	found := false
	for i, rating := range ratings {
		if rating.UserID == newRating.UserID {
			ratings[i] = *newRating
			found = true
			break
		}
	}

	// Если не найден, добавляем новый
	if !found {
		ratings = append(ratings, *newRating)
	}

	return r.saveToFile("ratings.json", ratings)
}
