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

	// TODO: Реализовать получение заявки по ID
	log.Printf("[INFO] Запрос заявки по ID: %d", orderID)
	h.sendJSONResponse(w, map[string]interface{}{
		"message":  "Эндпоинт в разработке",
		"order_id": orderID,
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

// =====================================================
// ОБРАБОТЧИКИ СДЕЛОК
// =====================================================

// handleGetDeals обрабатывает получение списка сделок пользователя
func (h *Handler) handleGetDeals(w http.ResponseWriter, r *http.Request) {
	log.Println("[INFO] Обработка запроса получения сделок пользователя")

	// TODO: Получить ID пользователя из JWT токена
	userID := int64(1) // Заглушка

	// Получаем сделки пользователя через сервис
	deals, err := h.service.GetUserDeals(userID)
	if err != nil {
		log.Printf("[ERROR] Ошибка при получении сделок: %v", err)
		h.sendErrorResponse(w, "Не удалось получить сделки", http.StatusInternalServerError)
		return
	}

	log.Printf("[INFO] Возвращено сделок пользователю: %d", len(deals))
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

	// TODO: Получить ID пользователя из JWT токена
	userID := int64(1) // Заглушка

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
	review, err := h.service.CreateReview(userID, &reviewData)
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
	// Оптимизированный мобильный интерфейс для Telegram мини-приложения
	html := `
<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0, user-scalable=no, maximum-scale=1.0">
    <title>P2P Крипто Биржа</title>
    <script src="https://telegram.org/js/telegram-web-app.js"></script>
    <style>
/* Telegram WebApp оптимизированные стили */
* {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
}

body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    background: var(--tg-theme-bg-color, #ffffff);
    color: var(--tg-theme-text-color, #000000);
    font-size: 14px;
    line-height: 1.4;
    overflow-x: hidden;
    -webkit-font-smoothing: antialiased;
}

.header {
    background: var(--tg-theme-header-bg-color, #f8f9fa);
    color: var(--tg-theme-text-color, #000000);
    padding: 8px 16px;
    text-align: center;
    border-bottom: 1px solid var(--tg-theme-section-separator-color, #e1e8ed);
    position: sticky;
    top: 0;
    z-index: 100;
}

.header h1 {
    font-size: 16px;
    font-weight: 600;
    margin-bottom: 2px;
}

.user-info {
    font-size: 12px;
    opacity: 0.7;
}

.navigation {
    display: flex;
    background: var(--tg-theme-secondary-bg-color, #f1f3f4);
    border-bottom: 1px solid var(--tg-theme-section-separator-color, #e1e8ed);
    overflow-x: auto;
    -webkit-overflow-scrolling: touch;
}

.nav-item {
    flex: 1;
    min-width: 70px;
    padding: 8px 4px;
    border: none;
    background: transparent;
    color: var(--tg-theme-hint-color, #708499);
    font-size: 11px;
    cursor: pointer;
    white-space: nowrap;
    transition: all 0.2s;
}

.nav-item.active {
    color: var(--tg-theme-link-color, #2481cc);
    background: var(--tg-theme-bg-color, #ffffff);
}

.container {
    padding: 12px;
    max-height: calc(100vh - 120px);
    overflow-y: auto;
}

.view {
    display: none;
}

.view:not(.hidden) {
    display: block;
}

.filters {
    margin-bottom: 12px;
}

.filter-row {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 8px;
    margin-bottom: 8px;
}

.form-select, .form-input {
    padding: 8px;
    border: 1px solid var(--tg-theme-section-separator-color, #e1e8ed);
    border-radius: 6px;
    background: var(--tg-theme-bg-color, #ffffff);
    color: var(--tg-theme-text-color, #000000);
    font-size: 13px;
    width: 100%;
}

.btn {
    padding: 8px 16px;
    border-radius: 6px;
    border: none;
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s;
}

.btn-primary {
    background: var(--tg-theme-button-color, #2481cc);
    color: var(--tg-theme-button-text-color, #ffffff);
    width: 100%;
}

.btn-primary:hover {
    opacity: 0.9;
}

.loading {
    text-align: center;
    padding: 20px;
    color: var(--tg-theme-hint-color, #708499);
}

.spinner {
    border: 2px solid var(--tg-theme-section-separator-color, #e1e8ed);
    border-top: 2px solid var(--tg-theme-link-color, #2481cc);
    border-radius: 50%;
    width: 20px;
    height: 20px;
    animation: spin 1s linear infinite;
    margin: 0 auto;
}

@keyframes spin {
    0% { transform: rotate(0deg); }
    100% { transform: rotate(360deg); }
}

.modal {
    display: none;
    position: fixed;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background: rgba(0,0,0,0.5);
    z-index: 1000;
}

.modal.show {
    display: flex;
    align-items: center;
    justify-content: center;
}

.modal-content {
    background: var(--tg-theme-bg-color, #ffffff);
    padding: 16px;
    border-radius: 12px;
    width: 90%;
    max-width: 400px;
    max-height: 80vh;
    overflow-y: auto;
}

.modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 16px;
}

.modal-title {
    font-size: 16px;
    font-weight: 600;
}

.modal-close {
    background: none;
    border: none;
    font-size: 20px;
    cursor: pointer;
    color: var(--tg-theme-hint-color, #708499);
}

.form-group {
    margin-bottom: 12px;
}

.form-label {
    display: block;
    margin-bottom: 4px;
    font-size: 12px;
    font-weight: 500;
    color: var(--tg-theme-hint-color, #708499);
}

.form-row {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 8px;
}

.form-textarea {
    padding: 8px;
    border: 1px solid var(--tg-theme-section-separator-color, #e1e8ed);
    border-radius: 6px;
    background: var(--tg-theme-bg-color, #ffffff);
    color: var(--tg-theme-text-color, #000000);
    font-size: 13px;
    width: 100%;
    resize: vertical;
    font-family: inherit;
}

.text-center {
    text-align: center;
}

.mt-md {
    margin-top: 24px;
}

.text-muted {
    color: var(--tg-theme-hint-color, #708499);
    font-size: 12px;
}

.hidden {
    display: none !important;
}

/* Адаптация для очень маленьких экранов */
@media (max-width: 375px) {
    .container {
        padding: 8px;
    }
    
    .modal-content {
        padding: 12px;
    }
    
    .form-row {
        grid-template-columns: 1fr;
    }
}
    </style>
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
            <div class="text-center mt-md">
                <h2>Мои заявки</h2>
                <p class="text-muted">Здесь будут отображаться ваши активные и завершенные заявки</p>
            </div>
        </div>

        <!-- Раздел "Сделки" -->
        <div id="dealsView" class="view hidden">
            <div class="text-center mt-md">
                <h2>История сделок</h2>
                <p class="text-muted">Здесь будет история ваших завершенных сделок</p>
            </div>
        </div>

        <!-- Раздел "Профиль" -->
        <div id="profileView" class="view hidden">
            <div class="text-center mt-md">
                <h2>Мой профиль</h2>
                <p class="text-muted">Информация о профиле, рейтинг и отзывы</p>
            </div>
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

    <script>
// Глобальные переменные
let currentUser = null;
let tg = window.Telegram?.WebApp;

// Инициализация Telegram WebApp
function initTelegramWebApp() {
    if (tg) {
        tg.ready();
        tg.expand();
        tg.disableVerticalSwipes();
        
        // Получаем данные пользователя из Telegram
        if (tg.initDataUnsafe?.user) {
            currentUser = tg.initDataUnsafe.user;
            document.querySelector('.user-info').textContent = 
                `👤 ${currentUser.first_name} ${currentUser.last_name || ''}`.trim();
        }
        
        // Применяем цветовую схему Telegram
        document.body.style.backgroundColor = tg.backgroundColor || '#ffffff';
        
        console.log('[INFO] Telegram WebApp инициализирован', currentUser);
    } else {
        console.warn('[WARN] Telegram WebApp API недоступен');
        document.querySelector('.user-info').textContent = '👤 Демо режим';
    }
}

// Навигация между разделами
function initNavigation() {
    const navItems = document.querySelectorAll('.nav-item');
    const views = document.querySelectorAll('.view');
    
    navItems.forEach(item => {
        item.addEventListener('click', () => {
            const viewName = item.dataset.view;
            
            // Обновляем активную навигацию
            navItems.forEach(nav => nav.classList.remove('active'));
            item.classList.add('active');
            
            // Показываем нужный раздел
            views.forEach(view => {
                view.style.display = view.id === viewName + 'View' ? 'block' : 'none';
            });
            
            // Загружаем данные для раздела
            if (viewName === 'orders') {
                loadOrders();
            }
        });
    });
}

// Модальное окно
function initModal() {
    const modal = document.getElementById('createOrderModal');
    const createBtn = document.getElementById('createOrderBtn');
    const closeBtn = document.querySelector('.modal-close');
    const form = document.getElementById('createOrderForm');
    
    createBtn.addEventListener('click', () => {
        modal.classList.add('show');
    });
    
    closeBtn.addEventListener('click', () => {
        modal.classList.remove('show');
    });
    
    modal.addEventListener('click', (e) => {
        if (e.target === modal) {
            modal.classList.remove('show');
        }
    });
    
    form.addEventListener('submit', handleCreateOrder);
}

// Создание заявки
async function handleCreateOrder(e) {
    e.preventDefault();
    
    if (!currentUser) {
        showError('Пользователь не авторизован');
        return;
    }
    
    const formData = new FormData(e.target);
    const paymentMethods = [];
    
    // Собираем выбранные способы оплаты
    formData.getAll('payment_methods').forEach(method => {
        paymentMethods.push(method);
    });
    
    const orderData = {
        type: formData.get('type'),
        cryptocurrency: formData.get('cryptocurrency'),
        fiat_currency: formData.get('fiat_currency'),
        amount: parseFloat(formData.get('amount')),
        price: parseFloat(formData.get('price')),
        payment_methods: paymentMethods,
        description: formData.get('description') || '',
        auto_match: formData.has('auto_match')
    };
    
    try {
        const response = await fetch('/api/v1/orders', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'X-Telegram-User-ID': currentUser.id.toString()
            },
            body: JSON.stringify(orderData)
        });
        
        const result = await response.json();
        
        if (result.success) {
            showSuccess('Заявка успешно создана!');
            document.getElementById('createOrderModal').classList.remove('show');
            e.target.reset();
            loadOrders(); // Перезагружаем список заявок
        } else {
            showError(result.error || 'Ошибка создания заявки');
        }
    } catch (error) {
        console.error('[ERROR] Ошибка создания заявки:', error);
        showError('Ошибка сети. Попробуйте позже.');
    }
}

// Загрузка заявок
async function loadOrders() {
    const content = document.getElementById('ordersContent');
    content.innerHTML = '<div class="loading"><div class="spinner"></div><p>Загрузка заявок...</p></div>';
    
    try {
        const response = await fetch('/api/v1/orders');
        const result = await response.json();
        
        if (result.success) {
            displayOrders(result.orders || []);
        } else {
            content.innerHTML = '<p class="text-center text-muted">Ошибка загрузки заявок</p>';
        }
    } catch (error) {
        console.error('[ERROR] Ошибка загрузки заявок:', error);
        content.innerHTML = '<p class="text-center text-muted">Ошибка сети</p>';
    }
}

// Отображение заявок
function displayOrders(orders) {
    const content = document.getElementById('ordersContent');
    
    if (orders.length === 0) {
        content.innerHTML = '<p class="text-center text-muted">Заявок пока нет</p>';
        return;
    }
    
    const ordersHTML = orders.map(order => `
        <div style="border: 1px solid var(--tg-theme-section-separator-color, #e1e8ed); 
                    border-radius: 8px; padding: 12px; margin-bottom: 8px;
                    background: var(--tg-theme-secondary-bg-color, #f8f9fa);">
            <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px;">
                <span style="font-weight: 600; color: ${order.type === 'buy' ? '#22c55e' : '#ef4444'};">
                    ${order.type === 'buy' ? '🟢 Покупка' : '🔴 Продажа'}
                </span>
                <span style="font-size: 12px; color: var(--tg-theme-hint-color, #708499);">
                    ${new Date(order.created_at).toLocaleString('ru')}
                </span>
            </div>
            <div style="margin-bottom: 8px;">
                <strong>${order.amount} ${order.cryptocurrency}</strong> за <strong>${order.price} ${order.fiat_currency}</strong>
            </div>
            <div style="font-size: 12px; color: var(--tg-theme-hint-color, #708499);">
                Способы оплаты: ${(order.payment_methods || []).join(', ') || 'Не указано'}
            </div>
            ${order.description ? `<div style="font-size: 12px; margin-top: 4px;">${order.description}</div>` : ''}
        </div>
    `).join('');
    
    content.innerHTML = ordersHTML;
}

// Уведомления
function showSuccess(message) {
    if (tg) {
        tg.showAlert(message);
    } else {
        alert('✅ ' + message);
    }
}

function showError(message) {
    if (tg) {
        tg.showAlert('❌ ' + message);
    } else {
        alert('❌ ' + message);
    }
}

// Инициализация приложения
document.addEventListener('DOMContentLoaded', () => {
    initTelegramWebApp();
    initNavigation();
    initModal();
    loadOrders(); // Загружаем заявки при старте
});
    </script>
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
