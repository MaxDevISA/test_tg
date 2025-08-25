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
// ЗАГЛУШКИ ДЛЯ ОСТАЛЬНЫХ МЕТОДОВ (ПОКА НЕ РЕАЛИЗОВАНЫ)
// =====================================================

// GetMatchingOrders - заглушка для совместимости
func (r *FileRepository) GetMatchingOrders(order *model.Order) ([]*model.Order, error) {
	log.Printf("[WARN] GetMatchingOrders не реализован для файлового хранилища")
	return []*model.Order{}, nil
}

// MatchOrders - заглушка для совместимости
func (r *FileRepository) MatchOrders(orderID1, orderID2 int64) error {
	log.Printf("[WARN] MatchOrders не реализован для файлового хранилища")
	return nil
}

// CreateDeal - заглушка для совместимости
func (r *FileRepository) CreateDeal(deal *model.Deal) error {
	log.Printf("[WARN] CreateDeal не реализован для файлового хранилища")
	return nil
}

// GetDealsByUserID - заглушка для совместимости
func (r *FileRepository) GetDealsByUserID(userID int64) ([]*model.Deal, error) {
	log.Printf("[WARN] GetDealsByUserID не реализован для файлового хранилища")
	return []*model.Deal{}, nil
}

// GetDealByID - заглушка для совместимости
func (r *FileRepository) GetDealByID(dealID int64) (*model.Deal, error) {
	log.Printf("[WARN] GetDealByID не реализован для файлового хранилища")
	return nil, fmt.Errorf("метод не реализован")
}

// ConfirmDeal - заглушка для совместимости
func (r *FileRepository) ConfirmDeal(dealID int64, userID int64, isPaymentProof bool, paymentProof string) error {
	log.Printf("[WARN] ConfirmDeal не реализован для файлового хранилища")
	return nil
}

// CreateReview - заглушка для совместимости
func (r *FileRepository) CreateReview(review *model.Review) error {
	log.Printf("[WARN] CreateReview не реализован для файлового хранилища")
	return nil
}

// GetReviewsByUserID - заглушка для совместимости
func (r *FileRepository) GetReviewsByUserID(userID int64, limit, offset int) ([]*model.Review, error) {
	log.Printf("[WARN] GetReviewsByUserID не реализован для файлового хранилища")
	return []*model.Review{}, nil
}

// GetUserRating - заглушка для совместимости
func (r *FileRepository) GetUserRating(userID int64) (*model.Rating, error) {
	log.Printf("[WARN] GetUserRating не реализован для файлового хранилища")
	return &model.Rating{UserID: userID, AverageRating: 0.0}, nil
}

// CheckCanReview - заглушка для совместимости
func (r *FileRepository) CheckCanReview(dealID, fromUserID, toUserID int64) (bool, error) {
	log.Printf("[WARN] CheckCanReview не реализован для файлового хранилища")
	return false, fmt.Errorf("метод не реализован")
}

// ReportReview - заглушка для совместимости
func (r *FileRepository) ReportReview(report *model.ReviewReport) error {
	log.Printf("[WARN] ReportReview не реализован для файлового хранилища")
	return nil
}

// GetUserReviewStats - заглушка для совместимости
func (r *FileRepository) GetUserReviewStats(userID int64) (*model.ReviewStats, error) {
	log.Printf("[WARN] GetUserReviewStats не реализован для файлового хранилища")
	return &model.ReviewStats{UserID: userID, AverageRating: 0.0}, nil
}
