package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"p2pTG-crypto-exchange/internal/model"
	"p2pTG-crypto-exchange/internal/repository"
)

// TelegramChatMemberResponse структура ответа от Telegram Bot API для проверки членства
type TelegramChatMemberResponse struct {
	OK     bool `json:"ok"`
	Result struct {
		Status string `json:"status"`
		User   struct {
			ID int64 `json:"id"`
		} `json:"user"`
	} `json:"result"`
}

// Service представляет слой бизнес-логики приложения
// Содержит все основные операции P2P криптобиржи
// Связывает слой обработчиков с репозиторием данных
type Service struct {
	repo                repository.RepositoryInterface // Интерфейс репозитория для работы с данными
	telegramToken       string                         // Токен Telegram бота для авторизации
	chatID              string                         // ID закрытого чата для проверки членства
	httpClient          *http.Client                   // HTTP клиент для запросов к Telegram Bot API
	notificationService *NotificationService           // Сервис уведомлений для отправки сообщений в Telegram
}

// NewService создает новый экземпляр сервиса (для обратной совместимости)
// Принимает интерфейс репозитория, токен бота и ID чата
func NewService(repo repository.RepositoryInterface, telegramToken, chatID string) *Service {
	// Используем локальный URL для уведомлений по умолчанию
	webAppURL := "https://localhost:8080"
	return NewServiceWithWebApp(repo, telegramToken, chatID, webAppURL)
}

// NewServiceWithWebApp создает новый экземпляр сервиса с URL веб-приложения
// Принимает интерфейс репозитория, токен бота, ID чата и URL веб-приложения
func NewServiceWithWebApp(repo repository.RepositoryInterface, telegramToken, chatID, webAppURL string) *Service {
	log.Println("[INFO] Инициализация сервиса бизнес-логики")

	// Инициализируем сервис уведомлений с переданным URL
	notificationService := NewNotificationService(telegramToken, webAppURL)

	return &Service{
		repo:                repo,
		telegramToken:       telegramToken,
		chatID:              chatID,
		httpClient:          &http.Client{Timeout: 10 * time.Second}, // HTTP клиент с таймаутом 10 сек
		notificationService: notificationService,
	}
}

// =====================================================
// АВТОРИЗАЦИЯ И АУТЕНТИФИКАЦИЯ
// =====================================================

// AuthenticateUser выполняет авторизацию пользователя через Telegram WebApp
// Проверяет подлинность данных от Telegram и создает/обновляет пользователя
func (s *Service) AuthenticateUser(authData *model.TelegramAuthData) (*model.User, error) {
	log.Printf("[INFO] Попытка авторизации пользователя: TelegramID=%d, Username=%s",
		authData.ID, authData.Username)

	// Проверяем подлинность данных от Telegram WebApp
	// ВРЕМЕННО: отключаем проверку подписи для тестирования
	if authData.Hash != "dummy_hash" && !s.validateTelegramAuth(authData) {
		log.Printf("[WARN] Неверная подпись авторизации для пользователя TelegramID=%d", authData.ID)
		return nil, fmt.Errorf("неверная подпись авторизации")
	}

	// Проверяем срок действия авторизации (не более 24 часов)
	authTime := time.Unix(authData.AuthDate, 0)
	if time.Since(authTime) > 24*time.Hour {
		log.Printf("[WARN] Истекший токен авторизации для пользователя TelegramID=%d", authData.ID)
		return nil, fmt.Errorf("срок действия авторизации истек")
	}

	// Пытаемся найти существующего пользователя
	user, err := s.repo.GetUserByTelegramID(authData.ID)
	if err != nil {
		// Если пользователь не найден, создаем нового
		if strings.Contains(err.Error(), "не найден") {
			log.Printf("[INFO] Создание нового пользователя: TelegramID=%d", authData.ID)
			user = &model.User{
				TelegramID:      authData.ID,
				TelegramUserID:  authData.Username,
				FirstName:       authData.FirstName,
				LastName:        authData.LastName,
				Username:        authData.Username,
				PhotoURL:        authData.PhotoURL,
				IsBot:           false,
				LanguageCode:    "ru",
				IsActive:        true,
				Rating:          0.0,
				TotalDeals:      0,
				SuccessfulDeals: 0,
				ChatMember:      false, // Будет обновлено после проверки членства
			}

			// Создаем пользователя в базе данных
			if err := s.repo.CreateUser(user); err != nil {
				log.Printf("[ERROR] Не удалось создать пользователя: %v", err)
				return nil, fmt.Errorf("не удалось создать пользователя: %w", err)
			}
		} else {
			log.Printf("[ERROR] Ошибка при поиске пользователя: %v", err)
			return nil, fmt.Errorf("ошибка при поиске пользователя: %w", err)
		}
	} else {
		log.Printf("[INFO] Найден существующий пользователь: ID=%d, TelegramID=%d",
			user.ID, user.TelegramID)
	}

	// Проверяем членство пользователя в закрытом чате через Telegram Bot API
	isChatMember, err := s.checkChatMembership(authData.ID)
	if err != nil {
		log.Printf("[ERROR] Не удалось проверить членство в чате для пользователя TelegramID=%d: %v",
			authData.ID, err)
		// При ошибке API считаем что пользователь не является членом чата для безопасности
		isChatMember = false
	}

	// Обновляем статус членства если изменился
	if user.ChatMember != isChatMember {
		if err := s.repo.UpdateUserChatMembership(user.TelegramID, isChatMember); err != nil {
			log.Printf("[WARN] Не удалось обновить статус членства: %v", err)
		} else {
			user.ChatMember = isChatMember
		}
	}

	// Проверяем, является ли пользователь членом чата
	if !user.ChatMember {
		log.Printf("[WARN] Пользователь TelegramID=%d не является членом закрытого чата", user.TelegramID)
		return nil, fmt.Errorf("доступ запрещен: вы не являетесь членом закрытого чата")
	}

	// Проверяем, активен ли пользователь (не заблокирован)
	if !user.IsActive {
		log.Printf("[WARN] Попытка входа заблокированного пользователя TelegramID=%d", user.TelegramID)
		return nil, fmt.Errorf("ваш аккаунт заблокирован")
	}

	log.Printf("[INFO] Успешная авторизация пользователя: ID=%d, TelegramID=%d",
		user.ID, user.TelegramID)

	return user, nil
}

// GetUserByTelegramID получает пользователя по его Telegram ID
func (s *Service) GetUserByTelegramID(telegramID int64) (*model.User, error) {
	log.Printf("[INFO] Получение пользователя по Telegram ID=%d", telegramID)

	user, err := s.repo.GetUserByTelegramID(telegramID)
	if err != nil {
		log.Printf("[ERROR] Пользователь с Telegram ID=%d не найден: %v", telegramID, err)
		return nil, fmt.Errorf("пользователь не найден")
	}

	log.Printf("[INFO] Пользователь найден: ID=%d, Telegram ID=%d", user.ID, user.TelegramID)
	return user, nil
}

// validateTelegramAuth проверяет подлинность данных авторизации от Telegram WebApp
// Использует HMAC-SHA256 для валидации подписи
func (s *Service) validateTelegramAuth(authData *model.TelegramAuthData) bool {
	// Создаем строку для подписи из данных авторизации
	// Порядок полей важен и должен соответствовать документации Telegram
	dataCheckString := fmt.Sprintf("auth_date=%d\nfirst_name=%s\nid=%d\nlast_name=%s\nphoto_url=%s\nusername=%s",
		authData.AuthDate,
		authData.FirstName,
		authData.ID,
		authData.LastName,
		authData.PhotoURL,
		authData.Username,
	)

	// Создаем HMAC ключ из токена бота
	secretKey := sha256.Sum256([]byte(s.telegramToken))

	// Вычисляем HMAC-SHA256 подпись
	h := hmac.New(sha256.New, secretKey[:])
	h.Write([]byte(dataCheckString))
	expectedHash := hex.EncodeToString(h.Sum(nil))

	// Сравниваем полученную подпись с ожидаемой
	return hmac.Equal([]byte(authData.Hash), []byte(expectedHash))
}

