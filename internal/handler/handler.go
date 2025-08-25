package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"p2pTG-crypto-exchange/internal/model"
	"p2pTG-crypto-exchange/internal/service"

	"github.com/gorilla/mux"
)

// Handler представляет слой обработки HTTP запросов
// Принимает HTTP запросы, вызывает соответствующие сервисы и возвращает ответы
type Handler struct {
	service *service.Service // Сервис бизнес-логики
}

// NewHandler создает новый экземпляр обработчика
func NewHandler(service *service.Service) *Handler {
	log.Println("[INFO] Инициализация HTTP обработчиков")
	return &Handler{
		service: service,
	}
}

// RegisterRoutes регистрирует все HTTP маршруты приложения
// Определяет какой обработчик вызывать для каждого URL
func (h *Handler) RegisterRoutes(router *mux.Router) {
	log.Println("[INFO] Регистрация HTTP маршрутов")

	// Маршруты для API
	api := router.PathPrefix("/api/v1").Subrouter()

	// Аутентификация и авторизация
	api.HandleFunc("/auth/login", h.handleLogin).Methods("POST")
	api.HandleFunc("/auth/me", h.handleGetCurrentUser).Methods("GET")

	// Управление заявками (ордерами)
	api.HandleFunc("/orders", h.handleGetOrders).Methods("GET")           // Получить список заявок
	api.HandleFunc("/orders", h.handleCreateOrder).Methods("POST")        // Создать новую заявку
	api.HandleFunc("/orders/{id}", h.handleGetOrder).Methods("GET")       // Получить заявку по ID
	api.HandleFunc("/orders/{id}", h.handleCancelOrder).Methods("DELETE") // Отменить заявку
	api.HandleFunc("/orders/my", h.handleGetMyOrders).Methods("GET")      // Получить мои заявки

	// Управление сделками
	api.HandleFunc("/deals", h.handleGetDeals).Methods("GET")                  // Получить список сделок пользователя
	api.HandleFunc("/deals/{id}", h.handleGetDeal).Methods("GET")              // Получить сделку по ID
	api.HandleFunc("/deals/{id}/confirm", h.handleConfirmDeal).Methods("POST") // Подтвердить сделку

	// Система отзывов и рейтингов
	api.HandleFunc("/reviews", h.handleGetReviews).Methods("GET")                // Получить отзывы пользователя
	api.HandleFunc("/reviews", h.handleCreateReview).Methods("POST")             // Оставить отзыв
	api.HandleFunc("/users/{id}/profile", h.handleGetUserProfile).Methods("GET") // Получить профиль пользователя

	// Информационные эндпоинты
	api.HandleFunc("/health", h.handleHealthCheck).Methods("GET") // Проверка состояния сервиса

	// Статические файлы для веб-интерфейса
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./web/static/"))))

	// Главная страница веб-приложения
	router.HandleFunc("/", h.handleIndex).Methods("GET")

	log.Println("[INFO] Все HTTP маршруты зарегистрированы")
}

// =====================================================
// ОБРАБОТЧИКИ АВТОРИЗАЦИИ
// =====================================================

