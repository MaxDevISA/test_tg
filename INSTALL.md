# Инструкция по установке и настройке P2P Криптобиржи

## 🚀 Пошаговая инструкция по запуску

### 1. Подготовка системы
```bash
# Убедитесь что у вас установлены:
# - Go 1.19 или выше
# - PostgreSQL 12 или выше

go version
psql --version
```

### 2. Настройка базы данных
```bash
# Создайте базу данных
createdb p2p_crypto_exchange

# Или через psql:
psql -U postgres
CREATE DATABASE p2p_crypto_exchange;
\q

# Выполните миграцию
psql -d p2p_crypto_exchange -f sql/migrations/001_initial_schema.sql
```

### 3. Создание Telegram бота
1. Найдите [@BotFather](https://t.me/BotFather) в Telegram
2. Создайте бота: `/newbot`  
3. Выберите имя: например "P2P Crypto Exchange Bot"
4. Выберите username: например `p2p_crypto_bot`
5. Получите токен бота (похож на `1234567890:AAEhBP0zabcdefghijklmnopqrstuvwxyz`)

### 4. Настройка чата
1. Создайте закрытый чат в Telegram
2. Добавьте бота в чат как администратора
3. Получите ID чата (можно через [@userinfobot](https://t.me/userinfobot))

### 5. Создание файла конфигурации
Создайте файл `.env` в корне проекта:
```env
# Основные настройки
PORT=8080
DATABASE_URL=postgres://username:password@localhost:5432/p2p_crypto_exchange?sslmode=disable

# Telegram настройки
TELEGRAM_BOT_TOKEN=ваш_токен_бота_здесь
TELEGRAM_CHAT_ID=ваш_id_чата_здесь

# Опциональные настройки для разработки
LOG_LEVEL=info
DEVELOPMENT_MODE=true
```

### 6. Запуск приложения
```bash
# Установка зависимостей
go mod tidy

# Сборка приложения
go build -o bot.exe main.go

# Запуск приложения
./bot.exe

# Или запуск без сборки (для разработки)
go run main.go
```

### 7. Проверка работы
1. Откройте в браузере: http://localhost:8080
2. Проверьте health check: http://localhost:8080/api/v1/health
3. Посмотрите логи в консоли

## 🔧 Настройка Telegram мини-приложения

### 1. Настройка кнопки меню
В чате с ботом отправьте команды:
```
/setmenubutton
<выберите вашего бота>
<введите текст кнопки: "Открыть биржу">
<введите URL: https://yourdomain.com или для тестирования http://localhost:8080>
```

### 2. Настройка описания бота
```
/setdescription
<выберите вашего бота>
<введите описание: "P2P криптобиржа для закрытого чата">
```

### 3. Настройка команд
```
/setcommands
<выберите вашего бота>
<введите команды:>
start - Запуск бота
help - Помощь
exchange - Открыть биржу
```

## 🌐 Настройка для продакшена

### 1. Получение SSL сертификата
```bash
# С помощью Let's Encrypt
sudo certbot certonly --standalone -d yourdomain.com
```

### 2. Настройка переменных окружения
```env
# Продакшен настройки
PORT=443
DATABASE_URL=postgres://user:pass@db-server:5432/p2p_crypto_exchange?sslmode=require
ENABLE_TLS=true
SSL_CERT_FILE=/path/to/fullchain.pem
SSL_KEY_FILE=/path/to/privkey.pem

# Telegram настройки для webhook
TELEGRAM_WEBHOOK_URL=https://yourdomain.com/telegram/webhook
TELEGRAM_WEBAPP_URL=https://yourdomain.com
```

### 3. Настройка nginx (опционально)
```nginx
server {
    listen 80;
    server_name yourdomain.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl;
    server_name yourdomain.com;
    
    ssl_certificate /etc/letsencrypt/live/yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/yourdomain.com/privkey.pem;
    
    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## 🐛 Устранение неполадок

### Проблема: "База данных недоступна"
```bash
# Проверьте соединение с PostgreSQL
psql -h localhost -U postgres -d p2p_crypto_exchange -c "SELECT 1;"

# Убедитесь что сервис запущен
sudo systemctl status postgresql
```

### Проблема: "Неверный токен бота"
- Проверьте правильность токена в `.env`
- Убедитесь что бот не был удален в @BotFather
- Проверьте что в токене нет лишних пробелов

### Проблема: "Чат не найден"  
- Убедитесь что бот добавлен в чат как администратор
- Проверьте правильность ID чата (должен начинаться с `-100`)
- ID можно получить через @userinfobot

### Проблема: "Ошибка авторизации в приложении"
- Проверьте что приложение открыто из Telegram
- В браузере используется тестовый режим
- Проверьте настройки CORS для локальной разработки

## 📊 Мониторинг

### Логи приложения
```bash
# Просмотр логов в реальном времени
tail -f app.log

# Поиск ошибок
grep ERROR app.log

# Статистика запросов
grep "HTTP" app.log | awk '{print $7}' | sort | uniq -c
```

### Health Check
```bash
# Автоматическая проверка доступности
curl -f http://localhost:8080/api/v1/health || echo "Сервис недоступен"
```

## 🔄 Обновление

```bash
# Остановите приложение
pkill bot

# Обновите код
git pull origin main

# Пересоберите приложение
go build -o bot.exe main.go

# Запустите снова
./bot.exe
```

## 🆘 Получение помощи

1. Проверьте логи приложения на ошибки
2. Убедитесь что все зависимости установлены
3. Проверьте настройки в `.env` файле
4. Создайте issue в репозитории с описанием проблемы

## 🔗 Полезные ссылки
- [Telegram Bot API](https://core.telegram.org/bots/api)
- [Telegram WebApp Guide](https://core.telegram.org/bots/webapps)  
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [Go Documentation](https://golang.org/doc/)
