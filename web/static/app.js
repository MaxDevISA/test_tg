
// Глобальные переменные для P2P криптобиржи
let currentUser = null;
let currentInternalUserId = null; // Внутренний ID пользователя в системе
let tg = window.Telegram?.WebApp;

// Инициализация Telegram WebApp
function initTelegramWebApp() {
    if (tg) {
        tg.ready();
        tg.expand();
        tg.disableVerticalSwipes();
        
        // Получаем данные пользователя из Telegram
        if (tg.initDataUnsafe?.user) {
            currentUser = tg.initDataUnsafe.user;
            document.querySelector('.user-info').textContent = 
                '👤 ' + currentUser.first_name + ' ' + (currentUser.last_name || '');
            
            // Автоматически авторизуем пользователя при входе
            authenticateUser();
        } else {
            showError('Ошибка получения данных пользователя из Telegram');
            return;
        }
        
        // Применяем цветовую схему Telegram
        document.body.style.backgroundColor = tg.backgroundColor || '#ffffff';
        
        console.log('[INFO] Telegram WebApp инициализирован', currentUser);
    } else {
        console.warn('[WARN] Telegram WebApp API недоступен - демо режим');
        document.querySelector('.user-info').textContent = '👤 Демо режим';
        // В демо режиме создаем фейкового пользователя для тестирования
        currentUser = {
            id: 123456789,
            first_name: 'Тестовый',
            last_name: 'Пользователь',
            username: 'testuser'
        };
    }
}

// Навигация между разделами
function initNavigation() {
    const navItems = document.querySelectorAll('.nav-item');
    const views = document.querySelectorAll('.view');
    
    navItems.forEach(item => {
        item.addEventListener('click', () => {
            const viewName = item.dataset.view;
            
            // Обновляем активную навигацию
            navItems.forEach(nav => nav.classList.remove('active'));
            item.classList.add('active');
            
            // Показываем нужный раздел
            views.forEach(view => {
                if (view.id === viewName + 'View') {
                    view.style.display = 'block';
                    view.classList.remove('hidden');
                } else {
                    view.style.display = 'none';
                    view.classList.add('hidden');
                }
            });
            
            // Загружаем данные для раздела
            if (viewName === 'orders') {
                loadOrders();
            } else if (viewName === 'my-orders') {
                loadMyOrders();
            } else if (viewName === 'responses') {
                loadResponses();
                initResponseTabs();
            } else if (viewName === 'profile') {
                loadProfile();
            }
        });
    });
}

// Модальное окно
function initModal() {
    const modal = document.getElementById('createOrderModal');
    const createBtn = document.getElementById('createOrderBtn');
    const closeBtn = document.querySelector('.modal-close');
    const form = document.getElementById('createOrderForm');
    
    createBtn.addEventListener('click', () => {
        modal.classList.add('show');
    });
    
    closeBtn.addEventListener('click', () => {
        modal.classList.remove('show');
    });
    
    modal.addEventListener('click', (e) => {
        if (e.target === modal) {
            modal.classList.remove('show');
        }
    });
    
    form.addEventListener('submit', handleCreateOrder);
}

// Создание заявки
async function handleCreateOrder(e) {
    e.preventDefault();
    
    if (!currentUser) {
        showError('Пользователь не авторизован');
        return;
    }
    
    const formData = new FormData(e.target);
    const paymentMethods = [];
    
    // Собираем выбранные способы оплаты
    formData.getAll('payment_methods').forEach(method => {
        paymentMethods.push(method);
    });
    
    const orderData = {
        type: formData.get('type'),
        cryptocurrency: formData.get('cryptocurrency'),
        fiat_currency: formData.get('fiat_currency'),
        amount: parseFloat(formData.get('amount')),
        price: parseFloat(formData.get('price')),
        payment_methods: paymentMethods,
        description: formData.get('description') || '',
        auto_match: formData.has('auto_match')
    };
    
    try {
        const response = await fetch('/api/v1/orders', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'X-Telegram-User-ID': currentUser.id.toString()
            },
            body: JSON.stringify(orderData)
        });
        
        const result = await response.json();
        
        // Отладочная информация
        console.log('[DEBUG] Результат создания заявки:', result);
        
        if (result.success) {
            console.log('[DEBUG] Заявка создана успешно, перезагружаем список...');
            showSuccess('Заявка успешно создана!');
            document.getElementById('createOrderModal').classList.remove('show');
            e.target.reset();
            loadOrders(); // Перезагружаем список заявок
        } else {
            showError(result.error || 'Ошибка создания заявки');
        }
    } catch (error) {
        console.error('[ERROR] Ошибка создания заявки:', error);
        showError('Ошибка сети. Попробуйте позже.');
    }
}

// Загрузка заявок
async function loadOrders() {
    const content = document.getElementById('ordersContent');
    content.innerHTML = '<div class="loading"><div class="spinner"></div><p>Загрузка заявок...</p></div>';
    
    try {
        // Добавляем фильтр status=active чтобы показывались только активные заявки
        const response = await fetch('/api/v1/orders?status=active');
        const result = await response.json();
        
        // Отладочная информация
        console.log('[DEBUG] Ответ сервера на загрузку заявок:', result);
        
        if (result.success) {
            console.log('[DEBUG] Количество заявок:', (result.orders || []).length);
            displayOrders(result.orders || []);
        } else {
            console.log('[DEBUG] Ошибка загрузки заявок:', result.error);
            content.innerHTML = '<p class="text-center text-muted">Ошибка загрузки заявок</p>';
        }
    } catch (error) {
        console.error('[ERROR] Ошибка загрузки заявок:', error);
        content.innerHTML = '<p class="text-center text-muted">Ошибка сети</p>';
    }
}

// Отображение заявок
function displayOrders(orders) {
    const content = document.getElementById('ordersContent');
    
    // Отладочная информация
    console.log('[DEBUG] Отображение заявок:', orders);
    
    if (orders.length === 0) {
        console.log('[DEBUG] Массив заявок пуст');
        content.innerHTML = '<p class="text-center text-muted">Заявок пока нет</p>';
        return;
    }
    
    const ordersHTML = orders.map((order, index) => {
        console.log(`[DEBUG] Обработка заявки ${index}:`, order);
        
        // Проверяем критические поля
        if (!order.type || !order.amount || !order.cryptocurrency || !order.price || !order.fiat_currency) {
            console.log(`[DEBUG] Заявка ${index} имеет пустые обязательные поля:`, {
                type: order.type,
                amount: order.amount, 
                cryptocurrency: order.cryptocurrency,
                price: order.price,
                fiat_currency: order.fiat_currency
            });
        }
        
        // Проверяем не наша ли это заявка
        const isMyOrder = currentInternalUserId && order.user_id === currentInternalUserId;
        
        // Подсчитываем общую сумму сделки
        const totalAmount = order.total_amount || (order.amount * order.price);
        
        // Определяем отображение автора
        const authorName = order.user_name || order.first_name || `Пользователь ${order.user_id}`;
        const authorUsername = order.username; 
        console.log('[DEBUG] Данные автора заявки:', { 
            authorName, 
            authorUsername, 
            user_name: order.user_name,
            first_name: order.first_name,
            username: order.username 
        });
        
        const authorDisplay = authorUsername ? 
            `<span onclick="openTelegramProfile('${authorUsername}')" style="color: var(--tg-theme-link-color, #0088cc); cursor: pointer; text-decoration: underline; font-weight: 500;">@${authorUsername}</span>` :
            `<span style="color: var(--tg-theme-text-color, #000); font-weight: 500;">${authorName}</span>`;
        
        return '<div class="order-card">' +
            '<div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px;">' +
                '<span style="font-weight: 600; color: ' + (order.type === 'buy' ? '#22c55e' : '#ef4444') + ';">' +
                    (order.type === 'buy' ? '🟢 Покупка' : '🔴 Продажа') +
                '</span>' +
                '<span style="font-size: 12px; color: var(--tg-theme-hint-color, #708499);">' +
                    (order.created_at ? new Date(order.created_at).toLocaleString('ru') : 'Дата неизвестна') +
                '</span>' +
            '</div>' +
            
            '<div style="margin-bottom: 10px;">' +
                '<div style="font-size: 14px; margin-bottom: 4px;">👤 Автор: ' + authorDisplay + '</div>' +
            '</div>' +
            
            '<div style="background: var(--tg-theme-secondary-bg-color, rgba(255,255,255,0.1)); border: 1px solid var(--tg-theme-section-separator-color, rgba(255,255,255,0.2)); padding: 10px; border-radius: 6px; margin-bottom: 10px;">' +
                '<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 8px; font-size: 13px;">' +
                    '<div>' +
                        '<span style="color: var(--tg-theme-hint-color, #708499);">📊 Объем:</span><br>' +
                        '<strong style="color: var(--tg-theme-text-color, #000);">' + (order.amount || '?') + ' ' + (order.cryptocurrency || '?') + '</strong>' +
                    '</div>' +
                    '<div>' +
                        '<span style="color: var(--tg-theme-hint-color, #708499);">💰 Курс:</span><br>' +
                        '<strong style="color: var(--tg-theme-text-color, #000);">' + (order.price || '?') + ' ' + (order.fiat_currency || '?') + ' за 1' + (order.cryptocurrency || '?') + '</strong>' +
                    '</div>' +
                '</div>' +
                '<div style="margin-top: 8px; padding-top: 8px; border-top: 1px solid var(--tg-theme-section-separator-color, #e2e8f0); font-size: 13px;">' +
                    '<span style="color: var(--tg-theme-hint-color, #708499);">💵 Общая сумма:</span> ' +
                    '<strong style="color: var(--tg-theme-text-color, #000); font-size: 15px;">' + totalAmount.toLocaleString('ru') + ' ' + (order.fiat_currency || '?') + '</strong>' +
                '</div>' +
            '</div>' +
            
            '<div style="font-size: 12px; color: var(--tg-theme-hint-color, #708499); margin-bottom: 10px;">' +
                '💳 Способы оплаты: ' + ((order.payment_methods || []).join(', ') || 'Не указано') +
            '</div>' +
            
            (order.description ? '<div style="font-size: 12px; margin-bottom: 10px; color: var(--tg-theme-text-color, #000);">' + order.description + '</div>' : '') +
            
            (!isMyOrder ? 
                '<div style="display: flex; gap: 8px; margin-top: 12px;">' +
                    '<button onclick="openUserProfile(' + (order.user_id || 0) + ')" ' +
                           'style="background: var(--tg-theme-hint-color, #6c757d); color: var(--tg-theme-button-text-color, white); border: none; padding: 8px 12px; ' +
                           'border-radius: 4px; font-size: 12px; flex: 1;">👤 Профиль</button>' +
                    '<button onclick="respondToOrder(' + (order.id || 0) + ')" ' +
                           'style="background: var(--tg-theme-button-color, #22c55e); color: var(--tg-theme-button-text-color, white); border: none; padding: 8px 12px; ' +
                           'border-radius: 4px; font-size: 12px; flex: 2;">🤝 Откликнуться</button>' +
                '</div>' : 
                '<div style="display: flex; gap: 8px; margin-top: 12px;">' +
                    '<div style="background: var(--tg-theme-secondary-bg-color, #e8f4fd); border: 1px solid var(--tg-theme-link-color, #007bff); border-radius: 4px; padding: 8px 12px; font-size: 12px; color: var(--tg-theme-link-color, #007bff); flex: 1; text-align: center; font-weight: 500;">📝 Ваша заявка</div>' +
                    '<button onclick="editOrder(' + (order.id || 0) + ')" ' +
                           'style="background: var(--tg-theme-button-color, #f59e0b); color: var(--tg-theme-button-text-color, white); border: none; padding: 8px 12px; ' +
                           'border-radius: 4px; font-size: 12px; flex: 1;">✏️ Редактировать</button>' +
                    '<button onclick="viewOrderResponses(' + (order.id || 0) + ')" ' +
                           'style="background: var(--tg-theme-button-color, #3b82f6); color: var(--tg-theme-button-text-color, white); border: none; padding: 8px 12px; ' +
                           'border-radius: 4px; font-size: 12px; flex: 1;">👀 Отклики (' + (order.response_count || 0) + ')</button>' +
                '</div>'
            ) +
        '</div>';
    }).join('');
    
    content.innerHTML = ordersHTML;
}

// Уведомления
function showSuccess(message) {
    if (tg) {
        tg.showAlert(message);
    } else {
        alert('✅ ' + message);
    }
}

function showError(message) {
    if (tg) {
        tg.showAlert('❌ ' + message);
    } else {
        alert('❌ ' + message);
    }
}

// Универсальная функция для показа уведомлений
function showAlert(message) {
    if (tg) {
        tg.showAlert(message);
    } else {
        alert(message);
    }
}

// Универсальная функция для HTTP запросов
async function apiRequest(url, method = 'GET', data = null) {
    const options = {
        method: method,
        headers: {
            'Content-Type': 'application/json'
        }
    };
    
    // Добавляем Telegram User ID в заголовки для авторизации
    if (currentUser && currentUser.id) {
        options.headers['X-Telegram-User-ID'] = currentUser.id.toString();
    }
    
    // Добавляем данные для POST/PUT запросов
    if (data && (method === 'POST' || method === 'PUT')) {
        options.body = JSON.stringify(data);
    }
    
    try {
        const response = await fetch(url, options);
        const result = await response.json();
        return result;
    } catch (error) {
        console.error('[ERROR] Ошибка HTTP запроса:', error);
        throw error;
    }
}

