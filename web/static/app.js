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

// Инициализация приложения
document.addEventListener('DOMContentLoaded', () => {
    initTelegramWebApp();
    initNavigation();
    initModal();
});