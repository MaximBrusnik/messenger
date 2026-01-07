-- Создание тестовых пользователей
INSERT INTO users (username, email, password, is_active) VALUES
                                                             ('alex', 'alex@example.com', '$2a$10$N9qo8uLOickgx2ZMRZoMy.MrYvJp1pJ.6YlQj5WYlA3vLc6JqK0Oa', true), -- password: password123
                                                             ('maria', 'maria@example.com', '$2a$10$N9qo8uLOickgx2ZMRZoMy.MrYvJp1pJ.6YlQj5WYlA3vLc6JqK0Oa', true), -- password: password123
                                                             ('ivan', 'ivan@example.com', '$2a$10$N9qo8uLOickgx2ZMRZoMy.MrYvJp1pJ.6YlQj5WYlA3vLc6JqK0Oa', true), -- password: password123
                                                             ('olga', 'olga@example.com', '$2a$10$N9qo8uLOickgx2ZMRZoMy.MrYvJp1pJ.6YlQj5WYlA3vLc6JqK0Oa', true), -- password: password123
                                                             ('test', 'test@example.com', '$2a$10$N9qo8uLOickgx2ZMRZoMy.MrYvJp1pJ.6YlQj5WYlA3vLc6JqK0Oa', true); -- password: password123

-- Создание настроек для пользователей
INSERT INTO user_settings (user_id, theme, language) VALUES
                                                         (1, 'light', 'ru'),
                                                         (2, 'dark', 'ru'),
                                                         (3, 'light', 'en'),
                                                         (4, 'dark', 'ru'),
                                                         (5, 'light', 'ru');

-- Создание контактов
INSERT INTO user_contacts (user_id, contact_id, nickname) VALUES
                                                              (1, 2, 'Мария'),
                                                              (1, 3, 'Иван'),
                                                              (2, 1, 'Алекс'),
                                                              (2, 4, 'Ольга'),
                                                              (3, 1, 'Алексей'),
                                                              (4, 2, 'Маша'),
                                                              (5, 1, 'Александр');

-- Создание приватных чатов
INSERT INTO chats (name, type) VALUES
                                   ('Алекс и Мария', 'private'),
                                   ('Алекс и Иван', 'private'),
                                   ('Мария и Ольга', 'private'),
                                   ('Общий чат', 'group');

-- Добавление участников в чаты
INSERT INTO chat_users (chat_id, user_id, is_admin) VALUES
                                                        -- Чат 1: Алекс и Мария
                                                        (1, 1, false),
                                                        (1, 2, false),

                                                        -- Чат 2: Алекс и Иван
                                                        (2, 1, false),
                                                        (2, 3, false),

                                                        -- Чат 3: Мария и Ольга
                                                        (3, 2, false),
                                                        (3, 4, false),

                                                        -- Групповой чат
                                                        (4, 1, true),  -- Алекс - администратор
                                                        (4, 2, false),
                                                        (4, 3, false),
                                                        (4, 4, false),
                                                        (4, 5, false);

-- Обновление владельца группового чата
UPDATE chats SET owner_id = 1 WHERE id = 4;

-- Создание тестовых сообщений
INSERT INTO messages (chat_id, sender_id, text, is_read, created_at) VALUES
                                                                         -- Чат 1
                                                                         (1, 1, 'Привет, Мария!', true, NOW() - INTERVAL '10 minutes'),
                                                                         (1, 2, 'Привет, Алекс! Как дела?', true, NOW() - INTERVAL '9 minutes'),
                                                                         (1, 1, 'Все отлично, спасибо! А у тебя?', true, NOW() - INTERVAL '8 minutes'),

                                                                         -- Чат 2
                                                                         (2, 3, 'Алекс, привет!', false, NOW() - INTERVAL '15 minutes'),
                                                                         (2, 1, 'Иван, здравствуй!', false, NOW() - INTERVAL '14 minutes'),

                                                                         -- Чат 3
                                                                         (3, 2, 'Оля, как настроение?', true, NOW() - INTERVAL '20 minutes'),
                                                                         (3, 4, 'Отлично! Давай сегодня встретимся?', true, NOW() - INTERVAL '19 minutes'),

                                                                         -- Групповой чат
                                                                         (4, 1, 'Всем привет!', true, NOW() - INTERVAL '5 minutes'),
                                                                         (4, 2, 'Привет всем!', true, NOW() - INTERVAL '4 minutes'),
                                                                         (4, 3, 'Здравствуйте!', true, NOW() - INTERVAL '3 minutes'),
                                                                         (4, 4, 'Приветствую!', true, NOW() - INTERVAL '2 minutes'),
                                                                         (4, 5, 'Всем доброго дня!', false, NOW() - INTERVAL '1 minute');

-- Создание тестовых звонков
INSERT INTO calls (chat_id, caller_id, started_at, ended_at, duration, call_type, status) VALUES
                                                                                              (1, 1, NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day' + INTERVAL '5 minutes', 300, 'audio', 'answered'),
                                                                                              (2, 3, NOW() - INTERVAL '2 days', NULL, NULL, 'video', 'missed');

INSERT INTO call_participants (call_id, user_id) VALUES
                                                     (1, 1),
                                                     (1, 2),
                                                     (2, 3);

-- Создание реакций на сообщения
INSERT INTO message_reactions (message_id, user_id, reaction) VALUES
                                                                  (1, 2, '❤️'),
                                                                  (8, 2, '👍'),
                                                                  (8, 3, '🔥');