// Авторизация пользователя через Telegram WebApp
async function authenticateUser() {
    if (!currentUser) {
        showError('Данные пользователя недоступны');
        return;
    }

    try {
        // Подготавливаем данные для авторизации
        const authData = {
            id: currentUser.id,
            first_name: currentUser.first_name || '',
            last_name: currentUser.last_name || '',
            username: currentUser.username || '',
            photo_url: currentUser.photo_url || '',
            auth_date: Math.floor(Date.now() / 1000),
            hash: 'dummy_hash' // В реальном приложении тут будет настоящий hash от Telegram
        };

        const response = await fetch('/api/v1/auth/login', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(authData)
        });

        const result = await response.json();

        if (result.success) {
            console.log('[INFO] Авторизация успешна:', result.user);
            
            // Сохраняем внутренний ID пользователя для проверки "моих заявок"
            currentInternalUserId = result.user.id;
            
            document.querySelector('.user-info').textContent = 
                '👤 ' + result.user.first_name + ' ⭐' + result.user.rating.toFixed(1);
            
            // Загружаем заявки после успешной авторизации
            loadOrders();
        } else {
            if (result.error && result.error.includes('не являетесь членом закрытого чата')) {
                showAccessDenied();
            } else {
                showError('Ошибка авторизации: ' + result.error);
            }
        }
    } catch (error) {
        console.error('[ERROR] Ошибка авторизации:', error);
        showError('Ошибка сети при авторизации');
    }
}

// Показывает сообщение об отказе в доступе
function showAccessDenied() {
    const container = document.querySelector('.container');
    container.innerHTML = `
        <div style="text-align: center; padding: 40px 20px; color: var(--tg-theme-hint-color, #708499);">
            <h2 style="color: var(--tg-theme-text-color, #000); margin-bottom: 16px;">🔒 Доступ ограничен</h2>
            <p style="margin-bottom: 16px; line-height: 1.5;">
                Доступ к P2P криптобирже разрешен только подписчикам закрытого чата.
            </p>
            <p style="font-size: 12px; opacity: 0.8;">
                Подпишитесь на наш закрытый чат, чтобы получить доступ к торговле.
            </p>
        </div>
    `;
    
    // Скрываем навигацию
    document.querySelector('.navigation').style.display = 'none';
}

// Загрузка моих заявок
async function loadMyOrders() {
    console.log('[DEBUG] Запрос загрузки моих заявок');
    
    if (!currentUser) {
        console.log('[DEBUG] Пользователь не авторизован');
        showError('Пользователь не авторизован');
        return;
    }

    console.log('[DEBUG] Загрузка заявок для пользователя:', currentUser);
    const content = document.getElementById('my-ordersView');
    content.innerHTML = '<div class="loading"><div class="spinner"></div><p>Загрузка ваших заявок...</p></div>';
    
    try {
        const response = await fetch('/api/v1/orders/my', {
            headers: {
                'X-Telegram-User-ID': currentUser.id.toString()
            }
        });
        
        const result = await response.json();
        console.log('[DEBUG] loadMyOrders: Ответ сервера:', result);
        
        if (result.success) {
            console.log('[DEBUG] loadMyOrders: Успешно, передаю заявки в displayMyOrders:', result.orders);
            displayMyOrders(result.orders || []);
        } else {
            console.error('[ERROR] loadMyOrders: Ошибка от сервера:', result.error);
            content.innerHTML = '<p class="text-center text-muted">Ошибка загрузки заявок</p>';
        }
    } catch (error) {
        console.error('[ERROR] Ошибка загрузки моих заявок:', error);
        content.innerHTML = '<p class="text-center text-muted">Ошибка сети</p>';
    }
}

// Отображение моих заявок
function displayMyOrders(orders) {
    console.log('[DEBUG] displayMyOrders вызвана с данными:', orders);
    console.log('[DEBUG] Количество заявок для отображения:', orders.length);
    
    const content = document.getElementById('my-ordersView');
    console.log('[DEBUG] displayMyOrders: Элемент my-ordersView найден?', !!content);
    
    if (!content) {
        console.error('[ERROR] displayMyOrders: Элемент my-ordersView не найден!');
        return;
    }
    
    if (orders.length === 0) {
        content.innerHTML = `
            <div style="text-align: center; padding: 40px 20px;">
                <div style="font-size: 48px; margin-bottom: 16px;">📋</div>
                <h3 style="margin-bottom: 12px; color: var(--tg-theme-text-color, #000000);">Мои заявки</h3>
                <p style="color: var(--tg-theme-hint-color, #708499); margin-bottom: 20px; line-height: 1.4;">
                    У вас пока нет заявок.<br/>
                    Создайте первую заявку на покупку или продажу криптовалюты!
                </p>
                <button class="btn btn-primary" id="createFirstOrderBtn" 
                        style="background: var(--tg-theme-button-color, #2481cc); color: var(--tg-theme-button-text-color, #ffffff); border: none; border-radius: 8px; padding: 12px 24px; font-size: 14px; cursor: pointer;">
                    🚀 Создать заявку
                </button>
            </div>
        `;
        
        // Добавляем обработчик для кнопки создания первой заявки
        document.getElementById('createFirstOrderBtn').addEventListener('click', () => {
            document.getElementById('createOrderModal').classList.add('show');
        });
        return;
    }
    
    console.log('[DEBUG] displayMyOrders: Переходим к группировке заявок...');
    
    // Группируем заявки по статусу
    const activeOrders = orders.filter(o => o.status === 'active');
    const inDealOrders = orders.filter(o => o.status === 'matched' || o.status === 'in_progress');
    
    console.log('[DEBUG] displayMyOrders: activeOrders =', activeOrders.length);
    console.log('[DEBUG] displayMyOrders: inDealOrders =', inDealOrders.length);  
    const completedOrders = orders.filter(o => o.status === 'completed');
    const cancelledOrders = orders.filter(o => o.status === 'cancelled');
    
    console.log('[DEBUG] displayMyOrders: completedOrders =', completedOrders.length);
    console.log('[DEBUG] displayMyOrders: cancelledOrders =', cancelledOrders.length);
    console.log('[DEBUG] displayMyOrders: Начинаем формировать HTML...');
    
    let html = `
        <div style="padding: 20px;">
            <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px;">
                <h2 style="margin: 0; color: var(--tg-theme-text-color, #000000);">📋 Мои заявки</h2>
                <button class="btn btn-primary" onclick="document.getElementById('createOrderModal').classList.add('show')" 
                        style="background: var(--tg-theme-button-color, #2481cc); color: var(--tg-theme-button-text-color, #ffffff); border: none; border-radius: 6px; padding: 8px 16px; font-size: 12px;">
                    ➕ Создать
                </button>
            </div>
    `;
    
    // Статистика
    html += `
        <div style="display: grid; grid-template-columns: 1fr 1fr 1fr; gap: 8px; margin-bottom: 20px;">
            <div style="text-align: center; padding: 12px; background: var(--tg-theme-secondary-bg-color, #f8f9fa); border-radius: 8px;">
                <div style="font-size: 18px; font-weight: 600; color: #22c55e;">${activeOrders.length}</div>
                <div style="font-size: 11px; color: var(--tg-theme-hint-color, #708499);">Активные</div>
            </div>
            <div style="text-align: center; padding: 12px; background: var(--tg-theme-secondary-bg-color, #f8f9fa); border-radius: 8px;">
                <div style="font-size: 18px; font-weight: 600; color: #f59e0b;">${inDealOrders.length}</div>
                <div style="font-size: 11px; color: var(--tg-theme-hint-color, #708499);">В сделке</div>
            </div>
            <div style="text-align: center; padding: 12px; background: var(--tg-theme-secondary-bg-color, #f8f9fa); border-radius: 8px;">
                <div style="font-size: 18px; font-weight: 600; color: #3b82f6;">${completedOrders.length}</div>
                <div style="font-size: 11px; color: var(--tg-theme-hint-color, #708499);">Завершено</div>
            </div>
        </div>
    `;
    
    // Активные заявки
    if (activeOrders.length > 0) {
        html += `<div style="margin-bottom: 20px;">
            <h3 style="font-size: 16px; margin-bottom: 12px; color: #22c55e;">🟢 Активные заявки</h3>`;
        
        activeOrders.forEach(order => {
            html += createOrderCard(order, 'active');
        });
        html += `</div>`;
    }
    
    // Заявки в сделке  
    if (inDealOrders.length > 0) {
        html += `<div style="margin-bottom: 20px;">
            <h3 style="font-size: 16px; margin-bottom: 12px; color: #f59e0b;">🤝 В процессе сделки</h3>`;
        
        inDealOrders.forEach(order => {
            html += createOrderCard(order, 'in_deal');
        });
        html += `</div>`;
    }
    
    // Завершенные заявки
    if (completedOrders.length > 0) {
        html += `<div style="margin-bottom: 20px;">
            <h3 style="font-size: 16px; margin-bottom: 12px; color: #3b82f6;">✅ Завершенные</h3>`;
        
        completedOrders.slice(0, 3).forEach(order => { // Показываем только последние 3
            html += createOrderCard(order, 'completed');
        });
        html += `</div>`;
    }
    
    html += `</div>`;
    
    console.log('[DEBUG] displayMyOrders: Готовый HTML длиной', html.length, 'символов');
    console.log('[DEBUG] displayMyOrders: Устанавливаем innerHTML...');
    
    content.innerHTML = html;
    
    console.log('[DEBUG] displayMyOrders: Завершено успешно!');
}

// Создание карточки заявки с действиями
function createOrderCard(order, category) {
    const typeIcon = order.type === 'buy' ? '🟢' : '🔴';
    const typeText = order.type === 'buy' ? 'Покупка' : 'Продажа';
    const typeColor = order.type === 'buy' ? '#22c55e' : '#ef4444';
    
    const statusColors = {
        active: '#22c55e',
        matched: '#f59e0b', 
        in_progress: '#f59e0b',
        completed: '#3b82f6',
        cancelled: '#6b7280'
    };
    
    const totalAmount = (order.amount * order.price).toFixed(2);
    
    let actions = '';
    
    switch (category) {
        case 'active':
            actions = `
                <div style="display: flex; gap: 8px; margin-top: 12px;">
                    <button onclick="editOrder(${order.id})" class="btn-small btn-secondary">
                        ✏️ Редактировать
                    </button>
                    <button onclick="viewOrderResponses(${order.id})" class="btn-small btn-info">
                        👀 Отклики
                    </button>
                    <button onclick="cancelOrder(${order.id})" class="btn-small btn-danger">
                        ❌ Удалить
                    </button>
                </div>
            `;
            break;
        case 'in_deal':
            actions = `
                <div style="display: flex; gap: 8px; margin-top: 12px;">
                    <button onclick="viewActiveDeals(${order.id})" class="btn-small btn-primary">
                        🤝 Перейти к сделке
                    </button>
                    <button onclick="viewOrderResponses(${order.id})" class="btn-small btn-info">
                        👀 Все отклики
                    </button>
                </div>
            `;
            break;
        case 'completed':
            actions = `
                <div style="margin-top: 12px;">
                    <button onclick="viewOrderHistory(${order.id})" class="btn-small btn-secondary">
                        📊 История
                    </button>
                </div>
            `;
            break;
    }
    
    return `
        <div class="order-card" style="border: 1px solid var(--tg-theme-section-separator-color, #e1e8ed); border-radius: 12px; padding: 16px; margin-bottom: 12px; background: var(--tg-theme-bg-color, #ffffff);">
            <div style="display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 12px;">
                <div>
                    <div style="font-weight: 600; color: ${typeColor}; margin-bottom: 4px;">
                        ${typeIcon} ${typeText}
                    </div>
                    <div style="font-size: 18px; font-weight: 700; color: var(--tg-theme-text-color, #000000);">
                        ${order.amount} ${order.cryptocurrency}
                    </div>
                    <div style="font-size: 14px; color: var(--tg-theme-hint-color, #708499);">
                        по ${order.price} ${order.fiat_currency} = ${totalAmount} ${order.fiat_currency}
                    </div>
                </div>
                <div style="text-align: right;">
                    <div style="font-size: 11px; padding: 4px 8px; border-radius: 12px; background: ${statusColors[order.status]}; color: white; margin-bottom: 4px;">
                        ${getStatusText(order.status)}
                    </div>
                    <div style="font-size: 10px; color: var(--tg-theme-hint-color, #708499);">
                        ID: ${order.id}
                    </div>
                </div>
            </div>
            
            ${order.description ? `
            <div style="font-size: 12px; color: var(--tg-theme-text-color, #000000); margin-bottom: 8px; padding: 8px; background: var(--tg-theme-secondary-bg-color, #f8f9fa); border-radius: 6px;">
                💬 ${order.description}
            </div>
            ` : ''}
            
            <div style="display: flex; justify-content: space-between; font-size: 11px; color: var(--tg-theme-hint-color, #708499); margin-bottom: 8px;">
                <span>💳 ${Array.isArray(order.payment_methods) ? order.payment_methods.join(', ') : order.payment_methods || 'Любой способ'}</span>
                <span>📅 ${new Date(order.created_at).toLocaleDateString('ru')}</span>
            </div>
            
            ${actions}
        </div>
    `;
}

// Загрузка сделок
async function loadDeals() {
    if (!currentUser) {
        showError('Пользователь не авторизован');
        return;
    }

    const content = document.getElementById('dealsView');
    content.innerHTML = '<div class="loading"><div class="spinner"></div><p>Загрузка сделок...</p></div>';
    
    try {
        const response = await fetch('/api/v1/deals', {
            headers: {
                'X-Telegram-User-ID': currentUser.id.toString()
            }
        });
        
        const result = await response.json();
        
        if (result.success) {
            displayDeals(result.deals || []);
        } else {
            content.innerHTML = '<p class="text-center text-muted">Ошибка загрузки сделок</p>';
        }
    } catch (error) {
        console.error('[ERROR] Ошибка загрузки сделок:', error);
        content.innerHTML = '<p class="text-center text-muted">Ошибка сети</p>';
    }
}

// Отображение сделок
function displayDeals(deals) {
    const content = document.getElementById('dealsView');
    
    if (deals.length === 0) {
        content.innerHTML = `
            <div class="text-center mt-md">
                <h2>История сделок</h2>
                <p class="text-muted">У вас пока нет завершенных сделок</p>
            </div>
        `;
        return;
    }
    
    let html = '<h2 style="margin-bottom: 16px;">История сделок</h2>';
    
    deals.forEach(deal => {
        const statusColor = deal.status === 'pending' ? '#f59e0b' : 
                           deal.status === 'completed' ? '#22c55e' : '#ef4444';
        
        html += `
            <div style="border: 1px solid var(--tg-theme-section-separator-color, #e1e8ed); 
                        border-radius: 8px; padding: 12px; margin-bottom: 12px;
                        background: var(--tg-theme-bg-color, #ffffff);">
                <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px;">
                    <span style="font-weight: 600;">
                        💼 Сделка #${deal.id}
                    </span>
                    <span style="font-size: 12px; padding: 4px 8px; border-radius: 12px; background: ${statusColor}; color: white;">
                        ${getDealStatusText(deal.status)}
                    </span>
                </div>
                <div style="margin-bottom: 8px;">
                    <strong>${deal.amount} ${deal.cryptocurrency}</strong> за <strong>${deal.total_amount} ${deal.fiat_currency}</strong>
                </div>
                <div style="font-size: 12px; color: var(--tg-theme-hint-color, #708499);">
                    Создано: ${new Date(deal.created_at).toLocaleString('ru')}
                </div>
            </div>
        `;
    });
    
    content.innerHTML = html;
}