// handleLogin обрабатывает авторизацию пользователя через Telegram WebApp
func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	log.Println("[INFO] Обработка запроса авторизации")

	// Читаем данные авторизации из тела запроса
	var authData model.TelegramAuthData
	if err := json.NewDecoder(r.Body).Decode(&authData); err != nil {
		log.Printf("[WARN] Неверный формат данных авторизации: %v", err)
		h.sendErrorResponse(w, "Неверный формат данных", http.StatusBadRequest)
		return
	}

	// Вызываем сервис авторизации
	user, err := h.service.AuthenticateUser(&authData)
	if err != nil {
		log.Printf("[WARN] Ошибка авторизации: %v", err)
		h.sendErrorResponse(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Возвращаем успешный ответ с данными пользователя
	log.Printf("[INFO] Успешная авторизация пользователя: ID=%d", user.ID)
	h.sendJSONResponse(w, map[string]interface{}{
		"success": true,
		"user":    user,
		"message": "Авторизация успешна",
	})
}

// handleGetCurrentUser возвращает информацию о текущем пользователе
func (h *Handler) handleGetCurrentUser(w http.ResponseWriter, r *http.Request) {
	// TODO: Реализовать получение пользователя из JWT токена
	// В реальной реализации здесь должна быть проверка JWT токена
	h.sendJSONResponse(w, map[string]interface{}{
		"message": "Эндпоинт в разработке - требуется JWT middleware",
	})
}

// =====================================================
// ОБРАБОТЧИКИ ЗАЯВОК
// =====================================================

// handleGetOrders обрабатывает получение списка заявок с фильтрацией
func (h *Handler) handleGetOrders(w http.ResponseWriter, r *http.Request) {
	log.Println("[INFO] Обработка запроса получения заявок")

	// Парсим параметры фильтрации из query string
	filter := &model.OrderFilter{}

	// Тип заявки (buy/sell)
	if orderType := r.URL.Query().Get("type"); orderType != "" {
		if orderType == "buy" || orderType == "sell" {
			t := model.OrderType(orderType)
			filter.Type = &t
		}
	}

	// Криптовалюта
	if crypto := r.URL.Query().Get("cryptocurrency"); crypto != "" {
		filter.Cryptocurrency = &crypto
	}

	// Фиатная валюта
	if fiat := r.URL.Query().Get("fiat_currency"); fiat != "" {
		filter.FiatCurrency = &fiat
	}

	// Статус заявки
	if status := r.URL.Query().Get("status"); status != "" {
		s := model.OrderStatus(status)
		filter.Status = &s
	}

	// Пагинация
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filter.Limit = limit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}

	// Сортировка
	filter.SortBy = r.URL.Query().Get("sort_by")
	filter.SortOrder = r.URL.Query().Get("sort_order")

	// Получаем заявки через сервис
	orders, err := h.service.GetOrders(filter)
	if err != nil {
		log.Printf("[ERROR] Ошибка при получении заявок: %v", err)
		h.sendErrorResponse(w, "Не удалось получить заявки", http.StatusInternalServerError)
		return
	}

	log.Printf("[INFO] Возвращено заявок: %d", len(orders))
	h.sendJSONResponse(w, map[string]interface{}{
		"success": true,
		"orders":  orders,
		"count":   len(orders),
	})
}

// handleCreateOrder обрабатывает создание новой заявки
func (h *Handler) handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	log.Println("[INFO] Обработка запроса создания заявки")

	// Получаем Telegram ID пользователя из заголовка
	telegramIDStr := r.Header.Get("X-Telegram-User-ID")
	if telegramIDStr == "" {
		log.Printf("[WARN] Не передан Telegram ID пользователя")
		h.sendErrorResponse(w, "Требуется авторизация", http.StatusUnauthorized)
		return
	}

	telegramID, err := strconv.ParseInt(telegramIDStr, 10, 64)
	if err != nil {
		log.Printf("[WARN] Неверный формат Telegram ID: %v", err)
		h.sendErrorResponse(w, "Неверный ID пользователя", http.StatusBadRequest)
		return
	}

	// Читаем данные заявки из тела запроса
	var orderData model.Order
	if err := json.NewDecoder(r.Body).Decode(&orderData); err != nil {
		log.Printf("[WARN] Неверный формат данных заявки: %v", err)
		h.sendErrorResponse(w, "Неверный формат данных заявки", http.StatusBadRequest)
		return
	}

	// Создаем заявку через сервис (передаем Telegram ID)
	order, err := h.service.CreateOrder(telegramID, &orderData)
	if err != nil {
		log.Printf("[WARN] Ошибка создания заявки: %v", err)
		h.sendErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("[INFO] Создана новая заявка: ID=%d", order.ID)
	h.sendJSONResponse(w, map[string]interface{}{
		"success": true,
		"order":   order,
		"message": "Заявка успешно создана",
	})
}

