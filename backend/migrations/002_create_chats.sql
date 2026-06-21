-- Создание таблицы чатов
CREATE TABLE IF NOT EXISTS chats (
                                     id SERIAL PRIMARY KEY,
                                     created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                                     updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                                     deleted_at TIMESTAMP WITH TIME ZONE,

                                     name VARCHAR(255),
    type VARCHAR(20) DEFAULT 'private' CHECK (type IN ('private', 'group')),
    avatar VARCHAR(500),
    description TEXT,

    -- Для групповых чатов
    owner_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    is_archived BOOLEAN DEFAULT FALSE
    );

-- Создание таблицы связи пользователей и чатов
CREATE TABLE IF NOT EXISTS chat_users (
                                          chat_id INTEGER NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    is_admin BOOLEAN DEFAULT FALSE,
    last_read TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                                                                                            notifications BOOLEAN DEFAULT TRUE,

                                                                                            PRIMARY KEY (chat_id, user_id)
    );

-- Индексы для чатов
CREATE INDEX idx_chats_type ON chats(type);
CREATE INDEX idx_chats_owner_id ON chats(owner_id);
CREATE INDEX idx_chats_updated_at ON chats(updated_at DESC);
CREATE INDEX idx_chats_deleted_at ON chats(deleted_at);
CREATE INDEX idx_chat_users_user_id ON chat_users(user_id);
CREATE INDEX idx_chat_users_chat_id ON chat_users(chat_id);
CREATE INDEX idx_chat_users_last_read ON chat_users(last_read);

-- Комментарии к таблицам
COMMENT ON TABLE chats IS 'Таблица чатов (приватных и групповых)';
COMMENT ON COLUMN chats.type IS 'Тип чата: private - приватный, group - групповой';
COMMENT ON COLUMN chats.owner_id IS 'Владелец группового чата';
COMMENT ON TABLE chat_users IS 'Таблица связи пользователей и чатов (участники)';
COMMENT ON COLUMN chat_users.last_read IS 'Время последнего прочтения сообщений пользователем';
COMMENT ON COLUMN chat_users.notifications IS 'Включены ли уведомления для пользователя';