// Загрузка профиля
async function loadProfile() {
    if (!currentUser) {
        showError('Пользователь не авторизован');
        return;
    }

    console.log('[DEBUG] Загрузка профиля для пользователя:', currentUser);
    
    const content = document.getElementById('profileView');
    content.innerHTML = '<div class="loading"><div class="spinner"></div><p>Загрузка профиля...</p></div>';
    
    try {
        console.log('[DEBUG] Отправка запросов для профиля...');
        
        // Получаем данные пользователя и статистику параллельно
        const [userResponse, statsResponse, reviewsResponse] = await Promise.all([
            fetch('/api/v1/auth/me', {
                headers: { 'X-Telegram-User-ID': currentUser.id.toString() }
            }).catch(err => {
                console.error('[DEBUG] Ошибка запроса /auth/me:', err);
                return null;
            }),
            fetch('/api/v1/auth/stats', {
                headers: { 'X-Telegram-User-ID': currentUser.id.toString() }
            }).catch(err => {
                console.error('[DEBUG] Ошибка запроса /auth/stats:', err);
                return null;
            }),
            fetch('/api/v1/auth/reviews?limit=5', {
                headers: { 'X-Telegram-User-ID': currentUser.id.toString() }
            }).catch(err => {
                console.error('[DEBUG] Ошибка запроса /auth/reviews:', err);
                return null;
            })
        ]);

        let userData = currentUser;
        let userStats = null;
        let userReviews = [];

        console.log('[DEBUG] Статусы ответов:', {
            user: userResponse ? userResponse.status : 'null',
            stats: statsResponse ? statsResponse.status : 'null', 
            reviews: reviewsResponse ? reviewsResponse.status : 'null'
        });

        // Парсим ответы
        if (userResponse && userResponse.ok) {
            const userResult = await userResponse.json();
            console.log('[DEBUG] Данные пользователя:', userResult);
            userData = userResult.user || currentUser;
        } else if (userResponse) {
            console.error('[DEBUG] Ошибка получения данных пользователя:', userResponse.status, await userResponse.text().catch(() => 'no text'));
        }

        if (statsResponse && statsResponse.ok) {
            const statsResult = await statsResponse.json();
            console.log('[DEBUG] Статистика пользователя:', statsResult);
            userStats = statsResult.stats;
        } else if (statsResponse) {
            console.error('[DEBUG] Ошибка получения статистики:', statsResponse.status, await statsResponse.text().catch(() => 'no text'));
        }

        if (reviewsResponse && reviewsResponse.ok) {
            const reviewsResult = await reviewsResponse.json();
            console.log('[DEBUG] Отзывы пользователя:', reviewsResult);
            userReviews = reviewsResult.reviews || [];
        } else if (reviewsResponse) {
            console.error('[DEBUG] Ошибка получения отзывов:', reviewsResponse.status, await reviewsResponse.text().catch(() => 'no text'));
        }

        console.log('[DEBUG] Передача данных в displayMyProfile:', { userData, userStats, userReviews });
        
        // Дополнительная отладка для пустого профиля
        if (!userStats || (userStats.total_orders === 0 && userStats.total_deals === 0)) {
            console.warn('[DEBUG] У пользователя пустая статистика!', userStats);
            console.log('[DEBUG] Данные пользователя:', userData);
            console.log('[DEBUG] currentInternalUserId:', currentInternalUserId);
            console.log('[DEBUG] Должен отображаться экран приветствия для нового пользователя');
        }
        
        console.log('[DEBUG] Вызываем displayMyProfile...');
        displayMyProfile(userData, userStats, userReviews);
        console.log('[DEBUG] displayMyProfile завершена');
    } catch (error) {
        console.error('[ERROR] Ошибка загрузки профиля:', error);
        displayMyProfile(currentUser, null, []);
    }
}

// Получение пользователя по Telegram ID (вспомогательная функция)
async function getUserByTelegramID() {
    try {
        const response = await fetch('/api/v1/auth/me', {
            headers: { 'X-Telegram-User-ID': currentUser.id.toString() }
        });
        
        if (response.ok) {
            const result = await response.json();
            return result.user;
        }
    } catch (error) {
        console.error('[ERROR] Ошибка получения пользователя:', error);
    }
    return null;
}

// Отображение профиля с отзывами
function displayProfileWithReviews(user, reviews, stats) {
    const content = document.getElementById('profileView');
    
    const rating = stats?.average_rating || user.rating || 0;
    const totalReviews = stats?.total_reviews || 0;
    const stars = '⭐'.repeat(Math.floor(rating)) + '☆'.repeat(5 - Math.floor(rating));
    const positivePercent = stats?.positive_percent || 0;
    
    let html = `
        <div style="padding: 16px;">
            <h2 style="margin-bottom: 16px; text-align: center;">👤 Мой профиль</h2>
            
            <!-- Основная информация -->
            <div style="text-align: center; margin-bottom: 24px;">
                <div style="font-size: 18px; font-weight: 600; margin-bottom: 8px;">
                    ${user.first_name} ${user.last_name || ''}
                </div>
                <div style="font-size: 14px; color: var(--tg-theme-hint-color, #708499); margin-bottom: 12px;">
                    @${user.username || 'пользователь'}
                </div>
                <div style="font-size: 16px; margin-bottom: 8px;">
                    ${stars} ${rating.toFixed(1)} (${totalReviews} отзывов)
                </div>
                ${positivePercent > 0 ? `
                <div style="font-size: 12px; color: #22c55e;">
                    ${positivePercent.toFixed(0)}% положительных отзывов
                </div>` : ''}
            </div>
            
            <!-- Статистика -->
            <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 12px; margin-bottom: 24px;">
                <div style="text-align: center; padding: 12px; border: 1px solid var(--tg-theme-section-separator-color, #e1e8ed); border-radius: 8px;">
                    <div style="font-size: 18px; font-weight: 600; color: var(--tg-theme-link-color, #2481cc);">
                        ${user.total_deals || 0}
                    </div>
                    <div style="font-size: 11px; color: var(--tg-theme-hint-color, #708499);">
                        Всего сделок
                    </div>
                </div>
                <div style="text-align: center; padding: 12px; border: 1px solid var(--tg-theme-section-separator-color, #e1e8ed); border-radius: 8px;">
                    <div style="font-size: 18px; font-weight: 600; color: #22c55e;">
                        ${user.successful_deals || 0}
                    </div>
                    <div style="font-size: 11px; color: var(--tg-theme-hint-color, #708499);">
                        Успешных
                    </div>
                </div>
            </div>
    `;
    
    // Отзывы
    if (reviews && reviews.length > 0) {
        html += `
            <div style="margin-bottom: 16px;">
                <h3 style="font-size: 16px; margin-bottom: 12px;">📝 Последние отзывы</h3>
        `;
        
        reviews.forEach(review => {
            const reviewStars = '⭐'.repeat(review.rating) + '☆'.repeat(5 - review.rating);
            const reviewDate = new Date(review.created_at).toLocaleDateString('ru');
            
            html += `
                <div style="border: 1px solid var(--tg-theme-section-separator-color, #e1e8ed); 
                            border-radius: 8px; padding: 12px; margin-bottom: 8px;
                            background: var(--tg-theme-bg-color, #ffffff);">
                    <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px;">
                        <span style="font-size: 14px;">${reviewStars}</span>
                        <span style="font-size: 11px; color: var(--tg-theme-hint-color, #708499);">
                            ${reviewDate}
                        </span>
                    </div>
                    ${review.comment ? `
                    <div style="font-size: 13px; line-height: 1.4;">
                        ${review.comment}
                    </div>
                    ` : ''}
                    ${!review.is_anonymous && review.from_user_name ? `
                    <div style="font-size: 11px; color: var(--tg-theme-hint-color, #708499); margin-top: 6px;">
                        От: ${review.from_user_username ? '@' + review.from_user_username : review.from_user_name}
                    </div>
                    ` : review.is_anonymous ? `
                    <div style="font-size: 11px; color: var(--tg-theme-hint-color, #708499); margin-top: 6px;">
                        Анонимный отзыв
                    </div>
                    ` : ''}
                </div>
            `;
        });
        
        html += `</div>`;
    } else if (totalReviews === 0) {
        html += `
            <div style="text-center; padding: 20px; color: var(--tg-theme-hint-color, #708499);">
                📝 Пока нет отзывов
            </div>
        `;
    }
    
    html += `
            <div style="font-size: 11px; color: var(--tg-theme-hint-color, #708499); text-align: center; margin-top: 16px;">
                Telegram ID: ${user.telegram_id || user.id}
            </div>
        </div>
    `;
    
    content.innerHTML = html;
}

// Отображение профиля (упрощенная версия без отзывов)
function displayProfile(user) {
    displayProfileWithReviews(user, [], null);
}

// Отображение моего профиля с полной статистикой
function displayMyProfile(user, stats, reviews) {
    console.log('[DEBUG] displayMyProfile вызвана с:', { user, stats, reviews });
    
    try {
        const content = document.getElementById('profileView');
        if (!content) {
            console.error('[ERROR] Элемент profileView не найден в DOM!');
            return;
        }
        
        // Проверяем что user содержит данные, а не ошибку
        if (!user || user.message) {
            console.error('[ERROR] Данные пользователя содержат ошибку:', user);
            user = currentUser; // Используем fallback
        }
        
        // Данные пользователя
        const avatarUrl = user.photo_url || '';
        const userName = user.first_name + (user.last_name ? ` ${user.last_name}` : '');
        const username = user.username ? `@${user.username}` : '';
        
        // Статистика рейтинга  
        const rating = stats?.average_rating || user.rating || 0;
        const totalReviews = stats?.total_reviews || 0;
        const stars = '⭐'.repeat(Math.floor(rating)) + '☆'.repeat(5 - Math.floor(rating));
        
        // Проверяем есть ли статистика
        const hasStats = stats && (stats.total_orders > 0 || stats.total_deals > 0 || totalReviews > 0);
        
        console.log('[DEBUG] hasStats =', hasStats, 'на основе stats =', stats, 'totalReviews =', totalReviews);
        console.log('[DEBUG] Начинаем формировать HTML...');
    
    let html = `
        <div style="padding: 20px;">
            <!-- Заголовок -->
            <div style="text-align: center; margin-bottom: 24px;">
                
                <!-- Аватар -->
                <div style="margin-bottom: 16px;">
                    ${avatarUrl ? 
                        `<img src="${avatarUrl}" style="width: 80px; height: 80px; border-radius: 50%; border: 3px solid var(--tg-theme-link-color, #2481cc);" alt="Аватар">` :
                        `<div style="width: 80px; height: 80px; border-radius: 50%; background: var(--tg-theme-link-color, #2481cc); display: flex; align-items: center; justify-content: center; margin: 0 auto; font-size: 32px; color: white;">
                            ${user.first_name ? user.first_name[0].toUpperCase() : '👤'}
                        </div>`
                    }
                </div>
                
                <!-- Имя и username -->
                <div style="margin-bottom: 12px;">
                    <div style="font-size: 20px; font-weight: 600; margin-bottom: 4px; color: var(--tg-theme-text-color, #000000);">
                        ${userName}
                    </div>
                    ${username ? `
                    <div style="font-size: 14px; color: var(--tg-theme-hint-color, #708499);">
                        ${username}
                    </div>` : ''}
                </div>
                
                <!-- Рейтинг -->
                <div style="font-size: 16px; margin-bottom: 8px;">
                    ${stars} ${rating.toFixed(1)}
                </div>
                <div style="font-size: 13px; color: var(--tg-theme-hint-color, #708499);">
                    ${totalReviews} отзыв${totalReviews === 1 ? '' : totalReviews > 4 ? 'ов' : 'а'}
                </div>
            </div>
            
            <!-- Статистика сделок или сообщение о начале работы -->
            ${hasStats ? `
            <div class="profile-stats-grid" style="margin-bottom: 24px;">
                <div class="profile-stat-card">
                    <div class="profile-stat-number" style="color: #22c55e;">${stats?.completed_deals || 0}</div>
                    <div class="profile-stat-label">Завершено сделок</div>
                </div>
                <div class="profile-stat-card">
                    <div class="profile-stat-number" style="color: #f59e0b;">${stats?.active_orders || 0}</div>
                    <div class="profile-stat-label">Активных заявок</div>
                </div>
                <div class="profile-stat-card">
                    <div class="profile-stat-number" style="color: #3b82f6;">${stats?.total_orders || 0}</div>
                    <div class="profile-stat-label">Всего заявок</div>
                </div>
                <div class="profile-stat-card">
                    <div class="profile-stat-number" style="color: #8b5cf6;">
                        ${stats?.success_rate ? stats.success_rate.toFixed(0) + '%' : '0%'}
                    </div>
                    <div class="profile-stat-label">Успешность</div>
                </div>
            </div>` : `
            <div style="background: linear-gradient(135deg, var(--tg-theme-secondary-bg-color, #f8f9fa) 0%, var(--tg-theme-bg-color, #ffffff) 100%); border-radius: 16px; padding: 24px; margin-bottom: 24px; text-align: center; border: 1px solid var(--tg-theme-section-separator-color, #e1e8ed);">
                <div style="font-size: 32px; margin-bottom: 12px;">🚀</div>
                <div style="font-size: 18px; font-weight: 600; margin-bottom: 8px; color: var(--tg-theme-text-color, #000000);">
                    Добро пожаловать на биржу!
                </div>
                <div style="font-size: 14px; color: var(--tg-theme-hint-color, #708499); line-height: 1.4;">
                    Пока у вас нет заявок и сделок.<br/>
                    Создайте первую заявку и начните торговать!
                </div>
                <div style="margin-top: 16px;">
                    <button onclick="showView('orders')" style="background: var(--tg-theme-button-color, #2481cc); color: var(--tg-theme-button-text-color, #ffffff); border: none; border-radius: 8px; padding: 10px 20px; font-size: 14px; cursor: pointer;">
                        📋 Перейти к заявкам
                    </button>
                </div>
            </div>
            `}
            

    `;
    
    // Отзывы
    if (reviews && reviews.length > 0) {
        html += `
            <div class="profile-reviews-section">
                <div class="profile-reviews-title">📝 Последние отзывы обо мне</div>
        `;
        
        reviews.slice(0, 3).forEach(review => {
            const reviewStars = '⭐'.repeat(review.rating) + '☆'.repeat(5 - review.rating);
            const reviewDate = new Date(review.created_at).toLocaleDateString('ru');
            
            html += `
                <div class="profile-review-card">
                    <div class="profile-review-header">
                        <span class="profile-review-stars">${reviewStars}</span>
                        <span class="profile-review-date">${reviewDate}</span>
                    </div>
                    ${review.comment ? `
                    <div class="profile-review-comment">
                        ${review.comment}
                    </div>
                    ` : ''}
                    ${!review.is_anonymous && review.from_user_name ? `
                    <div class="profile-review-author" style="font-size: 11px; color: var(--tg-theme-hint-color, #708499); margin-top: 6px;">
                        От: ${review.from_user_username ? '@' + review.from_user_username : review.from_user_name}
                    </div>
                    ` : review.is_anonymous ? `
                    <div class="profile-review-author" style="font-size: 11px; color: var(--tg-theme-hint-color, #708499); margin-top: 6px;">
                        Анонимный отзыв
                    </div>
                    ` : ''}
                </div>
            `;
        });
        
        html += `</div>`;
    } else if (hasStats) {
        html += `
            <div style="text-align: center; padding: 20px; color: var(--tg-theme-hint-color, #666); font-size: 13px;">
                📝 Пока нет отзывов обо мне
            </div>
        `;
    }
    
    // Дата регистрации
    html += `
            <div style="margin-top: 24px; text-align: center; padding-top: 16px; border-top: 1px solid var(--tg-theme-section-separator-color, #e1e8ed); font-size: 12px; color: var(--tg-theme-hint-color, #708499);">
                🗓️ Участник с ${new Date(user.created_at || Date.now()).toLocaleDateString('ru')}
                ${stats?.first_deal_date ? ` • Первая сделка: ${new Date(stats.first_deal_date).toLocaleDateString('ru')}` : ''}
            </div>
        </div>
    `;
    
    console.log('[DEBUG] HTML сформирован, длина:', html.length, 'символов');
    console.log('[DEBUG] Устанавливаем innerHTML для элемента:', content);
    
    if (!content) {
        console.error('[ERROR] Элемент profileView не найден!');
        return;
    }
    
    content.innerHTML = html;
    
    // Убираем класс hidden чтобы профиль был виден
    content.classList.remove('hidden');
    console.log('[DEBUG] Убран класс hidden, профиль теперь должен быть видим');
    
    console.log('[DEBUG] innerHTML установлен, профиль должен отображаться');
    
    // Проверяем что контент действительно установлен
    setTimeout(() => {
        if (content.innerHTML.length > 0) {
            console.log('[DEBUG] Профиль успешно отображен!');
        } else {
            console.error('[ERROR] Профиль не отобразился - innerHTML пустой');
        }
    }, 100);
    
    } catch (error) {
        console.error('[ERROR] Ошибка в displayMyProfile:', error);
        console.error('[ERROR] Stack trace:', error.stack);
        
        // Показываем базовое сообщение об ошибке
        const content = document.getElementById('profileView');
        if (content) {
            content.innerHTML = `
                <div style="padding: 20px; text-align: center;">
                    <h2>⚠️ Ошибка загрузки профиля</h2>
                    <p style="color: #666; margin-top: 10px;">
                        Произошла ошибка при отображении профиля.<br/>
                        Попробуйте обновить страницу.
                    </p>
                </div>
            `;
        }
    }
}

