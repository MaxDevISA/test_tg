// JavaScript для P2P криптобиржи в Telegram мини-приложении
// Отвечает за взаимодействие с API и управление интерфейсом

// ===== ГЛОБАЛЬНЫЕ ПЕРЕМЕННЫЕ =====
let currentUser = null;           // Данные текущего пользователя
let currentView = 'orders';       // Текущий активный раздел
let orders = [];                  // Кэш загруженных заявок
let isLoading = false;            // Флаг загрузки данных

// Конфигурация API
const API_BASE = '/api/v1';

// ===== ИНИЦИАЛИЗАЦИЯ ПРИЛОЖЕНИЯ =====
document.addEventListener('DOMContentLoaded', function() {
    console.log('[INFO] Инициализация P2P криптобиржи');
    
    // Инициализируем Telegram WebApp API
    initTelegramWebApp();
    
    // Настраиваем обработчики событий
    setupEventListeners();
    
    // Загружаем начальные данные
    loadInitialData();
});

// ===== РАБОТА С TELEGRAM WEBAPP API =====
function initTelegramWebApp() {
    console.log('[INFO] Инициализация Telegram WebApp');
    
    // Проверяем доступность Telegram WebApp
    if (typeof Telegram !== 'undefined' && Telegram.WebApp) {
        const tg = Telegram.WebApp;
        
        // Настраиваем тему приложения под Telegram
        setTelegramTheme(tg);
        
        // Показываем кнопку "Назад" если нужно
        tg.BackButton.show();
        tg.BackButton.onClick(function() {
            // Обработка нажатия кнопки "Назад"
            if (currentView !== 'orders') {
                showView('orders');
            } else {
                tg.close();
            }
        });
        
        // Разворачиваем приложение на весь экран
        tg.expand();
        
        // Готовим приложение к показу
        tg.ready();
        
        console.log('[INFO] Telegram WebApp инициализирован');
        
        // Пытаемся авторизовать пользователя
        authenticateUser();
    } else {
        console.log('[WARN] Telegram WebApp недоступен, работаем в режиме браузера');
        
        // В браузере показываем заглушку для тестирования
        showBrowserTestMode();
    }
}

// Устанавливает тему приложения из Telegram
function setTelegramTheme(tg) {
    const root = document.documentElement;
    
    // Применяем цвета из темы Telegram
    if (tg.themeParams) {
        root.style.setProperty('--tg-theme-bg-color', tg.themeParams.bg_color || '#ffffff');
        root.style.setProperty('--tg-theme-text-color', tg.themeParams.text_color || '#000000');
        root.style.setProperty('--tg-theme-hint-color', tg.themeParams.hint_color || '#999999');
        root.style.setProperty('--tg-theme-link-color', tg.themeParams.link_color || '#0088cc');
        root.style.setProperty('--tg-theme-button-color', tg.themeParams.button_color || '#0088cc');
        root.style.setProperty('--tg-theme-button-text-color', tg.themeParams.button_text_color || '#ffffff');
        root.style.setProperty('--tg-theme-secondary-bg-color', tg.themeParams.secondary_bg_color || '#f1f1f1');
    }
}

// Режим тестирования в браузере
function showBrowserTestMode() {
    const header = document.querySelector('.header');
    if (header) {
        header.innerHTML = `
            <h1>🔄 P2P Криптобиржа</h1>
            <div class="user-info">Режим тестирования в браузере</div>
        `;
    }
    
    // Загружаем тестовые данные
    loadTestData();
}

// ===== АВТОРИЗАЦИЯ ПОЛЬЗОВАТЕЛЯ =====
function authenticateUser() {
    console.log('[INFO] Попытка авторизации пользователя');
    
    const tg = Telegram.WebApp;
    const initData = tg.initData;
    
    if (!initData) {
        console.log('[WARN] Данные авторизации недоступны');
        showError('Ошибка авторизации. Перезапустите приложение.');
        return;
    }
    
    // Парсим данные авторизации
    const authData = parseInitData(initData);
    
    if (!authData.user) {
        console.log('[ERROR] Пользователь не найден в данных авторизации');
        showError('Ошибка авторизации пользователя.');
        return;
    }
    
    // Отправляем запрос на авторизацию
    fetch(`${API_BASE}/auth/login`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(authData)
    })
    .then(response => response.json())
    .then(data => {
        if (data.success && data.user) {
            currentUser = data.user;
            console.log('[INFO] Пользователь успешно авторизован:', currentUser.first_name);
            
            // Обновляем интерфейс
            updateUserInfo();
            
            // Загружаем данные пользователя
            loadUserData();
        } else {
            console.log('[ERROR] Ошибка авторизации:', data.error);
            showError(data.error || 'Ошибка авторизации');
        }
    })
    .catch(error => {
        console.error('[ERROR] Ошибка сети при авторизации:', error);
        showError('Ошибка соединения с сервером');
    });
}