// checkChatMembership проверяет является ли пользователь членом закрытого чата
// Использует Telegram Bot API метод getChatMember
func (s *Service) checkChatMembership(userTelegramID int64) (bool, error) {
	log.Printf("[INFO] Проверка членства пользователя TelegramID=%d в чате %s", userTelegramID, s.chatID)

	// URL для запроса к Telegram Bot API
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getChatMember?chat_id=%s&user_id=%d",
		s.telegramToken, s.chatID, userTelegramID)

	// Выполняем GET запрос
	resp, err := s.httpClient.Get(url)
	if err != nil {
		log.Printf("[ERROR] Ошибка при запросе к Telegram Bot API: %v", err)
		return false, fmt.Errorf("не удалось проверить членство в чате: %w", err)
	}
	defer resp.Body.Close()

	// Парсим ответ от API
	var response TelegramChatMemberResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Printf("[ERROR] Ошибка декодирования ответа от Telegram Bot API: %v", err)
		return false, fmt.Errorf("ошибка обработки ответа Telegram API: %w", err)
	}

	// Проверяем успешность запроса
	if !response.OK {
		log.Printf("[WARN] Telegram Bot API вернул ошибку для пользователя TelegramID=%d", userTelegramID)
		// Если пользователь не найден в чате, считаем что он не является членом
		return false, nil
	}

	// Проверяем статус пользователя в чате
	// Валидные статусы: "creator", "administrator", "member"
	// Невалидные: "left", "kicked", "restricted"
	isMember := response.Result.Status == "creator" ||
		response.Result.Status == "administrator" ||
		response.Result.Status == "member"

	log.Printf("[INFO] Статус пользователя TelegramID=%d в чате: %s, член чата: %v",
		userTelegramID, response.Result.Status, isMember)

	return isMember, nil
}

// =====================================================
// УПРАВЛЕНИЕ ЗАЯВКАМИ
// =====================================================

// CreateOrder создает новую заявку на покупку или продажу криптовалюты
// Проверяет валидность данных и сохраняет заявку в базе данных
func (s *Service) CreateOrder(userID int64, orderData *model.Order) (*model.Order, error) {
	log.Printf("[INFO] Создание заявки пользователем ID=%d: Type=%s, Crypto=%s, Amount=%.8f",
		userID, orderData.Type, orderData.Cryptocurrency, orderData.Amount)

	// Проверяем права пользователя на создание заявки
	user, err := s.repo.GetUserByTelegramID(userID)
	if err != nil {
		log.Printf("[ERROR] Пользователь ID=%d не найден при создании заявки", userID)
		return nil, fmt.Errorf("пользователь не найден")
	}

	if !user.IsActive || !user.ChatMember {
		log.Printf("[WARN] Попытка создания заявки неактивным пользователем ID=%d", userID)
		return nil, fmt.Errorf("недостаточно прав для создания заявки")
	}

	// Валидируем данные заявки
	if err := s.validateOrderData(orderData); err != nil {
		log.Printf("[WARN] Невалидные данные заявки от пользователя ID=%d: %v", userID, err)
		return nil, err
	}

	// Заполняем системные поля заявки
	orderData.UserID = user.ID
	orderData.Status = model.OrderStatusActive
	orderData.IsActive = true
	orderData.TotalAmount = orderData.Amount * orderData.Price
	// Устанавливаем ExpiresAt в далекое будущее - таймеры больше не используются
	orderData.ExpiresAt = time.Now().Add(365 * 24 * time.Hour) // 1 год

	// Если не указан минимальный и максимальный лимит, устанавливаем их равными общей сумме
	if orderData.MinAmount == 0 {
		orderData.MinAmount = orderData.TotalAmount
	}
	if orderData.MaxAmount == 0 {
		orderData.MaxAmount = orderData.TotalAmount
	}

	// Сохраняем заявку в базе данных
	if err := s.repo.CreateOrder(orderData); err != nil {
		log.Printf("[ERROR] Не удалось сохранить заявку пользователя ID=%d: %v", userID, err)
		return nil, fmt.Errorf("не удалось создать заявку: %w", err)
	}

	// Автоматическое сопоставление больше не используется
	// В новой логике пользователи создают отклики, а авторы их принимают

	log.Printf("[INFO] Успешно создана заявка: ID=%d, UserID=%d, Type=%s",
		orderData.ID, userID, orderData.Type)

	return orderData, nil
}

// GetOrders получает список заявок с фильтрацией и пагинацией
// Позволяет найти подходящие заявки для пользователя
func (s *Service) GetOrders(filter *model.OrderFilter) ([]*model.Order, error) {
	log.Printf("[INFO] Поиск заявок с фильтром: Type=%v, Crypto=%v, Status=%v",
		filter.Type, filter.Cryptocurrency, filter.Status)

	// Устанавливаем значения по умолчанию для пагинации
	if filter.Limit == 0 {
		filter.Limit = 50 // По умолчанию показываем 50 заявок
	}
	if filter.Limit > 100 {
		filter.Limit = 100 // Максимум 100 заявок за запрос
	}

	// Получаем заявки из репозитория
	orders, err := s.repo.GetOrdersByFilter(filter)
	if err != nil {
		log.Printf("[ERROR] Ошибка при поиске заявок: %v", err)
		return nil, fmt.Errorf("не удалось получить заявки: %w", err)
	}

	// Обогащаем заявки данными пользователей для отображения на фронтенде
	for _, order := range orders {
		if user, err := s.repo.GetUserByID(order.UserID); err == nil {
			// Добавляем данные пользователя к заявке (создаем дополнительные поля)
			// Эти поля не входят в основную модель Order, но нужны для фронтенда
			order.UserName = user.FirstName
			if user.LastName != "" {
				order.UserName += " " + user.LastName
			}
			order.Username = user.Username
			order.FirstName = user.FirstName
			order.LastName = user.LastName

			log.Printf("[DEBUG] Обогащена заявка ID=%d данными пользователя: Name=%s, Username=%s",
				order.ID, order.UserName, order.Username)
		} else {
			log.Printf("[WARN] Не удалось получить данные пользователя ID=%d для заявки ID=%d: %v",
				order.UserID, order.ID, err)
		}
	}

	log.Printf("[INFO] Найдено заявок: %d", len(orders))
	return orders, nil
}

// GetOrder получает заявку по ID
func (s *Service) GetOrder(orderID int64) (*model.Order, error) {
	log.Printf("[INFO] Получение заявки по ID=%d", orderID)

	// Получаем все заявки и ищем нужную (пока нет отдельного метода GetOrderByID)
	filter := &model.OrderFilter{
		Limit:  1000, // Получаем больше заявок для поиска
		Offset: 0,
	}

	orders, err := s.repo.GetOrdersByFilter(filter)
	if err != nil {
		log.Printf("[ERROR] Ошибка при поиске заявки ID=%d: %v", orderID, err)
		return nil, fmt.Errorf("ошибка поиска заявки")
	}

	// Ищем заявку с нужным ID
	for _, order := range orders {
		if order.ID == orderID {
			log.Printf("[INFO] Заявка найдена: ID=%d, Type=%s", order.ID, order.Type)
			return order, nil
		}
	}

	log.Printf("[WARN] Заявка ID=%d не найдена", orderID)
	return nil, fmt.Errorf("заявка с ID=%d не найдена", orderID)
}

// CancelOrder отменяет активную заявку пользователя
// Только создатель заявки может ее отменить
func (s *Service) CancelOrder(userID, orderID int64) error {
	log.Printf("[INFO] Отмена заявки ID=%d пользователем ID=%d", orderID, userID)

	// TODO: Добавить проверку прав пользователя на отмену заявки
	// Нужно проверить, что заявка принадлежит пользователю и имеет статус "active"

	// Обновляем статус заявки на "cancelled"
	err := s.repo.UpdateOrderStatus(orderID, model.OrderStatusCancelled)
	if err != nil {
		log.Printf("[ERROR] Не удалось отменить заявку ID=%d: %v", orderID, err)
		return fmt.Errorf("не удалось отменить заявку: %w", err)
	}

	log.Printf("[INFO] Заявка ID=%d успешно отменена", orderID)
	return nil
}

// =====================================================
// ВСПОМОГАТЕЛЬНЫЕ МЕТОДЫ
// =====================================================