// Вспомогательные функции
function getStatusText(status) {
    const statusMap = {
        'active': 'Активна',
        'matched': 'Сопоставлена',
        'completed': 'Завершена',
        'cancelled': 'Отменена'
    };
    return statusMap[status] || status;
}

function getDealStatusText(status) {
    const statusMap = {
        'pending': 'Ожидает',
        'completed': 'Завершена',
        'cancelled': 'Отменена'
    };
    return statusMap[status] || status;
}

// Отмена заявки
async function cancelOrder(orderId) {
    if (!currentUser) {
        showError('Пользователь не авторизован');
        return;
    }
    
    if (!confirm('Вы уверены что хотите отменить эту заявку?')) {
        return;
    }
    
    try {
        const response = await fetch(`/api/v1/orders/${orderId}`, {
            method: 'DELETE',
            headers: {
                'X-Telegram-User-ID': currentUser.id.toString()
            }
        });
        
        const result = await response.json();
        
        if (result.success) {
            showSuccess('Заявка успешно отменена');
            loadMyOrders(); // Перезагружаем список заявок
        } else {
            showError('Ошибка отмены заявки: ' + result.error);
        }
    } catch (error) {
        console.error('[ERROR] Ошибка отмены заявки:', error);
        showError('Ошибка сети при отмене заявки');
    }
}

// Создание отзыва
async function createReview(dealId, toUserId, rating, comment, isAnonymous) {
    if (!currentUser) {
        showError('Пользователь не авторизован');
        return;
    }
    
    try {
        const response = await fetch('/api/v1/reviews', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'X-Telegram-User-ID': currentUser.id.toString()
            },
            body: JSON.stringify({
                deal_id: dealId,
                to_user_id: toUserId,
                rating: rating,
                comment: comment,
                is_anonymous: isAnonymous
            })
        });
        
        const result = await response.json();
        
        if (result.success) {
            showSuccess('Отзыв успешно создан!');
            loadDeals(); // Обновляем список сделок
            closeReviewModal();
        } else {
            showError('Ошибка создания отзыва: ' + (result.error || 'Неизвестная ошибка'));
        }
    } catch (error) {
        console.error('[ERROR] Ошибка создания отзыва:', error);
        showError('Ошибка сети при создании отзыва');
    }
}

// Открытие модального окна для создания отзыва
function openReviewModal(dealId, toUserId) {
    const modal = document.getElementById('reviewModal');
    if (modal) {
        modal.classList.add('show');
        
        // Сохраняем данные в модальном окне
        modal.dataset.dealId = dealId;
        modal.dataset.toUserId = toUserId;
        
        // Очищаем форму
        document.getElementById('reviewRating').value = '5';
        document.getElementById('reviewComment').value = '';
        document.getElementById('reviewAnonymous').checked = false;
        
        // Обновляем звездочки
        updateStarRating(5);
    }
}

// Закрытие модального окна отзыва
function closeReviewModal() {
    const modal = document.getElementById('reviewModal');
    if (modal) {
        modal.classList.remove('show');
    }
}

// Обновление визуального отображения звездного рейтинга
function updateStarRating(rating) {
    const stars = document.querySelectorAll('.star-rating .star');
    stars.forEach((star, index) => {
        if (index < rating) {
            star.textContent = '⭐';
            star.classList.add('active');
        } else {
            star.textContent = '☆';
            star.classList.remove('active');
        }
    });
}

// Обработчик отправки отзыва
function handleReviewSubmit() {
    const modal = document.getElementById('reviewModal');
    const dealId = parseInt(modal.dataset.dealId);
    const toUserId = parseInt(modal.dataset.toUserId);
    const rating = parseInt(document.getElementById('reviewRating').value);
    const comment = document.getElementById('reviewComment').value.trim();
    const isAnonymous = document.getElementById('reviewAnonymous').checked;
    
    // Валидация
    if (rating < 1 || rating > 5) {
        showError('Рейтинг должен быть от 1 до 5 звезд');
        return;
    }
    
    if (rating <= 2 && !comment) {
        showError('Для оценки 1-2 звезды необходимо указать комментарий');
        return;
    }
    
    if (comment.length > 500) {
        showError('Комментарий не должен превышать 500 символов');
        return;
    }
    
    createReview(dealId, toUserId, rating, comment, isAnonymous);
}

// Инициализация приложения
document.addEventListener('DOMContentLoaded', () => {
    initTelegramWebApp();
    initNavigation();
    initModal();
    initReviewModal();
});

// Инициализация модального окна для отзывов
function initReviewModal() {
    // Создаем HTML для модального окна отзыва если его еще нет
    if (!document.getElementById('reviewModal')) {
        const modalHTML = `
            <div id="reviewModal" class="modal" style="display: none;">
                <div class="modal-content" style="max-width: 400px;">
                    <div class="modal-header">
                        <h2>📝 Оставить отзыв</h2>
                        <span class="close" onclick="closeReviewModal()">&times;</span>
                    </div>
                    <div class="modal-body">
                        <div style="margin-bottom: 16px;">
                            <label style="display: block; margin-bottom: 8px; font-weight: 600;">Рейтинг:</label>
                            <div class="star-rating" style="font-size: 24px; margin-bottom: 8px;">
                                <span class="star active" data-rating="1" onclick="setRating(1)">⭐</span>
                                <span class="star active" data-rating="2" onclick="setRating(2)">⭐</span>
                                <span class="star active" data-rating="3" onclick="setRating(3)">⭐</span>
                                <span class="star active" data-rating="4" onclick="setRating(4)">⭐</span>
                                <span class="star active" data-rating="5" onclick="setRating(5)">⭐</span>
                            </div>
                            <input type="hidden" id="reviewRating" value="5">
                        </div>
                        
                        <div style="margin-bottom: 16px;">
                            <label for="reviewComment" style="display: block; margin-bottom: 8px; font-weight: 600;">
                                Комментарий:
                            </label>
                            <textarea id="reviewComment" 
                                    style="width: 100%; height: 80px; padding: 8px; border: 1px solid #ddd; border-radius: 4px; resize: vertical;" 
                                    placeholder="Опишите ваш опыт сотрудничества (необязательно для оценок 3-5 звезд)"
                                    maxlength="500"></textarea>
                            <div style="font-size: 12px; color: #666; text-align: right;">
                                <span id="commentLength">0</span>/500
                            </div>
                        </div>
                        
                        <div style="margin-bottom: 16px;">
                            <label style="display: flex; align-items: center; font-size: 14px;">
                                <input type="checkbox" id="reviewAnonymous" style="margin-right: 8px;">
                                Анонимный отзыв
                            </label>
                        </div>
                    </div>
                    <div class="modal-footer">
                        <button type="button" onclick="closeReviewModal()" 
                                style="background: #6c757d; color: white; border: none; padding: 8px 16px; border-radius: 4px; margin-right: 8px;">
                            Отмена
                        </button>
                        <button type="button" onclick="handleReviewSubmit()" 
                                style="background: #22c55e; color: white; border: none; padding: 8px 16px; border-radius: 4px;">
                            Отправить отзыв
                        </button>
                    </div>
                </div>
            </div>
        `;
        
        document.body.insertAdjacentHTML('beforeend', modalHTML);
        
        // Добавляем обработчик для подсчета символов
        const commentField = document.getElementById('reviewComment');
        const lengthCounter = document.getElementById('commentLength');
        
        commentField.addEventListener('input', () => {
            lengthCounter.textContent = commentField.value.length;
        });
    }
}

// Установка рейтинга при клике на звезду
function setRating(rating) {
    document.getElementById('reviewRating').value = rating;
    updateStarRating(rating);
}

// =====================================================
// ОТКЛИКИ НА ЗАЯВКИ И ПРОСМОТР ПРОФИЛЕЙ
// =====================================================

// Открытие профиля пользователя
async function openUserProfile(userId) {
    if (!userId || userId === 0) {
        showError('Неверный ID пользователя');
        return;
    }

    console.log('[DEBUG] Открытие профиля пользователя ID:', userId);
    
    // Создаем модальное окно программно если его нет
    if (!document.getElementById('profileModal')) {
        createProfileModal();
    }
    
    const modal = document.getElementById('profileModal');
    const content = document.getElementById('profileModalContent');
    
    // Показываем модальное окно
    modal.classList.add('show');
    
    // Показываем загрузку
    content.innerHTML = `
        <div class="loading">
            <div class="spinner"></div>
            <p>Загрузка профиля пользователя...</p>
        </div>
    `;
    
    try {
        // Загружаем профиль пользователя
        const [profileResponse, reviewsResponse] = await Promise.all([
            fetch(`/api/v1/users/${userId}/profile`, {
                headers: { 'X-Telegram-User-ID': currentUser.id.toString() }
            }),
            fetch(`/api/v1/reviews?user_id=${userId}&limit=5`, {
                headers: { 'X-Telegram-User-ID': currentUser.id.toString() }
            }).catch(() => null)
        ]);

        let profileData = null;
        let reviews = [];

        if (profileResponse && profileResponse.ok) {
            const result = await profileResponse.json();
            profileData = result.profile || result;
        }

        if (reviewsResponse && reviewsResponse.ok) {
            const result = await reviewsResponse.json();
            reviews = result.reviews || [];
        }

        displayUserProfileModal(profileData, reviews);
        
    } catch (error) {
        console.error('[ERROR] Ошибка загрузки профиля:', error);
        content.innerHTML = `
            <div class="text-center" style="padding: 20px;">
                <p>❌ Ошибка загрузки профиля</p>
                <p style="font-size: 12px; color: #666;">Попробуйте позже</p>
            </div>
        `;
    }
}