// Парсит данные инициализации от Telegram
function parseInitData(initData) {
    const params = new URLSearchParams(initData);
    const user = JSON.parse(params.get('user') || '{}');
    
    return {
        id: user.id,
        first_name: user.first_name,
        last_name: user.last_name,
        username: user.username,
        photo_url: user.photo_url,
        auth_date: parseInt(params.get('auth_date')),
        hash: params.get('hash')
    };
}

// ===== УПРАВЛЕНИЕ ИНТЕРФЕЙСОМ =====
function setupEventListeners() {
    console.log('[INFO] Настройка обработчиков событий');
    
    // Навигационные кнопки
    document.querySelectorAll('.nav-item').forEach(item => {
        item.addEventListener('click', function() {
            const view = this.dataset.view;
            showView(view);
        });
    });
    
    // Кнопка создания новой заявки
    const createOrderBtn = document.getElementById('createOrderBtn');
    if (createOrderBtn) {
        createOrderBtn.addEventListener('click', showCreateOrderModal);
    }
    
    // Закрытие модальных окон
    document.querySelectorAll('.modal-close').forEach(closeBtn => {
        closeBtn.addEventListener('click', function() {
            closeModal(this.closest('.modal').id);
        });
    });
    
    // Форма создания заявки
    const createOrderForm = document.getElementById('createOrderForm');
    if (createOrderForm) {
        createOrderForm.addEventListener('submit', handleCreateOrder);
    }
    
    // Фильтры заявок
    const filters = document.querySelectorAll('.filter-select');
    filters.forEach(filter => {
        filter.addEventListener('change', applyFilters);
    });
}

// Переключает активный раздел приложения
function showView(viewName) {
    console.log(`[INFO] Переключение на раздел: ${viewName}`);
    
    currentView = viewName;
    
    // Обновляем активную навигационную кнопку
    document.querySelectorAll('.nav-item').forEach(item => {
        item.classList.remove('active');
        if (item.dataset.view === viewName) {
            item.classList.add('active');
        }
    });
    
    // Скрываем все разделы
    document.querySelectorAll('.view').forEach(view => {
        view.classList.add('hidden');
    });
    
    // Показываем нужный раздел
    const targetView = document.getElementById(viewName + 'View');
    if (targetView) {
        targetView.classList.remove('hidden');
    }
    
    // Загружаем данные для раздела
    loadViewData(viewName);
}

// ===== РАБОТА С ДАННЫМИ =====
function loadInitialData() {
    console.log('[INFO] Загрузка начальных данных');
    showView('orders');
}

function loadUserData() {
    if (!currentUser) return;
    
    console.log('[INFO] Загрузка данных пользователя');
    // TODO: Загрузить профиль пользователя, его заявки и статистику
}

function loadViewData(viewName) {
    switch (viewName) {
        case 'orders':
            loadOrders();
            break;
        case 'my-orders':
            loadMyOrders();
            break;
        case 'deals':
            loadDeals();
            break;
        case 'profile':
            loadProfile();
            break;
        default:
            console.log(`[WARN] Неизвестный раздел: ${viewName}`);
    }
}

// Загружает список всех заявок
function loadOrders() {
    console.log('[INFO] Загрузка списка заявок');
    
    if (isLoading) return;
    isLoading = true;
    
    showLoading('ordersContent');
    
    fetch(`${API_BASE}/orders?limit=20&offset=0`)
        .then(response => response.json())
        .then(data => {
            isLoading = false;
            hideLoading('ordersContent');
            
            if (data.success && data.orders) {
                orders = data.orders;
                renderOrders(orders);
                console.log(`[INFO] Загружено заявок: ${orders.length}`);
            } else {
                console.log('[ERROR] Ошибка загрузки заявок:', data.error);
                showError('Не удалось загрузить заявки');
            }
        })
        .catch(error => {
            isLoading = false;
            hideLoading('ordersContent');
            console.error('[ERROR] Ошибка сети при загрузке заявок:', error);
            showError('Ошибка соединения с сервером');
        });
}