// validateOrderData проверяет корректность данных заявки
func (s *Service) validateOrderData(order *model.Order) error {
	// Проверяем тип заявки
	if order.Type != model.OrderTypeBuy && order.Type != model.OrderTypeSell {
		return fmt.Errorf("неверный тип заявки: %s", order.Type)
	}

	// Проверяем криптовалюту
	supportedCryptos := []string{"BTC", "ETH", "USDT", "USDC", "LTC"}
	isValidCrypto := false
	for _, crypto := range supportedCryptos {
		if order.Cryptocurrency == crypto {
			isValidCrypto = true
			break
		}
	}
	if !isValidCrypto {
		return fmt.Errorf("неподдерживаемая криптовалюта: %s", order.Cryptocurrency)
	}

	// Проверяем фиатную валюту
	supportedFiats := []string{"RUB", "USD", "EUR", "UAH"}
	isValidFiat := false
	for _, fiat := range supportedFiats {
		if order.FiatCurrency == fiat {
			isValidFiat = true
			break
		}
	}
	if !isValidFiat {
		return fmt.Errorf("неподдерживаемая фиатная валюта: %s", order.FiatCurrency)
	}

	// Проверяем количество и цену
	if order.Amount <= 0 {
		return fmt.Errorf("количество должно быть больше нуля")
	}
	if order.Price <= 0 {
		return fmt.Errorf("цена должна быть больше нуля")
	}

	// Проверяем лимиты
	if order.MinAmount < 0 {
		return fmt.Errorf("минимальная сумма не может быть отрицательной")
	}
	if order.MaxAmount > 0 && order.MaxAmount < order.MinAmount {
		return fmt.Errorf("максимальная сумма не может быть меньше минимальной")
	}

	// Проверяем способы оплаты
	if len(order.PaymentMethods) == 0 {
		return fmt.Errorf("необходимо указать хотя бы один способ оплаты")
	}

	supportedPayments := []string{"bank_transfer", "sberbank", "tinkoff", "qiwi", "yandex_money", "cash"}
	for _, method := range order.PaymentMethods {
		isValidPayment := false
		for _, supported := range supportedPayments {
			if method == supported {
				isValidPayment = true
				break
			}
		}
		if !isValidPayment {
			return fmt.Errorf("неподдерживаемый способ оплаты: %s", method)
		}
	}

	return nil
}

// HealthCheck проверяет состояние сервиса и его зависимостей
func (s *Service) HealthCheck() error {
	// Проверяем соединение с базой данных
	if err := s.repo.HealthCheck(); err != nil {
		return fmt.Errorf("сервис недоступен: %w", err)
	}

	return nil
}

// =====================================================
// ЛОГИКА P2P ТОРГОВЛИ
// =====================================================

// tryAutoMatchOrder пытается автоматически сопоставить заявку с подходящими
// Выполняется в отдельной горутине для неблокирующей работы
func (s *Service) tryAutoMatchOrder(order *model.Order) {
	log.Printf("[INFO] Попытка автоматического сопоставления заявки ID=%d", order.ID)

	// Ищем подходящие заявки
	matchingOrders, err := s.repo.GetMatchingOrders(order)
	if err != nil {
		log.Printf("[ERROR] Не удалось найти подходящие заявки для Order ID=%d: %v", order.ID, err)
		return
	}

	if len(matchingOrders) == 0 {
		log.Printf("[INFO] Подходящие заявки для Order ID=%d не найдены", order.ID)
		return
	}

	// Пытаемся сопоставить с первой подходящей заявкой
	matchingOrder := matchingOrders[0]

	// Проверяем совместимость цен
	if !s.isPriceCompatible(order, matchingOrder) {
		log.Printf("[INFO] Цены заявок ID=%d и ID=%d несовместимы", order.ID, matchingOrder.ID)
		return
	}

	// Проверяем совместимость лимитов
	if !s.isAmountCompatible(order, matchingOrder) {
		log.Printf("[INFO] Лимиты заявок ID=%d и ID=%d несовместимы", order.ID, matchingOrder.ID)
		return
	}

	// Проверяем совместимость способов оплаты
	if !s.hasCommonPaymentMethods(order, matchingOrder) {
		log.Printf("[INFO] У заявок ID=%d и ID=%d нет общих способов оплаты", order.ID, matchingOrder.ID)
		return
	}

	// Сопоставляем заявки
	if err := s.repo.MatchOrders(order.ID, matchingOrder.ID); err != nil {
		log.Printf("[ERROR] Не удалось сопоставить заявки ID=%d и ID=%d: %v", order.ID, matchingOrder.ID, err)
		return
	}

	// Автосопоставление отключено в новой логике откликов
	log.Printf("[INFO] Автосопоставление отключено, используйте систему откликов")

	// TODO: Отправить уведомления участникам через Telegram бота
	// s.notifyUsersAboutDeal(deal)
}

// isPriceCompatible проверяет совместимость цен двух заявок
func (s *Service) isPriceCompatible(order1, order2 *model.Order) bool {
	// Для покупки: цена покупателя должна быть >= цены продавца
	// Для продажи: цена продавца должна быть <= цены покупателя

	if order1.Type == model.OrderTypeBuy && order2.Type == model.OrderTypeSell {
		return order1.Price >= order2.Price // Покупатель готов платить >= чем просит продавец
	}

	if order1.Type == model.OrderTypeSell && order2.Type == model.OrderTypeBuy {
		return order1.Price <= order2.Price // Продавец готов продать <= чем готов платить покупатель
	}

	return false
}

// isAmountCompatible проверяет совместимость сумм двух заявок
func (s *Service) isAmountCompatible(order1, order2 *model.Order) bool {
	// Проверяем что суммы заявок пересекаются в допустимых диапазонах

	// Общее количество не должно превышать минимум из двух заявок
	minAmount := order1.Amount
	if order2.Amount < minAmount {
		minAmount = order2.Amount
	}

	// Проверяем минимальные лимиты
	if order1.MinAmount > 0 && minAmount*order1.Price < order1.MinAmount {
		return false
	}

	if order2.MinAmount > 0 && minAmount*order2.Price < order2.MinAmount {
		return false
	}

	// Проверяем максимальные лимиты
	if order1.MaxAmount > 0 && minAmount*order1.Price > order1.MaxAmount {
		return false
	}

	if order2.MaxAmount > 0 && minAmount*order2.Price > order2.MaxAmount {
		return false
	}

	return true
}

// hasCommonPaymentMethods проверяет есть ли общие способы оплаты у двух заявок
func (s *Service) hasCommonPaymentMethods(order1, order2 *model.Order) bool {
	// Создаем карту способов оплаты первой заявки для быстрого поиска
	methods1 := make(map[string]bool)
	for _, method := range order1.PaymentMethods {
		methods1[method] = true
	}

	// Проверяем есть ли пересечения с методами второй заявки
	for _, method := range order2.PaymentMethods {
		if methods1[method] {
			return true // Найден общий способ оплаты
		}
	}

	return false
}

// УДАЛЕНО: createDealFromOrders - устаревший метод автосопоставления
// В новой логике откликов автоматическое сопоставление не используется

// =====================================================
// УПРАВЛЕНИЕ СДЕЛКАМИ
// =====================================================

// GetUserDeals получает все сделки пользователя
func (s *Service) GetUserDeals(userID int64) ([]*model.Deal, error) {
	log.Printf("[INFO] Получение сделок для пользователя ID=%d", userID)

	deals, err := s.repo.GetDealsByUserID(userID)
	if err != nil {
		log.Printf("[ERROR] Не удалось получить сделки пользователя ID=%d: %v", userID, err)
		return nil, fmt.Errorf("не удалось получить сделки: %w", err)
	}

	// Обогащаем сделки данными пользователей для отображения на фронтенде
	for _, deal := range deals {
		// Добавляем данные автора заявки
		if author, err := s.repo.GetUserByID(deal.AuthorID); err == nil {
			deal.AuthorName = author.FirstName
			if author.LastName != "" {
				deal.AuthorName += " " + author.LastName
			}
			deal.AuthorUsername = author.Username

			log.Printf("[DEBUG] Обогащена сделка ID=%d данными автора: Name=%s, Username=%s",
				deal.ID, deal.AuthorName, deal.AuthorUsername)
		} else {
			log.Printf("[WARN] Не удалось получить данные автора ID=%d для сделки ID=%d: %v",
				deal.AuthorID, deal.ID, err)
		}

		// Добавляем данные контрагента
		if counterparty, err := s.repo.GetUserByID(deal.CounterpartyID); err == nil {
			deal.CounterpartyName = counterparty.FirstName
			if counterparty.LastName != "" {
				deal.CounterpartyName += " " + counterparty.LastName
			}
			deal.CounterpartyUsername = counterparty.Username

			log.Printf("[DEBUG] Обогащена сделка ID=%d данными контрагента: Name=%s, Username=%s",
				deal.ID, deal.CounterpartyName, deal.CounterpartyUsername)
		} else {
			log.Printf("[WARN] Не удалось получить данные контрагента ID=%d для сделки ID=%d: %v",
				deal.CounterpartyID, deal.ID, err)
		}
	}

	log.Printf("[INFO] Найдено сделок для пользователя ID=%d: %d", userID, len(deals))
	return deals, nil
}

