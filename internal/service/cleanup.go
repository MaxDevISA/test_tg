package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"p2pTG-crypto-exchange/internal/model"
)

// CleanupService обеспечивает автоматическую очистку устаревших заявок и сделок
// Запускается в фоновом режиме и периодически проверяет таймауты
type CleanupService struct {
	service       *Service           // Основной сервис для доступа к данным
	checkInterval time.Duration      // Интервал между проверками (по умолчанию 30 минут)
	orderTimeout  time.Duration      // Таймаут для заявок (по умолчанию 7 дней)
	dealTimeout   time.Duration      // Таймаут для сделок (по умолчанию 1 день)
	ctx           context.Context    // Контекст для управления жизненным циклом
	cancel        context.CancelFunc // Функция отмены
}

// NewCleanupService создает новый сервис автоматической очистки
func NewCleanupService(service *Service) *CleanupService {
	ctx, cancel := context.WithCancel(context.Background())

	return &CleanupService{
		service:       service,
		checkInterval: 30 * time.Minute,   // Проверяем каждые 30 минут
		orderTimeout:  7 * 24 * time.Hour, // Заявки истекают через 7 дней
		dealTimeout:   6 * time.Hour,      // Сделки истекают через 1 день
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Start запускает background процесс автоматической очистки
// Безопасно для вызова в отдельной горутине
func (cs *CleanupService) Start() {
	log.Printf("[INFO] Запуск службы автоматической очистки (проверка каждые %v)", cs.checkInterval)

	// Немедленно выполняем первую проверку
	cs.runCleanupCycle()

	// Создаем таймер для периодических проверок
	ticker := time.NewTicker(cs.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Время для очередной проверки
			cs.runCleanupCycle()

		case <-cs.ctx.Done():
			// Получен сигнал на остановку
			log.Printf("[INFO] Служба автоматической очистки остановлена")
			return
		}
	}
}

// Stop останавливает службу автоматической очистки
func (cs *CleanupService) Stop() {
	log.Printf("[INFO] Остановка службы автоматической очистки...")
	cs.cancel()
}

// runCleanupCycle выполняет один цикл проверки и очистки
func (cs *CleanupService) runCleanupCycle() {
	log.Printf("[INFO] Начинается цикл автоматической очистки")

	// Безопасно выполняем очистку с обработкой паники
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[ERROR] Паника в цикле очистки: %v", r)
		}
	}()

	// Очищаем устаревшие заявки
	expiredOrders := cs.cleanupExpiredOrders()

	// Очищаем устаревшие сделки
	expiredDeals := cs.cleanupExpiredDeals()

	log.Printf("[INFO] Цикл очистки завершен: заявок обработано=%d, сделок обработано=%d",
		expiredOrders, expiredDeals)
}

// cleanupExpiredOrders находит и обрабатывает устаревшие заявки
func (cs *CleanupService) cleanupExpiredOrders() int {
	log.Printf("[DEBUG] Поиск заявок старше %v", cs.orderTimeout)

	// Вычисляем пороговое время (текущее время - таймаут)
	cutoffTime := time.Now().Add(-cs.orderTimeout)

	log.Printf("[DEBUG] Пороговое время для заявок: %v (сейчас: %v, таймаут: %v)",
		cutoffTime, time.Now(), cs.orderTimeout)

	// Получаем все активные заявки созданные до порогового времени
	filter := &model.OrderFilter{
		Status:        (*model.OrderStatus)(&[]model.OrderStatus{model.OrderStatusActive}[0]),
		CreatedBefore: &cutoffTime,
		Limit:         100, // Обрабатываем порциями для безопасности
	}

	orders, err := cs.service.GetOrders(filter)
	if err != nil {
		log.Printf("[ERROR] Не удалось получить устаревшие заявки: %v", err)
		return 0
	}

	expiredCount := 0
	for _, order := range orders {
		if cs.expireOrder(order) {
			expiredCount++
		}
	}

	if expiredCount > 0 {
		log.Printf("[INFO] Обработано устаревших заявок: %d", expiredCount)
	}

	return expiredCount
}

// cleanupExpiredDeals находит и обрабатывает устаревшие сделки
func (cs *CleanupService) cleanupExpiredDeals() int {
	log.Printf("[DEBUG] Поиск сделок старше %v", cs.dealTimeout)

	// Вычисляем пороговое время
	cutoffTime := time.Now().Add(-cs.dealTimeout)

	// Получаем активные сделки (в процессе или ожидающие подтверждения)
	activeDeals, err := cs.getActiveDealsOlderThan(cutoffTime)
	if err != nil {
		log.Printf("[ERROR] Не удалось получить устаревшие сделки: %v", err)
		return 0
	}

	expiredCount := 0
	for _, deal := range activeDeals {
		if cs.expireDeal(deal) {
			expiredCount++
		}
	}

	if expiredCount > 0 {
		log.Printf("[INFO] Обработано устаревших сделок: %d", expiredCount)
	}

	return expiredCount
}