// handleGetOrder обрабатывает получение заявки по ID
func (h *Handler) handleGetOrder(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		h.sendErrorResponse(w, "Неверный ID заявки", http.StatusBadRequest)
		return
	}

	log.Printf("[INFO] Запрос заявки по ID: %d", orderID)

	// Получаем заявку через сервис
	order, err := h.service.GetOrder(orderID)
	if err != nil {
		log.Printf("[WARN] Заявка ID=%d не найдена: %v", orderID, err)
		h.sendErrorResponse(w, "Заявка не найдена", http.StatusNotFound)
		return
	}

	log.Printf("[INFO] Заявка ID=%d успешно получена", orderID)
	h.sendJSONResponse(w, map[string]interface{}{
		"success": true,
		"order":   order,
	})
}

// handleCancelOrder обрабатывает отмену заявки
func (h *Handler) handleCancelOrder(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		h.sendErrorResponse(w, "Неверный ID заявки", http.StatusBadRequest)
		return
	}

	// TODO: Получить ID пользователя из JWT токена
	userID := int64(1) // Заглушка

	// Отменяем заявку через сервис
	if err := h.service.CancelOrder(userID, orderID); err != nil {
		log.Printf("[WARN] Ошибка отмены заявки ID=%d: %v", orderID, err)
		h.sendErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("[INFO] Заявка ID=%d отменена пользователем ID=%d", orderID, userID)
	h.sendJSONResponse(w, map[string]interface{}{
		"success": true,
		"message": "Заявка успешно отменена",
	})
}

// handleGetMyOrders обрабатывает получение заявок пользователя (страница "Мои")
func (h *Handler) handleGetMyOrders(w http.ResponseWriter, r *http.Request) {
	log.Println("[INFO] Обработка запроса получения заявок пользователя")

	// Получаем Telegram ID пользователя из заголовка
	telegramIDStr := r.Header.Get("X-Telegram-User-ID")
	if telegramIDStr == "" {
		log.Printf("[WARN] Не передан Telegram ID пользователя")
		h.sendErrorResponse(w, "Требуется авторизация", http.StatusUnauthorized)
		return
	}

	telegramID, err := strconv.ParseInt(telegramIDStr, 10, 64)
	if err != nil {
		log.Printf("[WARN] Неверный формат Telegram ID: %v", err)
		h.sendErrorResponse(w, "Неверный ID пользователя", http.StatusBadRequest)
		return
	}

	// Получаем пользователя по Telegram ID
	user, err := h.service.GetUserByTelegramID(telegramID)
	if err != nil {
		log.Printf("[ERROR] Пользователь не найден: %v", err)
		h.sendErrorResponse(w, "Пользователь не найден", http.StatusNotFound)
		return
	}

	// Создаем фильтр для заявок пользователя
	filter := &model.OrderFilter{
		UserID: &user.ID, // Фильтруем по ID пользователя
		Limit:  50,       // Ограничиваем 50 заявками
		Offset: 0,
	}

	// Получаем заявки пользователя
	orders, err := h.service.GetOrders(filter)
	if err != nil {
		log.Printf("[ERROR] Ошибка при получении заявок пользователя: %v", err)
		h.sendErrorResponse(w, "Не удалось получить заявки", http.StatusInternalServerError)
		return
	}

	log.Printf("[INFO] Возвращено заявок пользователю ID=%d: %d", user.ID, len(orders))
	h.sendJSONResponse(w, map[string]interface{}{
		"success": true,
		"orders":  orders,
		"count":   len(orders),
	})
}

// =====================================================
// ОБРАБОТЧИКИ СДЕЛОК
// =====================================================