// Отображение профиля в модальном окне
function displayUserProfileModal(profileData, reviews) {
    const content = document.getElementById('profileModalContent');
    
    if (!profileData) {
        content.innerHTML = `
            <div class="text-center" style="padding: 20px;">
                <p>❌ Профиль не найден</p>
            </div>
        `;
        return;
    }

    // Информация о пользователе
    const user = profileData.user || {};
    const stats = profileData.stats || profileData;
    
    // Используем данные из stats если profileData содержит только статистику (для обратной совместимости)
    const userId = user.id || stats.user_id;
    const rating = stats.average_rating || 0;
    const totalReviews = stats.total_reviews || 0;
    const positivePercent = stats.positive_percent || 0;
    const stars = '⭐'.repeat(Math.floor(rating)) + '☆'.repeat(5 - Math.floor(rating));
    
    // Формируем отображаемое имя пользователя
    let userDisplayName = `Пользователь #${userId}`;
    if (user.username) {
        userDisplayName = `@${user.username}`;
    } else if (user.first_name) {
        userDisplayName = user.first_name;
        if (user.last_name) {
            userDisplayName += ` ${user.last_name}`;
        }
    }

    let html = `
        <div class="text-center" style="margin-bottom: 20px;">
            <div style="font-size: 18px; font-weight: 600; margin-bottom: 8px; color: var(--tg-theme-text-color, #000000);">
                ${user.username ? 
                    `<a href="https://t.me/${user.username}" target="_blank" style="color: var(--tg-theme-link-color, #2481cc); text-decoration: none;">
                        ${userDisplayName}
                    </a>` : 
                    userDisplayName
                }
            </div>
            <div style="font-size: 16px; margin-bottom: 8px;">
                ${stars} ${rating.toFixed(1)} (${totalReviews} отзывов)
            </div>
            ${positivePercent > 0 ? `
            <div style="font-size: 13px; color: #22c55e;">
                ${positivePercent.toFixed(0)}% положительных отзывов
            </div>` : ''}
        </div>
        
        <div class="profile-stats-grid">
            <div class="profile-stat-card">
                <div class="profile-stat-number">${totalReviews}</div>
                <div class="profile-stat-label">Всего отзывов</div>
            </div>
            <div class="profile-stat-card">
                <div class="profile-stat-number" style="color: #22c55e;">${Math.round(positivePercent)}%</div>
                <div class="profile-stat-label">Положительных</div>
            </div>
        </div>
    `;
    
    // Отзывы (используем переданные reviews или recent_reviews из stats)
    const reviewsToShow = reviews && reviews.length > 0 ? reviews : (stats.recent_reviews || []);
    
    if (reviewsToShow.length > 0) {
        html += `
            <div class="profile-reviews-section">
                <div class="profile-reviews-title">📝 Последние отзывы</div>
        `;
        
        reviewsToShow.slice(0, 3).forEach(review => {
            const reviewStars = '⭐'.repeat(review.rating) + '☆'.repeat(5 - review.rating);
            const reviewDate = new Date(review.created_at).toLocaleDateString('ru');
            
            html += `
                <div class="profile-review-card">
                    <div class="profile-review-header">
                        <span class="profile-review-stars">${reviewStars}</span>
                        <span class="profile-review-date">${reviewDate}</span>
                    </div>
                    ${review.comment ? `
                    <div class="profile-review-comment">
                        ${review.comment}
                    </div>
                    ` : ''}
                    ${!review.is_anonymous && review.from_user_name ? `
                    <div class="profile-review-author" style="font-size: 11px; color: var(--tg-theme-hint-color, #708499); margin-top: 6px;">
                        От: ${review.from_user_username ? '@' + review.from_user_username : review.from_user_name}
                    </div>
                    ` : review.is_anonymous ? `
                    <div class="profile-review-author" style="font-size: 11px; color: var(--tg-theme-hint-color, #708499); margin-top: 6px;">
                        Анонимный отзыв
                    </div>
                    ` : ''}
                </div>
            `;
        });
        
        html += `</div>`;
    } else {
        html += `
            <div class="text-center" style="padding: 20px; color: var(--tg-theme-hint-color, #666); font-size: 13px;">
                📝 Пока нет отзывов
            </div>
        `;
    }
    
    content.innerHTML = html;
}

// Отклик на заявку
async function respondToOrder(orderId) {
    if (!currentUser) {
        showError('Требуется авторизация');
        return;
    }

    if (!orderId || orderId === 0) {
        showError('Неверный ID заявки');
        return;
    }

    console.log('[DEBUG] Отклик на заявку ID:', orderId);
    
    // Создаем модальное окно программно если его нет
    if (!document.getElementById('respondModal')) {
        createRespondModal();
    }
    
    const modal = document.getElementById('respondModal');
    const orderDetails = document.getElementById('respondOrderDetails');
    
    // Показываем модальное окно
    modal.classList.add('show');
    modal.dataset.orderId = orderId;
    
    // Показываем загрузку
    orderDetails.innerHTML = `
        <div class="loading">
            <div class="spinner"></div>
            <p>Загрузка информации о заявке...</p>
        </div>
    `;
    
    try {
        // Загружаем детали заявки
        const response = await fetch(`/api/v1/orders/${orderId}`, {
            headers: { 'X-Telegram-User-ID': currentUser.id.toString() }
        });
        
        const result = await response.json();
        
        if (result.success && result.order) {
            displayOrderDetails(result.order);
        } else {
            orderDetails.innerHTML = '<p>❌ Ошибка загрузки заявки</p>';
        }
        
    } catch (error) {
        console.error('[ERROR] Ошибка загрузки заявки:', error);
        orderDetails.innerHTML = '<p>❌ Ошибка сети</p>';
    }
}

// Отображение деталей заявки в модальном окне отклика
function displayOrderDetails(order) {
    const orderDetails = document.getElementById('respondOrderDetails');
    
    const totalAmount = order.total_amount || (order.amount * order.price);
    
    orderDetails.innerHTML = `
        <div class="order-info-card">
            <div class="order-info-title">
                ${order.type === 'buy' ? '🟢 Заявка на покупку' : '🔴 Заявка на продажу'}
            </div>
            <div class="order-info-row">
                <span class="order-info-label">Количество:</span>
                <span class="order-info-value">${order.amount} ${order.cryptocurrency}</span>
            </div>
            <div class="order-info-row">
                <span class="order-info-label">Курс:</span>
                <span class="order-info-value">${order.price} ${order.fiat_currency}</span>
            </div>
            <div class="order-info-row">
                <span class="order-info-label">Общая сумма:</span>
                <span class="order-info-value" style="color: #22c55e; font-size: 16px;">
                    ${totalAmount.toFixed(2)} ${order.fiat_currency}
                </span>
            </div>
            <div class="order-info-row">
                <span class="order-info-label">Способы оплаты:</span>
                <span class="order-info-value">${(order.payment_methods || []).join(', ') || 'Не указано'}</span>
            </div>
            ${order.description ? `
            <div style="margin-top: 12px; padding-top: 12px; border-top: 1px solid var(--tg-theme-section-separator-color, #e1e8ed);">
                <div class="order-info-label" style="margin-bottom: 4px;">Описание:</div>
                <div style="font-size: 13px; color: var(--tg-theme-text-color, #000000);">
                    ${order.description}
                </div>
            </div>
            ` : ''}
        </div>
    `;
}

// Отправка отклика (обновлено для новой логики)
async function submitResponse() {
    const modal = document.getElementById('respondModal');
    const orderId = parseInt(modal.dataset.orderId);
    const message = document.getElementById('respondMessage').value.trim();
    
    if (!currentUser) {
        showAlert('❌ Требуется авторизация');
        return;
    }
    
    if (!orderId || orderId === 0) {
        showAlert('❌ Неверный ID заявки');
        return;
    }
    
    // Блокируем кнопку на время отправки
    const submitBtn = modal.querySelector('button[onclick="submitResponse()"]');
    const originalText = submitBtn.textContent;
    submitBtn.disabled = true;
    submitBtn.textContent = 'Отправка...';
    
    try {
        console.log('[DEBUG] Создание отклика на заявку:', { orderId, message });
        
        // Создаем отклик через новый API
        const result = await apiRequest('/api/v1/responses', 'POST', {
            order_id: orderId,
            message: message
        });
        
        if (result.success) {
            console.log('[INFO] Отклик создан:', result.response);
            
            if (tg) {
                tg.showPopup({
                    message: 'Отклик отправлен!\n\nВы откликнулись на заявку. Автор заявки рассмотрит ваш отклик и примет решение.'
                });
            } else {
                showAlert('✅ Отклик успешно отправлен! Автор заявки рассмотрит ваш отклик.');
            }
            
            closeRespondModal();
            
            // Обновляем список заявок
            loadOrders();
            
        } else {
            showAlert('❌ ' + (result.message || 'Не удалось создать отклик'));
        }
        
    } catch (error) {
        console.error('[ERROR] Ошибка создания отклика:', error);
        showAlert('❌ Ошибка сети при отправке отклика');
    } finally {
        // Восстанавливаем кнопку
        submitBtn.disabled = false;
        submitBtn.textContent = originalText;
    }
}

// Создание модального окна профиля программно
function createProfileModal() {
    const modalHTML = `
        <div id="profileModal" class="modal">
            <div class="modal-content">
                <div class="modal-header">
                    <div class="modal-title">👤 Профиль пользователя</div>
                    <button class="modal-close" onclick="closeProfileModal()">&times;</button>
                </div>
                <div class="modal-body" id="profileModalContent">
                    <div class="loading">
                        <div class="spinner"></div>
                        <p>Загрузка профиля...</p>
                    </div>
                </div>
            </div>
        </div>
    `;
    document.body.insertAdjacentHTML('beforeend', modalHTML);
}

// Создание модального окна отклика программно  
function createRespondModal() {
    const modalHTML = `
        <div id="respondModal" class="modal">
            <div class="modal-content">
                <div class="modal-header">
                    <div class="modal-title">🤝 Откликнуться на заявку</div>
                    <button class="modal-close" onclick="closeRespondModal()">&times;</button>
                </div>
                <div class="modal-body">
                    <div id="respondOrderDetails">
                        <div class="loading">
                            <div class="spinner"></div>
                            <p>Загрузка информации о заявке...</p>
                        </div>
                    </div>
                    
                    <div class="form-group">
                        <label class="form-label">Сообщение контрагенту (необязательно):</label>
                        <textarea id="respondMessage" class="form-textarea" rows="3" maxlength="200" 
                                  placeholder="Например: Готов к сделке, жду контакта"></textarea>
                    </div>
                    
                    <!-- Автоматическое принятие отключено в новой логике откликов -->
                    
                    <div class="modal-footer">
                        <button type="button" onclick="closeRespondModal()" class="btn btn-secondary">
                            Отмена
                        </button>
                        <button type="button" onclick="submitResponse()" class="btn btn-success">
                            🚀 Откликнуться
                        </button>
                    </div>
                </div>
            </div>
        </div>
    `;
    document.body.insertAdjacentHTML('beforeend', modalHTML);
}

// Закрытие модальных окон
function closeProfileModal() {
    const modal = document.getElementById('profileModal');
    if (modal) {
        modal.classList.remove('show');
    }
}

function closeRespondModal() {
    console.log('[DEBUG] Закрываем модальное окно отклика');
    try {
        const modal = document.getElementById('respondModal');
        if (modal) {
            modal.classList.remove('show');
            
            // Безопасно очищаем форму
            const messageField = document.getElementById('respondMessage');
            if (messageField && messageField.value !== undefined) {
                messageField.value = '';
                console.log('[DEBUG] Очистили поле сообщения');
            }
            
            console.log('[DEBUG] Модальное окно закрыто успешно');
        } else {
            console.log('[DEBUG] Модальное окно не найдено');
        }
    } catch (error) {
        console.error('[ERROR] Ошибка при закрытии модального окна:', error);
    }
}

// =====================================================
// ФУНКЦИИ УПРАВЛЕНИЯ ЗАЯВКАМИ  
// =====================================================

// Редактирование заявки
async function editOrder(orderId) {
    if (!currentUser) {
        showError('Пользователь не авторизован');
        return;
    }

    try {
        // Получаем данные заявки
        const response = await fetch(`/api/v1/orders/${orderId}`, {
            headers: { 'X-Telegram-User-ID': currentUser.id.toString() }
        });

        if (!response.ok) {
            showError('Заявка не найдена');
            return;
        }

        const result = await response.json();
        const order = result.order;

        // Заполняем форму данными заявки
        document.querySelector('[name="type"]').value = order.type;
        document.querySelector('[name="cryptocurrency"]').value = order.cryptocurrency;
        document.querySelector('[name="fiat_currency"]').value = order.fiat_currency;
        document.querySelector('[name="amount"]').value = order.amount;
        document.querySelector('[name="price"]').value = order.price;
        document.querySelector('[name="description"]').value = order.description || '';

        // Устанавливаем способы оплаты
        const paymentMethods = Array.isArray(order.payment_methods) ? order.payment_methods : [];
        document.querySelectorAll('[name="payment_methods"]').forEach(checkbox => {
            checkbox.checked = paymentMethods.includes(checkbox.value);
        });

        // Показываем модальное окно
        document.getElementById('createOrderModal').classList.add('show');
        
        // Меняем заголовок и кнопку
        document.querySelector('.modal-title').textContent = 'Редактировать заявку';
        const submitBtn = document.querySelector('#createOrderForm button[type="submit"]');
        submitBtn.textContent = 'Сохранить изменения';
        
        // Добавляем ID для обновления
        document.getElementById('createOrderForm').dataset.editId = orderId;
        
    } catch (error) {
        console.error('[ERROR] Ошибка загрузки заявки:', error);
        showError('Ошибка загрузки данных заявки');
    }
}

// Просмотр откликов на заявку
async function viewOrderResponses(orderId) {
    if (!currentUser) {
        showError('Пользователь не авторизован');
        return;
    }

    try {
        // Получаем сделки связанные с заявкой
        const response = await fetch(`/api/v1/deals?order_id=${orderId}`, {
            headers: { 'X-Telegram-User-ID': currentUser.id.toString() }
        });

        const result = await response.json();
        
        if (result.success) {
            displayOrderResponses(orderId, result.deals || []);
        } else {
            showError('Ошибка загрузки откликов');
        }
    } catch (error) {
        console.error('[ERROR] Ошибка загрузки откликов:', error);
        
        // Показываем заглушку если API пока не готов
        displayOrderResponses(orderId, []);
    }
}

