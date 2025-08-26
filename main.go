package main

import (
	"log"
	"net/http"
	"os"

	"p2pTG-crypto-exchange/internal/handler"
	"p2pTG-crypto-exchange/internal/repository"
	"p2pTG-crypto-exchange/internal/service"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
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

	// Получаем путь к папке данных для JSON файлов
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

	log.Println("[INFO] Запуск P2P криптобиржи...")
	log.Printf("[INFO] Порт сервера: %s", port)
	log.Printf("[INFO] Папка данных: %s", dataDir)
	log.Printf("[INFO] URL веб-приложения: %s", webAppURL)

	// Инициализируем файловый репозиторий для работы с JSON данными
	// Репозиторий отвечает за все операции с данными в JSON файлах
	repo, err := repository.NewFileRepository(dataDir)
	if err != nil {
		log.Fatalf("[ERROR] Не удалось инициализировать файловый репозиторий: %v", err)
	}
	defer repo.Close() // Закрываем репозиторий при завершении работы
	log.Println("[INFO] Файловое хранилище JSON данных готово")

	// Инициализируем слой сервисов для бизнес-логики
	// Сервисы содержат всю логику работы с заявками, пользователями и отзывами
	svc := service.NewServiceWithWebApp(repo, telegramToken, chatID, webAppURL)
	log.Println("[INFO] Сервисы инициализированы")
	log.Println("[INFO] Система уведомлений готова к отправке сообщений участникам сделок")

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
	// Сервер будет обрабатывать все входящие HTTP запросы
	log.Printf("[INFO] Запуск HTTP сервера на порту %s", port)
	log.Printf("[INFO] Веб-интерфейс доступен по адресу: http://localhost:%s", port)

	// ListenAndServe блокирует выполнение и обрабатывает HTTP запросы
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatalf("[ERROR] Не удалось запустить HTTP сервер: %v", err)
	}
}