// handleGetDeals обрабатывает получение списка сделок пользователя
func (h *Handler) handleGetDeals(w http.ResponseWriter, r *http.Request) {
	log.Println("[INFO] Обработка запроса получения сделок пользователя")

	// Получаем Telegram ID пользователя из заголовка
	telegramIDStr := r.Header.Get("X-Telegram-User-ID")
	if telegramIDStr == "" {
		log.Printf("[WARN] Не передан Telegram ID пользователя")
		h.sendErrorResponse(w, "Требуется авторизация", http.StatusUnauthorized)
		return
	}

	telegramID, err := strconv.ParseInt(telegramIDStr, 10, 64)
	if err != nil {
		log.Printf("[WARN] Неверный формат Telegram ID: %v", err)
		h.sendErrorResponse(w, "Неверный ID пользователя", http.StatusBadRequest)
		return
	}

	// Получаем пользователя по Telegram ID для получения внутреннего ID
	user, err := h.service.GetUserByTelegramID(telegramID)
	if err != nil {
		log.Printf("[ERROR] Пользователь не найден: %v", err)
		h.sendErrorResponse(w, "Пользователь не найден", http.StatusNotFound)
		return
	}

	// Получаем сделки пользователя через сервис
	deals, err := h.service.GetUserDeals(user.ID)
	if err != nil {
		log.Printf("[ERROR] Ошибка при получении сделок: %v", err)
		h.sendErrorResponse(w, "Не удалось получить сделки", http.StatusInternalServerError)
		return
	}

	log.Printf("[INFO] Возвращено сделок пользователю ID=%d: %d", user.ID, len(deals))
	h.sendJSONResponse(w, map[string]interface{}{
		"success": true,
		"deals":   deals,
		"count":   len(deals),
	})
}

// handleGetDeal обрабатывает получение сделки по ID
func (h *Handler) handleGetDeal(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	dealID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		h.sendErrorResponse(w, "Неверный ID сделки", http.StatusBadRequest)
		return
	}

	// TODO: Получить ID пользователя из JWT токена
	userID := int64(1) // Заглушка

	log.Printf("[INFO] Запрос сделки ID=%d пользователем ID=%d", dealID, userID)

	// Получаем сделку через сервис с проверкой прав доступа
	deal, err := h.service.GetDeal(dealID, userID)
	if err != nil {
		log.Printf("[WARN] Ошибка получения сделки ID=%d: %v", dealID, err)
		// Возвращаем общую ошибку чтобы не раскрывать детали
		h.sendErrorResponse(w, "Сделка не найдена или доступ запрещен", http.StatusNotFound)
		return
	}

	log.Printf("[INFO] Сделка ID=%d успешно получена", dealID)
	h.sendJSONResponse(w, map[string]interface{}{
		"success": true,
		"deal":    deal,
	})
}

// handleConfirmDeal обрабатывает подтверждение сделки пользователем
func (h *Handler) handleConfirmDeal(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	dealID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		h.sendErrorResponse(w, "Неверный ID сделки", http.StatusBadRequest)
		return
	}

	// TODO: Получить ID пользователя из JWT токена
	userID := int64(1) // Заглушка

	log.Printf("[INFO] Подтверждение сделки ID=%d пользователем ID=%d", dealID, userID)

	// Читаем данные запроса (может содержать доказательство оплаты)
	var requestData struct {
		PaymentProof string `json:"payment_proof"` // Доказательство оплаты (ссылка на скриншот)
		Notes        string `json:"notes"`         // Дополнительные заметки
	}

	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		log.Printf("[WARN] Неверный формат данных подтверждения: %v", err)
		// Игнорируем ошибку декодирования - подтверждение без доказательств
	}

	// Подтверждаем сделку через сервис
	if err := h.service.ConfirmDeal(dealID, userID, requestData.PaymentProof); err != nil {
		log.Printf("[WARN] Ошибка подтверждения сделки ID=%d: %v", dealID, err)
		h.sendErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("[INFO] Сделка ID=%d подтверждена пользователем ID=%d", dealID, userID)
	h.sendJSONResponse(w, map[string]interface{}{
		"success": true,
		"message": "Сделка успешно подтверждена",
	})
}