// GetDeal получает сделку по ID с проверкой прав доступа
func (s *Service) GetDeal(dealID, userID int64) (*model.Deal, error) {
	log.Printf("[INFO] Получение сделки ID=%d пользователем ID=%d", dealID, userID)

	deal, err := s.repo.GetDealByID(dealID)
	if err != nil {
		log.Printf("[ERROR] Не удалось получить сделку ID=%d: %v", dealID, err)
		return nil, fmt.Errorf("сделка не найдена")
	}

	// Проверяем что пользователь участвует в сделке
	if deal.AuthorID != userID && deal.CounterpartyID != userID {
		log.Printf("[WARN] Пользователь ID=%d пытается получить доступ к чужой сделке ID=%d", userID, dealID)
		return nil, fmt.Errorf("доступ запрещен: вы не участвуете в данной сделке")
	}

	return deal, nil
}

// ConfirmDeal подтверждает сделку со стороны пользователя
func (s *Service) ConfirmDeal(dealID, userID int64, paymentProof string) error {
	log.Printf("[INFO] Подтверждение сделки ID=%d пользователем ID=%d", dealID, userID)

	// Проверяем доступ к сделке
	deal, err := s.GetDeal(dealID, userID)
	if err != nil {
		return err
	}

	// Проверяем что сделка в подходящем статусе
	if deal.Status != model.DealStatusInProgress && deal.Status != model.DealStatusWaitingConfirmation {
		return fmt.Errorf("сделка в статусе '%s' не может быть подтверждена", deal.Status)
	}

	// В новой логике доказательства могут предоставлять обе стороны
	var isPaymentProof bool
	if paymentProof != "" {
		isPaymentProof = true
	}

	// Подтверждаем сделку
	if err := s.repo.ConfirmDeal(dealID, userID, isPaymentProof, paymentProof); err != nil {
		log.Printf("[ERROR] Не удалось подтвердить сделку ID=%d: %v", dealID, err)
		return fmt.Errorf("не удалось подтвердить сделку: %w", err)
	}

	log.Printf("[INFO] Сделка ID=%d подтверждена пользователем ID=%d", dealID, userID)
	return nil
}

// ConfirmDealWithRole подтверждает сделку со стороны пользователя с указанием роли
func (s *Service) ConfirmDealWithRole(dealID, userID int64, isAuthor bool, paymentProof string) error {
	log.Printf("[INFO] Подтверждение сделки ID=%d пользователем ID=%d (isAuthor=%v)", dealID, userID, isAuthor)

	// Проверяем доступ к сделке
	deal, err := s.GetDeal(dealID, userID)
	if err != nil {
		return err
	}

	// Проверяем что сделка в подходящем статусе
	if deal.Status != model.DealStatusInProgress && deal.Status != model.DealStatusWaitingConfirmation {
		return fmt.Errorf("сделка в статусе '%s' не может быть подтверждена", deal.Status)
	}

	// Проверяем соответствие роли
	if isAuthor && deal.AuthorID != userID {
		return fmt.Errorf("пользователь не является автором сделки")
	}
	if !isAuthor && deal.CounterpartyID != userID {
		return fmt.Errorf("пользователь не является контрагентом сделки")
	}

	// Подтверждаем сделку с указанием роли
	if err := s.repo.ConfirmDealWithRole(dealID, userID, isAuthor, paymentProof); err != nil {
		log.Printf("[ERROR] Не удалось подтвердить сделку ID=%d: %v", dealID, err)
		return fmt.Errorf("не удалось подтвердить сделку: %w", err)
	}

	// Получаем обновленную сделку для проверки статуса
	updatedDeal, err := s.repo.GetDealByID(dealID)
	if err != nil {
		log.Printf("[WARN] Не удалось получить обновленную сделку ID=%d: %v", dealID, err)
	} else {
		// Определяем кто подтвердил и кто ждет подтверждения
		var confirmedByUserID, waitingForUserID int64

		if isAuthor {
			confirmedByUserID = deal.AuthorID
			waitingForUserID = deal.CounterpartyID
		} else {
			confirmedByUserID = deal.CounterpartyID
			waitingForUserID = deal.AuthorID
		}

		// Проверяем завершена ли сделка полностью
		if updatedDeal.Status == model.DealStatusCompleted {
			// Сделка завершена - отправляем уведомления о завершении обеим сторонам
			go s.sendDealCompletedNotifications(updatedDeal)
		} else if updatedDeal.Status == model.DealStatusWaitingConfirmation {
			// Одна сторона подтвердила, вторая еще нет - отправляем уведомление ожидающему
			go s.sendDealConfirmedNotification(updatedDeal, confirmedByUserID, waitingForUserID)
		}
	}

	log.Printf("[INFO] Сделка ID=%d подтверждена пользователем ID=%d как %s", dealID, userID,
		map[bool]string{true: "автор", false: "контрагент"}[isAuthor])
	return nil
}

// =====================================================
// СИСТЕМА ОТЗЫВОВ И РЕЙТИНГОВ
// =====================================================

// CreateReview создает новый отзыв после завершения сделки
func (s *Service) CreateReview(userID int64, reviewData *model.CreateReviewRequest) (*model.Review, error) {
	log.Printf("[INFO] Создание отзыва от пользователя ID=%d для сделки ID=%d", userID, reviewData.DealID)

	// Проверяем права на создание отзыва
	canReview, err := s.repo.CheckCanReview(reviewData.DealID, userID, reviewData.ToUserID)
	if err != nil {
		log.Printf("[WARN] Пользователь ID=%d не может оставить отзыв: %v", userID, err)
		return nil, err
	}

	if !canReview {
		return nil, fmt.Errorf("отзыв не может быть создан")
	}

	// Валидируем данные отзыва
	if err := s.validateReviewData(reviewData); err != nil {
		log.Printf("[WARN] Невалидные данные отзыва от пользователя ID=%d: %v", userID, err)
		return nil, err
	}

	// Создаем отзыв
	review := &model.Review{
		DealID:      reviewData.DealID,
		FromUserID:  userID,
		ToUserID:    reviewData.ToUserID,
		Rating:      reviewData.Rating,
		Comment:     reviewData.Comment,
		IsAnonymous: reviewData.IsAnonymous,
		IsVisible:   true,
	}

	// Сохраняем отзыв в базе данных
	if err := s.repo.CreateReview(review); err != nil {
		log.Printf("[ERROR] Не удалось создать отзыв: %v", err)
		return nil, fmt.Errorf("не удалось создать отзыв: %w", err)
	}

	log.Printf("[INFO] Отзыв создан успешно: ID=%d, Rating=%d", review.ID, review.Rating)
	return review, nil
}

// GetUserReviews получает отзывы о пользователе с пагинацией
func (s *Service) GetUserReviews(userID int64, limit, offset int) ([]*model.Review, error) {
	log.Printf("[INFO] Получение отзывов для пользователя ID=%d (limit=%d, offset=%d)", userID, limit, offset)

	// Устанавливаем разумные лимиты
	if limit <= 0 || limit > 50 {
		limit = 20 // По умолчанию 20 отзывов
	}
	if offset < 0 {
		offset = 0
	}

	// Получаем отзывы из репозитория
	reviews, err := s.repo.GetReviewsByUserID(userID, limit, offset)
	if err != nil {
		log.Printf("[ERROR] Не удалось получить отзывы для пользователя ID=%d: %v", userID, err)
		return nil, fmt.Errorf("не удалось получить отзывы: %w", err)
	}

	log.Printf("[INFO] Получено отзывов: %d", len(reviews))
	return reviews, nil
}

// GetUserRating получает рейтинг пользователя
func (s *Service) GetUserRating(userID int64) (*model.Rating, error) {
	log.Printf("[INFO] Получение рейтинга пользователя ID=%d", userID)

	rating, err := s.repo.GetUserRating(userID)
	if err != nil {
		log.Printf("[ERROR] Не удалось получить рейтинг пользователя ID=%d: %v", userID, err)
		return nil, fmt.Errorf("не удалось получить рейтинг: %w", err)
	}

	return rating, nil
}

// GetUserProfile получает полный профиль пользователя включая рейтинг и статистику
func (s *Service) GetUserProfile(userID int64) (*model.ReviewStats, error) {
	log.Printf("[INFO] Получение профиля пользователя ID=%d", userID)

	stats, err := s.repo.GetUserReviewStats(userID)
	if err != nil {
		log.Printf("[ERROR] Не удалось получить статистику пользователя ID=%d: %v", userID, err)
		return nil, fmt.Errorf("не удалось получить профиль пользователя: %w", err)
	}

	return stats, nil
}