// Отображение откликов на заявку
function displayOrderResponses(orderId, responses) {
    const modalHTML = `
        <div id="responsesModal" class="modal show">
            <div class="modal-content">
                <div class="modal-header">
                    <div class="modal-title">👥 Отклики на заявку #${orderId}</div>
                    <button class="modal-close" onclick="closeResponsesModal()">&times;</button>
                </div>
                <div class="modal-body">
                    ${responses.length === 0 ? `
                        <div style="text-align: center; padding: 30px; color: var(--tg-theme-hint-color, #708499);">
                            <div style="font-size: 48px; margin-bottom: 16px;">🤷‍♂️</div>
                            <h3 style="margin-bottom: 8px;">Пока никто не откликнулся</h3>
                            <p style="font-size: 14px; line-height: 1.4;">
                                Ваша заявка активна и видна другим пользователям.<br/>
                                Ожидайте откликов или поделитесь ссылкой на заявку.
                            </p>
                        </div>
                    ` : responses.map(response => `
                        <div style="border: 1px solid var(--tg-theme-section-separator-color, #e1e8ed); border-radius: 12px; padding: 16px; margin-bottom: 12px; background: var(--tg-theme-bg-color, #ffffff);">
                            <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px;">
                                <div>
                                    <div style="font-weight: 600; color: var(--tg-theme-text-color, #000000);">
                                        👤 ${response.buyer_id === currentInternalUserId ? 'Покупатель' : 'Продавец'} #${response.buyer_id === currentInternalUserId ? response.seller_id : response.buyer_id}
                                    </div>
                                    <div style="font-size: 12px; color: var(--tg-theme-hint-color, #708499);">
                                        ${new Date(response.created_at).toLocaleString('ru')}
                                    </div>
                                </div>
                                <div style="font-size: 11px; padding: 4px 8px; border-radius: 12px; background: #f59e0b; color: white;">
                                    ${response.status || 'pending'}
                                </div>
                            </div>
                            
                            <div style="margin-bottom: 12px;">
                                <div style="font-weight: 600; margin-bottom: 4px;">
                                    ${response.amount} ${response.cryptocurrency} за ${(response.amount * response.price).toFixed(2)} ${response.fiat_currency}
                                </div>
                                <div style="font-size: 12px; color: var(--tg-theme-hint-color, #708499);">
                                    Курс: ${response.price} ${response.fiat_currency}
                                </div>
                            </div>
                            
                            ${response.notes ? `
                                <div style="margin-bottom: 12px; padding: 8px; background: var(--tg-theme-secondary-bg-color, #f8f9fa); border-radius: 6px; font-size: 13px;">
                                    💬 ${response.notes}
                                </div>
                            ` : ''}
                            
                            <div style="display: flex; gap: 8px;">
                                <button onclick="viewDealDetails(${response.id})" class="btn-small btn-info">
                                    📋 Детали сделки
                                </button>
                                <button onclick="openUserProfile(${response.buyer_id === currentInternalUserId ? response.seller_id : response.buyer_id})" class="btn-small btn-secondary">
                                    👤 Профиль
                                </button>
                            </div>
                        </div>
                    `).join('')}
                </div>
            </div>
        </div>
    `;
    
    document.body.insertAdjacentHTML('beforeend', modalHTML);
}

// Переход к активным сделкам по заявке
async function viewActiveDeals(orderId) {
    // Переключаемся на вкладку откликов
    const responsesTab = document.querySelector('[data-view="responses"]');
    if (responsesTab) {
        responsesTab.click();
        
        // Переключаемся на вкладку активных сделок
        setTimeout(() => {
            switchResponseTab('active-deals');
            highlightDealsByOrder(orderId);
        }, 500);
    }
}

// Подсветка сделок по заявке
function highlightDealsByOrder(orderId) {
    const dealCards = document.querySelectorAll('.deal-card');
    dealCards.forEach(card => {
        const cardOrderId = card.dataset.orderId;
        if (cardOrderId === orderId.toString()) {
            card.style.border = '2px solid #f59e0b';
            card.style.background = 'rgba(245, 158, 11, 0.1)';
        }
    });
}

// История заявки (заглушка)
async function viewOrderHistory(orderId) {
    showInfo(`История заявки #${orderId} будет доступна в следующих обновлениях`);
}

// Детали сделки
async function viewDealDetails(dealId) {
    if (!currentUser) {
        showError('Пользователь не авторизован');
        return;
    }

    try {
        const response = await fetch(`/api/v1/deals/${dealId}`, {
            headers: { 'X-Telegram-User-ID': currentUser.id.toString() }
        });

        if (response.ok) {
            const result = await response.json();
            displayDealDetails(result.deal);
        } else {
            showError('Сделка не найдена');
        }
    } catch (error) {
        console.error('[ERROR] Ошибка загрузки сделки:', error);
        showError('Ошибка загрузки данных сделки');
    }
}

// Отображение деталей сделки
function displayDealDetails(deal) {
    const modalHTML = `
        <div id="dealDetailsModal" class="modal show">
            <div class="modal-content">
                <div class="modal-header">
                    <div class="modal-title">🤝 Сделка #${deal.id}</div>
                    <button class="modal-close" onclick="closeDealDetailsModal()">&times;</button>
                </div>
                <div class="modal-body">
                    <div style="padding: 16px;">
                        <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 16px; margin-bottom: 20px;">
                            <div>
                                <div style="font-size: 12px; color: var(--tg-theme-hint-color, #708499); margin-bottom: 4px;">Покупатель</div>
                                <div style="font-weight: 600;">👤 Пользователь #${deal.buyer_id}</div>
                            </div>
                            <div>
                                <div style="font-size: 12px; color: var(--tg-theme-hint-color, #708499); margin-bottom: 4px;">Продавец</div>
                                <div style="font-weight: 600;">👤 Пользователь #${deal.seller_id}</div>
                            </div>
                        </div>
                        
                        <div style="background: var(--tg-theme-secondary-bg-color, #f8f9fa); border-radius: 12px; padding: 16px; margin-bottom: 20px;">
                            <div style="font-size: 18px; font-weight: 700; margin-bottom: 8px;">
                                ${deal.amount} ${deal.cryptocurrency}
                            </div>
                            <div style="color: var(--tg-theme-hint-color, #708499);">
                                по ${deal.price} ${deal.fiat_currency} = ${deal.total_amount} ${deal.fiat_currency}
                            </div>
                        </div>
                        
                        <div style="margin-bottom: 20px;">
                            <div style="font-size: 12px; color: var(--tg-theme-hint-color, #708499); margin-bottom: 8px;">Статус сделки</div>
                            <div style="display: inline-block; padding: 6px 12px; border-radius: 12px; background: #f59e0b; color: white; font-size: 12px;">
                                ${getDealStatusText(deal.status)}
                            </div>
                        </div>
                        
                        ${deal.notes ? `
                        <div style="margin-bottom: 20px;">
                            <div style="font-size: 12px; color: var(--tg-theme-hint-color, #708499); margin-bottom: 8px;">Комментарий</div>
                            <div style="padding: 12px; background: var(--tg-theme-secondary-bg-color, #f8f9fa); border-radius: 8px;">
                                ${deal.notes}
                            </div>
                        </div>
                        ` : ''}
                        
                        <div style="display: flex; gap: 8px;">
                            ${deal.status === 'pending' ? `
                                <button onclick="confirmDeal(${deal.id})" class="btn btn-success" style="flex: 1;">
                                    ✅ Подтвердить
                                </button>
                            ` : ''}
                            <button onclick="openUserProfile(${deal.buyer_id === currentInternalUserId ? deal.seller_id : deal.buyer_id})" class="btn btn-secondary" style="flex: 1;">
                                👤 Профиль контрагента
                            </button>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    `;
    
    document.body.insertAdjacentHTML('beforeend', modalHTML);
}

// Закрытие модальных окон
function closeResponsesModal() {
    const modal = document.getElementById('responsesModal');
    if (modal) {
        modal.remove();
    }
}

function closeDealDetailsModal() {
    const modal = document.getElementById('dealDetailsModal');
    if (modal) {
        modal.remove();
    }
}

// Подтверждение сделки
async function confirmDeal(dealId) {
    if (!currentUser) {
        showError('Пользователь не авторизован');
        return;
    }

    if (!confirm('Вы уверены что хотите подтвердить эту сделку?')) {
        return;
    }

    try {
        const response = await fetch(`/api/v1/deals/${dealId}/confirm`, {
            method: 'POST',
            headers: {
                'X-Telegram-User-ID': currentUser.id.toString(),
                'Content-Type': 'application/json'
            }
        });

        const result = await response.json();
        
        if (result.success) {
            showSuccess('Сделка подтверждена!');
            closeDealDetailsModal();
            loadMyOrders(); // Обновляем заявки
            loadDeals(); // Обновляем сделки
        } else {
            showError('Ошибка подтверждения сделки: ' + result.error);
        }
    } catch (error) {
        console.error('[ERROR] Ошибка подтверждения сделки:', error);
        showError('Ошибка сети');
    }
}

// Информационное сообщение
function showInfo(message) {
    const alertHTML = `
        <div style="position: fixed; top: 50%; left: 50%; transform: translate(-50%, -50%); 
                    background: var(--tg-theme-bg-color, #ffffff); border: 1px solid var(--tg-theme-section-separator-color, #e1e8ed); 
                    border-radius: 12px; padding: 20px; z-index: 10000; min-width: 280px; text-align: center;">
            <div style="font-size: 32px; margin-bottom: 12px;">ℹ️</div>
            <div style="font-size: 14px; margin-bottom: 16px;">${message}</div>
            <button onclick="this.parentElement.remove()" style="background: var(--tg-theme-button-color, #2481cc); color: var(--tg-theme-button-text-color, #ffffff); border: none; border-radius: 6px; padding: 8px 16px; cursor: pointer;">
                Понятно
            </button>
        </div>
    `;
    
    document.body.insertAdjacentHTML('beforeend', alertHTML);
}

// =====================================================
// ФУНКЦИИ ДЛЯ РАБОТЫ С ОТКЛИКАМИ
// =====================================================

// Основная функция загрузки раздела откликов
async function loadResponses() {
    console.log('[DEBUG] Загрузка раздела откликов');
    
    // По умолчанию загружаем мои отклики
    await loadMyResponses();
}

// Инициализация табов откликов
function initResponseTabs() {
    console.log('[DEBUG] Инициализация табов откликов');
    
    const tabs = document.querySelectorAll('.response-tab');
    tabs.forEach(tab => {
        tab.addEventListener('click', () => {
            const tabName = tab.dataset.tab;
            switchResponseTab(tabName);
        });
    });
}

// Переключение между табами откликов
async function switchResponseTab(tabName) {
    console.log('[DEBUG] Переключение на таб:', tabName);
    
    // Обновляем активные табы
    const tabs = document.querySelectorAll('.response-tab');
    const contents = document.querySelectorAll('.response-tab-content');
    
    tabs.forEach(tab => {
        if (tab.dataset.tab === tabName) {
            tab.classList.add('active');
        } else {
            tab.classList.remove('active');
        }
    });
    
    contents.forEach(content => {
        if (content.id === tabName + '-content') {
            content.classList.add('active');
        } else {
            content.classList.remove('active');
        }
    });
    
    // Загружаем данные для активного таба
    switch(tabName) {
        case 'my-responses':
            await loadMyResponses();
            break;
        case 'responses-to-my':
            await loadResponsesToMyOrders();
            break;
        case 'active-deals':
            await loadActiveDeals();
            break;
    }
}

// Загрузка моих откликов
async function loadMyResponses() {
    console.log('[DEBUG] Загрузка моих откликов');
    
    const container = document.getElementById('myResponsesList');
    container.innerHTML = '<div class="loading"><div class="spinner"></div></div>';
    
    try {
        const result = await apiRequest('/api/v1/responses/my', 'GET');
        
        if (result.success) {
            displayMyResponses(result.responses || []);
        } else {
            container.innerHTML = `<div class="error-message">❌ ${result.message}</div>`;
        }
    } catch (error) {
        console.error('[ERROR] Ошибка загрузки моих откликов:', error);
        container.innerHTML = '<div class="error-message">❌ Ошибка загрузки откликов</div>';
    }
}

// Загрузка откликов на мои заявки
async function loadResponsesToMyOrders() {
    console.log('[DEBUG] Загрузка откликов на мои заявки');
    
    const container = document.getElementById('responsesToMyList');
    container.innerHTML = '<div class="loading"><div class="spinner"></div></div>';
    
    try {
        const result = await apiRequest('/api/v1/responses/to-my', 'GET');
        
        if (result.success) {
            displayResponsesToMyOrders(result.responses || []);
        } else {
            container.innerHTML = `<div class="error-message">❌ ${result.message}</div>`;
        }
    } catch (error) {
        console.error('[ERROR] Ошибка загрузки откликов на заявки:', error);
        container.innerHTML = '<div class="error-message">❌ Ошибка загрузки откликов</div>';
    }
}

// Загрузка активных сделок
async function loadActiveDeals() {
    console.log('[DEBUG] Загрузка активных сделок');
    
    const container = document.getElementById('activeDealsList');
    container.innerHTML = '<div class="loading"><div class="spinner"></div></div>';
    
    try {
        const result = await apiRequest('/api/v1/deals', 'GET');
        
        if (result.success) {
            displayActiveDeals(result.deals || []);
        } else {
            container.innerHTML = `<div class="error-message">❌ ${result.message}</div>`;
        }
    } catch (error) {
        console.error('[ERROR] Ошибка загрузки активных сделок:', error);
        container.innerHTML = '<div class="error-message">❌ Ошибка загрузки сделок</div>';
    }
}

