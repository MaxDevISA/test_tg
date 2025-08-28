package main

import (
	"database/sql"
	"io/ioutil"
	"log"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// migrate_responses.go - утилита для добавления таблицы responses в PostgreSQL
func main() {
	log.Println("🚀 Добавление таблицы responses в PostgreSQL")

	// Загружаем переменные окружения
	if err := godotenv.Load(); err != nil {
		log.Println("[WARN] Файл .env не найден, используются переменные окружения системы")
	}

	// Получаем URL базы данных
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("[ERROR] DATABASE_URL не задан в переменных окружения")
	}

	log.Printf("[INFO] Подключение к PostgreSQL...")

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

	// Выполняем миграцию для responses
	log.Println("[INFO] 📋 Создание таблицы responses...")
	migrationFile := "sql/migrations/002_add_responses_table.sql"
	
	// Читаем файл миграции
	migrationSQL, err := ioutil.ReadFile(migrationFile)
	if err != nil {
		log.Fatalf("[ERROR] Не удалось прочитать файл миграции %s: %v", migrationFile, err)
	}

	// Выполняем миграцию
	if _, err := db.Exec(string(migrationSQL)); err != nil {
		log.Fatalf("[ERROR] Не удалось выполнить миграцию: %v", err)
	}
	
	log.Println("[INFO] ✅ Таблица responses создана успешно")

	// Проверяем результат
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM responses").Scan(&count); err != nil {
		log.Printf("[WARN] Не удалось проверить таблицу responses: %v", err)
	} else {
		log.Printf("[INFO] Таблица responses: %d записей", count)
	}

	log.Println("[INFO] 🎉 Готово! Теперь можно реализовывать методы для работы с откликами")
}
