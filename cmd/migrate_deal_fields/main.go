package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	// Получаем DATABASE_URL из переменных окружения
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL не установлен в переменных окружения")
	}

	// Подключаемся к базе данных
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatalf("Не удалось подключиться к базе данных: %v", err)
	}
	defer db.Close()

	// Проверяем соединение
	if err := db.Ping(); err != nil {
		log.Fatalf("Не удалось выполнить ping базы данных: %v", err)
	}

	log.Println("✅ Подключение к PostgreSQL успешно")

	// Читаем миграцию
	migrationPath := "sql/migrations/005_rename_deal_confirmation_fields.sql"
	migrationSQL, err := ioutil.ReadFile(migrationPath)
	if err != nil {
		log.Fatalf("Не удалось прочитать файл миграции %s: %v", migrationPath, err)
	}

	log.Printf("📄 Применяем миграцию: %s", migrationPath)

	// Выполняем миграцию
	_, err = db.Exec(string(migrationSQL))
	if err != nil {
		log.Fatalf("❌ Ошибка выполнения миграции: %v", err)
	}

	log.Println("✅ Миграция успешно применена!")

	// Проверяем что поля переименованы
	var authorConfirmedExists, counterConfirmedExists bool

	checkAuthorSQL := `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_name = 'deals' AND column_name = 'author_confirmed'
		)`

	checkCounterSQL := `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_name = 'deals' AND column_name = 'counter_confirmed'
		)`

	err = db.QueryRow(checkAuthorSQL).Scan(&authorConfirmedExists)
	if err != nil {
		log.Printf("⚠️ Не удалось проверить поле author_confirmed: %v", err)
	}

	err = db.QueryRow(checkCounterSQL).Scan(&counterConfirmedExists)
	if err != nil {
		log.Printf("⚠️ Не удалось проверить поле counter_confirmed: %v", err)
	}

	if authorConfirmedExists && counterConfirmedExists {
		log.Println("✅ Поля успешно переименованы: author_confirmed, counter_confirmed")
	} else {
		log.Println("⚠️ Поля могут быть не переименованы корректно")
	}

	fmt.Println()
	fmt.Println("🎯 Миграция завершена! Теперь можно:")
	fmt.Println("   1. Подтверждать сделки без ошибок полей")
	fmt.Println("   2. Использовать author_confirmed/counter_confirmed")
	fmt.Println("   3. Тестировать подтверждение и завершение сделок")
}