// =====================================================
// ОБРАБОТЧИКИ ОТЗЫВОВ
// =====================================================

// handleGetReviews обрабатывает получение отзывов о пользователе
func (h *Handler) handleGetReviews(w http.ResponseWriter, r *http.Request) {
	log.Println("[INFO] Обработка запроса получения отзывов")

	// Получаем ID пользователя из параметров запроса
	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		h.sendErrorResponse(w, "Необходимо указать ID пользователя", http.StatusBadRequest)
		return
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		h.sendErrorResponse(w, "Неверный ID пользователя", http.StatusBadRequest)
		return
	}

	// Параметры пагинации
	limit := 20
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Получаем отзывы через сервис
	reviews, err := h.service.GetUserReviews(userID, limit, offset)
	if err != nil {
		log.Printf("[ERROR] Ошибка при получении отзывов: %v", err)
		h.sendErrorResponse(w, "Не удалось получить отзывы", http.StatusInternalServerError)
		return
	}

	log.Printf("[INFO] Возвращено отзывов: %d", len(reviews))
	h.sendJSONResponse(w, map[string]interface{}{
		"success": true,
		"reviews": reviews,
		"count":   len(reviews),
		"pagination": map[string]interface{}{
			"limit":  limit,
			"offset": offset,
		},
	})
}

// handleCreateReview обрабатывает создание нового отзыва
func (h *Handler) handleCreateReview(w http.ResponseWriter, r *http.Request) {
	log.Println("[INFO] Обработка создания отзыва")

	// Получаем Telegram ID пользователя из заголовка
	telegramIDStr := r.Header.Get("X-Telegram-User-ID")
	if telegramIDStr == "" {
		log.Printf("[WARN] Не передан Telegram ID пользователя")
		h.sendErrorResponse(w, "Требуется авторизация", http.StatusUnauthorized)
		return
	}

	telegramID, err := strconv.ParseInt(telegramIDStr, 10, 64)
	if err != nil {
		log.Printf("[WARN] Неверный формат Telegram ID: %v", err)
		h.sendErrorResponse(w, "Неверный ID пользователя", http.StatusBadRequest)
		return
	}

	// Получаем пользователя по Telegram ID для получения внутреннего ID
	user, err := h.service.GetUserByTelegramID(telegramID)
	if err != nil {
		log.Printf("[ERROR] Пользователь не найден: %v", err)
		h.sendErrorResponse(w, "Пользователь не найден", http.StatusNotFound)
		return
	}

	// Читаем данные отзыва из тела запроса
	var reviewData model.CreateReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&reviewData); err != nil {
		log.Printf("[WARN] Неверный формат данных отзыва: %v", err)
		h.sendErrorResponse(w, "Неверный формат данных отзыва", http.StatusBadRequest)
		return
	}

	log.Printf("[INFO] Создание отзыва: DealID=%d, ToUserID=%d, Rating=%d",
		reviewData.DealID, reviewData.ToUserID, reviewData.Rating)

	// Создаем отзыв через сервис
	review, err := h.service.CreateReview(user.ID, &reviewData)
	if err != nil {
		log.Printf("[WARN] Ошибка создания отзыва: %v", err)
		h.sendErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("[INFO] Отзыв создан успешно: ID=%d, Rating=%d", review.ID, review.Rating)
	h.sendJSONResponse(w, map[string]interface{}{
		"success": true,
		"review":  review,
		"message": "Отзыв успешно создан",
	})
}

