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
        // Получаем пользователя по Telegram ID
        const user = await getUserByTelegramID();
        if (!user) {
            displayProfile(currentUser);
            return;
        }

        // Параллельно загружаем профиль, отзывы и рейтинг
        const [profileResponse, reviewsResponse, ratingResponse] = await Promise.all([
            fetch('/api/v1/auth/me', {
                headers: { 'X-Telegram-User-ID': currentUser.id.toString() }
            }).catch(() => null),
            fetch(`/api/v1/reviews?user_id=${user.id}&limit=5`, {
                headers: { 'X-Telegram-User-ID': currentUser.id.toString() }
            }).catch(() => null),
            fetch(`/api/v1/users/${user.id}/profile`, {
                headers: { 'X-Telegram-User-ID': currentUser.id.toString() }
            }).catch(() => null)
        ]);

        let profileData = currentUser;
        let reviews = [];
        let stats = null;

        // Парсим ответы
        if (profileResponse && profileResponse.ok) {
            const profileResult = await profileResponse.json();
            profileData = profileResult.user || currentUser;
        }

        if (reviewsResponse && reviewsResponse.ok) {
            const reviewsResult = await reviewsResponse.json();
            reviews = reviewsResult.reviews || [];
        }

        if (ratingResponse && ratingResponse.ok) {
            const statsResult = await ratingResponse.json();
            stats = statsResult;
        }

        displayProfileWithReviews(profileData, reviews, stats);
    } catch (error) {
        console.error('[ERROR] Ошибка загрузки профиля:', error);
        displayProfile(currentUser);
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
                    ${!review.is_anonymous ? `
                    <div style="font-size: 11px; color: var(--tg-theme-hint-color, #708499); margin-top: 6px;">
                        От: Пользователь #${review.from_user_id}
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
        modal.style.display = 'flex';
        
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
        modal.style.display = 'none';
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