function loadTestData() {
    console.log('[INFO] Загрузка тестовых данных');
    
    // Тестовые заявки для демонстрации
    const testOrders = [
        {
            id: 1,
            type: 'buy',
            cryptocurrency: 'BTC',
            fiat_currency: 'RUB',
            amount: 0.001,
            price: 2800000.00,
            total_amount: 2800.00,
            payment_methods: ['sberbank', 'tinkoff'],
            description: 'Покупаю BTC, быстрая оплата',
            created_at: new Date().toISOString(),
            user: {
                first_name: 'Иван',
                rating: 4.8,
                total_deals: 15
            }
        },
        {
            id: 2,
            type: 'sell',
            cryptocurrency: 'USDT',
            fiat_currency: 'RUB',
            amount: 1000,
            price: 91.50,
            total_amount: 91500.00,
            payment_methods: ['qiwi', 'yandex_money'],
            description: 'Продаю USDT, только проверенным',
            created_at: new Date().toISOString(),
            user: {
                first_name: 'Мария',
                rating: 5.0,
                total_deals: 32
            }
        }
    ];
    
    orders = testOrders;
    renderOrders(orders);
}

// ===== ОТОБРАЖЕНИЕ ДАННЫХ =====
function updateUserInfo() {
    if (!currentUser) return;
    
    const userInfo = document.querySelector('.user-info');
    if (userInfo) {
        userInfo.innerHTML = `
            👤 ${currentUser.first_name} ${currentUser.last_name || ''}
            ⭐ ${currentUser.rating.toFixed(1)} 
            📊 ${currentUser.total_deals} сделок
        `;
    }
}

function renderOrders(ordersList) {
    const content = document.getElementById('ordersContent');
    if (!content) return;
    
    if (ordersList.length === 0) {
        content.innerHTML = `
            <div class="text-center mt-md">
                <p class="text-muted">Заявок пока нет</p>
                <button class="btn btn-primary" onclick="showCreateOrderModal()">
                    Создать первую заявку
                </button>
            </div>
        `;
        return;
    }
    
    const ordersHTML = ordersList.map(order => {
        return `
            <div class="card order-card">
                <div class="order-type ${order.type}">${order.type === 'buy' ? 'Покупка' : 'Продажа'}</div>
                
                <div class="order-header">
                    <div class="crypto-pair">${order.cryptocurrency}/${order.fiat_currency}</div>
                    <div class="price">${formatPrice(order.price, order.fiat_currency)}</div>
                </div>
                
                <div class="order-details">
                    <div class="detail-item">
                        <span class="detail-label">Количество:</span>
                        <span class="detail-value">${order.amount} ${order.cryptocurrency}</span>
                    </div>
                    <div class="detail-item">
                        <span class="detail-label">Сумма:</span>
                        <span class="detail-value">${formatPrice(order.total_amount, order.fiat_currency)}</span>
                    </div>
                </div>
                
                <div class="payment-methods">
                    ${order.payment_methods.map(method => 
                        `<span class="payment-method">${getPaymentMethodName(method)}</span>`
                    ).join('')}
                </div>
                
                ${order.description ? `<p class="text-muted mb-sm">${order.description}</p>` : ''}
                
                <div class="user-rating">
                    <div class="stars">⭐⭐⭐⭐⭐</div>
                    <div class="rating-text">
                        ${order.user ? order.user.first_name : 'Пользователь'} 
                        (${order.user ? order.user.rating.toFixed(1) : '0.0'})
                        • ${order.user ? order.user.total_deals : 0} сделок
                    </div>
                </div>
                
                <button class="btn btn-primary mt-sm" onclick="contactUser(${order.id})">
                    Связаться
                </button>
            </div>
        `;
    }).join('');
    
    content.innerHTML = ordersHTML;
}

// ===== МОДАЛЬНЫЕ ОКНА =====
function showCreateOrderModal() {
    console.log('[INFO] Открытие формы создания заявки');
    showModal('createOrderModal');
}

function showModal(modalId) {
    const modal = document.getElementById(modalId);
    if (modal) {
        modal.classList.add('show');
    }
}