// GetFullUserProfile получает полную информацию о пользователе включая данные профиля и статистику отзывов
func (s *Service) GetFullUserProfile(userID int64) (*model.FullUserProfile, error) {
	log.Printf("[INFO] Получение полного профиля пользователя ID=%d", userID)

	// Получаем данные пользователя
	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		log.Printf("[ERROR] Пользователь ID=%d не найден: %v", userID, err)
		return nil, fmt.Errorf("пользователь не найден")
	}

	// Получаем статистику отзывов
	stats, err := s.repo.GetUserReviewStats(userID)
	if err != nil {
		log.Printf("[ERROR] Не удалось получить статистику пользователя ID=%d: %v", userID, err)
		return nil, fmt.Errorf("не удалось получить статистику пользователя: %w", err)
	}

	// Собираем полный профиль
	userProfile := &model.FullUserProfile{
		User:  user,
		Stats: stats,
	}

	log.Printf("[INFO] Полный профиль пользователя ID=%d получен успешно", userID)
	return userProfile, nil
}

// GetUserStats получает подробную статистику пользователя
func (s *Service) GetUserStats(userID int64) (*model.UserStats, error) {
	log.Printf("[INFO] Получение статистики пользователя ID=%d", userID)

	// Получаем статистику отзывов
	reviewStats, err := s.repo.GetUserReviewStats(userID)
	if err != nil {
		log.Printf("[ERROR] Не удалось получить статистику отзывов пользователя ID=%d: %v", userID, err)
		return nil, fmt.Errorf("не удалось получить статистику отзывов: %w", err)
	}

	// Получаем заявки пользователя для подсчета статистики
	orderFilter := &model.OrderFilter{
		UserID: &userID,
		Limit:  1000, // Получаем все заявки
		Offset: 0,
	}

	log.Printf("[DEBUG] Поиск заявок пользователя ID=%d с фильтром: %+v", userID, orderFilter)
	orders, err := s.repo.GetOrdersByFilter(orderFilter)
	if err != nil {
		log.Printf("[ERROR] Не удалось получить заявки пользователя ID=%d: %v", userID, err)
		return nil, fmt.Errorf("не удалось получить заявки: %w", err)
	}
	log.Printf("[DEBUG] Найдено заявок для пользователя ID=%d: %d", userID, len(orders))

	// Получаем сделки пользователя
	deals, err := s.repo.GetDealsByUserID(userID)
	if err != nil {
		log.Printf("[ERROR] Не удалось получить сделки пользователя ID=%d: %v", userID, err)
		return nil, fmt.Errorf("не удалось получить сделки: %w", err)
	}

	// Подсчитываем статистику заявок
	totalOrders := len(orders)
	activeOrders := 0
	completedOrders := 0

	for _, order := range orders {
		switch order.Status {
		case model.OrderStatusActive:
			activeOrders++
		case model.OrderStatusCompleted:
			completedOrders++
		}
	}

	// Подсчитываем статистику сделок
	totalDeals := len(deals)
	completedDeals := 0
	cancelledDeals := 0
	totalVolume := float64(0)
	var firstDealDate *time.Time
	var lastActivityDate *time.Time

	for _, deal := range deals {
		// Обновляем даты
		if firstDealDate == nil || deal.CreatedAt.Before(*firstDealDate) {
			firstDealDate = &deal.CreatedAt
		}
		if lastActivityDate == nil || deal.CreatedAt.After(*lastActivityDate) {
			lastActivityDate = &deal.CreatedAt
		}

		// Подсчитываем статистику по статусам
		switch deal.Status {
		case "completed":
			completedDeals++
			totalVolume += deal.TotalAmount
		case "cancelled":
			cancelledDeals++
		}
	}

	// Вычисляем процент успешных сделок
	successRate := float32(0)
	if totalDeals > 0 {
		successRate = float32(completedDeals) / float32(totalDeals) * 100
	}

	// Средняя продолжительность сделки (упрощенно)
	avgDealTime := 0
	if completedDeals > 0 {
		avgDealTime = 60 // Пока заглушка - 60 минут
	}

	// Собираем статистику
	stats := &model.UserStats{
		UserID:           userID,
		TotalOrders:      totalOrders,
		ActiveOrders:     activeOrders,
		CompletedOrders:  completedOrders,
		TotalDeals:       totalDeals,
		CompletedDeals:   completedDeals,
		CancelledDeals:   cancelledDeals,
		TotalTradeVolume: totalVolume,
		AvgDealTime:      avgDealTime,
		FirstDealDate:    firstDealDate,
		LastActivityDate: lastActivityDate,
		SuccessRate:      successRate,
		AverageRating:    reviewStats.AverageRating,
		TotalReviews:     reviewStats.TotalReviews,
	}

	log.Printf("[INFO] Статистика пользователя ID=%d собрана: %d заявок, %d сделок", userID, totalOrders, totalDeals)
	return stats, nil
}

// ReportReview создает жалобу на неподходящий отзыв
func (s *Service) ReportReview(userID, reviewID int64, reason, comment string) error {
	log.Printf("[INFO] Жалоба на отзыв ID=%d от пользователя ID=%d", reviewID, userID)

	// Валидируем данные жалобы
	if reason == "" {
		return fmt.Errorf("необходимо указать причину жалобы")
	}

	supportedReasons := []string{
		"spam",
		"inappropriate_language",
		"fake_review",
		"personal_attack",
		"irrelevant_content",
		"other",
	}

	isValidReason := false
	for _, validReason := range supportedReasons {
		if reason == validReason {
			isValidReason = true
			break
		}
	}

	if !isValidReason {
		return fmt.Errorf("неподдерживаемая причина жалобы")
	}

	// Создаем жалобу
	report := &model.ReviewReport{
		ReviewID: reviewID,
		UserID:   userID,
		Reason:   reason,
		Comment:  comment,
		Status:   "pending",
	}

	// Сохраняем жалобу в базе данных
	if err := s.repo.ReportReview(report); err != nil {
		log.Printf("[ERROR] Не удалось создать жалобу: %v", err)
		return fmt.Errorf("не удалось создать жалобу: %w", err)
	}

	log.Printf("[INFO] Жалоба создана успешно: ID=%d", report.ID)
	return nil
}

// validateReviewData валидирует данные отзыва
func (s *Service) validateReviewData(review *model.CreateReviewRequest) error {
	// Проверяем рейтинг
	if review.Rating < 1 || review.Rating > 5 {
		return fmt.Errorf("рейтинг должен быть от 1 до 5 звезд")
	}

	// Проверяем длину комментария
	if len(review.Comment) > 500 {
		return fmt.Errorf("комментарий не должен превышать 500 символов")
	}

	// Проверяем что комментарий не пустой для низких оценок
	if review.Rating <= 2 && strings.TrimSpace(review.Comment) == "" {
		return fmt.Errorf("для оценки 1-2 звезды необходимо указать комментарий")
	}

	return nil
}

// УДАЛЕНО: CreateDealFromOrder - устаревший метод
// В новой логике сделки создаются только через AcceptResponse после принятия отклика автором заявки

// =====================================================
// МЕТОДЫ ДЛЯ РАБОТЫ С ОТКЛИКАМИ
// =====================================================

// CreateResponse создает новый отклик на заявку
func (s *Service) CreateResponse(userID int64, responseData *model.CreateResponseRequest) (*model.Response, error) {
	log.Printf("[INFO] Создание отклика пользователем ID=%d на заявку ID=%d", userID, responseData.OrderID)

	// Проверяем что заявка существует и активна
	order, err := s.GetOrder(responseData.OrderID)
	if err != nil {
		log.Printf("[ERROR] Заявка не найдена: %v", err)
		return nil, fmt.Errorf("заявка не найдена")
	}

	// Проверяем что это не своя заявка
	if order.UserID == userID {
		log.Printf("[WARN] Пользователь ID=%d пытается откликнуться на свою заявку", userID)
		return nil, fmt.Errorf("нельзя откликаться на собственную заявку")
	}

	// Проверяем статус заявки
	if order.Status != model.OrderStatusActive {
		log.Printf("[WARN] Заявка ID=%d имеет статус %s, нельзя откликаться", responseData.OrderID, order.Status)
		return nil, fmt.Errorf("заявка недоступна для откликов")
	}

	// Убираем проверку срока истечения - таймеры больше не используются

	// Создаем отклик
	response := &model.Response{
		OrderID: responseData.OrderID,
		UserID:  userID,
		Message: responseData.Message,
	}

	// Сохраняем отклик в репозитории
	if err := s.repo.CreateResponse(response); err != nil {
		log.Printf("[ERROR] Не удалось создать отклик: %v", err)
		return nil, fmt.Errorf("не удалось создать отклик: %w", err)
	}

	// Отправляем уведомление автору заявки о новом отклике
	go s.sendNewResponseNotification(order, response, userID)

	log.Printf("[INFO] Отклик создан успешно: ID=%d", response.ID)
	return response, nil
}

