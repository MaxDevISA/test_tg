package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"p2pTG-crypto-exchange/internal/handler"
	"p2pTG-crypto-exchange/internal/repository"
	"p2pTG-crypto-exchange/internal/service"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq" // PostgreSQL драйвер
)

// main является точкой входа в приложение P2P криптобиржи
// Инициализирует все компоненты и запускает HTTP сервер
func main() {
	// Загружаем переменные окружения из .env файла (если существует)
	// Это позволяет настраивать приложение без пересборки
	if err := godotenv.Load(); err != nil {
		log.Println("[WARN] Файл .env не найден, используются переменные окружения системы")
	}

	// Получаем порт для запуска сервера из переменной окружения
	// По умолчанию используется порт 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Println("[INFO] Порт не задан, используется порт по умолчанию: 8080")
	}

	// Получаем URL базы данных PostgreSQL
	databaseURL := os.Getenv("DATABASE_URL")

	// Получаем путь к папке данных для JSON файлов (резервный вариант)
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		// Папка данных по умолчанию
		dataDir = "data"
		log.Println("[INFO] DATA_DIR не задан, используется папка по умолчанию: data")
	}

	// Получаем токен Telegram бота из переменных окружения
	telegramToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if telegramToken == "" {
		log.Fatal("[ERROR] TELEGRAM_BOT_TOKEN обязательная переменная окружения не задана")
	}

	// Получаем ID закрытого чата из переменных окружения
	chatID := os.Getenv("TELEGRAM_CHAT_ID")
	if chatID == "" {
		log.Fatal("[ERROR] TELEGRAM_CHAT_ID обязательная переменная окружения не задана")
	}

	// Получаем URL веб-приложения из переменных окружения (необязательно)
	webAppURL := os.Getenv("TELEGRAM_WEBAPP_URL")
	if webAppURL == "" {
		webAppURL = "https://localhost:" + port // Устанавливаем значение по умолчанию
		log.Printf("[INFO] TELEGRAM_WEBAPP_URL не задан, используется по умолчанию: %s", webAppURL)
	}

	// Получаем URL бота из переменных окружения для ссылок в уведомлениях (необязательно)
	botURL := os.Getenv("TELEGRAM_BOT_URL")
	if botURL == "" {
		// Если не задан, используем стандартную ссылку на Telegram (без конкретного бота)
		botURL = "https://t.me" // По умолчанию просто ссылка на Telegram
		log.Printf("[INFO] TELEGRAM_BOT_URL не задан, используется по умолчанию: %s", botURL)
	}

	// Получаем ID группового чата для публикации новых заявок (необязательно)
	groupChatID := os.Getenv("TELEGRAM_GROUP_CHAT_ID")
	if groupChatID != "" {
		log.Printf("[INFO] Групповые уведомления будут отправляться в чат ID: %s", groupChatID)
	} else {
		log.Println("[INFO] TELEGRAM_GROUP_CHAT_ID не задан, групповые уведомления отключены")
	}

	// Получаем ID темы в групповом чате (необязательно)
	groupTopicID := os.Getenv("TELEGRAM_GROUP_TOPIC_ID")
	if groupTopicID != "" && groupChatID != "" {
		log.Printf("[INFO] Групповые уведомления будут отправляться в тему ID: %s", groupTopicID)
	}

	log.Println("[INFO] Запуск P2P криптобиржи...")
	log.Printf("[INFO] Порт сервера: %s", port)
	log.Printf("[INFO] URL веб-приложения: %s", webAppURL)

	// Инициализируем репозиторий для работы с данными
	var repo repository.RepositoryInterface
	var err error

	if databaseURL != "" {
		// Используем PostgreSQL базу данных
		log.Printf("[INFO] 🐘 Подключение к PostgreSQL базе данных...")
		repo, err = repository.NewRepository(databaseURL)
		if err != nil {
			log.Fatalf("[ERROR] Не удалось подключиться к PostgreSQL: %v", err)
		}
		log.Println("[INFO] ✅ PostgreSQL репозиторий инициализирован")
		log.Println("[INFO] 🎯 Все данные теперь сохраняются в базе данных!")
		log.Println("[INFO] 🔄 При деплоях данные НЕ БУДУТ пропадать!")
	} else {
		// Резервный вариант - файловое хранилище
		log.Printf("[WARN] DATABASE_URL не задан, используем файловое хранилище: %s", dataDir)
		log.Printf("[WARN] ⚠️  При деплоях данные БУДУТ пропадать!")
		repo, err = repository.NewFileRepository(dataDir)
		if err != nil {
			log.Fatalf("[ERROR] Не удалось инициализировать файловый репозиторий: %v", err)
		}
		log.Println("[INFO] Файловое хранилище JSON данных готово")
	}
	defer repo.Close() // Закрываем репозиторий при завершении работы

	// Инициализируем слой сервисов для бизнес-логики
	// Сервисы содержат всю логику работы с заявками, пользователями и отзывами
	svc := service.NewServiceWithGroup(repo, telegramToken, chatID, webAppURL, groupChatID, groupTopicID, botURL)
	log.Println("[INFO] Сервисы инициализированы")
	log.Println("[INFO] Система уведомлений готова к отправке сообщений участникам сделок")

	// Запускаем сервис автоматической очистки в фоновом режиме
	// Он будет периодически проверять и удалять устаревшие заявки и сделки
	svc.StartCleanupService()
	log.Println("[INFO] 🧹 Система автоматической очистки активна (заявки: 7 дней, сделки: 1 день)")

	// Инициализируем слой обработчиков HTTP запросов
	// Обработчики принимают HTTP запросы и вызывают соответствующие сервисы
	handlers := handler.NewHandler(svc)
	log.Println("[INFO] Обработчики HTTP запросов инициализированы")

	// Создаем HTTP маршрутизатор с использованием gorilla/mux
	// Маршрутизатор определяет какой обработчик вызывать для каждого URL
	router := mux.NewRouter()

	// Регистрируем все маршруты приложения
	handlers.RegisterRoutes(router)
	log.Println("[INFO] HTTP маршруты зарегистрированы")

	// Запускаем HTTP сервер на указанном порту
	// Создаем HTTP сервер с graceful shutdown
	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// Сервер будет обрабатывать все входящие HTTP запросы
	log.Printf("[INFO] Запуск HTTP сервера на порту %s", port)
	log.Printf("[INFO] Веб-интерфейс доступен по адресу: http://localhost:%s", port)

	// Создаем канал для получения системных сигналов
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Запускаем сервер в отдельной горутине
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[ERROR] Не удалось запустить HTTP сервер: %v", err)
		}
	}()

	log.Println("[INFO] ✅ Сервер запущен! Нажмите Ctrl+C для остановки")

	// Ожидаем сигнал остановки
	<-quit
	log.Println("[INFO] 🛑 Получен сигнал остановки, начинаем graceful shutdown...")

	// Останавливаем сервис автоматической очистки
	svc.StopCleanupService()

	// Создаем контекст с таймаутом для graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Graceful shutdown HTTP сервера
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("[ERROR] Ошибка при остановке сервера: %v", err)
	}

	log.Println("[INFO] ✅ Сервер остановлен корректно")
}
