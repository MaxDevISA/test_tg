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
	migrationPath := "sql/migrations/004_update_orders_status_constraint.sql"
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

	// Проверяем что constraint обновился
	var constraintExists bool
	checkSQL := `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.check_constraints 
			WHERE constraint_name = 'orders_status_check'
			AND check_clause LIKE '%in_deal%'
		)`

	err = db.QueryRow(checkSQL).Scan(&constraintExists)
	if err != nil {
		log.Printf("⚠️ Не удалось проверить constraint: %v", err)
	} else if constraintExists {
		log.Println("✅ CHECK constraint успешно обновлен с новыми статусами заявок!")
	} else {
		log.Println("⚠️ CHECK constraint может быть не обновлен корректно")
	}

	fmt.Println()
	fmt.Println("🎯 Миграция завершена! Теперь можно:")
	fmt.Println("   1. Обновлять статусы заявок на 'in_deal'")
	fmt.Println("   2. Использовать все статусы из Go модели OrderStatus")
	fmt.Println("   3. Тестировать принятие откликов и создание сделок")
}