// handleGetUserProfile обрабатывает получение профиля пользователя
func (h *Handler) handleGetUserProfile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		h.sendErrorResponse(w, "Неверный ID пользователя", http.StatusBadRequest)
		return
	}

	log.Printf("[INFO] Запрос профиля пользователя ID=%d", userID)

	// Получаем статистику профиля пользователя
	profile, err := h.service.GetUserProfile(userID)
	if err != nil {
		log.Printf("[ERROR] Ошибка получения профиля пользователя ID=%d: %v", userID, err)
		h.sendErrorResponse(w, "Не удалось получить профиль пользователя", http.StatusInternalServerError)
		return
	}

	log.Printf("[INFO] Профиль пользователя ID=%d получен успешно", userID)
	h.sendJSONResponse(w, map[string]interface{}{
		"success": true,
		"profile": profile,
	})
}

// =====================================================
// ИНФОРМАЦИОННЫЕ ОБРАБОТЧИКИ
// =====================================================

// handleHealthCheck обрабатывает проверку состояния сервиса
func (h *Handler) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	// Проверяем состояние сервиса и его зависимостей
	if err := h.service.HealthCheck(); err != nil {
		log.Printf("[ERROR] Проблемы со здоровьем сервиса: %v", err)
		h.sendErrorResponse(w, "Сервис недоступен", http.StatusServiceUnavailable)
		return
	}

	h.sendJSONResponse(w, map[string]interface{}{
		"status":    "ok",
		"message":   "Сервис работает нормально",
		"timestamp": strings.Split(log.Prefix(), " ")[0], // Простая временная метка
	})
}

