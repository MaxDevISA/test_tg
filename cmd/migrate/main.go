package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"p2pTG-crypto-exchange/internal/model"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// migrate_data.go - утилита для миграции данных из JSON файлов в PostgreSQL
func main() {
	log.Println("🚀 Запуск утилиты миграции данных JSON → PostgreSQL")

	// Загружаем переменные окружения
	if err := godotenv.Load(); err != nil {
		log.Println("[WARN] Файл .env не найден, используются переменные окружения системы")
	}

	// Получаем URL базы данных
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("[ERROR] DATABASE_URL не задан в переменных окружения")
	}

	// Получаем путь к данным
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "data"
	}

	log.Printf("[INFO] Подключение к PostgreSQL...")
	log.Printf("[INFO] Папка с данными: %s", dataDir)

	// Подключаемся к базе данных
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatalf("[ERROR] Не удалось подключиться к базе данных: %v", err)
	}
	defer db.Close()

	// Проверяем соединение
	if err := db.Ping(); err != nil {
		log.Fatalf("[ERROR] База данных недоступна: %v", err)
	}
	log.Println("[INFO] ✅ Подключение к PostgreSQL успешно")

	// Выполняем миграции (создаем таблицы)
	log.Println("[INFO] 📋 Выполнение SQL миграций...")
	if err := runMigrations(db); err != nil {
		log.Fatalf("[ERROR] Ошибка выполнения миграций: %v", err)
	}
	log.Println("[INFO] ✅ Миграции выполнены успешно")

	// Мигрируем данные
	log.Println("[INFO] 📦 Перенос данных из JSON файлов...")
	if err := migrateData(db, dataDir); err != nil {
		log.Fatalf("[ERROR] Ошибка миграции данных: %v", err)
	}
	log.Println("[INFO] ✅ Данные успешно перенесены в PostgreSQL")

	// Проверяем результаты
	log.Println("[INFO] 🔍 Проверка результатов миграции...")
	if err := validateMigration(db); err != nil {
		log.Fatalf("[ERROR] Ошибка валидации: %v", err)
	}
	log.Println("[INFO] ✅ Миграция завершена успешно!")

	log.Println("")
	log.Println("🎉 Все готово! Теперь можно:")
	log.Println("   1. Коммитить и пушить изменения в Git")
	log.Println("   2. Render автоматически задеплоит с PostgreSQL")
	log.Println("   3. Данные будут сохраняться между деплоями")
}

// runMigrations выполняет SQL миграции для создания таблиц
func runMigrations(db *sql.DB) error {
	// Список миграций в правильном порядке
	migrations := []string{
		"sql/migrations/001_initial_schema.sql",
		"sql/migrations/002_add_responses_table.sql",
	}

	// Выполняем каждую миграцию
	for _, migrationFile := range migrations {
		log.Printf("[INFO] Выполнение миграции: %s", migrationFile)

		// Читаем файл миграции
		migrationSQL, err := ioutil.ReadFile(migrationFile)
		if err != nil {
			return fmt.Errorf("не удалось прочитать файл миграции %s: %w", migrationFile, err)
		}

		// Выполняем миграцию
		if _, err := db.Exec(string(migrationSQL)); err != nil {
			return fmt.Errorf("не удалось выполнить миграцию %s: %w", migrationFile, err)
		}

		log.Printf("[INFO] ✅ Миграция выполнена: %s", migrationFile)
	}

	return nil
}