// GetMyResponses получает отклики пользователя (которые он оставлял)
func (s *Service) GetMyResponses(userID int64) ([]*model.Response, error) {
	log.Printf("[INFO] Получение откликов пользователя ID=%d", userID)

	responses, err := s.repo.GetResponsesFromUser(userID)
	if err != nil {
		log.Printf("[ERROR] Не удалось получить отклики пользователя: %v", err)
		return nil, fmt.Errorf("не удалось получить отклики: %w", err)
	}

	// Обогащаем отклики данными пользователей и заявок
	for _, response := range responses {
		// Добавляем данные откликнувшегося пользователя
		if user, err := s.repo.GetUserByID(response.UserID); err == nil {
			response.UserName = user.FirstName
			if user.LastName != "" {
				response.UserName += " " + user.LastName
			}
			response.Username = user.Username
		}

		// Получаем данные заявки и её автора
		if order, err := s.GetOrder(response.OrderID); err == nil {
			response.OrderType = string(order.Type)
			response.Cryptocurrency = order.Cryptocurrency
			response.FiatCurrency = order.FiatCurrency
			response.Amount = order.Amount
			response.Price = order.Price
			response.TotalAmount = order.TotalAmount

			// Добавляем данные автора заявки
			if author, err := s.repo.GetUserByID(order.UserID); err == nil {
				response.AuthorName = author.FirstName
				if author.LastName != "" {
					response.AuthorName += " " + author.LastName
				}
				response.AuthorUsername = author.Username

				log.Printf("[DEBUG] Обогащен отклик ID=%d: User=%s(@%s) -> Author=%s(@%s), Order=%s %s",
					response.ID, response.UserName, response.Username,
					response.AuthorName, response.AuthorUsername,
					response.OrderType, response.Cryptocurrency)
			}
		}
	}

	log.Printf("[INFO] Найдено откликов пользователя ID=%d: %d", userID, len(responses))
	return responses, nil
}

// GetResponsesToMyOrders получает отклики на заявки пользователя (которые ему оставляли)
func (s *Service) GetResponsesToMyOrders(authorID int64) ([]*model.Response, error) {
	log.Printf("[INFO] Получение откликов на заявки автора ID=%d", authorID)

	responses, err := s.repo.GetResponsesForAuthor(authorID)
	if err != nil {
		log.Printf("[ERROR] Не удалось получить отклики на заявки: %v", err)
		return nil, fmt.Errorf("не удалось получить отклики: %w", err)
	}

	// Обогащаем отклики данными пользователей и заявок
	for _, response := range responses {
		// Добавляем данные откликнувшегося пользователя
		if user, err := s.repo.GetUserByID(response.UserID); err == nil {
			response.UserName = user.FirstName
			if user.LastName != "" {
				response.UserName += " " + user.LastName
			}
			response.Username = user.Username
		}

		// Получаем данные заявки и её автора (это сам authorID)
		if order, err := s.GetOrder(response.OrderID); err == nil {
			response.OrderType = string(order.Type)
			response.Cryptocurrency = order.Cryptocurrency
			response.FiatCurrency = order.FiatCurrency
			response.Amount = order.Amount
			response.Price = order.Price
			response.TotalAmount = order.TotalAmount

			// Добавляем данные автора заявки (сам authorID)
			if author, err := s.repo.GetUserByID(authorID); err == nil {
				response.AuthorName = author.FirstName
				if author.LastName != "" {
					response.AuthorName += " " + author.LastName
				}
				response.AuthorUsername = author.Username

				log.Printf("[DEBUG] Обогащен отклик на заявку ID=%d: %s(@%s) -> Author=%s(@%s), Order=%s %s",
					response.ID, response.UserName, response.Username,
					response.AuthorName, response.AuthorUsername,
					response.OrderType, response.Cryptocurrency)
			}
		}
	}

	log.Printf("[INFO] Найдено откликов на заявки автора ID=%d: %d", authorID, len(responses))
	return responses, nil
}

// AcceptResponse принимает отклик и создает сделку
func (s *Service) AcceptResponse(responseID, authorID int64) (*model.Deal, error) {
	log.Printf("[INFO] Принятие отклика ID=%d автором ID=%d", responseID, authorID)

	// Получаем отклик через фильтр
	filter := &model.ResponseFilter{Limit: 100}
	responses, err := s.repo.GetResponsesByFilter(filter)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить отклик: %w", err)
	}

	var response *model.Response
	for _, r := range responses {
		if r.ID == responseID {
			response = r
			break
		}
	}

	if response == nil {
		return nil, fmt.Errorf("отклик не найден")
	}

	// Получаем заявку
	order, err := s.GetOrder(response.OrderID)
	if err != nil {
		return nil, fmt.Errorf("заявка не найдена: %w", err)
	}

	// Проверяем что автор заявки принимает отклик
	if order.UserID != authorID {
		return nil, fmt.Errorf("только автор заявки может принимать отклики")
	}

	// Проверяем статус отклика
	if response.Status != model.ResponseStatusWaiting {
		return nil, fmt.Errorf("отклик уже был рассмотрен")
	}

	// Принимаем отклик
	if err := s.repo.UpdateResponseStatus(responseID, model.ResponseStatusAccepted); err != nil {
		return nil, fmt.Errorf("не удалось принять отклик: %w", err)
	}

	// Создаем сделку
	deal := &model.Deal{
		ResponseID:     responseID,
		OrderID:        order.ID,
		AuthorID:       authorID,
		CounterpartyID: response.UserID,
		Cryptocurrency: order.Cryptocurrency,
		FiatCurrency:   order.FiatCurrency,
		Amount:         order.Amount,
		Price:          order.Price,
		TotalAmount:    order.TotalAmount,
		PaymentMethods: order.PaymentMethods,
		OrderType:      order.Type,
		Status:         model.DealStatusInProgress,
		// Убираем ExpiresAt - таймеры больше не используются
	}

	if err := s.repo.CreateDeal(deal); err != nil {
		return nil, fmt.Errorf("не удалось создать сделку: %w", err)
	}

	// Обновляем статус заявки на "в сделке"
	if err := s.repo.UpdateOrderStatus(order.ID, model.OrderStatusInDeal); err != nil {
		log.Printf("[WARN] Не удалось обновить статус заявки: %v", err)
	}

	// Отклоняем все остальные отклики на эту заявку
	s.rejectOtherResponses(order.ID, responseID)

	// Отправляем уведомления участникам
	go s.sendResponseAcceptedNotification(order, response, deal)
	go s.sendDealCreatedNotifications(deal)

	log.Printf("[INFO] Отклик принят, создана сделка ID=%d", deal.ID)
	return deal, nil
}

// RejectResponse отклоняет отклик
func (s *Service) RejectResponse(responseID, authorID int64, reason string) error {
	log.Printf("[INFO] Отклонение отклика ID=%d автором ID=%d", responseID, authorID)

	// Получаем отклик для отправки уведомления
	filter := &model.ResponseFilter{Limit: 100}
	responses, err := s.repo.GetResponsesByFilter(filter)
	if err != nil {
		return fmt.Errorf("не удалось получить отклик: %w", err)
	}

	var response *model.Response
	for _, r := range responses {
		if r.ID == responseID {
			response = r
			break
		}
	}

	if response == nil {
		return fmt.Errorf("отклик не найден")
	}

	// Получаем заявку
	order, err := s.GetOrder(response.OrderID)
	if err != nil {
		return fmt.Errorf("заявка не найдена: %w", err)
	}

	// Проверяем что автор заявки отклоняет отклик
	if order.UserID != authorID {
		return fmt.Errorf("только автор заявки может отклонять отклики")
	}

	// Проверяем статус отклика
	if response.Status != model.ResponseStatusWaiting {
		return fmt.Errorf("отклик уже был рассмотрен")
	}

	// Отклоняем отклик
	if err := s.repo.UpdateResponseStatus(responseID, model.ResponseStatusRejected); err != nil {
		return fmt.Errorf("не удалось отклонить отклик: %w", err)
	}

	// Отправляем уведомление пользователю об отклонении его отклика
	go s.sendResponseRejectedNotification(order, response)

	log.Printf("[INFO] Отклик ID=%d отклонен", responseID)
	return nil
}

