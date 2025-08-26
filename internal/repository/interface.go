package repository

import "p2pTG-crypto-exchange/internal/model"

// RepositoryInterface определяет методы для работы с хранилищем данных
// Этот интерфейс может быть реализован для PostgreSQL, файлового хранилища или других БД
type RepositoryInterface interface {
	// Управление жизненным циклом репозитория
	Close() error
	HealthCheck() error

	// Методы для работы с пользователями
	CreateUser(user *model.User) error
	GetUserByID(userID int64) (*model.User, error)
	GetUserByTelegramID(telegramID int64) (*model.User, error)
	UpdateUserChatMembership(telegramID int64, isMember bool) error

	// Методы для работы с заявками
	CreateOrder(order *model.Order) error
	GetOrdersByFilter(filter *model.OrderFilter) ([]*model.Order, error)
	UpdateOrderStatus(orderID int64, status model.OrderStatus) error
	GetMatchingOrders(order *model.Order) ([]*model.Order, error)
	MatchOrders(orderID1, orderID2 int64) error

	// Методы для работы со сделками
	CreateDeal(deal *model.Deal) error
	GetDealsByUserID(userID int64) ([]*model.Deal, error)
	GetDealByID(dealID int64) (*model.Deal, error)
	ConfirmDeal(dealID int64, userID int64, isPaymentProof bool, paymentProof string) error

	// Методы для работы с откликами
	CreateResponse(response *model.Response) error
	GetResponsesByFilter(filter *model.ResponseFilter) ([]*model.Response, error)
	UpdateResponseStatus(responseID int64, status model.ResponseStatus) error
	GetResponsesForOrder(orderID int64) ([]*model.Response, error)
	GetResponsesFromUser(userID int64) ([]*model.Response, error)
	GetResponsesForAuthor(authorID int64) ([]*model.Response, error)

	// Методы для работы с отзывами и рейтингами
	CreateReview(review *model.Review) error
	GetReviewsByUserID(userID int64, limit, offset int) ([]*model.Review, error)
	GetUserRating(userID int64) (*model.Rating, error)
	CheckCanReview(dealID, fromUserID, toUserID int64) (bool, error)
	ReportReview(report *model.ReviewReport) error
	GetUserReviewStats(userID int64) (*model.ReviewStats, error)
}