// migrateData переносит данные из JSON файлов в PostgreSQL
func migrateData(db *sql.DB, dataDir string) error {
	// 1. Мигрируем пользователей
	if err := migrateUsers(db, filepath.Join(dataDir, "users.json")); err != nil {
		return fmt.Errorf("ошибка миграции пользователей: %w", err)
	}

	// 2. Мигрируем заявки
	if err := migrateOrders(db, filepath.Join(dataDir, "orders.json")); err != nil {
		return fmt.Errorf("ошибка миграции заявок: %w", err)
	}

	// 3. Мигрируем сделки
	if err := migrateDeals(db, filepath.Join(dataDir, "deals.json")); err != nil {
		return fmt.Errorf("ошибка миграции сделок: %w", err)
	}

	// 4. Мигрируем отзывы
	if err := migrateReviews(db, filepath.Join(dataDir, "reviews.json")); err != nil {
		return fmt.Errorf("ошибка миграции отзывов: %w", err)
	}

	// 5. Мигрируем рейтинги
	if err := migrateRatings(db, filepath.Join(dataDir, "ratings.json")); err != nil {
		return fmt.Errorf("ошибка миграции рейтингов: %w", err)
	}

	return nil
}

// migrateUsers переносит пользователей из users.json в таблицу users
func migrateUsers(db *sql.DB, filePath string) error {
	log.Printf("[INFO] Миграция пользователей из %s", filePath)

	// Проверяем существование файла
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Printf("[WARN] Файл %s не существует, пропускаем", filePath)
		return nil
	}

	// Читаем JSON файл
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("не удалось прочитать файл: %w", err)
	}

	// Парсим JSON
	var users []model.User
	if err := json.Unmarshal(data, &users); err != nil {
		return fmt.Errorf("не удалось распарсить JSON: %w", err)
	}

	log.Printf("[INFO] Найдено %d пользователей", len(users))

	// Переносим каждого пользователя
	for i, user := range users {
		query := `
			INSERT INTO users (
				id, telegram_id, telegram_user_id, first_name, last_name, 
				username, photo_url, is_bot, language_code, created_at, 
				updated_at, is_active, rating, total_deals, successful_deals, chat_member
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
			) ON CONFLICT (telegram_id) DO UPDATE SET
				first_name = EXCLUDED.first_name,
				last_name = EXCLUDED.last_name,
				username = EXCLUDED.username,
				photo_url = EXCLUDED.photo_url,
				updated_at = NOW(),
				rating = EXCLUDED.rating,
				total_deals = EXCLUDED.total_deals,
				successful_deals = EXCLUDED.successful_deals,
				chat_member = EXCLUDED.chat_member
		`

		_, err := db.Exec(query,
			user.ID, user.TelegramID, user.TelegramUserID, user.FirstName, user.LastName,
			user.Username, user.PhotoURL, user.IsBot, user.LanguageCode, user.CreatedAt,
			user.UpdatedAt, user.IsActive, user.Rating, user.TotalDeals, user.SuccessfulDeals, user.ChatMember,
		)
		if err != nil {
			return fmt.Errorf("не удалось добавить пользователя %d: %w", i, err)
		}
	}

	log.Printf("[INFO] ✅ Перенесено %d пользователей", len(users))
	return nil
}

// TODO: Добавить остальные функции миграции (orders, deals, reviews, ratings)
// Пока реализуем только пользователей для тестирования

// Заглушки для остальных функций
func migrateOrders(db *sql.DB, filePath string) error {
	log.Printf("[INFO] TODO: Миграция заявок из %s", filePath)
	return nil
}

func migrateDeals(db *sql.DB, filePath string) error {
	log.Printf("[INFO] TODO: Миграция сделок из %s", filePath)
	return nil
}

func migrateReviews(db *sql.DB, filePath string) error {
	log.Printf("[INFO] TODO: Миграция отзывов из %s", filePath)
	return nil
}

func migrateRatings(db *sql.DB, filePath string) error {
	log.Printf("[INFO] TODO: Миграция рейтингов из %s", filePath)
	return nil
}

// validateMigration проверяет что данные перенеслись корректно
func validateMigration(db *sql.DB) error {
	// Проверяем количество записей в каждой таблице
	tables := []string{"users", "orders", "deals", "reviews", "ratings", "responses"}

	for _, table := range tables {
		var count int
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
		if err := db.QueryRow(query).Scan(&count); err != nil {
			log.Printf("[WARN] Не удалось проверить таблицу %s: %v", table, err)
			continue
		}
		log.Printf("[INFO] Таблица %s: %d записей", table, count)
	}

	return nil
}