// rejectOtherResponses отклоняет все остальные отклики на заявку кроме принятого
func (s *Service) rejectOtherResponses(orderID, acceptedResponseID int64) {
	responses, err := s.repo.GetResponsesForOrder(orderID)
	if err != nil {
		log.Printf("[WARN] Не удалось получить отклики для отклонения: %v", err)
		return
	}

	// Получаем заявку для отправки уведомлений
	order, err := s.GetOrder(orderID)
	if err != nil {
		log.Printf("[WARN] Не удалось получить заявку ID=%d для уведомлений об отклонении: %v", orderID, err)
	}

	for _, response := range responses {
		if response.ID != acceptedResponseID && response.Status == model.ResponseStatusWaiting {
			// Отклоняем отклик
			if err := s.repo.UpdateResponseStatus(response.ID, model.ResponseStatusRejected); err != nil {
				log.Printf("[WARN] Не удалось отклонить отклик ID=%d: %v", response.ID, err)
			} else {
				// Отправляем уведомление пользователю об отклонении его отклика (автоматически)
				if order != nil {
					go s.sendResponseRejectedNotification(order, response)
					log.Printf("[INFO] Отправлено уведомление об автоматическом отклонении отклика ID=%d", response.ID)
				}
			}
		}
	}
}

// =====================================================
// МЕТОДЫ ДЛЯ УВЕДОМЛЕНИЙ
// =====================================================

// sendNewResponseNotification отправляет уведомление автору заявки о новом отклике
func (s *Service) sendNewResponseNotification(order *model.Order, response *model.Response, responderUserID int64) {
	log.Printf("[INFO] Отправка уведомления о новом отклике автору заявки ID=%d", order.UserID)

	// Получаем данные автора заявки по внутреннему ID
	author, err := s.repo.GetUserByID(order.UserID)
	if err != nil {
		log.Printf("[ERROR] Не удалось найти автора заявки ID=%d: %v", order.UserID, err)
		return
	}

	// Получаем данные пользователя, который откликнулся
	responder, err := s.repo.GetUserByID(responderUserID)
	if err != nil {
		log.Printf("[ERROR] Не удалось найти откликнувшегося пользователя ID=%d: %v", responderUserID, err)
		return
	}

	// Формируем имя откликнувшегося пользователя
	responderName := responder.FirstName
	if responder.LastName != "" {
		responderName += " " + responder.LastName
	}
	if responder.Username != "" {
		responderName += " (@" + responder.Username + ")"
	}

	// Форматируем уведомление с использованием шаблона
	title, message := s.notificationService.FormatResponseNotification(order, response, responderName)

	// Создаем уведомление
	notificationReq := &model.CreateNotificationRequest{
		UserID:     author.ID,
		Type:       model.NotificationTypeNewResponse,
		Title:      title,
		Message:    message,
		OrderID:    &order.ID,
		ResponseID: &response.ID,
		Data: map[string]interface{}{
			"order_type":       string(order.Type),
			"cryptocurrency":   order.Cryptocurrency,
			"fiat_currency":    order.FiatCurrency,
			"amount":           order.Amount,
			"price":            order.Price,
			"total_amount":     order.TotalAmount,
			"responder_name":   responderName,
			"responder_id":     responderUserID,
			"response_message": response.Message,
		},
	}

	notification, err := s.notificationService.CreateNotification(notificationReq)
	if err != nil {
		log.Printf("[ERROR] Не удалось создать уведомление: %v", err)
		return
	}

	// Отправляем уведомление в Telegram
	if err := s.notificationService.SendNotification(notification, author.TelegramID); err != nil {
		log.Printf("[ERROR] Не удалось отправить уведомление: %v", err)
		return
	}

	log.Printf("[INFO] Уведомление о новом отклике отправлено автору TelegramID=%d", author.TelegramID)
}

// sendResponseAcceptedNotification отправляет уведомление участнику о принятом отклике
func (s *Service) sendResponseAcceptedNotification(order *model.Order, response *model.Response, deal *model.Deal) {
	log.Printf("[INFO] Отправка уведомления о принятии отклика пользователю ID=%d", response.UserID)

	// Получаем данные пользователя, которому отправляем уведомление
	responder, err := s.repo.GetUserByID(response.UserID)
	if err != nil {
		log.Printf("[ERROR] Не удалось найти пользователя ID=%d: %v", response.UserID, err)
		return
	}

	// Получаем данные автора заявки
	author, err := s.repo.GetUserByID(order.UserID)
	if err != nil {
		log.Printf("[ERROR] Не удалось найти автора заявки ID=%d: %v", order.UserID, err)
		return
	}

	// Формируем имя автора заявки
	authorName := author.FirstName
	if author.LastName != "" {
		authorName += " " + author.LastName
	}
	if author.Username != "" {
		authorName += " (@" + author.Username + ")"
	}

	// Форматируем уведомление
	title, message := s.notificationService.FormatAcceptedResponseNotification(order, authorName)

	// Создаем уведомление
	notificationReq := &model.CreateNotificationRequest{
		UserID:     responder.ID,
		Type:       model.NotificationTypeResponseAccepted,
		Title:      title,
		Message:    message,
		OrderID:    &order.ID,
		ResponseID: &response.ID,
		DealID:     &deal.ID,
		Data: map[string]interface{}{
			"order_type":     string(order.Type),
			"cryptocurrency": order.Cryptocurrency,
			"fiat_currency":  order.FiatCurrency,
			"amount":         order.Amount,
			"price":          order.Price,
			"total_amount":   order.TotalAmount,
			"author_name":    authorName,
			"deal_id":        deal.ID,
		},
	}

	notification, err := s.notificationService.CreateNotification(notificationReq)
	if err != nil {
		log.Printf("[ERROR] Не удалось создать уведомление: %v", err)
		return
	}

	// Отправляем уведомление в Telegram
	if err := s.notificationService.SendNotification(notification, responder.TelegramID); err != nil {
		log.Printf("[ERROR] Не удалось отправить уведомление: %v", err)
		return
	}

	log.Printf("[INFO] Уведомление о принятии отклика отправлено пользователю TelegramID=%d", responder.TelegramID)
}

// sendResponseRejectedNotification отправляет уведомление участнику об отклоненном отклике
func (s *Service) sendResponseRejectedNotification(order *model.Order, response *model.Response) {
	log.Printf("[INFO] Отправка уведомления об отклонении отклика пользователю ID=%d", response.UserID)

	// Получаем данные пользователя, которому отправляем уведомление
	responder, err := s.repo.GetUserByID(response.UserID)
	if err != nil {
		log.Printf("[ERROR] Не удалось найти пользователя ID=%d: %v", response.UserID, err)
		return
	}

	// Получаем данные автора заявки
	author, err := s.repo.GetUserByID(order.UserID)
	if err != nil {
		log.Printf("[ERROR] Не удалось найти автора заявки ID=%d: %v", order.UserID, err)
		return
	}

	// Формируем имя автора заявки
	authorName := author.FirstName
	if author.LastName != "" {
		authorName += " " + author.LastName
	}
	if author.Username != "" {
		authorName += " (@" + author.Username + ")"
	}

	// Форматируем уведомление
	title, message := s.notificationService.FormatRejectedResponseNotification(order, authorName)

	// Создаем уведомление
	notificationReq := &model.CreateNotificationRequest{
		UserID:     responder.ID,
		Type:       model.NotificationTypeResponseRejected,
		Title:      title,
		Message:    message,
		OrderID:    &order.ID,
		ResponseID: &response.ID,
		Data: map[string]interface{}{
			"order_type":       string(order.Type),
			"cryptocurrency":   order.Cryptocurrency,
			"fiat_currency":    order.FiatCurrency,
			"amount":           order.Amount,
			"price":            order.Price,
			"total_amount":     order.TotalAmount,
			"author_name":      authorName,
			"response_message": response.Message,
		},
	}

	notification, err := s.notificationService.CreateNotification(notificationReq)
	if err != nil {
		log.Printf("[ERROR] Не удалось создать уведомление об отклонении: %v", err)
		return
	}

	// Отправляем уведомление в Telegram
	if err := s.notificationService.SendNotification(notification, responder.TelegramID); err != nil {
		log.Printf("[ERROR] Не удалось отправить уведомление об отклонении: %v", err)
		return
	}

	log.Printf("[INFO] Уведомление об отклонении отклика отправлено пользователю TelegramID=%d", responder.TelegramID)
}

