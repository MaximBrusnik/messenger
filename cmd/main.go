package main

import (
	"MessangerMax/internal/config"
	"MessangerMax/internal/entity"
	"MessangerMax/internal/handlers"
	"MessangerMax/internal/logic"
	"MessangerMax/internal/middleware"
	"MessangerMax/internal/repo"
	db "MessangerMax/pkg"
	"MessangerMax/utils"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"log"
)

func main() {
	// Загрузка .env файла
	if err := godotenv.Load(); err != nil {
		log.Println("Файл .env не найден, используются переменные окружения")
	}

	// Загрузка конфигурации
	cfg := config.Load()

	// Инициализация базы данных
	db, err := db.NewDB(cfg)
	if err != nil {
		log.Fatal("Ошибка подключения к базе данных:", err)
	}

	// Автомиграция
	if err := db.AutoMigrate(
		&entity.User{},
		&entity.Chat{},
		&entity.Message{},
		&entity.ChatUser{},
	); err != nil {
		log.Fatal("Ошибка миграции базы данных:", err)
	}
	log.Println("Миграции базы данных выполнены успешно")

	// Инициализация зависимостей
	userRepo := repo.NewUserRepository(db)
	chatRepo := repo.NewChatRepository(db)
	messageRepo := repo.NewMessageRepository(db)

	jwtUtils := utils.NewJWTUtils(cfg.JWTSecret)

	authService := logic.NewAuthService(userRepo, jwtUtils)
	userService := logic.NewUserService(userRepo)
	chatService := logic.NewChatService(chatRepo, messageRepo, userRepo)

	authHandler := handlers.NewAuthHandler(authService)
	userHandler := handlers.NewUserHandler(userService, authService)
	chatHandler := handlers.NewChatHandler(chatService)
	wsHandler := handlers.NewWSHandler(jwtUtils)

	// Настройка роутера
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// Middleware
	router.Use(middleware.CORSMiddleware())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
			"env":    cfg.Env,
		})
	})

	// API v1
	v1 := router.Group("/api/v1")
	{
		// Публичные маршруты
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
		}

		// Защищенные маршруты
		protected := v1.Group("/")
		protected.Use(middleware.AuthMiddleware(jwtUtils))
		{
			// Авторизация
			protected.GET("/auth/profile", authHandler.GetProfile)
			protected.POST("/auth/logout", authHandler.Logout)
			protected.PUT("/auth/profile", userHandler.UpdateProfile)
			protected.POST("/auth/change-password", userHandler.ChangePassword)

			// Чаты
			protected.GET("/chats", chatHandler.GetChats)
			protected.POST("/chats", chatHandler.CreateChat)
			protected.GET("/chats/:id", chatHandler.GetChat)
			protected.GET("/chats/:id/messages", chatHandler.GetMessages)
			protected.POST("/chats/:id/messages", chatHandler.SendMessage)
			protected.POST("/chats/:id/read", chatHandler.MarkAsRead)

			// Пользователи
			protected.GET("/users", userHandler.GetAllUsers)
			protected.GET("/users/search", userHandler.SearchUsers)
			protected.GET("/contacts", userHandler.GetContacts)
			protected.POST("/contacts", userHandler.AddContact)
		}
	}

	// WebSocket
	router.GET("/ws", wsHandler.HandleWebSocket)

	// Запуск сервера
	port := ":" + cfg.ServerPort
	log.Printf("Сервер запущен на порту %s в режиме %s", port, cfg.Env)

	if err := router.Run(port); err != nil {
		log.Fatal("Ошибка запуска сервера:", err)
	}
}