// expireOrder помечает заявку как истекшую и отправляет уведомление
func (cs *CleanupService) expireOrder(order *model.Order) bool {
	log.Printf("[INFO] Истекает заявка ID=%d (создана: %v, назад: %v, пороговое время: %v)",
		order.ID, order.CreatedAt, time.Since(order.CreatedAt), time.Now().Add(-cs.orderTimeout))

	// Обновляем статус заявки на "истекла"
	err := cs.service.repo.UpdateOrderStatus(order.ID, model.OrderStatusExpired)
	if err != nil {
		log.Printf("[ERROR] Не удалось обновить статус заявки ID=%d: %v", order.ID, err)
		return false
	}

	// Отправляем уведомление автору заявки
	cs.sendOrderExpiredNotification(order)

	log.Printf("[INFO] Заявка ID=%d помечена как истекшая", order.ID)
	return true
}

// expireDeal помечает сделку как истекшую и отправляет уведомления
func (cs *CleanupService) expireDeal(deal *model.Deal) bool {
	log.Printf("[INFO] Истекает сделка ID=%d (создана %v назад)",
		deal.ID, time.Since(deal.CreatedAt))

	// Обновляем статус сделки на "истекла"
	err := cs.service.repo.UpdateDealStatus(deal.ID, string(model.DealStatusExpired))
	if err != nil {
		log.Printf("[ERROR] Не удалось обновить статус сделки ID=%d: %v", deal.ID, err)
		return false
	}

	// Завершаем связанные заявки (они больше не нужны)
	cs.completeOrdersFromExpiredDeal(deal)

	// Отправляем уведомления обеим сторонам
	cs.sendDealExpiredNotifications(deal)

	log.Printf("[INFO] Сделка ID=%d помечена как истекшая", deal.ID)
	return true
}

// sendOrderExpiredNotification отправляет уведомление об истекшей заявке
func (cs *CleanupService) sendOrderExpiredNotification(order *model.Order) {
	message := generateOrderExpiredMessage(order)

	notificationReq := &model.CreateNotificationRequest{
		UserID:  order.UserID,
		Type:    model.NotificationTypeSystemMessage,
		Title:   "Заявка автоматически удалена",
		Message: message,
		OrderID: &order.ID,
		Data: map[string]interface{}{
			"order_id": order.ID,
			"reason":   "expired",
		},
	}

	notification, err := cs.service.notificationService.CreateNotification(notificationReq)
	if err != nil {
		log.Printf("[ERROR] Не удалось создать уведомление об истекшей заявке ID=%d: %v",
			order.ID, err)
		return
	}

	// Получаем данные пользователя для отправки в Telegram
	user, err := cs.service.repo.GetUserByID(order.UserID)
	if err != nil {
		log.Printf("[ERROR] Не удалось получить пользователя ID=%d для отправки уведомления: %v", order.UserID, err)
		return
	}

	// Отправляем уведомление в Telegram
	if err := cs.service.notificationService.SendNotification(notification, user.TelegramID); err != nil {
		log.Printf("[ERROR] Не удалось отправить уведомление в Telegram пользователю ID=%d: %v", order.UserID, err)
		return
	}

	log.Printf("[INFO] Отправлено уведомление пользователю ID=%d об истекшей заявке ID=%d", order.UserID, order.ID)
}

