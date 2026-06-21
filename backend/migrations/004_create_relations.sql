-- Создание таблицы контактов
CREATE TABLE IF NOT EXISTS user_contacts (
                                             user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    contact_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                                                                                               nickname VARCHAR(100),
    is_blocked BOOLEAN DEFAULT FALSE,
    is_favorite BOOLEAN DEFAULT FALSE,

    PRIMARY KEY (user_id, contact_id),
    CHECK (user_id != contact_id) -- Нельзя добавить себя в контакты
    );

-- Создание таблицы настроек пользователей
CREATE TABLE IF NOT EXISTS user_settings (
                                             user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    theme VARCHAR(20) DEFAULT 'light' CHECK (theme IN ('light', 'dark')),
    language VARCHAR(10) DEFAULT 'ru',
    notifications BOOLEAN DEFAULT TRUE,
    sound BOOLEAN DEFAULT TRUE,
    show_online_status BOOLEAN DEFAULT TRUE,
    last_seen_privacy VARCHAR(20) DEFAULT 'everyone' CHECK (last_seen_privacy IN ('everyone', 'contacts', 'nobody')),
    read_receipts BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
                                                                                                  );

-- Создание таблицы отложенных сообщений
CREATE TABLE IF NOT EXISTS scheduled_messages (
                                                  id SERIAL PRIMARY KEY,
                                                  chat_id INTEGER NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    sender_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    text TEXT NOT NULL,
    scheduled_for TIMESTAMP WITH TIME ZONE NOT NULL,
    sent BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
                                                                                                    );

-- Создание таблицы реакций на сообщения
CREATE TABLE IF NOT EXISTS message_reactions (
                                                 message_id INTEGER NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reaction VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

                                                                                                         PRIMARY KEY (message_id, user_id)
    );

-- Создание таблицы звонков
CREATE TABLE IF NOT EXISTS calls (
                                     id SERIAL PRIMARY KEY,
                                     chat_id INTEGER REFERENCES chats(id) ON DELETE SET NULL,
    caller_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    ended_at TIMESTAMP WITH TIME ZONE,
                                                                              duration INTEGER, -- в секундах
                                                                              call_type VARCHAR(20) CHECK (call_type IN ('audio', 'video')),
    status VARCHAR(20) CHECK (status IN ('missed', 'answered', 'rejected', 'cancelled'))
    );

-- Создание таблицы участников звонков
CREATE TABLE IF NOT EXISTS call_participants (
                                                 call_id INTEGER NOT NULL REFERENCES calls(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    left_at TIMESTAMP WITH TIME ZONE,

                                                                                                   PRIMARY KEY (call_id, user_id)
    );

-- Индексы для новых таблиц
CREATE INDEX idx_user_contacts_user_id ON user_contacts(user_id);
CREATE INDEX idx_user_contacts_contact_id ON user_contacts(contact_id);
CREATE INDEX idx_user_contacts_is_blocked ON user_contacts(is_blocked) WHERE is_blocked;
CREATE INDEX idx_scheduled_messages_scheduled_for ON scheduled_messages(scheduled_for) WHERE NOT sent;
CREATE INDEX idx_scheduled_messages_sent ON scheduled_messages(sent);
CREATE INDEX idx_message_reactions_message_id ON message_reactions(message_id);
CREATE INDEX idx_calls_chat_id ON calls(chat_id);
CREATE INDEX idx_calls_caller_id ON calls(caller_id);
CREATE INDEX idx_calls_started_at ON calls(started_at DESC);
CREATE INDEX idx_call_participants_call_id ON call_participants(call_id);
CREATE INDEX idx_call_participants_user_id ON call_participants(user_id);

-- Комментарии к таблицам
COMMENT ON TABLE user_contacts IS 'Таблица контактов пользователей';
COMMENT ON COLUMN user_contacts.nickname IS 'Кастомное имя для контакта';
COMMENT ON COLUMN user_contacts.is_blocked IS 'Заблокирован ли контакт';
COMMENT ON COLUMN user_contacts.is_favorite IS 'Добавлен ли в избранное';

COMMENT ON TABLE user_settings IS 'Настройки пользователей';

COMMENT ON TABLE scheduled_messages IS 'Отложенные сообщения';

COMMENT ON TABLE message_reactions IS 'Реакции на сообщения (лайки, эмодзи)';

COMMENT ON TABLE calls IS 'История звонков';
COMMENT ON COLUMN calls.duration IS 'Длительность звонка в секундах';