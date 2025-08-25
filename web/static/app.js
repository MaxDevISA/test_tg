// Глобальные переменные для P2P криптобиржи
let currentUser = null;
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
                view.style.display = view.id === viewName + 'View' ? 'block' : 'none';
            });
            
            // Загружаем данные для раздела
            if (viewName === 'orders') {
                loadOrders();
            } else if (viewName === 'my-orders') {
                loadMyOrders();
            } else if (viewName === 'deals') {
                loadDeals();
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
        
        if (result.success) {
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
        const response = await fetch('/api/v1/orders');
        const result = await response.json();
        
        if (result.success) {
            displayOrders(result.orders || []);
        } else {
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
    
    if (orders.length === 0) {
        content.innerHTML = '<p class="text-center text-muted">Заявок пока нет</p>';
        return;
    }
    
    const ordersHTML = orders.map(order => 
        '<div style="border: 1px solid var(--tg-theme-section-separator-color, #e1e8ed); ' +
                    'border-radius: 8px; padding: 12px; margin-bottom: 8px; ' +
                    'background: var(--tg-theme-secondary-bg-color, #f8f9fa);">' +
            '<div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px;">' +
                '<span style="font-weight: 600; color: ' + (order.type === 'buy' ? '#22c55e' : '#ef4444') + ';">' +
                    (order.type === 'buy' ? '🟢 Покупка' : '🔴 Продажа') +
                '</span>' +
                '<span style="font-size: 12px; color: var(--tg-theme-hint-color, #708499);">' +
                    new Date(order.created_at).toLocaleString('ru') +
                '</span>' +
            '</div>' +
            '<div style="margin-bottom: 8px;">' +
                '<strong>' + order.amount + ' ' + order.cryptocurrency + '</strong> за <strong>' + order.price + ' ' + order.fiat_currency + '</strong>' +
            '</div>' +
            '<div style="font-size: 12px; color: var(--tg-theme-hint-color, #708499);">' +
                'Способы оплаты: ' + ((order.payment_methods || []).join(', ') || 'Не указано') +
            '</div>' +
            (order.description ? '<div style="font-size: 12px; margin-top: 4px;">' + order.description + '</div>' : '') +
        '</div>'
    ).join('');
    
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
    if (!currentUser) {
        showError('Пользователь не авторизован');
        return;
    }

    const content = document.getElementById('my-ordersView');
    content.innerHTML = '<div class="loading"><div class="spinner"></div><p>Загрузка ваших заявок...</p></div>';
    
    try {
        const response = await fetch('/api/v1/orders/my', {
            headers: {
                'X-Telegram-User-ID': currentUser.id.toString()
            }
        });
        
        const result = await response.json();
        
        if (result.success) {
            displayMyOrders(result.orders || []);
        } else {
            content.innerHTML = '<p class="text-center text-muted">Ошибка загрузки заявок</p>';
        }
    } catch (error) {
        console.error('[ERROR] Ошибка загрузки моих заявок:', error);
        content.innerHTML = '<p class="text-center text-muted">Ошибка сети</p>';
    }
}

// Отображение моих заявок
function displayMyOrders(orders) {
    const content = document.getElementById('my-ordersView');
    
    if (orders.length === 0) {
        content.innerHTML = `
            <div class="text-center mt-md">
                <h2>Мои заявки</h2>
                <p class="text-muted">У вас пока нет активных заявок</p>
                <button class="btn btn-primary" id="createFirstOrderBtn">Создать первую заявку</button>
            </div>
        `;
        
        // Добавляем обработчик для кнопки создания первой заявки
        document.getElementById('createFirstOrderBtn').addEventListener('click', () => {
            document.getElementById('createOrderModal').classList.add('show');
        });
        return;
    }
    
    let html = '<h2 style="margin-bottom: 16px;">Мои заявки</h2>';
    
    orders.forEach(order => {
        const statusColor = order.status === 'active' ? '#22c55e' : 
                           order.status === 'matched' ? '#f59e0b' :
                           order.status === 'completed' ? '#3b82f6' : '#ef4444';
        
        html += `
            <div style="border: 1px solid var(--tg-theme-section-separator-color, #e1e8ed); 
                        border-radius: 8px; padding: 12px; margin-bottom: 12px;
                        background: var(--tg-theme-bg-color, #ffffff);">
                <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px;">
                    <span style="font-weight: 600; color: ${order.type === 'buy' ? '#22c55e' : '#ef4444'};">
                        ${order.type === 'buy' ? '🟢 Покупка' : '🔴 Продажа'}
                    </span>
                    <span style="font-size: 12px; padding: 4px 8px; border-radius: 12px; background: ${statusColor}; color: white;">
                        ${getStatusText(order.status)}
                    </span>
                </div>
                <div style="margin-bottom: 8px;">
                    <strong>${order.amount} ${order.cryptocurrency}</strong> за <strong>${order.price} ${order.fiat_currency}</strong>
                </div>
                <div style="font-size: 12px; color: var(--tg-theme-hint-color, #708499);">
                    Создано: ${new Date(order.created_at).toLocaleString('ru')}
                </div>
                ${order.status === 'active' ? `
                <div style="margin-top: 8px;">
                    <button onclick="cancelOrder(${order.id})" style="background: #ef4444; color: white; border: none; padding: 4px 8px; border-radius: 4px; font-size: 12px;">
                        Отменить
                    </button>
                </div>
                ` : ''}
            </div>
        `;
    });
    
    content.innerHTML = html;
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

    const content = document.getElementById('profileView');
    content.innerHTML = '<div class="loading"><div class="spinner"></div><p>Загрузка профиля...</p></div>';
    
    try {
        const response = await fetch('/api/v1/auth/me', {
            headers: {
                'X-Telegram-User-ID': currentUser.id.toString()
            }
        });
        
        const result = await response.json();
        
        if (result.success || result.user) {
            displayProfile(result.user || currentUser);
        } else {
            displayProfile(currentUser); // Показываем базовую информацию
        }
    } catch (error) {
        console.error('[ERROR] Ошибка загрузки профиля:', error);
        displayProfile(currentUser); // Показываем базовую информацию
    }
}

// Отображение профиля
function displayProfile(user) {
    const content = document.getElementById('profileView');
    
    const rating = user.rating || 0;
    const stars = '⭐'.repeat(Math.floor(rating)) + '☆'.repeat(5 - Math.floor(rating));
    
    content.innerHTML = `
        <div style="text-align: center; padding: 20px;">
            <h2 style="margin-bottom: 16px;">👤 Мой профиль</h2>
            
            <div style="margin-bottom: 24px;">
                <div style="font-size: 18px; font-weight: 600; margin-bottom: 8px;">
                    ${user.first_name} ${user.last_name || ''}
                </div>
                <div style="font-size: 14px; color: var(--tg-theme-hint-color, #708499); margin-bottom: 12px;">
                    @${user.username || 'пользователь'}
                </div>
                <div style="font-size: 16px; margin-bottom: 8px;">
                    ${stars} ${rating.toFixed(1)}
                </div>
            </div>
            
            <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 16px; margin-bottom: 24px;">
                <div style="text-align: center; padding: 12px; border: 1px solid var(--tg-theme-section-separator-color, #e1e8ed); border-radius: 8px;">
                    <div style="font-size: 20px; font-weight: 600; color: var(--tg-theme-link-color, #2481cc);">
                        ${user.total_deals || 0}
                    </div>
                    <div style="font-size: 12px; color: var(--tg-theme-hint-color, #708499);">
                        Всего сделок
                    </div>
                </div>
                <div style="text-align: center; padding: 12px; border: 1px solid var(--tg-theme-section-separator-color, #e1e8ed); border-radius: 8px;">
                    <div style="font-size: 20px; font-weight: 600; color: #22c55e;">
                        ${user.successful_deals || 0}
                    </div>
                    <div style="font-size: 12px; color: var(--tg-theme-hint-color, #708499);">
                        Успешных
                    </div>
                </div>
            </div>
            
            <div style="font-size: 12px; color: var(--tg-theme-hint-color, #708499);">
                Telegram ID: ${user.id}
            </div>
        </div>
    `;
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

// Инициализация приложения
document.addEventListener('DOMContentLoaded', () => {
    initTelegramWebApp();
    initNavigation();
    initModal();
});