// Отображение моих откликов
function displayMyResponses(responses) {
    console.log('[DEBUG] Отображение моих откликов:', responses.length);
    
    const container = document.getElementById('myResponsesList');
    
    if (!responses || responses.length === 0) {
        container.innerHTML = `
            <div class="empty-state">
                <div class="empty-icon">📤</div>
                <h3>Пока нет откликов</h3>
                <p>Перейдите в раздел "Рынок" и откликнитесь на интересную заявку</p>
            </div>
        `;
        return;
    }
    
    // Группируем отклики по статусу
    const waiting = responses.filter(r => r.status === 'waiting');
    const accepted = responses.filter(r => r.status === 'accepted');
    const rejected = responses.filter(r => r.status === 'rejected');
    
    let html = '';
    
    if (waiting.length > 0) {
        html += `<div class="response-group">
            <h3 class="group-title">🟡 Ожидают рассмотрения (${waiting.length})</h3>
            ${waiting.map(response => createMyResponseCard(response)).join('')}
        </div>`;
    }
    
    if (accepted.length > 0) {
        html += `<div class="response-group">
            <h3 class="group-title">🟢 Приняты (${accepted.length})</h3>
            ${accepted.map(response => createMyResponseCard(response)).join('')}
        </div>`;
    }
    
    if (rejected.length > 0) {
        html += `<div class="response-group">
            <h3 class="group-title">🔴 Отклонены (${rejected.length})</h3>
            ${rejected.map(response => createMyResponseCard(response)).join('')}
        </div>`;
    }
    
    container.innerHTML = html;
}

// Отображение откликов на мои заявки
function displayResponsesToMyOrders(responses) {
    console.log('[DEBUG] Отображение откликов на мои заявки:', responses.length);
    
    const container = document.getElementById('responsesToMyList');
    
    if (!responses || responses.length === 0) {
        container.innerHTML = `
            <div class="empty-state">
                <div class="empty-icon">📥</div>
                <h3>Пока нет откликов</h3>
                <p>Создайте заявку и ждите откликов от других пользователей</p>
            </div>
        `;
        return;
    }
    
    // Группируем отклики по заявкам
    const responsesByOrder = {};
    responses.forEach(response => {
        if (!responsesByOrder[response.order_id]) {
            responsesByOrder[response.order_id] = [];
        }
        responsesByOrder[response.order_id].push(response);
    });
    
    let html = '';
    Object.entries(responsesByOrder).forEach(([orderId, orderResponses]) => {
        const waitingResponses = orderResponses.filter(r => r.status === 'waiting');
        
        // Берём первый отклик для получения информации о заявке
        const firstResponse = orderResponses[0];
        const orderTypeText = firstResponse.order_type === 'buy' ? '🟢 Покупка' : '🔴 Продажа';
        const totalAmount = firstResponse.total_amount || (firstResponse.amount * firstResponse.price);
        
        html += `<div class="order-responses-group">
            <div class="order-info">
                <h4>📋 Заявка #${orderId} - ${orderTypeText}</h4>
                ${firstResponse.cryptocurrency ? `
                    <div style="font-size: 12px; color: var(--tg-theme-hint-color, #708499); margin-top: 4px;">
                        💰 ${firstResponse.amount || '?'} ${firstResponse.cryptocurrency || '?'} за ${firstResponse.price || '?'} ${firstResponse.fiat_currency || '?'} = ${totalAmount.toLocaleString('ru')} ${firstResponse.fiat_currency || '?'}
                    </div>
                ` : ''}
                <span class="response-count">${waitingResponses.length} новых откликов</span>
            </div>
            ${orderResponses.map(response => createOrderResponseCard(response)).join('')}
        </div>`;
    });
    
    container.innerHTML = html;
}

// Создание карточки моего отклика
function createMyResponseCard(response) {
    const statusConfig = {
        waiting: { icon: '🟡', text: 'Ожидает', color: '#f59e0b' },
        accepted: { icon: '🟢', text: 'Принят', color: '#22c55e' },
        rejected: { icon: '🔴', text: 'Отклонен', color: '#ef4444' }
    };
    
    const status = statusConfig[response.status] || statusConfig.waiting;
    const createdDate = new Date(response.created_at).toLocaleString('ru-RU');
    
    return `
        <div class="response-card my-response">
            <div class="response-header">
                <div class="response-status" style="color: ${status.color}">
                    ${status.icon} ${status.text}
                </div>
                <div class="response-date">${createdDate}</div>
            </div>
            
            <div class="response-order-info">
                <h4 class="order-title">📋 Заявка #${response.order_id} - ${response.order_type === 'buy' ? '🟢 Покупка' : '🔴 Продажа'}</h4>
                <div style="font-size: 13px; color: var(--tg-theme-hint-color, #708499); margin-top: 4px;">
                    👤 Автор: ${response.author_username ? 
                        `<span onclick="openTelegramProfile('${response.author_username}')" style="color: var(--tg-theme-link-color, #0088cc); cursor: pointer; text-decoration: underline; font-weight: 500;">@${response.author_username}</span>` :
                        `<span style="color: var(--tg-theme-text-color, #000); font-weight: 500;">${response.author_name || 'Неизвестен'}</span>`
                    }
                </div>
                ${response.cryptocurrency ? `
                    <div style="font-size: 12px; color: var(--tg-theme-hint-color, #708499); margin-top: 6px;">
                        💰 ${response.amount || '?'} ${response.cryptocurrency || '?'} за ${response.price || '?'} ${response.fiat_currency || '?'} = ${(response.total_amount || (response.amount * response.price)).toLocaleString('ru')} ${response.fiat_currency || '?'}
                    </div>
                ` : ''}
            </div>
            
            <div class="response-message">
                <strong>💬 Ваше сообщение:</strong>
                <p>${response.message || 'Без сообщения'}</p>
            </div>
            
            ${response.status === 'accepted' ? `
                <div class="response-actions">
                    <button onclick="goToDeal(${response.id})" class="btn btn-primary">
                        🤝 Перейти к сделке
                    </button>
                </div>
            ` : ''}
        </div>
    `;
}

// Создание карточки отклика на мою заявку
function createOrderResponseCard(response) {
    const statusConfig = {
        waiting: { icon: '🟡', text: 'Ожидает', color: '#f59e0b' },
        accepted: { icon: '🟢', text: 'Принят', color: '#22c55e' },
        rejected: { icon: '🔴', text: 'Отклонен', color: '#ef4444' }
    };
    
    const status = statusConfig[response.status] || statusConfig.waiting;
    const createdDate = new Date(response.created_at).toLocaleString('ru-RU');
    
    return `
        <div class="response-card order-response">
            <div class="response-header">
                <div class="response-user">👤 ${response.username ? 
                    `<span onclick="openTelegramProfile('${response.username}')" style="color: var(--tg-theme-link-color, #0088cc); cursor: pointer; text-decoration: underline; font-weight: 500;">@${response.username}</span>` :
                    `<span style="color: var(--tg-theme-text-color, #000); font-weight: 500;">${response.user_name || `Пользователь #${response.user_id}`}</span>`
                }</div>
                <div class="response-status" style="color: ${status.color}">
                    ${status.icon} ${status.text}
                </div>
            </div>
            
            <div class="response-date">${createdDate}</div>
            
            <div style="font-size: 12px; color: var(--tg-theme-hint-color, #708499); margin: 8px 0;">
                📋 ${response.order_type === 'buy' ? '🟢 Покупка' : '🔴 Продажа'} ${response.cryptocurrency || '?'} - ${response.amount || '?'} ${response.cryptocurrency || '?'} за ${response.price || '?'} ${response.fiat_currency || '?'} = ${(response.total_amount || (response.amount * response.price)).toLocaleString('ru')} ${response.fiat_currency || '?'}
            </div>
            
            <div class="response-message">
                <strong>💬 Сообщение:</strong>
                <p>${response.message || 'Без сообщения'}</p>
            </div>
            
            ${response.status === 'waiting' ? `
                <div class="response-actions">
                    <button onclick="acceptResponse(${response.id})" class="btn btn-success">
                        ✅ Принять
                    </button>
                    <button onclick="rejectResponse(${response.id})" class="btn btn-danger">
                        ❌ Отклонить
                    </button>
                </div>
            ` : ''}
        </div>
    `;
}

// Принятие отклика
async function acceptResponse(responseId) {
    console.log('[DEBUG] Принятие отклика:', responseId);
    
    try {
        const result = await apiRequest(`/api/v1/responses/${responseId}/accept`, 'POST');
        
        if (result.success) {
            showAlert('✅ Отклик принят! Создана сделка.');
            // Перезагружаем отклики на мои заявки
            await loadResponsesToMyOrders();
            // Также загружаем активные сделки
            await loadActiveDeals();
        } else {
            showAlert('❌ ' + (result.message || 'Ошибка при принятии отклика'));
        }
    } catch (error) {
        console.error('[ERROR] Ошибка принятия отклика:', error);
        showAlert('❌ Ошибка при принятии отклика');
    }
}

// Отклонение отклика
async function rejectResponse(responseId) {
    console.log('[DEBUG] Отклонение отклика:', responseId);
    
    try {
        const result = await apiRequest(`/api/v1/responses/${responseId}/reject`, 'POST');
        
        if (result.success) {
            showAlert('❌ Отклик отклонен');
            // Перезагружаем отклики на мои заявки
            await loadResponsesToMyOrders();
        } else {
            showAlert('❌ ' + (result.message || 'Ошибка при отклонении отклика'));
        }
    } catch (error) {
        console.error('[ERROR] Ошибка отклонения отклика:', error);
        showAlert('❌ Ошибка при отклонении отклика');
    }
}

// Переход к сделке
async function goToDeal(responseId) {
    console.log('[DEBUG] Переход к сделке по отклику:', responseId);
    
    // Переключаемся на таб активных сделок
    switchResponseTab('active-deals');
}

// Отображение активных сделок
function displayActiveDeals(deals) {
    console.log('[DEBUG] Отображение активных сделок:', deals.length, deals);
    
    const container = document.getElementById('activeDealsList');
    
    if (!deals || deals.length === 0) {
        container.innerHTML = `
            <div class="empty-state">
                <div class="empty-icon">⏰</div>
                <div class="empty-title">Активных сделок пока нет</div>
                <div class="empty-subtitle">Когда вы примете отклик или ваш отклик будет принят,<br>здесь появятся активные сделки</div>
            </div>
        `;
        return;
    }
    
    // Отображаем реальные сделки
    const dealsHTML = deals.map(deal => createDealCard(deal)).join('');
    container.innerHTML = dealsHTML;
}

// Создание карточки активной сделки
function createDealCard(deal) {
    console.log('[DEBUG] Создание карточки сделки:', deal);
    
    // Определяем роль пользователя в сделке
    const isAuthor = currentInternalUserId === deal.author_id;
    
    // Получаем данные автора и контрагента
    const authorName = deal.author_name || `Пользователь ${deal.author_id}`;
    const authorUsername = deal.author_username ? `@${deal.author_username}` : '';
    const counterpartyName = deal.counterparty_name || `Пользователь ${deal.counterparty_id}`;
    const counterpartyUsername = deal.counterparty_username ? `@${deal.counterparty_username}` : '';
    
    // Определяем контрагента для текущего пользователя
    const counterpartyDisplayName = isAuthor ? counterpartyName : authorName;
    const counterpartyDisplayUsername = isAuthor ? counterpartyUsername : authorUsername;
    const counterpartyTelegramUsername = isAuthor ? deal.counterparty_username : deal.author_username;
    const counterpartyUserId = isAuthor ? deal.counterparty_id : deal.author_id;
    
    console.log('[DEBUG] Telegram usernames:', {
        authorUsername: deal.author_username,
        counterpartyUsername: deal.counterparty_username,
        counterpartyTelegramUsername: counterpartyTelegramUsername,
        isAuthor: isAuthor
    });
    
    // Статус сделки
    const statusConfig = {
        in_progress: { icon: '⏳', text: 'В процессе', color: '#f59e0b' },
        waiting_payment: { icon: '💰', text: 'Ожидание оплаты', color: '#3b82f6' },
        completed: { icon: '✅', text: 'Завершена', color: '#22c55e' },
        cancelled: { icon: '❌', text: 'Отменена', color: '#ef4444' },
        expired: { icon: '⏰', text: 'Истекла', color: '#6b7280' }
    };
    
    const status = statusConfig[deal.status] || statusConfig.in_progress;
    
    // Убираем таймер - больше не используем
    
    // Подтверждения участников
    const authorConfirmed = deal.author_confirmed || false;
    const counterConfirmed = deal.counter_confirmed || false;
    const myConfirmed = isAuthor ? authorConfirmed : counterConfirmed;
    const partnerConfirmed = isAuthor ? counterConfirmed : authorConfirmed;
    
    return `
        <div class="order-card">
            <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px;">
                <span style="font-weight: 600; color: ${deal.order_type === 'buy' ? '#22c55e' : '#ef4444'};">
                    ${deal.order_type === 'buy' ? '🟢 Покупка' : '🔴 Продажа'}
                </span>
                <div style="display: flex; align-items: center; gap: 8px;">
                    <span style="color: ${status.color}; font-weight: 500; font-size: 14px;">
                        ${status.icon} ${status.text}
                    </span>
                </div>
            </div>
            
            <div style="margin-bottom: 12px;">
                <strong style="font-size: 18px; color: var(--tg-theme-text-color, #000);">${deal.amount || '?'} ${deal.cryptocurrency || '?'}</strong> 
                <span style="color: var(--tg-theme-hint-color, #708499);">за</span>
                <strong style="font-size: 16px; color: var(--tg-theme-text-color, #000);">${deal.price || '?'} ${deal.fiat_currency || '?'}</strong>
            </div>
            
            <div style="background: var(--tg-theme-secondary-bg-color, #f1f5f9); padding: 12px; border-radius: 6px; margin-bottom: 12px; font-size: 13px;">
                <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 8px;">
                    <div>
                        <div style="color: var(--tg-theme-hint-color, #708499); margin-bottom: 4px;">📝 Автор:</div>
                        <div style="font-weight: 500; color: var(--tg-theme-text-color, #000);">${authorName}</div>
                        <div style="color: var(--tg-theme-link-color, #3b82f6); font-size: 12px;">${authorUsername}</div>
                    </div>
                    <div>
                        <div style="color: var(--tg-theme-hint-color, #708499); margin-bottom: 4px;">🤝 Откликнулся:</div>
                        <div style="font-weight: 500; color: var(--tg-theme-text-color, #000);">${counterpartyName}</div>
                        <div style="color: var(--tg-theme-link-color, #3b82f6); font-size: 12px;">${counterpartyUsername}</div>
                    </div>
                </div>
                
                <div style="margin-top: 12px; padding-top: 8px; border-top: 1px solid var(--tg-theme-section-separator-color, #e2e8f0);">
                    <div style="color: var(--tg-theme-hint-color, #708499); margin-bottom: 4px;">💳 Способ оплаты:</div>
                    <div style="font-weight: 500; color: var(--tg-theme-text-color, #000);">${(deal.payment_methods || []).join(', ') || 'Не указано'}</div>
                </div>
                
                <div style="margin-top: 8px; display: grid; grid-template-columns: 1fr 1fr; gap: 12px; font-size: 12px;">
                    <div>
                        <span style="color: var(--tg-theme-hint-color, #708499);">💰 Курс:</span>
                        <span style="font-weight: 500; color: var(--tg-theme-text-color, #000);">${deal.price} ${deal.fiat_currency}</span>
                    </div>
                    <div>
                        <span style="color: var(--tg-theme-hint-color, #708499);">💵 Сумма:</span>
                        <span style="font-weight: 500; color: var(--tg-theme-text-color, #000);">${deal.total_amount || (deal.amount * deal.price).toFixed(2)} ${deal.fiat_currency}</span>
                    </div>
                </div>
            </div>
            

            
            <div style="background: var(--tg-theme-secondary-bg-color, #f8fafc); border-radius: 6px; padding: 8px; margin-bottom: 12px;">
                <div style="font-size: 12px; color: var(--tg-theme-hint-color, #708499); margin-bottom: 6px;">Подтверждения:</div>
                <div style="display: flex; justify-content: space-between;">
                    <div style="display: flex; align-items: center; gap: 4px;">
                        <span>${myConfirmed ? '✅' : '⏳'}</span>
                        <span style="font-size: 12px; color: var(--tg-theme-text-color, #000);">Вы</span>
                    </div>
                    <div style="display: flex; align-items: center; gap: 4px;">
                        <span>${partnerConfirmed ? '✅' : '⏳'}</span>
                        <span style="font-size: 12px; color: var(--tg-theme-text-color, #000);">Контрагент</span>
                    </div>
                </div>
            </div>
            
            <div style="display: flex; gap: 8px;">
                ${counterpartyTelegramUsername ? `
                    <button onclick="contactCounterparty('${counterpartyTelegramUsername}')" style="background: var(--tg-theme-button-color, #0088cc); color: var(--tg-theme-button-text-color, white); border: none; padding: 8px 12px; border-radius: 4px; font-size: 12px; flex: 1;">
                        💬 Написать
                    </button>
                ` : ''}
                
                ${deal.status === 'completed' ? `
                    <button onclick="openReviewModal(${deal.id}, ${counterpartyUserId}, '${counterpartyDisplayName}')" style="background: var(--tg-theme-button-color, #f59e0b); color: var(--tg-theme-button-text-color, white); border: none; padding: 8px 12px; border-radius: 4px; font-size: 12px; flex: 1;">
                        ⭐ Оставить отзыв
                    </button>
                ` : `
                    <button onclick="confirmPayment(${deal.id}, ${isAuthor})" style="background: ${myConfirmed ? 'var(--tg-theme-hint-color, #6c757d)' : 'var(--tg-theme-button-color, #22c55e)'}; color: var(--tg-theme-button-text-color, white); border: none; padding: 8px 12px; border-radius: 4px; font-size: 12px; flex: 1;" ${myConfirmed ? 'disabled' : ''}>
                        ${myConfirmed ? '✅ Подтверждено' : '✅ Подтвердить'}
                    </button>
                `}
            </div>
        </div>
    `;
}