// sendDealExpiredNotifications отправляет уведомления обеим сторонам об истекшей сделке
func (cs *CleanupService) sendDealExpiredNotifications(deal *model.Deal) {
	// Форматируем уведомление используя правильный шаблон
	title, message := cs.service.notificationService.FormatDealCancelledNotification(deal)

	// Уведомление автору
	authorNotificationReq := &model.CreateNotificationRequest{
		UserID:  deal.AuthorID,
		Type:    model.NotificationTypeDealCancelled,
		Title:   title,
		Message: message,
		DealID:  &deal.ID,
		Data: map[string]interface{}{
			"deal_id": deal.ID,
			"reason":  "expired",
		},
	}

	// Уведомление контрагенту
	counterpartyNotificationReq := &model.CreateNotificationRequest{
		UserID:  deal.CounterpartyID,
		Type:    model.NotificationTypeDealCancelled,
		Title:   title,
		Message: message,
		DealID:  &deal.ID,
		Data: map[string]interface{}{
			"deal_id": deal.ID,
			"reason":  "expired",
		},
	}

	// Отправляем уведомления
	// Получаем данные пользователей
	author, err := cs.service.repo.GetUserByID(deal.AuthorID)
	if err != nil {
		log.Printf("[ERROR] Не удалось получить автора ID=%d для отправки уведомления: %v", deal.AuthorID, err)
	} else {
		// Создаем и отправляем уведомление автору
		authorNotification, err := cs.service.notificationService.CreateNotification(authorNotificationReq)
		if err != nil {
			log.Printf("[ERROR] Не удалось создать уведомление автору о сделке ID=%d: %v", deal.ID, err)
		} else {
			if err := cs.service.notificationService.SendNotification(authorNotification, author.TelegramID); err != nil {
				log.Printf("[ERROR] Не удалось отправить уведомление в Telegram автору ID=%d: %v", deal.AuthorID, err)
			} else {
				log.Printf("[INFO] Отправлено уведомление автору ID=%d о сделке ID=%d", deal.AuthorID, deal.ID)
			}
		}
	}

	counterparty, err := cs.service.repo.GetUserByID(deal.CounterpartyID)
	if err != nil {
		log.Printf("[ERROR] Не удалось получить контрагента ID=%d для отправки уведомления: %v", deal.CounterpartyID, err)
	} else {
		// Создаем и отправляем уведомление контрагенту
		counterpartyNotification, err := cs.service.notificationService.CreateNotification(counterpartyNotificationReq)
		if err != nil {
			log.Printf("[ERROR] Не удалось создать уведомление контрагенту о сделке ID=%d: %v", deal.ID, err)
		} else {
			if err := cs.service.notificationService.SendNotification(counterpartyNotification, counterparty.TelegramID); err != nil {
				log.Printf("[ERROR] Не удалось отправить уведомление в Telegram контрагенту ID=%d: %v", deal.CounterpartyID, err)
			} else {
				log.Printf("[INFO] Отправлено уведомление контрагенту ID=%d о сделке ID=%d", deal.CounterpartyID, deal.ID)
			}
		}
	}

	log.Printf("[INFO] Процесс отправки уведомлений завершен для сделки ID=%d", deal.ID)
}

// completeOrdersFromExpiredDeal завершает заявки после истечения связанной сделки
func (cs *CleanupService) completeOrdersFromExpiredDeal(deal *model.Deal) {
	log.Printf("[INFO] Завершение заявок для истекшей сделки ID=%d (участники: %d и %d)",
		deal.ID, deal.AuthorID, deal.CounterpartyID)

	// Завершаем ВСЕ заявки автора в статусе in_deal
	cs.completeUserOrdersInDeal(deal.AuthorID, "автора")

	// Завершаем ВСЕ заявки контрагента в статусе in_deal
	cs.completeUserOrdersInDeal(deal.CounterpartyID, "контрагента")
}

// completeUserOrdersInDeal завершает все заявки пользователя в статусе in_deal
func (cs *CleanupService) completeUserOrdersInDeal(userID int64, role string) {
	log.Printf("[INFO] Поиск заявок %s (ID=%d) в статусе in_deal", role, userID)

	// Ищем все заявки пользователя в статусе in_deal
	filter := &model.OrderFilter{
		UserID: &userID,
		Status: (*model.OrderStatus)(&[]model.OrderStatus{model.OrderStatusInDeal}[0]),
		Limit:  50, // На случай если у пользователя много заявок
	}

	orders, err := cs.service.GetOrders(filter)
	if err != nil {
		log.Printf("[ERROR] Не удалось найти заявки %s ID=%d: %v", role, userID, err)
		return
	}

	if len(orders) == 0 {
		log.Printf("[DEBUG] У %s ID=%d нет заявок в статусе in_deal", role, userID)
		return
	}

	// Завершаем каждую найденную заявку
	for _, order := range orders {
		err := cs.service.repo.UpdateOrderStatus(order.ID, model.OrderStatusCancelled)
		if err != nil {
			log.Printf("[ERROR] Не удалось завершить заявку %s ID=%d: %v", role, order.ID, err)
		} else {
			log.Printf("[INFO] Заявка %s ID=%d завершена (статус: cancelled)", role, order.ID)
		}
	}

	log.Printf("[INFO] Завершено заявок %s: %d", role, len(orders))
}

// getActiveDealsOlderThan получает активные сделки старше указанного времени
func (cs *CleanupService) getActiveDealsOlderThan(cutoffTime time.Time) ([]*model.Deal, error) {
	// Используем прямой вызов к repository для поиска устаревших сделок
	deals, err := cs.service.repo.GetExpiredDeals(cutoffTime)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить устаревшие сделки: %w", err)
	}
	return deals, nil
}

// generateOrderExpiredMessage формирует сообщение об истекшей заявке
func generateOrderExpiredMessage(order *model.Order) string {
	orderType := "продажу"
	if order.Type == model.OrderTypeBuy {
		orderType = "покупку"
	}

	return fmt.Sprintf(
		"⏰ Ваша заявка на %s %.8f %s по цене %.2f %s была автоматически удалена после 7 дней без активности.\n\n"+
			"💡 Вы можете создать новую заявку в любое время.",
		orderType, order.Amount, order.Cryptocurrency, order.Price, order.FiatCurrency,
	)
}
