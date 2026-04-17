    const API_BASE = window.location.origin;
    const CACHE_KEY = 'user_urls_cache';
    const CACHE_TIMESTAMP_KEY = 'user_urls_cache_timestamp';
    const CACHE_DURATION = 5 * 60 * 1000;
    
    let isLoading = false;
    let copyTimeout = null; // для таймера кнопки копирования
    
    // Функция склонения слова "ссылка/ссылки/ссылок"
    function getLinksCountText(count) {
        if (count % 10 === 1 && count % 100 !== 11) {
            return `${count} ссылка`;
        } else if (count % 10 >= 2 && count % 10 <= 4 && (count % 100 < 10 || count % 100 >= 20)) {
            return `${count} ссылки`;
        } else {
            return `${count} ссылок`;
        }
    }
    
    function showNotification(message, type = 'info') {
        const oldNotif = document.querySelector('.custom-notification');
        if (oldNotif) oldNotif.remove();
        const notif = document.createElement('div');
        notif.className = 'custom-notification';
        notif.textContent = message;
        const colors = { success: '#10b981', error: '#ef4444', info: '#4f46e5' };
        notif.style.backgroundColor = colors[type] || '#334155';
        document.body.appendChild(notif);
        setTimeout(() => {
            notif.style.opacity = '0';
            notif.style.transform = 'translateX(100%)';
            setTimeout(() => notif.remove(), 300);
        }, 2800);
    }
    
    function getCachedUrls() {
        try {
            const cached = localStorage.getItem(CACHE_KEY);
            const timestamp = localStorage.getItem(CACHE_TIMESTAMP_KEY);
            if (!cached || !timestamp) return null;
            const now = Date.now();
            if (now - parseInt(timestamp) > CACHE_DURATION) {
                localStorage.removeItem(CACHE_KEY);
                localStorage.removeItem(CACHE_TIMESTAMP_KEY);
                return null;
            }
            return JSON.parse(cached);
        } catch (e) {
            console.warn('Ошибка чтения кэша', e);
            return null;
        }
    }
    
    function saveUrlsToCache(urls) {
        try {
            localStorage.setItem(CACHE_KEY, JSON.stringify(urls));
            localStorage.setItem(CACHE_TIMESTAMP_KEY, Date.now().toString());
            updateCacheUI();
        } catch(e) { console.error(e); }
    }
    
    function clearCache() {
        localStorage.removeItem(CACHE_KEY);
        localStorage.removeItem(CACHE_TIMESTAMP_KEY);
        updateCacheUI();
        showNotification('🧹 Кэш очищен, следующие данные будут взяты с сервера', 'info');
    }
    
    // Флаг, показывающий, было ли реальное изменение из другой вкладки
    let externalChangeDetected = false;
    // Храним последнее известное состояние кэша для сравнения
    let lastKnownCacheState = null;
    
    function updateCacheUI() {
        const cached = getCachedUrls();
        const indicator = document.getElementById('cacheIndicator');
        const warningDiv = document.getElementById('remoteChangeWarning');
        if (!indicator) return;
        
        if (cached !== null) {
            const timestamp = localStorage.getItem(CACHE_TIMESTAMP_KEY);
            let timeStr = '';
            if (timestamp) {
                const date = new Date(parseInt(timestamp));
                timeStr = ` (${date.toLocaleTimeString()})`;
            }
            const countText = getLinksCountText(cached.length);
            indicator.innerHTML = `📦 Кэш активен${timeStr} • ${countText}`;
            // Предупреждение показываем ТОЛЬКО если был детектирован внешний change
            if (externalChangeDetected && warningDiv) {
                warningDiv.style.display = 'inline-flex';
            } else if (warningDiv) {
                warningDiv.style.display = 'none';
            }
        } else {
            indicator.innerHTML = `🟢 Актуальные данные с сервера`;
            if (warningDiv) warningDiv.style.display = 'none';
            externalChangeDetected = false;
        }
    }
    
    async function createShortUrl(originalUrl) {
        const response = await fetch(`${API_BASE}/api/shorten`, {
            method: 'POST',
            credentials: 'include',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ url: originalUrl })
        });
        if (response.status === 409) {
            const data = await response.json();
            return data.result;
        }
        if (!response.ok) throw new Error(`Ошибка ${response.status}`);
        const data = await response.json();
        return data.result;
    }
    
    async function getUserUrls(forceRefresh = false) {
        if (!forceRefresh) {
            const cached = getCachedUrls();
            if (cached !== null) {
                console.log('Используем кэш ссылок');
                return cached;
            }
        }
        if (isLoading) {
            await new Promise(resolve => {
                const interval = setInterval(() => {
                    if (!isLoading) {
                        clearInterval(interval);
                        resolve();
                    }
                }, 80);
            });
            return getUserUrls(forceRefresh);
        }
        isLoading = true;
        try {
            const response = await fetch(`${API_BASE}/api/user/urls`, {
                method: 'GET',
                credentials: 'include',
            });
            if (response.status === 204) {
                saveUrlsToCache([]);
                return [];
            }
            if (!response.ok) throw new Error(`Ошибка сервера ${response.status}`);
            const data = await response.json();
            saveUrlsToCache(data);
            // При успешной загрузке с сервера сбрасываем флаг внешнего изменения
            externalChangeDetected = false;
            // Обновляем сохранённое состояние
            lastKnownCacheState = JSON.stringify(data);
            return data;
        } catch (err) {
            console.error('Ошибка загрузки с сервера', err);
            const staleCache = getCachedUrls();
            if (staleCache && staleCache.length) {
                showNotification('⚠️ Ошибка соединения, показаны сохранённые ссылки (кэш)', 'info');
                return staleCache;
            }
            throw err;
        } finally {
            isLoading = false;
        }
    }
    
    async function deleteUrls(shortCodes) {
        const response = await fetch(`${API_BASE}/api/user/urls`, {
            method: 'DELETE',
            credentials: 'include',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(shortCodes)
        });
        if (response.status !== 202 && !response.ok) throw new Error(`Ошибка удаления ${response.status}`);
        const freshResp = await fetch(`${API_BASE}/api/user/urls`, {
            method: 'GET',
            credentials: 'include',
        });
        if (freshResp.ok && freshResp.status !== 204) {
            const freshData = await freshResp.json();
            saveUrlsToCache(freshData);
            externalChangeDetected = false;
            lastKnownCacheState = JSON.stringify(freshData);
            return freshData;
        } else if (freshResp.status === 204) {
            saveUrlsToCache([]);
            externalChangeDetected = false;
            lastKnownCacheState = JSON.stringify([]);
            return [];
        }
        clearCache();
        return [];
    }
    
    function createUrlElement(urlItem) {
        const div = document.createElement('div');
        div.className = 'url-item';
        const shortCode = urlItem.short_url.split('/').pop();
        div.setAttribute('data-short', shortCode);
        const displayOriginal = urlItem.original_url.length > 80 ? urlItem.original_url.slice(0, 80) + '…' : urlItem.original_url;
        div.innerHTML = `
            <div class="url-info">
                <div class="url-short">
                    <a href="${urlItem.short_url}" target="_blank">🔗 ${urlItem.short_url}</a>
                </div>
                <div class="url-original" title="${urlItem.original_url.replace(/&/g, '&amp;')}">📄 ${escapeHtml(displayOriginal)}</div>
            </div>
            <button class="delete-btn" data-short="${shortCode}">🗑️ Удалить</button>
        `;
        const delBtn = div.querySelector('.delete-btn');
        delBtn.addEventListener('click', async (e) => {
            e.stopPropagation();
            if (!confirm(`Удалить ссылку?\n${urlItem.short_url}`)) return;
            delBtn.disabled = true;
            delBtn.textContent = '⏳ ...';
            try {
                await deleteUrls([shortCode]);
                showNotification('✅ Ссылка удалена, список обновлён', 'success');
                await renderUrlsList(true);
            } catch (err) {
                showNotification('❌ Ошибка удаления', 'error');
                delBtn.disabled = false;
                delBtn.textContent = '🗑️ Удалить';
            }
        });
        return div;
    }
    
    function escapeHtml(str) {
        if (!str) return '';
        return str.replace(/[&<>]/g, function(m) {
            if (m === '&') return '&amp;';
            if (m === '<') return '&lt;';
            if (m === '>') return '&gt;';
            return m;
        });
    }
    
    async function renderUrlsList(forceRefresh = false) {
        const container = document.getElementById('urlsList');
        const refreshBtn = document.getElementById('refreshBtn');
        if (!container) return;
        
        const hasCache = !forceRefresh && getCachedUrls() !== null;
        if (!hasCache || forceRefresh) {
            container.innerHTML = '<div class="loading">🔄 Загрузка ссылок...</div>';
        }
        
        if (forceRefresh && refreshBtn) {
            refreshBtn.classList.add('loading');
        }
        
        try {
            const urls = await getUserUrls(forceRefresh);
            if (!urls || urls.length === 0) {
                container.innerHTML = '<div class="loading">✨ У вас пока нет ссылок. Создайте первую!</div>';
                updateCacheUI();
                return;
            }
            container.innerHTML = '';
            urls.forEach(url => {
                if (url && url.short_url && url.original_url) {
                    container.appendChild(createUrlElement(url));
                }
            });
            updateCacheUI();
            if (forceRefresh) {
                showNotification('📡 Список обновлён с сервера', 'success');
                // После обновления сбрасываем флаг внешнего изменения, предупреждение скроется
                externalChangeDetected = false;
                updateCacheUI();
            }
        } catch (err) {
            console.error('Ошибка renderUrlsList:', err);
            const cachedFallback = getCachedUrls();
            if (cachedFallback && cachedFallback.length) {
                container.innerHTML = '';
                cachedFallback.forEach(url => {
                    if (url && url.short_url) container.appendChild(createUrlElement(url));
                });
                updateCacheUI();
                showNotification('⚠️ Ошибка соединения, отображены кэшированные ссылки', 'info');
            } else {
                container.innerHTML = '<div class="error">❌ Не удалось загрузить ссылки. Проверьте соединение с сервером.</div>';
            }
        } finally {
            if (refreshBtn) refreshBtn.classList.remove('loading');
        }
    }
    
    // Событие создания ссылки
    document.getElementById('createForm').addEventListener('submit', async (e) => {
        e.preventDefault();
        const originalUrl = document.getElementById('originalUrl').value.trim();
        const submitBtn = e.target.querySelector('button');
        const resultDiv = document.getElementById('result');
        const shortUrlLink = document.getElementById('shortUrl');
        
        resultDiv.classList.add('hidden');
        if (!originalUrl || !originalUrl.match(/^https?:\/\/.+/)) {
            showNotification('❌ Введите корректный URL (http:// или https://)', 'error');
            return;
        }
        
        try {
            submitBtn.disabled = true;
            submitBtn.textContent = '⏳ Создание...';
            const fullShort = await createShortUrl(originalUrl);
            shortUrlLink.href = fullShort;
            shortUrlLink.textContent = fullShort;
            resultDiv.classList.remove('hidden');
            document.getElementById('originalUrl').value = '';
            showNotification('✅ Короткая ссылка создана!', 'success');
            // Сбрасываем текст кнопки копирования на исходный, если она была изменена ранее
            const copyBtn = document.getElementById('copyBtn');
            if (copyBtn && copyTimeout) {
                clearTimeout(copyTimeout);
                copyBtn.textContent = '📋 Копировать';
            }
            await renderUrlsList(true);
        } catch (error) {
            let msg = 'Ошибка создания ссылки';
            if (error.message.includes('409')) msg = '⚠️ Такая ссылка уже существует';
            else if (error.message.includes('400')) msg = '❌ Неверный формат URL';
            showNotification(msg, 'error');
        } finally {
            submitBtn.disabled = false;
            submitBtn.textContent = '✨ Сократить';
        }
    });
    
    // Копирование с корректным возвратом текста
    const copyBtn = document.getElementById('copyBtn');
    if (copyBtn) {
        copyBtn.addEventListener('click', async () => {
            const shortUrlLink = document.getElementById('shortUrl');
            const url = shortUrlLink.href;
            if (!url || url.includes('undefined') || url === '#') {
                showNotification('❌ Нет ссылки для копирования. Сначала создайте ссылку.', 'error');
                return;
            }
            try {
                await navigator.clipboard.writeText(url);
                showNotification('📋 Ссылка скопирована в буфер обмена!', 'success');
                const originalText = copyBtn.textContent;
                copyBtn.textContent = '✅ Скопировано!';
                if (copyTimeout) clearTimeout(copyTimeout);
                copyTimeout = setTimeout(() => {
                    copyBtn.textContent = originalText === '✅ Скопировано!' ? '📋 Копировать' : originalText;
                    if (copyBtn.textContent === '📋 Копировать') copyTimeout = null;
                }, 2000);
            } catch (err) {
                console.error('Ошибка копирования:', err);
                showNotification('❌ Не удалось скопировать ссылку', 'error');
            }
        });
    }
    
    // Кнопка обновления
    const refreshBtn = document.getElementById('refreshBtn');
    if (refreshBtn) {
        refreshBtn.addEventListener('click', async () => {
            await renderUrlsList(true);
        });
    }
    
    // Очистка кэша вручную
    const clearCacheSpan = document.getElementById('manualClearCache');
    if (clearCacheSpan) {
        clearCacheSpan.addEventListener('click', () => {
            clearCache();
            externalChangeDetected = false;
            renderUrlsList(true);
        });
    }
    
    // Инициализируем начальное состояние кэша при загрузке страницы
    const initialCache = getCachedUrls();
    if (initialCache !== null) {
        lastKnownCacheState = JSON.stringify(initialCache);
    }
    
    // Обработка события storage — только если в другой вкладке изменили те же ключи
    window.addEventListener('storage', (event) => {
        if (event.key === CACHE_KEY || event.key === CACHE_TIMESTAMP_KEY) {
            // Небольшая задержка, чтобы дать localStorage обновиться полностью
            setTimeout(() => {
                const newCached = getCachedUrls();
                const newState = JSON.stringify(newCached);
                
                // Сравниваем с сохранённым состоянием
                if (lastKnownCacheState !== newState) {
                    // Данные изменились в другой вкладке
                    externalChangeDetected = true;
                    lastKnownCacheState = newState;
                    updateCacheUI();
                    
                    // Обновляем отображение списка из кэша
                    const container = document.getElementById('urlsList');
                    if (container) {
                        if (newCached && newCached.length > 0) {
                            container.innerHTML = '';
                            newCached.forEach(url => {
                                if (url && url.short_url) container.appendChild(createUrlElement(url));
                            });
                        } else if (newCached && newCached.length === 0) {
                            container.innerHTML = '<div class="loading">✨ У вас пока нет ссылок. Создайте первую!</div>';
                        }
                    }
                    showNotification('🔄 Данные обновлены в другой вкладке. Нажмите "Обновить" для синхронизации.', 'info');
                }
            }, 10);
        }
    });
    
    // При возврате на вкладку — не спамим предупреждение, просто обновим UI
    document.addEventListener('visibilitychange', () => {
        if (!document.hidden) {
            updateCacheUI();
        }
    });
    
    // Инициализация
    renderUrlsList(false);