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
	api.HandleFunc("/orders/my", h.handleGetMyOrders).Methods("GET")      // Получить мои заявки (ВАЖНО: должно быть ДО {id})
	api.HandleFunc("/orders/{id}", h.handleGetOrder).Methods("GET")       // Получить заявку по ID
	api.HandleFunc("/orders/{id}", h.handleUpdateOrder).Methods("PUT")    // Обновить заявку
	api.HandleFunc("/orders/{id}", h.handleCancelOrder).Methods("DELETE") // Отменить заявку

	// Управление сделками
	api.HandleFunc("/deals", h.handleGetDeals).Methods("GET")                  // Получить список сделок пользователя
	api.HandleFunc("/deals", h.handleCreateDeal).Methods("POST")               // Создать новую сделку (отклик)
	api.HandleFunc("/deals/{id}", h.handleGetDeal).Methods("GET")              // Получить сделку по ID
	api.HandleFunc("/deals/{id}/confirm", h.handleConfirmDeal).Methods("POST") // Подтвердить сделку

	// Система откликов
	api.HandleFunc("/responses", h.handleCreateResponse).Methods("POST")              // Создать отклик на заявку
	api.HandleFunc("/responses/my", h.handleGetMyResponses).Methods("GET")            // Получить мои отклики
	api.HandleFunc("/responses/to-my", h.handleGetResponsesToMyOrders).Methods("GET") // Получить отклики на мои заявки
	api.HandleFunc("/responses/{id}/accept", h.handleAcceptResponse).Methods("POST")  // Принять отклик
	api.HandleFunc("/responses/{id}/reject", h.handleRejectResponse).Methods("POST")  // Отклонить отклик

	// Система отзывов и рейтингов
	api.HandleFunc("/reviews", h.handleGetReviews).Methods("GET")    // Получить отзывы пользователя
	api.HandleFunc("/reviews", h.handleCreateReview).Methods("POST") // Оставить отзыв

	api.HandleFunc("/users/{id}/profile", h.handleGetUserProfile).Methods("GET") // Получить профиль пользователя
	api.HandleFunc("/auth/stats", h.handleGetMyStats).Methods("GET")             // Получить статистику текущего пользователя
	api.HandleFunc("/auth/reviews", h.handleGetMyReviews).Methods("GET")         // Получить отзывы текущего пользователя

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
	log.Printf("[INFO] Обработка запроса получения текущего пользователя")

	// Получаем Telegram ID пользователя из заголовка
	telegramIDStr := r.Header.Get("X-Telegram-User-ID")
	if telegramIDStr == "" {
		log.Printf("[WARN] Не передан Telegram ID пользователя в /auth/me")
		h.sendErrorResponse(w, "Требуется авторизация", http.StatusUnauthorized)
		return
	}

	telegramID, err := strconv.ParseInt(telegramIDStr, 10, 64)
	if err != nil {
		log.Printf("[WARN] Неверный формат Telegram ID в /auth/me: %v", err)
		h.sendErrorResponse(w, "Неверный ID пользователя", http.StatusBadRequest)
		return
	}

	// Получаем пользователя по Telegram ID
	user, err := h.service.GetUserByTelegramID(telegramID)
	if err != nil {
		log.Printf("[ERROR] Пользователь не найден в /auth/me: %v", err)
		h.sendErrorResponse(w, "Пользователь не найден", http.StatusNotFound)
		return
	}

	log.Printf("[INFO] Данные текущего пользователя получены: ID=%d, TelegramID=%d", user.ID, user.TelegramID)
	h.sendJSONResponse(w, map[string]interface{}{
		"success": true,
		"user":    user,
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

	// Включить неактивные заявки (для фильтрации откликов)
	if includeInactive := r.URL.Query().Get("include_inactive"); includeInactive == "true" {
		filter.IncludeInactive = true
		log.Printf("[DEBUG] Включены неактивные заявки для фильтрации")
	}

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

// handleUpdateOrder обрабатывает обновление существующей заявки
func (h *Handler) handleUpdateOrder(w http.ResponseWriter, r *http.Request) {
	log.Println("[INFO] Обработка запроса обновления заявки")

	// Получаем ID заявки из URL
	vars := mux.Vars(r)
	orderID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		log.Printf("[WARN] Неверный ID заявки: %v", err)
		h.sendErrorResponse(w, "Неверный ID заявки", http.StatusBadRequest)
		return
	}

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

	// Обновляем заявку через сервис
	order, err := h.service.UpdateOrder(orderID, telegramID, &orderData)
	if err != nil {
		log.Printf("[WARN] Ошибка обновления заявки: %v", err)
		h.sendErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("[INFO] Заявка успешно обновлена: ID=%d", order.ID)
	h.sendJSONResponse(w, map[string]interface{}{
		"success": true,
		"order":   order,
		"message": "Заявка успешно обновлена",
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

// handleCreateDeal обрабатывает создание нового отклика на заявку (в новой логике)
func (h *Handler) handleCreateDeal(w http.ResponseWriter, r *http.Request) {
	log.Println("[INFO] Обработка запроса создания сделки")

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

	// Читаем данные для создания сделки
	var createDealRequest struct {
		OrderID    int64  `json:"order_id"`
		Message    string `json:"message"`
		AutoAccept bool   `json:"auto_accept"`
	}

	if err := json.NewDecoder(r.Body).Decode(&createDealRequest); err != nil {
		log.Printf("[WARN] Неверный формат данных сделки: %v", err)
		h.sendErrorResponse(w, "Неверный формат данных", http.StatusBadRequest)
		return
	}

	if createDealRequest.OrderID == 0 {
		log.Printf("[WARN] Не указан ID заявки")
		h.sendErrorResponse(w, "Требуется ID заявки", http.StatusBadRequest)
		return
	}

	// Получаем пользователя по Telegram ID
	user, err := h.service.GetUserByTelegramID(telegramID)
	if err != nil {
		log.Printf("[ERROR] Пользователь не найден: %v", err)
		h.sendErrorResponse(w, "Пользователь не найден", http.StatusNotFound)
		return
	}

	// Создаем отклик через новую систему
	responseData := &model.CreateResponseRequest{
		OrderID: createDealRequest.OrderID,
		Message: createDealRequest.Message,
	}

	response, err := h.service.CreateResponse(user.ID, responseData)
	if err != nil {
		log.Printf("[WARN] Ошибка создания отклика: %v", err)
		h.sendErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("[INFO] Создан новый отклик: ID=%d", response.ID)
	h.sendJSONResponse(w, map[string]interface{}{
		"success":  true,
		"response": response,
		"message":  "Отклик успешно создан",
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

	// Получаем данные пользователя из Telegram заголовка
	telegramUserIDStr := r.Header.Get("X-Telegram-User-ID")
	if telegramUserIDStr == "" {
		log.Printf("[ERROR] Отсутствует заголовок X-Telegram-User-ID")
		h.sendErrorResponse(w, "Не авторизован", http.StatusUnauthorized)
		return
	}

	telegramUserID, err := strconv.ParseInt(telegramUserIDStr, 10, 64)
	if err != nil {
		log.Printf("[ERROR] Неверный формат Telegram User ID: %s", telegramUserIDStr)
		h.sendErrorResponse(w, "Неверный формат пользователя", http.StatusBadRequest)
		return
	}

	// Получаем пользователя по Telegram ID
	user, err := h.service.GetUserByTelegramID(telegramUserID)
	if err != nil {
		log.Printf("[ERROR] Ошибка получения пользователя: %v", err)
		h.sendErrorResponse(w, "Не авторизован", http.StatusUnauthorized)
		return
	}

	log.Printf("[INFO] Подтверждение сделки ID=%d пользователем ID=%d", dealID, user.ID)

	// Читаем данные запроса
	var requestData struct {
		IsAuthor     bool   `json:"is_author"`     // Является ли пользователь автором заявки
		PaymentProof string `json:"payment_proof"` // Доказательство оплаты (ссылка на скриншот)
		Notes        string `json:"notes"`         // Дополнительные заметки
	}

	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		log.Printf("[WARN] Неверный формат данных подтверждения: %v", err)
		// Игнорируем ошибку декодирования - подтверждение без доказательств
	}

	// Используем базовое подтверждение если не указано
	paymentProof := requestData.PaymentProof
	if paymentProof == "" {
		paymentProof = "confirmed_via_webapp"
	}

	// Подтверждаем сделку через сервис с указанием роли пользователя
	if err := h.service.ConfirmDealWithRole(dealID, user.ID, requestData.IsAuthor, paymentProof); err != nil {
		log.Printf("[WARN] Ошибка подтверждения сделки ID=%d: %v", dealID, err)
		h.sendErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("[INFO] Сделка ID=%d подтверждена пользователем ID=%d", dealID, user.ID)

	// Проверяем завершена ли сделка (оба участника подтвердили)
	deal, err := h.service.GetDeal(dealID, user.ID)
	dealCompleted := false
	if err == nil {
		dealCompleted = deal.Status == "completed"
	}
	// Возвращаем успешный ответ
	h.sendJSONResponse(w, map[string]interface{}{
		"success":        true,
		"message":        "Сделка успешно подтверждена",
		"deal_completed": dealCompleted,
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
	log.Printf("[DEBUG] Отправитель отзыва: TelegramID=%d, InternalID=%d", telegramID, user.ID)

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

	// Получаем данные пользователя и статистику профиля
	userProfile, err := h.service.GetFullUserProfile(userID)
	if err != nil {
		log.Printf("[ERROR] Ошибка получения профиля пользователя ID=%d: %v", userID, err)
		h.sendErrorResponse(w, "Не удалось получить профиль пользователя", http.StatusInternalServerError)
		return
	}

	log.Printf("[INFO] Профиль пользователя ID=%d получен успешно", userID)
	h.sendJSONResponse(w, map[string]interface{}{
		"success": true,
		"profile": userProfile,
	})
}

// handleGetMyReviews обрабатывает получение отзывов текущего пользователя
func (h *Handler) handleGetMyReviews(w http.ResponseWriter, r *http.Request) {
	log.Printf("[INFO] Обработка запроса получения отзывов текущего пользователя")

	// Получаем Telegram ID пользователя из заголовка
	telegramIDStr := r.Header.Get("X-Telegram-User-ID")
	if telegramIDStr == "" {
		log.Printf("[WARN] Не передан Telegram ID пользователя для отзывов")
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
		log.Printf("[ERROR] Пользователь не найден для отзывов: %v", err)
		h.sendErrorResponse(w, "Пользователь не найден", http.StatusNotFound)
		return
	}

	// Параметры пагинации
	limit := 20
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	log.Printf("[INFO] Получение отзывов для пользователя ID=%d (limit=%d, offset=%d)", user.ID, limit, offset)

	// Получаем отзывы пользователя
	reviews, err := h.service.GetUserReviews(user.ID, limit, offset)
	if err != nil {
		log.Printf("[ERROR] Ошибка получения отзывов: %v", err)
		h.sendErrorResponse(w, "Не удалось получить отзывы", http.StatusInternalServerError)
		return
	}

	log.Printf("[INFO] Получено отзывов: %d", len(reviews))
	h.sendJSONResponse(w, map[string]interface{}{
		"success": true,
		"reviews": reviews,
		"count":   len(reviews),
	})
}

// handleGetMyStats обрабатывает получение статистики текущего пользователя
func (h *Handler) handleGetMyStats(w http.ResponseWriter, r *http.Request) {
	log.Printf("[INFO] Обработка запроса получения статистики пользователя")

	// Получаем Telegram ID пользователя из заголовка
	telegramIDStr := r.Header.Get("X-Telegram-User-ID")
	if telegramIDStr == "" {
		log.Printf("[WARN] Не передан Telegram ID пользователя для статистики")
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

	log.Printf("[INFO] Запрос статистики пользователя ID=%d", user.ID)

	// Получаем статистику пользователя
	stats, err := h.service.GetUserStats(user.ID)
	if err != nil {
		log.Printf("[ERROR] Ошибка получения статистики пользователя ID=%d: %v", user.ID, err)
		h.sendErrorResponse(w, "Не удалось получить статистику", http.StatusInternalServerError)
		return
	}

	log.Printf("[INFO] Статистика пользователя ID=%d получена успешно", user.ID)
	h.sendJSONResponse(w, map[string]interface{}{
		"success": true,
		"stats":   stats,
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
	// Отдаем HTML файл из папки templates
	http.ServeFile(w, r, "web/templates/index.html")
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

// =====================================================
// ОБРАБОТЧИКИ ДЛЯ ОТКЛИКОВ
// =====================================================

// handleCreateResponse создает новый отклик на заявку
func (h *Handler) handleCreateResponse(w http.ResponseWriter, r *http.Request) {
	log.Printf("[INFO] %s %s - Создание отклика", r.Method, r.URL.Path)

	// Получаем данные пользователя
	// Извлекаем пользователя из заголовка Telegram
	telegramUserIDStr := r.Header.Get("X-Telegram-User-ID")
	if telegramUserIDStr == "" {
		log.Printf("[ERROR] Отсутствует заголовок X-Telegram-User-ID")
		http.Error(w, "Не авторизован", http.StatusUnauthorized)
		return
	}

	telegramUserID, err := strconv.ParseInt(telegramUserIDStr, 10, 64)
	if err != nil {
		log.Printf("[ERROR] Неверный формат Telegram User ID: %s", telegramUserIDStr)
		http.Error(w, "Неверный формат пользователя", http.StatusBadRequest)
		return
	}

	// Получаем пользователя по Telegram ID
	user, err := h.service.GetUserByTelegramID(telegramUserID)
	if err != nil {
		log.Printf("[ERROR] Ошибка получения пользователя: %v", err)
		http.Error(w, "Не авторизован", http.StatusUnauthorized)
		return
	}

	// Декодируем запрос
	var requestData model.CreateResponseRequest
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		log.Printf("[ERROR] Некорректный JSON в запросе создания отклика: %v", err)
		http.Error(w, "Некорректный формат данных", http.StatusBadRequest)
		return
	}

	// Создаем отклик через сервис
	response, err := h.service.CreateResponse(user.ID, &requestData)
	if err != nil {
		log.Printf("[ERROR] Ошибка создания отклика: %v", err)
		h.sendJSONResponse(w, map[string]interface{}{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// Успешный ответ
	h.sendJSONResponse(w, map[string]interface{}{
		"success":  true,
		"message":  "Отклик успешно создан",
		"response": response,
	})

	log.Printf("[INFO] Отклик создан: ID=%d, UserID=%d, OrderID=%d",
		response.ID, user.ID, requestData.OrderID)
}

// handleGetMyResponses получает отклики пользователя
func (h *Handler) handleGetMyResponses(w http.ResponseWriter, r *http.Request) {
	log.Printf("[INFO] %s %s - Получение моих откликов", r.Method, r.URL.Path)

	// Получаем данные пользователя
	// Извлекаем пользователя из заголовка Telegram
	telegramUserIDStr := r.Header.Get("X-Telegram-User-ID")
	if telegramUserIDStr == "" {
		log.Printf("[ERROR] Отсутствует заголовок X-Telegram-User-ID")
		http.Error(w, "Не авторизован", http.StatusUnauthorized)
		return
	}

	telegramUserID, err := strconv.ParseInt(telegramUserIDStr, 10, 64)
	if err != nil {
		log.Printf("[ERROR] Неверный формат Telegram User ID: %s", telegramUserIDStr)
		http.Error(w, "Неверный формат пользователя", http.StatusBadRequest)
		return
	}

	// Получаем пользователя по Telegram ID
	user, err := h.service.GetUserByTelegramID(telegramUserID)
	if err != nil {
		log.Printf("[ERROR] Ошибка получения пользователя: %v", err)
		http.Error(w, "Не авторизован", http.StatusUnauthorized)
		return
	}

	// Получаем отклики через сервис
	responses, err := h.service.GetMyResponses(user.ID)
	if err != nil {
		log.Printf("[ERROR] Ошибка получения откликов: %v", err)
		h.sendJSONResponse(w, map[string]interface{}{
			"success": false,
			"message": "Не удалось загрузить отклики",
		})
		return
	}

	// Успешный ответ
	h.sendJSONResponse(w, map[string]interface{}{
		"success":   true,
		"responses": responses,
		"count":     len(responses),
	})

	log.Printf("[INFO] Возвращено откликов пользователя ID=%d: %d", user.ID, len(responses))
}

// handleGetResponsesToMyOrders получает отклики на заявки пользователя
func (h *Handler) handleGetResponsesToMyOrders(w http.ResponseWriter, r *http.Request) {
	log.Printf("[INFO] %s %s - Получение откликов на мои заявки", r.Method, r.URL.Path)

	// Получаем данные пользователя
	// Извлекаем пользователя из заголовка Telegram
	telegramUserIDStr := r.Header.Get("X-Telegram-User-ID")
	if telegramUserIDStr == "" {
		log.Printf("[ERROR] Отсутствует заголовок X-Telegram-User-ID")
		http.Error(w, "Не авторизован", http.StatusUnauthorized)
		return
	}

	telegramUserID, err := strconv.ParseInt(telegramUserIDStr, 10, 64)
	if err != nil {
		log.Printf("[ERROR] Неверный формат Telegram User ID: %s", telegramUserIDStr)
		http.Error(w, "Неверный формат пользователя", http.StatusBadRequest)
		return
	}

	// Получаем пользователя по Telegram ID
	user, err := h.service.GetUserByTelegramID(telegramUserID)
	if err != nil {
		log.Printf("[ERROR] Ошибка получения пользователя: %v", err)
		http.Error(w, "Не авторизован", http.StatusUnauthorized)
		return
	}

	// Получаем отклики через сервис
	responses, err := h.service.GetResponsesToMyOrders(user.ID)
	if err != nil {
		log.Printf("[ERROR] Ошибка получения откликов на заявки: %v", err)
		h.sendJSONResponse(w, map[string]interface{}{
			"success": false,
			"message": "Не удалось загрузить отклики",
		})
		return
	}

	// Успешный ответ
	h.sendJSONResponse(w, map[string]interface{}{
		"success":   true,
		"responses": responses,
		"count":     len(responses),
	})

	log.Printf("[INFO] Возвращено откликов на заявки автора ID=%d: %d", user.ID, len(responses))
}

// handleAcceptResponse принимает отклик
func (h *Handler) handleAcceptResponse(w http.ResponseWriter, r *http.Request) {
	log.Printf("[INFO] %s %s - Принятие отклика", r.Method, r.URL.Path)

	// Получаем данные пользователя
	// Извлекаем пользователя из заголовка Telegram
	telegramUserIDStr := r.Header.Get("X-Telegram-User-ID")
	if telegramUserIDStr == "" {
		log.Printf("[ERROR] Отсутствует заголовок X-Telegram-User-ID")
		http.Error(w, "Не авторизован", http.StatusUnauthorized)
		return
	}

	telegramUserID, err := strconv.ParseInt(telegramUserIDStr, 10, 64)
	if err != nil {
		log.Printf("[ERROR] Неверный формат Telegram User ID: %s", telegramUserIDStr)
		http.Error(w, "Неверный формат пользователя", http.StatusBadRequest)
		return
	}

	// Получаем пользователя по Telegram ID
	user, err := h.service.GetUserByTelegramID(telegramUserID)
	if err != nil {
		log.Printf("[ERROR] Ошибка получения пользователя: %v", err)
		http.Error(w, "Не авторизован", http.StatusUnauthorized)
		return
	}

	// Получаем ID отклика из URL
	vars := mux.Vars(r)
	responseIDStr := vars["id"]
	responseID, err := strconv.ParseInt(responseIDStr, 10, 64)
	if err != nil {
		log.Printf("[ERROR] Неверный ID отклика: %s", responseIDStr)
		http.Error(w, "Неверный ID отклика", http.StatusBadRequest)
		return
	}

	// Принимаем отклик через сервис
	deal, err := h.service.AcceptResponse(responseID, user.ID)
	if err != nil {
		log.Printf("[ERROR] Ошибка принятия отклика: %v", err)
		h.sendJSONResponse(w, map[string]interface{}{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// Успешный ответ
	h.sendJSONResponse(w, map[string]interface{}{
		"success": true,
		"message": "Отклик принят, создана сделка",
		"deal":    deal,
	})

	log.Printf("[INFO] Отклик принят: ResponseID=%d, DealID=%d", responseID, deal.ID)
}

// handleRejectResponse отклоняет отклик
func (h *Handler) handleRejectResponse(w http.ResponseWriter, r *http.Request) {
	log.Printf("[INFO] %s %s - Отклонение отклика", r.Method, r.URL.Path)

	// Получаем данные пользователя
	// Извлекаем пользователя из заголовка Telegram
	telegramUserIDStr := r.Header.Get("X-Telegram-User-ID")
	if telegramUserIDStr == "" {
		log.Printf("[ERROR] Отсутствует заголовок X-Telegram-User-ID")
		http.Error(w, "Не авторизован", http.StatusUnauthorized)
		return
	}

	telegramUserID, err := strconv.ParseInt(telegramUserIDStr, 10, 64)
	if err != nil {
		log.Printf("[ERROR] Неверный формат Telegram User ID: %s", telegramUserIDStr)
		http.Error(w, "Неверный формат пользователя", http.StatusBadRequest)
		return
	}

	// Получаем пользователя по Telegram ID
	user, err := h.service.GetUserByTelegramID(telegramUserID)
	if err != nil {
		log.Printf("[ERROR] Ошибка получения пользователя: %v", err)
		http.Error(w, "Не авторизован", http.StatusUnauthorized)
		return
	}

	// Получаем ID отклика из URL
	vars := mux.Vars(r)
	responseIDStr := vars["id"]
	responseID, err := strconv.ParseInt(responseIDStr, 10, 64)
	if err != nil {
		log.Printf("[ERROR] Неверный ID отклика: %s", responseIDStr)
		http.Error(w, "Неверный ID отклика", http.StatusBadRequest)
		return
	}

	// Отклоняем отклик через сервис
	if err := h.service.RejectResponse(responseID, user.ID, "Отклонен автором"); err != nil {
		log.Printf("[ERROR] Ошибка отклонения отклика: %v", err)
		h.sendJSONResponse(w, map[string]interface{}{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// Успешный ответ
	h.sendJSONResponse(w, map[string]interface{}{
		"success": true,
		"message": "Отклик отклонен",
	})

	log.Printf("[INFO] Отклик отклонен: ResponseID=%d", responseID)
}