// sendDealCreatedNotifications отправляет уведомления обеим сторонам о создании сделки
func (s *Service) sendDealCreatedNotifications(deal *model.Deal) {
	log.Printf("[INFO] Отправка уведомлений о создании сделки ID=%d", deal.ID)

	// Получаем данные автора заявки
	author, err := s.repo.GetUserByID(deal.AuthorID)
	if err != nil {
		log.Printf("[ERROR] Не удалось найти автора сделки ID=%d: %v", deal.AuthorID, err)
		return
	}

	// Получаем данные контрагента
	counterparty, err := s.repo.GetUserByID(deal.CounterpartyID)
	if err != nil {
		log.Printf("[ERROR] Не удалось найти контрагента сделки ID=%d: %v", deal.CounterpartyID, err)
		return
	}

	// Отправляем уведомление автору заявки
	go s.sendDealCreatedNotification(deal, author, counterparty, true)

	// Отправляем уведомление контрагенту
	go s.sendDealCreatedNotification(deal, counterparty, author, false)
}

// sendDealCreatedNotification отправляет уведомление о создании сделки конкретному пользователю
func (s *Service) sendDealCreatedNotification(deal *model.Deal, recipient *model.User, counterparty *model.User, isAuthor bool) {
	// Формируем имя контрагента
	counterpartyName := counterparty.FirstName
	if counterparty.LastName != "" {
		counterpartyName += " " + counterparty.LastName
	}
	if counterparty.Username != "" {
		counterpartyName += " (@" + counterparty.Username + ")"
	}

	// Форматируем уведомление
	title, message := s.notificationService.FormatDealCreatedNotification(deal, counterpartyName)

	// Создаем уведомление
	notificationReq := &model.CreateNotificationRequest{
		UserID:  recipient.ID,
		Type:    model.NotificationTypeDealCreated,
		Title:   title,
		Message: message,
		DealID:  &deal.ID,
		Data: map[string]interface{}{
			"deal_id":           deal.ID,
			"order_type":        string(deal.OrderType),
			"cryptocurrency":    deal.Cryptocurrency,
			"fiat_currency":     deal.FiatCurrency,
			"amount":            deal.Amount,
			"price":             deal.Price,
			"total_amount":      deal.TotalAmount,
			"counterparty_name": counterpartyName,
			"is_author":         isAuthor,
			"author_id":         deal.AuthorID,
			"counterparty_id":   deal.CounterpartyID,
		},
	}

	notification, err := s.notificationService.CreateNotification(notificationReq)
	if err != nil {
		log.Printf("[ERROR] Не удалось создать уведомление о сделке: %v", err)
		return
	}

	// Отправляем уведомление в Telegram
	if err := s.notificationService.SendNotification(notification, recipient.TelegramID); err != nil {
		log.Printf("[ERROR] Не удалось отправить уведомление о сделке: %v", err)
		return
	}

	log.Printf("[INFO] Уведомление о создании сделки отправлено пользователю TelegramID=%d", recipient.TelegramID)
}

// sendDealConfirmedNotification отправляет уведомление о подтверждении сделки
func (s *Service) sendDealConfirmedNotification(deal *model.Deal, confirmedByUserID int64, waitingForUserID int64) {
	log.Printf("[INFO] Отправка уведомления о подтверждении сделки ID=%d", deal.ID)

	// Получаем данные пользователя, который подтвердил
	confirmedBy, err := s.repo.GetUserByID(confirmedByUserID)
	if err != nil {
		log.Printf("[ERROR] Не удалось найти пользователя который подтвердил ID=%d: %v", confirmedByUserID, err)
		return
	}

	// Получаем данные пользователя, который ждет подтверждения
	waitingFor, err := s.repo.GetUserByID(waitingForUserID)
	if err != nil {
		log.Printf("[ERROR] Не удалось найти пользователя который ждет подтверждения ID=%d: %v", waitingForUserID, err)
		return
	}

	// Формируем имена пользователей
	confirmedByName := confirmedBy.FirstName
	if confirmedBy.LastName != "" {
		confirmedByName += " " + confirmedBy.LastName
	}

	waitingForName := waitingFor.FirstName
	if waitingFor.LastName != "" {
		waitingForName += " " + waitingFor.LastName
	}

	// Форматируем уведомление
	title, message := s.notificationService.FormatDealConfirmedNotification(deal, confirmedByName, waitingForName)

	// Создаем уведомление для пользователя, который должен подтвердить
	notificationReq := &model.CreateNotificationRequest{
		UserID:  waitingFor.ID,
		Type:    model.NotificationTypeDealConfirmed,
		Title:   title,
		Message: message,
		DealID:  &deal.ID,
		Data: map[string]interface{}{
			"deal_id":           deal.ID,
			"confirmed_by_id":   confirmedByUserID,
			"confirmed_by_name": confirmedByName,
			"waiting_for_id":    waitingForUserID,
			"waiting_for_name":  waitingForName,
			"order_type":        string(deal.OrderType),
			"cryptocurrency":    deal.Cryptocurrency,
			"fiat_currency":     deal.FiatCurrency,
			"amount":            deal.Amount,
			"price":             deal.Price,
			"total_amount":      deal.TotalAmount,
		},
	}

	notification, err := s.notificationService.CreateNotification(notificationReq)
	if err != nil {
		log.Printf("[ERROR] Не удалось создать уведомление о подтверждении сделки: %v", err)
		return
	}

	// Отправляем уведомление в Telegram
	if err := s.notificationService.SendNotification(notification, waitingFor.TelegramID); err != nil {
		log.Printf("[ERROR] Не удалось отправить уведомление о подтверждении сделки: %v", err)
		return
	}

	log.Printf("[INFO] Уведомление о подтверждении сделки отправлено пользователю TelegramID=%d", waitingFor.TelegramID)
}

// sendDealCompletedNotifications отправляет уведомления о завершении сделки обеим участникам
func (s *Service) sendDealCompletedNotifications(deal *model.Deal) {
	log.Printf("[INFO] Отправка уведомлений о завершении сделки ID=%d", deal.ID)

	// Получаем данные автора заявки
	author, err := s.repo.GetUserByID(deal.AuthorID)
	if err != nil {
		log.Printf("[ERROR] Не удалось найти автора сделки ID=%d: %v", deal.AuthorID, err)
		return
	}

	// Получаем данные контрагента
	counterparty, err := s.repo.GetUserByID(deal.CounterpartyID)
	if err != nil {
		log.Printf("[ERROR] Не удалось найти контрагента сделки ID=%d: %v", deal.CounterpartyID, err)
		return
	}

	// Формируем имена участников
	authorName := author.FirstName
	if author.LastName != "" {
		authorName += " " + author.LastName
	}

	counterpartyName := counterparty.FirstName
	if counterparty.LastName != "" {
		counterpartyName += " " + counterparty.LastName
	}

	// Форматируем уведомление
	title, message := s.notificationService.FormatDealCompletedNotification(deal, authorName, counterpartyName)

	// Отправляем уведомление автору заявки
	go s.sendDealCompletedNotification(deal, author, title, message)

	// Отправляем уведомление контрагенту
	go s.sendDealCompletedNotification(deal, counterparty, title, message)
}

// sendDealCompletedNotification отправляет уведомление о завершении сделки конкретному пользователю
func (s *Service) sendDealCompletedNotification(deal *model.Deal, recipient *model.User, title, message string) {
	// Создаем уведомление
	notificationReq := &model.CreateNotificationRequest{
		UserID:  recipient.ID,
		Type:    model.NotificationTypeDealCompleted,
		Title:   title,
		Message: message,
		DealID:  &deal.ID,
		Data: map[string]interface{}{
			"deal_id":         deal.ID,
			"order_type":      string(deal.OrderType),
			"cryptocurrency":  deal.Cryptocurrency,
			"fiat_currency":   deal.FiatCurrency,
			"amount":          deal.Amount,
			"price":           deal.Price,
			"total_amount":    deal.TotalAmount,
			"author_id":       deal.AuthorID,
			"counterparty_id": deal.CounterpartyID,
			"completed_at":    deal.CompletedAt,
		},
	}

	notification, err := s.notificationService.CreateNotification(notificationReq)
	if err != nil {
		log.Printf("[ERROR] Не удалось создать уведомление о завершении сделки: %v", err)
		return
	}

	// Отправляем уведомление в Telegram
	if err := s.notificationService.SendNotification(notification, recipient.TelegramID); err != nil {
		log.Printf("[ERROR] Не удалось отправить уведомление о завершении сделки: %v", err)
		return
	}

	log.Printf("[INFO] Уведомление о завершении сделки отправлено пользователю TelegramID=%d", recipient.TelegramID)
}