// Функция таймера удалена - больше не используем таймеры в сделках

// Связь с контрагентом в Telegram
function contactCounterparty(username) {
    console.log('[DEBUG] Открытие чата с контрагентом:', username);
    
    if (username) {
        const telegramUrl = `https://t.me/${username}`;
        
        if (tg && tg.openTelegramLink) {
            // Используем Telegram WebApp API для открытия ссылки
            tg.openTelegramLink(telegramUrl);
        } else {
            // Резервный вариант - открываем в новом окне
            window.open(telegramUrl, '_blank');
        }
    } else {
        showAlert('❌ Не удалось найти username контрагента');
    }
}

// Открытие Telegram профиля пользователя (для кликабельного username в заявках)
function openTelegramProfile(username) {
    console.log('[DEBUG] Открытие Telegram профиля:', username);
    
    if (username) {
        const telegramUrl = `https://t.me/${username}`;
        
        if (tg && tg.openTelegramLink) {
            // Используем Telegram WebApp API для открытия ссылки
            tg.openTelegramLink(telegramUrl);
        } else {
            // Резервный вариант - открываем в новом окне
            window.open(telegramUrl, '_blank');
        }
    } else {
        showAlert('❌ Не удалось найти username пользователя');
    }
}

// Просмотр откликов на заявку
function viewOrderResponses(orderId) {
    console.log('[DEBUG] Просмотр откликов на заявку:', orderId);
    
    // Переключаемся на раздел откликов и таб "На мои заявки"
    const responsesTab = document.querySelector('[data-view="responses"]');
    if (responsesTab) {
        responsesTab.click(); // Переходим в раздел "Отклики"
        
        // Небольшая задержка, чтобы раздел успел загрузиться
        setTimeout(() => {
            switchResponseTab('responses-to-my'); // Переключаемся на таб "На мои заявки"
        }, 100);
    } else {
        showAlert('❌ Не удалось перейти к откликам');
    }
}



// Подтверждение платежа/получения в сделке  
async function confirmPayment(dealId, isAuthor) {
    console.log('[DEBUG] Подтверждение сделки:', { dealId, isAuthor });
    
    if (!currentUser) {
        showAlert('❌ Требуется авторизация');
        return;
    }
    
    try {
        // Подтверждаем через API
        const result = await apiRequest(`/api/v1/deals/${dealId}/confirm`, 'POST', {
            is_author: isAuthor
        });
        
        if (result.success) {
            const message = isAuthor ? 
                '✅ Вы подтвердили получение средств!' : 
                '✅ Вы подтвердили отправку средств!';
            
            showAlert(message);
            
            // Перезагружаем активные сделки
            await loadActiveDeals();
            
            // Проверяем завершена ли сделка
            if (result.deal_completed) {
                showAlert('🎉 Сделка успешно завершена!\n\nВы можете оставить отзыв о контрагенте.');
                
                // Можно добавить переход к форме отзыва
                setTimeout(() => {
                    // switchResponseTab('completed-deals'); // если будет такой таб
                }, 2000);
            }
            
        } else {
            showAlert('❌ ' + (result.message || 'Ошибка при подтверждении сделки'));
        }
        
    } catch (error) {
        console.error('[ERROR] Ошибка подтверждения сделки:', error);
        showAlert('❌ Ошибка сети при подтверждении сделки');
    }
}

// =====================================================
// ФУНКЦИИ ДЛЯ РАБОТЫ С ОТЗЫВАМИ
// =====================================================

// Переменная для хранения текущего рейтинга
let currentReviewRating = 0;

// Открытие модального окна для оставления отзыва
function openReviewModal(dealId, toUserId, counterpartyName) {
    console.log('[DEBUG] Открытие модального окна отзыва:', { dealId, toUserId, counterpartyName });
    
    // Заполняем скрытые поля
    document.getElementById('reviewDealId').value = dealId;
    document.getElementById('reviewToUserId').value = toUserId;
    document.getElementById('reviewCounterpartyName').textContent = counterpartyName || 'Неизвестный пользователь';
    
    // Сбрасываем форму
    resetReviewForm();
    
    // Показываем модальное окно
    const modal = document.getElementById('reviewModal');
    modal.classList.add('show');
    document.body.classList.add('modal-open');
    
    // Инициализируем звездный рейтинг
    initializeStarRating();
}

// Закрытие модального окна отзыва
function closeReviewModal() {
    const modal = document.getElementById('reviewModal');
    modal.classList.remove('show');
    document.body.classList.remove('modal-open');
    
    // Сбрасываем форму при закрытии
    resetReviewForm();
}

// Сброс формы отзыва к исходному состоянию
function resetReviewForm() {
    // Очищаем рейтинг
    currentReviewRating = 0;
    document.getElementById('reviewRating').value = '';
    
    // Сбрасываем звезды
    const stars = document.querySelectorAll('#starRating .star');
    stars.forEach(star => {
        star.classList.remove('active', 'hovered', 'just-selected');
    });
    
    // Сбрасываем текст рейтинга
    document.getElementById('ratingValue').textContent = 'Выберите оценку от 1 до 5 звезд';
    
    // Очищаем комментарий
    document.getElementById('reviewComment').value = '';
    
    // Сбрасываем чекбокс анонимности
    document.getElementById('reviewAnonymous').checked = false;
}

// Инициализация звездного рейтинга
function initializeStarRating() {
    const stars = document.querySelectorAll('#starRating .star');
    
    stars.forEach((star, index) => {
        const rating = parseInt(star.getAttribute('data-rating'));
        
        // Обработчик клика по звезде
        star.addEventListener('click', function() {
            selectStarRating(rating);
        });
        
        // Обработчик наведения мыши
        star.addEventListener('mouseenter', function() {
            hoverStarRating(rating);
        });
    });
    
    // Обработчик покидания области звездного рейтинга
    const starRating = document.getElementById('starRating');
    starRating.addEventListener('mouseleave', function() {
        clearHoverStarRating();
    });
}

// Выбор рейтинга по звездам
function selectStarRating(rating) {
    console.log('[DEBUG] Выбран рейтинг:', rating);
    
    currentReviewRating = rating;
    document.getElementById('reviewRating').value = rating;
    
    // Обновляем визуальное отображение звезд
    updateStarsDisplay(rating, true);
    
    // Обновляем текст рейтинга
    const ratingTexts = {
        1: '1 звезда - Очень плохо',
        2: '2 звезды - Плохо', 
        3: '3 звезды - Нормально',
        4: '4 звезды - Хорошо',
        5: '5 звезд - Отлично'
    };
    
    document.getElementById('ratingValue').textContent = ratingTexts[rating];
    document.getElementById('ratingValue').style.color = rating >= 4 ? '#22c55e' : rating === 3 ? '#f59e0b' : '#ef4444';
    
    // Добавляем анимацию к выбранной звезде
    const selectedStar = document.querySelector(`#starRating .star[data-rating="${rating}"]`);
    selectedStar.classList.add('just-selected');
    setTimeout(() => {
        selectedStar.classList.remove('just-selected');
    }, 300);
}

// Отображение hover эффекта для звезд
function hoverStarRating(rating) {
    updateStarsDisplay(rating, false, true);
}

// Очистка hover эффекта
function clearHoverStarRating() {
    updateStarsDisplay(currentReviewRating, true);
}

// Обновление визуального отображения звезд
function updateStarsDisplay(rating, isSelected = false, isHovered = false) {
    const stars = document.querySelectorAll('#starRating .star');
    
    stars.forEach((star, index) => {
        const starRating = parseInt(star.getAttribute('data-rating'));
        
        // Удаляем все классы
        star.classList.remove('active', 'hovered');
        
        if (starRating <= rating) {
            if (isSelected) {
                star.classList.add('active');
            } else if (isHovered) {
                star.classList.add('hovered');
            }
        }
    });
}

// Обработчик отправки формы отзыва
document.addEventListener('DOMContentLoaded', function() {
    const reviewForm = document.getElementById('reviewForm');
    if (reviewForm) {
        reviewForm.addEventListener('submit', handleReviewSubmit);
    }
    
    // Обработчик закрытия модального окна по клику вне его
    const reviewModal = document.getElementById('reviewModal');
    if (reviewModal) {
        reviewModal.addEventListener('click', function(e) {
            if (e.target === reviewModal) {
                closeReviewModal();
            }
        });
    }
});

// Обработка отправки отзыва
async function handleReviewSubmit(event) {
    event.preventDefault();
    
    console.log('[DEBUG] Отправка отзыва');
    
    if (!currentUser) {
        showAlert('❌ Требуется авторизация');
        return;
    }
    
    // Проверяем обязательные поля
    const dealId = parseInt(document.getElementById('reviewDealId').value);
    const toUserId = parseInt(document.getElementById('reviewToUserId').value);
    const rating = currentReviewRating;
    
    if (!dealId || !toUserId || !rating) {
        showAlert('❌ Пожалуйста, заполните все обязательные поля');
        return;
    }
    
    if (rating < 1 || rating > 5) {
        showAlert('❌ Рейтинг должен быть от 1 до 5 звезд');
        return;
    }
    
    // Собираем данные формы
    const reviewData = {
        deal_id: dealId,
        to_user_id: toUserId,
        rating: rating,
        comment: document.getElementById('reviewComment').value.trim(),
        is_anonymous: document.getElementById('reviewAnonymous').checked
    };
    
    console.log('[DEBUG] Данные отзыва для отправки:', reviewData);
    
    try {
        // Блокируем кнопку отправки
        const submitButton = document.querySelector('#reviewForm button[type="submit"]');
        const originalText = submitButton.textContent;
        submitButton.disabled = true;
        submitButton.textContent = 'Отправка...';
        
        // Отправляем отзыв через API
        const result = await apiRequest('/api/v1/reviews', 'POST', reviewData);
        
        if (result.success) {
            showAlert('✅ Отзыв успешно отправлен!\n\nСпасибо за ваше мнение.');
            closeReviewModal();
            
            // Обновляем активные сделки
            await loadActiveDeals();
        } else {
            console.error('[ERROR] Ошибка создания отзыва:', result.message);
            showAlert('❌ Ошибка при отправке отзыва: ' + (result.message || 'Неизвестная ошибка'));
        }
        
    } catch (error) {
        console.error('[ERROR] Ошибка отправки отзыва:', error);
        showAlert('❌ Ошибка сети при отправке отзыва');
    } finally {
        // Разблокируем кнопку
        const submitButton = document.querySelector('#reviewForm button[type="submit"]');
        if (submitButton) {
            submitButton.disabled = false;
            submitButton.textContent = originalText;
        }
    }
}