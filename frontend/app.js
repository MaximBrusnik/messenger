// Главное приложение Vue
const { createApp } = Vue;

createApp({
    data() {
        return {
            //mobile
            isMobile: false,
            sidebarVisible: true,
            // модалка контактов
            newContactModal: false,
            contactSearch: '',
            // Состояние приложения
            loading: true,
            isAuthenticated: false,
            authView: 'login',
            authLoading: false,

            // Текущий пользователь
            currentUser: null,

            // Навигация
            currentView: 'chats',
            activeChat: null,

            // Поиск
            searchQuery: '',
            newChatSearch: '',

            // Данные
            chats: [],
            contacts: [],
            users: [],

            // Формы
            loginForm: {
                username: '',
                password: ''
            },
            registerForm: {
                username: '',
                email: '',
                password: '',
                confirmPassword: ''
            },
            profileForm: {
                username: '',
                email: ''
            },
            passwordForm: {
                oldPassword: '',
                newPassword: '',
                confirmPassword: ''
            },
            newMessage: '',

            // Модальные окна
            newChat: false,
            changePasswordModal: false,

            // Уведомления
            notification: {
                show: false,
                type: 'info',
                message: '',
                icon: 'fas fa-info-circle'
            },

            // WebSocket для реального времени
            ws: null
        };
    },

    computed: {
        //
        allUsersForContacts() {
            return this.users.filter(user =>
                user.id !== this.currentUser.id &&
                (user.username.toLowerCase().includes(this.contactSearch.toLowerCase()) ||
                    user.email.toLowerCase().includes(this.contactSearch.toLowerCase()))
            );
        },
        // Фильтрованные чаты
        filteredChats() {
            if (!this.searchQuery) return this.chats;
            return this.chats.filter(chat =>
                chat.name.toLowerCase().includes(this.searchQuery.toLowerCase()) ||
                chat.lastMessage?.text.toLowerCase().includes(this.searchQuery.toLowerCase())
            );
        },

        // Фильтрованные контакты
        filteredContacts() {
            if (!this.searchQuery) return this.contacts;
            return this.contacts.filter(contact =>
                contact.username.toLowerCase().includes(this.searchQuery.toLowerCase()) ||
                contact.email.toLowerCase().includes(this.searchQuery.toLowerCase())
            );
        },

        // Поиск пользователей для нового чата
        searchUsers() {
            return this.users.filter(user =>
                user.id !== this.currentUser.id &&
                user.username.toLowerCase().includes(this.newChatSearch.toLowerCase())
            );
        },

        // Непрочитанные сообщения
        unreadCount() {
            if (!this.chats || !Array.isArray(this.chats)) {
                return 0
            }
            return this.chats.reduce((sum, chat) => sum + (chat.unread || 0), 0)
        }
    },

    async created() {
        // Проверяем мобильное устройство
        this.checkMobile();
        window.addEventListener('resize', this.checkMobile);

        await this.checkAuth();
        this.loading = false;
    },

    methods: {
        // Проверить мобильное устройство
        checkMobile() {
            this.isMobile = window.innerWidth <= 768;

            // На мобильных при открытии чата скрываем сайдбар
            if (this.isMobile && this.currentView === 'chat') {
                this.sidebarVisible = false;
                // Проверяем, что элемент существует перед использованием classList
                const messengerElement = document.querySelector('.messenger');
                if (messengerElement) {
                    messengerElement.classList.add('chat-open');
                }
            } else {
                this.sidebarVisible = true;
                const messengerElement = document.querySelector('.messenger');
                if (messengerElement) {
                    messengerElement.classList.remove('chat-open');
                }
            }
        },

        // Переключить видимость сайдбара на мобильных
        toggleSidebar() {
            if (this.isMobile) {
                this.sidebarVisible = !this.sidebarVisible;
                const messengerElement = document.querySelector('.messenger');
                if (messengerElement) {
                    if (this.sidebarVisible) {
                        messengerElement.classList.remove('chat-open');
                    } else {
                        messengerElement.classList.add('chat-open');
                    }
                }
            }
        },

        // Вернуться к списку чатов на мобильных
        backToChats() {
            if (this.isMobile) {
                this.currentView = 'chats';
                this.activeChat = null;
                this.sidebarVisible = true;
                const messengerElement = document.querySelector('.messenger');
                if (messengerElement) {
                    messengerElement.classList.remove('chat-open');
                }
            } else {
                this.currentView = 'chats';
            }
        },

        async addContact(userId) {
            try {
                await MessengerAPI.users.addContact(userId);
                // Обновляем список контактов
                const contactsResponse = await MessengerAPI.users.getContacts();
                this.contacts = contactsResponse.data.data;

                this.showNotification('Контакт добавлен', 'success');
            } catch (error) {
                const message = error.response?.data?.error || 'Ошибка добавления контакта';
                this.showNotification(message, 'error');
            }
        },

        // Удалить контакт
        async removeContact(contactId) {
            if (!confirm('Удалить контакт?')) return;

            try {
                await MessengerAPI.users.removeContact(contactId);
                // Обновляем список контактов
                this.contacts = this.contacts.filter(c => c.id !== contactId);

                this.showNotification('Контакт удален', 'success');
            } catch (error) {
                const message = error.response?.data?.error || 'Ошибка удаления контакта';
                this.showNotification(message, 'error');
            }
        },

        // Проверить, есть ли пользователь в контактах
        isContact(userId) {
            return this.contacts.some(contact => contact.id === userId);
        },

        // Проверить, является ли пользователь текущим пользователем
        isCurrentUser(userId) {
            return this.currentUser && this.currentUser.id === userId;
        },
        // Показать уведомление
        showNotification(message, type = 'info') {
            const icons = {
                success: 'fas fa-check-circle',
                error: 'fas fa-exclamation-circle',
                info: 'fas fa-info-circle'
            };

            this.notification = {
                show: true,
                type,
                message,
                icon: icons[type] || 'fas fa-info-circle'
            };

            setTimeout(() => {
                this.notification.show = false;
            }, 3000);
        },

        // Инициалы для аватара
        getInitials(name) {
            if (!name) return '?';
            return name
                .split(' ')
                .map(part => part.charAt(0))
                .join('')
                .toUpperCase()
                .substring(0, 2);
        },

        // Форматирование времени
        formatTime(dateString) {
            if (!dateString) return '';
            const date = new Date(dateString);
            const now = new Date();
            const diff = now - date;

            if (diff < 24 * 60 * 60 * 1000) {
                // Сегодня
                return date.toLocaleTimeString('ru-RU', {
                    hour: '2-digit',
                    minute: '2-digit'
                });
            } else if (diff < 7 * 24 * 60 * 60 * 1000) {
                // На этой неделе
                return date.toLocaleDateString('ru-RU', {
                    weekday: 'short'
                });
            } else {
                // Ранее
                return date.toLocaleDateString('ru-RU', {
                    day: 'numeric',
                    month: 'short'
                });
            }
        },

        // Форматирование даты
        formatDate(dateString) {
            if (!dateString) return '';
            return new Date(dateString).toLocaleDateString('ru-RU', {
                year: 'numeric',
                month: 'long',
                day: 'numeric',
                hour: '2-digit',
                minute: '2-digit'
            });
        },

        // Проверить авторизацию
        async checkAuth() {
            const token = localStorage.getItem('token');
            const user = localStorage.getItem('user');

            if (!token) return;

            try {
                const response = await MessengerAPI.auth.getProfile();
                this.isAuthenticated = true;
                this.currentUser = response.data.data;
                this.profileForm = { ...response.data.data };

                // Загружаем данные
                await this.loadData();

                // Подключаем WebSocket
                this.connectWebSocket();

            } catch (error) {
                this.clearAuth();
            }
        },

        // Загрузить данные
        async loadData() {
            try {
                // Загружаем чаты
                const chatsResponse = await MessengerAPI.chats.getAll();
                this.chats = chatsResponse.data.data;

                // Загружаем контакты
                const contactsResponse = await MessengerAPI.users.getContacts();
                this.contacts = contactsResponse.data.data;

                // Загружаем пользователей
                const usersResponse = await MessengerAPI.users.getAll();
                this.users = usersResponse.data.data;

            } catch (error) {
                console.error('Error loading data:', error);
            }
        },

        // Подключить WebSocket
        connectWebSocket() {
            const token = localStorage.getItem('token');
            if (!token) return;

            // Если уже подключены, закрываем
            if (this.ws) {
                this.ws.close();
            }

            // Подключаемся к WebSocket серверу
            this.ws = new WebSocket(`ws://localhost:8080/ws?token=${token}`);

            this.ws.onopen = () => {
                console.log('WebSocket connected');
            };

            this.ws.onmessage = (event) => {
                const data = JSON.parse(event.data);
                this.handleWebSocketMessage(data);
            };

            this.ws.onclose = () => {
                console.log('WebSocket disconnected');
                // Пробуем переподключиться через 5 секунд
                setTimeout(() => this.connectWebSocket(), 5000);
            };
        },

        // Обработка сообщений WebSocket
        handleWebSocketMessage(data) {
            switch (data.type) {
                case 'NEW_MESSAGE':
                    this.handleNewMessage(data.message);
                    break;
                case 'CHAT_UPDATED':
                    this.updateChat(data.chat);
                    break;
                case 'USER_STATUS':
                    this.updateUserStatus(data.userId, data.status);
                    break;
            }
        },

        // Новое сообщение
        handleNewMessage(message) {
            // Обновляем активный чат
            if (this.activeChat && this.activeChat.id === message.chat_id) {
                this.activeChat.messages.push(message);
                this.$nextTick(() => {
                    this.scrollToBottom();
                });
            }

            // Обновляем список чатов
            const chat = this.chats.find(c => c.id === message.chat_id);
            if (chat) {
                chat.lastMessage = message;
                chat.unread = (chat.unread || 0) + 1;

                // Перемещаем чат вверх
                this.chats = [
                    chat,
                    ...this.chats.filter(c => c.id !== chat.id)
                ];
            }
        },

        // Обновить чат
        updateChat(updatedChat) {
            const index = this.chats.findIndex(c => c.id === updatedChat.id);
            if (index !== -1) {
                this.chats[index] = updatedChat;
            } else {
                this.chats.unshift(updatedChat);
            }
        },

        // Обновить статус пользователя
        updateUserStatus(userId, status) {
            // Обновляем статус в контактах
            const contact = this.contacts.find(c => c.id === userId);
            if (contact) {
                contact.status = status;
            }

            // Обновляем статус в активном чате
            if (this.activeChat && this.activeChat.participants) {
                const participant = this.activeChat.participants.find(p => p.id === userId);
                if (participant) {
                    participant.status = status;
                }
            }
        },

        // Открыть чат
        async openChat(chat) {
            this.currentView = 'chat';
            this.activeChat = chat;

            try {
                // Загружаем сообщения
                const response = await MessengerAPI.chats.getMessages(chat.id);
                this.activeChat.messages = response.data.data;

                // Помечаем как прочитанные
                await MessengerAPI.chats.markAsRead(chat.id);
                chat.unread = 0;

                this.$nextTick(() => {
                    this.scrollToBottom();
                });

            } catch (error) {
                this.showNotification('Ошибка загрузки сообщений', 'error');
            }
        },

        // Начать новый чат
        async startChat(user) {
            try {
                const response = await MessengerAPI.chats.createChat(user.id);
                const newChat = response.data.data;

                this.chats.unshift(newChat);
                this.openChat(newChat);
                this.newChat = false;
                this.newChatSearch = '';

                this.showNotification('Чат создан', 'success');

            } catch (error) {
                this.showNotification('Ошибка создания чата', 'error');
            }
        },

        // Создать чат (из модального окна)
        createChat(user) {
            this.startChat(user);
        },

        // Отправить сообщение
        async sendMessage() {
            if (!this.newMessage.trim() || !this.activeChat) return;

            try {
                const response = await MessengerAPI.chats.sendMessage(
                    this.activeChat.id,
                    this.newMessage.trim()
                );

                const message = response.data.data;
                this.activeChat.messages.push(message);
                this.activeChat.lastMessage = message;

                this.newMessage = '';
                this.scrollToBottom();

            } catch (error) {
                this.showNotification('Ошибка отправки сообщения', 'error');
            }
        },

        // Прокрутить вниз
        scrollToBottom() {
            const container = this.$refs.messagesContainer;
            if (container) {
                container.scrollTop = container.scrollHeight;
            }
        },

        // Вход
        async handleLogin() {
            this.authLoading = true;

            try {
                const response = await MessengerAPI.auth.login(this.loginForm);
                const { token, data: user } = response.data;

                // Сохраняем
                localStorage.setItem('token', token);
                localStorage.setItem('user', JSON.stringify(user));

                // Обновляем состояние
                this.isAuthenticated = true;
                this.currentUser = user;
                this.profileForm = { ...user };

                // Загружаем данные
                await this.loadData();

                // Подключаем WebSocket
                this.connectWebSocket();

                this.showNotification('Успешный вход!', 'success');

            } catch (error) {
                const message = error.response?.data?.error || 'Ошибка входа';
                this.showNotification(message, 'error');
            } finally {
                this.authLoading = false;
            }
        },

        // Регистрация
        async handleRegister() {
            // Валидация
            if (this.registerForm.password !== this.registerForm.confirmPassword) {
                this.showNotification('Пароли не совпадают', 'error');
                return;
            }

            if (this.registerForm.password.length < 6) {
                this.showNotification('Пароль должен быть минимум 6 символов', 'error');
                return;
            }

            this.authLoading = true;

            try {
                await MessengerAPI.auth.register({
                    username: this.registerForm.username,
                    email: this.registerForm.email,
                    password: this.registerForm.password
                });

                // Переключаемся на вход
                this.authView = 'login';
                this.registerForm = {
                    username: '',
                    email: '',
                    password: '',
                    confirmPassword: ''
                };

                this.showNotification('Регистрация успешна! Теперь войдите.', 'success');

            } catch (error) {
                const message = error.response?.data?.error || 'Ошибка регистрации';
                this.showNotification(message, 'error');
            } finally {
                this.authLoading = false;
            }
        },

        // Обновить профиль
        async updateProfile() {
            try {
                await MessengerAPI.auth.updateProfile(this.profileForm);

                // Обновляем текущего пользователя
                Object.assign(this.currentUser, this.profileForm);
                localStorage.setItem('user', JSON.stringify(this.currentUser));

                this.showNotification('Профиль обновлен', 'success');

            } catch (error) {
                this.showNotification('Ошибка обновления профиля', 'error');
            }
        },

        // Сменить пароль
        async changePassword() {
            // Валидация
            if (this.passwordForm.newPassword !== this.passwordForm.confirmPassword) {
                this.showNotification('Пароли не совпадают', 'error');
                return;
            }

            if (this.passwordForm.newPassword.length < 6) {
                this.showNotification('Пароль должен быть минимум 6 символов', 'error');
                return;
            }

            try {
                await MessengerAPI.auth.changePassword({
                    old_password: this.passwordForm.oldPassword,
                    new_password: this.passwordForm.newPassword
                });

                this.passwordForm = {
                    oldPassword: '',
                    newPassword: '',
                    confirmPassword: ''
                };
                this.changePasswordModal = false;

                this.showNotification('Пароль успешно изменен', 'success');

            } catch (error) {
                const message = error.response?.data?.error || 'Ошибка смены пароля';
                this.showNotification(message, 'error');
            }
        },

        // Выход
        async logout() {
            try {
                await MessengerAPI.auth.logout();
            } catch (error) {
                // Игнорируем ошибки
            }

            this.clearAuth();
            this.showNotification('Вы вышли из системы', 'success');
        },

        // Очистить авторизацию
        clearAuth() {
            this.isAuthenticated = false;
            this.currentUser = null;
            this.chats = [];
            this.contacts = [];
            this.users = [];

            localStorage.removeItem('token');
            localStorage.removeItem('user');

            // Закрываем WebSocket
            if (this.ws) {
                this.ws.close();
                this.ws = null;
            }

            // Сбрасываем вид
            this.currentView = 'chats';
            this.authView = 'login';
        },

        // Загрузка аватара
        uploadAvatar() {
            // В реальном приложении здесь была бы загрузка файла
            this.showNotification('Функция загрузки аватара в разработке', 'info');
        }
    }
}).mount('#app');