function closeModal(modalId) {
    const modal = document.getElementById(modalId);
    if (modal) {
        modal.classList.remove('show');
    }
}

// ===== ОБРАБОТКА ФОРМ =====
function handleCreateOrder(e) {
    e.preventDefault();
    
    console.log('[INFO] Создание новой заявки');
    
    const form = e.target;
    const formData = new FormData(form);
    
    const orderData = {
        type: formData.get('type'),
        cryptocurrency: formData.get('cryptocurrency'),
        fiat_currency: formData.get('fiat_currency'),
        amount: parseFloat(formData.get('amount')),
        price: parseFloat(formData.get('price')),
        payment_methods: formData.getAll('payment_methods'),
        description: formData.get('description'),
        auto_match: formData.get('auto_match') === 'on'
    };
    
    // Валидация данных
    if (!validateOrderData(orderData)) {
        return;
    }
    
    // Отправка на сервер
    fetch(`${API_BASE}/orders`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(orderData)
    })
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            console.log('[INFO] Заявка успешно создана:', data.order.id);
            showSuccess('Заявка создана успешно!');
            closeModal('createOrderModal');
            form.reset();
            loadOrders(); // Перезагружаем список заявок
        } else {
            console.log('[ERROR] Ошибка создания заявки:', data.error);
            showError(data.error || 'Не удалось создать заявку');
        }
    })
    .catch(error => {
        console.error('[ERROR] Ошибка сети при создании заявки:', error);
        showError('Ошибка соединения с сервером');
    });
}

// ===== ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ =====
function validateOrderData(data) {
    if (!data.type || !data.cryptocurrency || !data.amount || !data.price) {
        showError('Заполните все обязательные поля');
        return false;
    }
    
    if (data.amount <= 0 || data.price <= 0) {
        showError('Количество и цена должны быть больше нуля');
        return false;
    }
    
    if (!data.payment_methods || data.payment_methods.length === 0) {
        showError('Выберите хотя бы один способ оплаты');
        return false;
    }
    
    return true;
}

function formatPrice(price, currency) {
    return new Intl.NumberFormat('ru-RU', {
        style: 'currency',
        currency: currency === 'RUB' ? 'RUB' : 'USD',
        minimumFractionDigits: 2
    }).format(price);
}

function getPaymentMethodName(method) {
    const names = {
        'bank_transfer': 'Банк',
        'sberbank': 'Сбербанк',
        'tinkoff': 'Тинькофф',
        'qiwi': 'QIWI',
        'yandex_money': 'ЮMoney',
        'cash': 'Наличные'
    };
    
    return names[method] || method;
}

function showLoading(containerId) {
    const container = document.getElementById(containerId);
    if (container) {
        container.innerHTML = `
            <div class="loading">
                <div class="spinner"></div>
            </div>
        `;
    }
}

function hideLoading(containerId) {
    // Загрузка скроется при рендере контента
}

function showError(message) {
    console.error('[ERROR]', message);
    
    // Создаем уведомление об ошибке
    const alert = document.createElement('div');
    alert.className = 'alert alert-danger';
    alert.textContent = message;
    
    // Добавляем в начало контейнера
    const container = document.querySelector('.container');
    if (container) {
        container.insertBefore(alert, container.firstChild);
        
        // Убираем через 5 секунд
        setTimeout(() => {
            alert.remove();
        }, 5000);
    }
}

function showSuccess(message) {
    console.log('[SUCCESS]', message);
    
    // Создаем уведомление об успехе
    const alert = document.createElement('div');
    alert.className = 'alert alert-success';
    alert.textContent = message;
    
    // Добавляем в начало контейнера
    const container = document.querySelector('.container');
    if (container) {
        container.insertBefore(alert, container.firstChild);
        
        // Убираем через 3 секунды
        setTimeout(() => {
            alert.remove();
        }, 3000);
    }
}

// Функция для связи с пользователем (заглушка)
function contactUser(orderId) {
    console.log(`[INFO] Связь с пользователем по заявке ${orderId}`);
    showSuccess('Функция в разработке. Скоро будет доступна!');
}

// Заглушки для других функций
function loadMyOrders() { showError('Раздел "Мои заявки" в разработке'); }
function loadDeals() { showError('Раздел "Сделки" в разработке'); }  
function loadProfile() { showError('Раздел "Профиль" в разработке'); }
function applyFilters() { console.log('[INFO] Применение фильтров'); }
