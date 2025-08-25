# 🚀 Быстрый запуск P2P Криптобиржи (JSON версия)

## Простой запуск без базы данных - всего 3 шага!

### 1️⃣ Запуск сервера 

```bash
# Запуск с минимальными настройками
go run main.go
```

Или собрать и запустить:

```bash  
go build -o bot.exe main.go
.\bot.exe
```

### 2️⃣ Что происходит при первом запуске

✅ Автоматически создается папка `data/` с JSON файлами:
- `users.json` - пользователи  
- `orders.json` - заявки
- `deals.json` - сделки
- `reviews.json` - отзывы
- `ratings.json` - рейтинги
- `counters.json` - счетчики ID

✅ Запускается сервер на порту 8080

✅ Веб-интерфейс доступен по адресу: http://localhost:8080

### 3️⃣ Настройка Telegram бота (опционально)

Для полной функциональности создайте файл `.env`:

```env
PORT=8080
DATA_DIR=data
TELEGRAM_BOT_TOKEN=ваш_токен_от_BotFather  
TELEGRAM_CHAT_ID=ваш_id_чата
```

## 🧪 Тестирование API

### Проверка здоровья сервиса
```bash
curl http://localhost:8080/api/v1/health
```

### Получение заявок  
```bash
curl http://localhost:8080/api/v1/orders
```

### Создание тестовой заявки
```bash
curl -X POST http://localhost:8080/api/v1/orders \
-H "Content-Type: application/json" \
-d '{
  "type": "buy",
  "cryptocurrency": "BTC", 
  "fiat_currency": "RUB",
  "amount": 0.001,
  "price": 2800000,
  "payment_methods": ["sberbank"],
  "description": "Тестовая заявка"
}'
```

## 📁 Структура JSON файлов

После запуска в папке `data/` появятся читаемые JSON файлы:

```json
// users.json - Пользователи
[
  {
    "id": 1,
    "telegram_id": 12345,
    "first_name": "Иван",
    "username": "ivan123", 
    "rating": 4.5,
    "total_deals": 10
  }
]

// orders.json - Заявки  
[
  {
    "id": 1,
    "user_id": 1,
    "type": "buy",
    "cryptocurrency": "BTC",
    "amount": 0.001,
    "price": 2800000,
    "status": "active"
  }
]
```

## 🔧 Преимущества JSON версии

✅ **Никаких зависимостей** - не нужно устанавливать PostgreSQL  
✅ **Мгновенный запуск** - один файл, одна команда  
✅ **Легкая отладка** - все данные в читаемых JSON файлах  
✅ **Портативность** - папку `data/` можно легко переносить  
✅ **Безопасность** - данные хранятся локально  

## 🌐 Доступные разделы

- **Главная**: http://localhost:8080 
- **API документация**: все эндпоинты работают
- **JSON данные**: папка `data/` с файлами

## 🛠️ Разработка

Для разработки просто редактируйте код и перезапускайте:

```bash
go run main.go  # Данные сохранятся в JSON файлах
```

JSON файлы можно редактировать напрямую для тестирования!
