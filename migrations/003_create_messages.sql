-- Создание таблицы сообщений
CREATE TABLE IF NOT EXISTS messages (
                                        id SERIAL PRIMARY KEY,
                                        created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                                        updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                                        deleted_at TIMESTAMP WITH TIME ZONE,

                                        chat_id INTEGER NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    sender_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    text TEXT NOT NULL,
    is_read BOOLEAN DEFAULT FALSE,

    -- Для сообщений с файлами
    attachment_type VARCHAR(50) CHECK (attachment_type IN ('image', 'video', 'audio', 'file', 'sticker')),
    attachment_url VARCHAR(500),
    attachment_name VARCHAR(255),
    attachment_size INTEGER,

    -- Для цитирования сообщений
    reply_to_id INTEGER REFERENCES messages(id) ON DELETE SET NULL,

    -- Метаданные
    edited BOOLEAN DEFAULT FALSE,
    edited_at TIMESTAMP WITH TIME ZONE
                                                                                          );

-- Индексы для сообщений
CREATE INDEX idx_messages_chat_id ON messages(chat_id);
CREATE INDEX idx_messages_sender_id ON messages(sender_id);
CREATE INDEX idx_messages_created_at ON messages(created_at DESC);
CREATE INDEX idx_messages_is_read ON messages(is_read) WHERE NOT is_read;
CREATE INDEX idx_messages_reply_to_id ON messages(reply_to_id);
CREATE INDEX idx_messages_deleted_at ON messages(deleted_at);

-- Функция для автоматического обновления updated_at у чата
CREATE OR REPLACE FUNCTION update_chat_timestamp()
RETURNS TRIGGER AS $$
BEGIN
UPDATE chats SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.chat_id;
RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Триггер для обновления времени чата при новом сообщении
CREATE TRIGGER update_chat_on_message
    AFTER INSERT ON messages
    FOR EACH ROW
    EXECUTE FUNCTION update_chat_timestamp();

-- Комментарии к таблице
COMMENT ON TABLE messages IS 'Таблица сообщений в чатах';
COMMENT ON COLUMN messages.text IS 'Текст сообщения';
COMMENT ON COLUMN messages.attachment_type IS 'Тип вложения (если есть)';
COMMENT ON COLUMN messages.attachment_url IS 'URL вложения';
COMMENT ON COLUMN messages.reply_to_id IS 'ID сообщения, на которое отвечают';
COMMENT ON COLUMN messages.edited IS 'Было ли сообщение отредактировано';
COMMENT ON COLUMN messages.edited_at IS 'Время последнего редактирования';