// handleIndex обрабатывает главную страницу веб-приложения
func (h *Handler) handleIndex(w http.ResponseWriter, r *http.Request) {
	// Чистый HTML для Telegram мини-приложения с внешними CSS/JS файлами
	html := `
<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0, user-scalable=no, maximum-scale=1.0">
    <title>P2P Крипто Биржа</title>
    <link rel="stylesheet" href="/static/style.css">
    <script src="https://telegram.org/js/telegram-web-app.js"></script>
</head>
<body>
    <!-- Заголовок приложения -->
    <div class="header">
        <h1>🔄 P2P Крипто Биржа</h1>
        <div class="user-info">Загрузка...</div>
    </div>

    <!-- Навигация -->
    <div class="navigation">
        <button class="nav-item active" data-view="orders">📋 Заявки</button>
        <button class="nav-item" data-view="my-orders">📝 Мои</button>
        <button class="nav-item" data-view="deals">🤝 Сделки</button>
        <button class="nav-item" data-view="profile">👤 Профиль</button>
    </div>

    <!-- Основной контент -->
    <div class="container">
        
        <!-- Раздел "Все заявки" -->
        <div id="ordersView" class="view">
            <div class="filters">
                <div class="filter-row">
                    <select class="form-select filter-select" data-filter="type">
                        <option value="">Все типы</option>
                        <option value="buy">Покупка</option>
                        <option value="sell">Продажа</option>
                    </select>
                    
                    <select class="form-select filter-select" data-filter="cryptocurrency">
                        <option value="">Все монеты</option>
                        <option value="BTC">Bitcoin (BTC)</option>
                        <option value="ETH">Ethereum (ETH)</option>
                        <option value="USDT">Tether (USDT)</option>
                        <option value="USDC">USD Coin (USDC)</option>
                    </select>
                </div>
                
                <button class="btn btn-primary" id="createOrderBtn">+ Создать заявку</button>
            </div>
            
            <div id="ordersContent">
                <div class="loading">
                    <div class="spinner"></div>
                </div>
            </div>
        </div>

        <!-- Раздел "Мои заявки" -->
        <div id="my-ordersView" class="view hidden">
            <!-- Контент будет загружен через JavaScript -->
        </div>

        <!-- Раздел "Сделки" -->
        <div id="dealsView" class="view hidden">
            <!-- Контент будет загружен через JavaScript -->
        </div>

        <!-- Раздел "Профиль" -->
        <div id="profileView" class="view hidden">
            <!-- Контент будет загружен через JavaScript -->
        </div>
    </div>

    <!-- Модальное окно создания заявки -->
    <div id="createOrderModal" class="modal">
        <div class="modal-content">
            <div class="modal-header">
                <h3 class="modal-title">Создать заявку</h3>
                <button class="modal-close">&times;</button>
            </div>
            
            <form id="createOrderForm">
                <div class="form-group">
                    <label class="form-label">Тип операции</label>
                    <select class="form-select" name="type" required>
                        <option value="">Выберите тип</option>
                        <option value="buy">Покупка</option>
                        <option value="sell">Продажа</option>
                    </select>
                </div>
                
                <div class="form-row">
                    <div class="form-group">
                        <label class="form-label">Криптовалюта</label>
                        <select class="form-select" name="cryptocurrency" required>
                            <option value="">Выберите монету</option>
                            <option value="BTC">Bitcoin (BTC)</option>
                            <option value="ETH">Ethereum (ETH)</option>
                            <option value="USDT">Tether (USDT)</option>
                            <option value="USDC">USD Coin (USDC)</option>
                        </select>
                    </div>
                    
                    <div class="form-group">
                        <label class="form-label">Валюта</label>
                        <select class="form-select" name="fiat_currency" required>
                            <option value="RUB">Рубли (RUB)</option>
                            <option value="USD">Доллары (USD)</option>
                            <option value="EUR">Евро (EUR)</option>
                        </select>
                    </div>
                </div>
                
                <div class="form-row">
                    <div class="form-group">
                        <label class="form-label">Количество</label>
                        <input type="number" class="form-input" name="amount" step="0.00000001" required>
                    </div>
                    
                    <div class="form-group">
                        <label class="form-label">Цена</label>
                        <input type="number" class="form-input" name="price" step="0.01" required>
                    </div>
                </div>
                
                <div class="form-group">
                    <label class="form-label">Способы оплаты</label>
                    <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 8px;">
                        <label><input type="checkbox" name="payment_methods" value="sberbank"> Сбербанк</label>
                        <label><input type="checkbox" name="payment_methods" value="tinkoff"> Тинькофф</label>
                        <label><input type="checkbox" name="payment_methods" value="qiwi"> QIWI</label>
                        <label><input type="checkbox" name="payment_methods" value="yandex_money"> ЮMoney</label>
                        <label><input type="checkbox" name="payment_methods" value="bank_transfer"> Банк</label>
                        <label><input type="checkbox" name="payment_methods" value="cash"> Наличные</label>
                    </div>
                </div>
                
                <div class="form-group">
                    <label class="form-label">Описание (необязательно)</label>
                    <textarea class="form-textarea" name="description" rows="3" maxlength="200"></textarea>
                </div>
                
                <div class="form-group">
                    <label>
                        <input type="checkbox" name="auto_match" checked> 
                        Автоматическое сопоставление
                    </label>
                </div>
                
                <button type="submit" class="btn btn-primary">Создать заявку</button>
            </form>
        </div>
    </div>



    <script src="/static/app.js"></script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(html))
}

// =====================================================
// ВСПОМОГАТЕЛЬНЫЕ МЕТОДЫ
// =====================================================

// sendJSONResponse отправляет JSON ответ клиенту
func (h *Handler) sendJSONResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("[ERROR] Ошибка кодирования JSON ответа: %v", err)
		http.Error(w, "Ошибка сервера", http.StatusInternalServerError)
	}
}

// sendErrorResponse отправляет JSON ответ с ошибкой
func (h *Handler) sendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResponse := map[string]interface{}{
		"success": false,
		"error":   message,
		"code":    statusCode,
	}

	if err := json.NewEncoder(w).Encode(errorResponse); err != nil {
		log.Printf("[ERROR] Ошибка кодирования JSON ошибки: %v", err)
	}
}
