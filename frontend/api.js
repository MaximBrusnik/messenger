// API конфигурация
const API_BASE = 'http://localhost:8080/api/v1';

// Создаем экземпляр axios с настройками
const api = axios.create({
    baseURL: API_BASE,
    headers: {
        'Content-Type': 'application/json'
    }
});

// Добавляем токен к каждому запросу
api.interceptors.request.use(config => {
    const token = localStorage.getItem('token');
    if (token) {
        config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
});

// Обработка ошибок
api.interceptors.response.use(
    response => response,
    error => {
        if (error.response?.status === 401) {
            localStorage.removeItem('token');
            localStorage.removeItem('user');
            // Можно перенаправить на страницу входа
            if (window.location.pathname !== '/') {
                window.location.reload();
            }
        }
        return Promise.reject(error);
    }
);

// API методы
const MessengerAPI = {
    // Аутентификация
    auth: {
        login(credentials) {
            return api.post('/auth/login', credentials);
        },
        register(userData) {
            return api.post('/auth/register', userData);
        },
        logout() {
            return api.post('/auth/logout');
        },
        getProfile() {
            return api.get('/auth/profile');
        },
        updateProfile(userData) {
            return api.put('/auth/profile', userData);
        },
        changePassword(passwords) {
            return api.post('/auth/change-password', passwords);
        }
    },

    // Чаты
    chats: {
        getAll() {
            return api.get('/chats');
        },
        getChat(chatId) {
            return api.get(`/chats/${chatId}`);
        },
        createChat(userId) {
            return api.post('/chats', { user_id: userId });
        },
        getMessages(chatId) {
            return api.get(`/chats/${chatId}/messages`);
        },
        sendMessage(chatId, text) {
            return api.post(`/chats/${chatId}/messages`, { text });
        },
        markAsRead(chatId) {
            return api.post(`/chats/${chatId}/read`);
        }
    },

    // Пользователи
    users: {
        getAll() {
            return api.get('/users');
        },
        search(query) {
            return api.get('/users/search', { params: { q: query } });
        },
        addContact(userId) {
            return api.post('/contacts', { user_id: userId });
        },
        getContacts() {
            return api.get('/contacts');
        },
        removeContact(contactId) {
            return api.delete(`/contacts/${contactId}`);
        }
    }
};

// Экспортируем для использования в app.js
window.MessengerAPI = MessengerAPI;