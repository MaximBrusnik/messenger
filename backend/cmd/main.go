package main

import (
	ai2 "MessangerMax/internal/ai"
	"MessangerMax/internal/config"
	entity2 "MessangerMax/internal/entity"
	handlers2 "MessangerMax/internal/handlers"
	logic2 "MessangerMax/internal/logic"
	"MessangerMax/internal/middleware"
	repo2 "MessangerMax/internal/repo"
	db "MessangerMax/pkg"
	"MessangerMax/utils"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"log"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Файл .env не найден, используются переменные окружения")
	}

	cfg := config.Load()

	db, err := db.NewDB(cfg)
	if err != nil {
		log.Fatal("Ошибка подключения к базе данных:", err)
	}

	if err := db.AutoMigrate(
		&entity2.User{},
		&entity2.Chat{},
		&entity2.Message{},
		&entity2.ChatUser{},
		&entity2.MessageReaction{},
	); err != nil {
		log.Fatal("Ошибка миграции базы данных:", err)
	}
	log.Println("Миграции базы данных выполнены успешно")

	userRepo := repo2.NewUserRepository(db)
	chatRepo := repo2.NewChatRepository(db)
	messageRepo := repo2.NewMessageRepository(db)
	reactionRepo := repo2.NewReactionRepository(db)

	// Создаём AI-ассистента, если его нет
	seedAIBot(userRepo)

	jwtUtils := utils.NewJWTUtils(cfg.JWTSecret)

	wsNotifier := handlers2.NewWSNotifier(chatRepo)

	emailService := logic2.NewEmailService(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPFrom, cfg.AppURL)

	geminiClient := ai2.NewGeminiClient(cfg.GeminiAPIKey)

	authService := logic2.NewAuthService(userRepo, jwtUtils, emailService)
	userService := logic2.NewUserService(userRepo, chatRepo)
	chatService := logic2.NewChatService(chatRepo, messageRepo, userRepo, wsNotifier, geminiClient)
	reactionService := logic2.NewReactionService(reactionRepo, messageRepo, wsNotifier)

	authHandler := handlers2.NewAuthHandler(authService)
	userHandler := handlers2.NewUserHandler(userService, authService)
	chatHandler := handlers2.NewChatHandler(chatService)
	wsHandler := handlers2.NewWSHandler(jwtUtils, userRepo)
	uploadHandler := handlers2.NewUploadHandler(cfg.UploadDir)
	reactionHandler := handlers2.NewReactionHandler(reactionService)

	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()
	router.Use(middleware.CORSMiddleware())

	// Статика для загруженных файлов
	router.Static("/uploads", "./"+cfg.UploadDir)

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
			"env":    cfg.Env,
		})
	})

	// Статика для фронтенда
	router.Static("/assets", "./dist/assets")

	v1 := router.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.GET("/verify-email", authHandler.VerifyEmail)
		}

		protected := v1.Group("/")
		protected.Use(middleware.AuthMiddleware(jwtUtils))
		{
			protected.GET("/auth/profile", authHandler.GetProfile)
			protected.POST("/auth/logout", authHandler.Logout)
			protected.PUT("/auth/profile", userHandler.UpdateProfile)
			protected.POST("/auth/change-password", userHandler.ChangePassword)
			protected.POST("/auth/resend-verification", authHandler.ResendVerification)

			protected.GET("/auth/settings", userHandler.GetSettings)
			protected.PUT("/auth/settings", userHandler.UpdateSettings)

			protected.GET("/chats", chatHandler.GetChats)
			protected.POST("/chats", chatHandler.CreateChat)
			protected.GET("/chats/:id", chatHandler.GetChat)
			protected.GET("/chats/:id/messages", chatHandler.GetMessages)
			protected.POST("/chats/:id/messages", chatHandler.SendMessage)
			protected.PUT("/chats/:id/messages/:msgId", chatHandler.EditMessage)
			protected.DELETE("/chats/:id/messages/:msgId", chatHandler.DeleteMessage)
			protected.POST("/chats/:id/read", chatHandler.MarkAsRead)
			protected.DELETE("/chats/:id", chatHandler.DeleteChat)

			protected.GET("/users", userHandler.GetAllUsers)
			protected.GET("/users/search", userHandler.SearchUsers)
			protected.GET("/users/:id", userHandler.GetUser)
			protected.GET("/contacts", userHandler.GetContacts)
			protected.POST("/contacts", userHandler.AddContact)

			protected.POST("/upload", uploadHandler.Upload)

			protected.GET("/chats/:id/messages/:msgId/reactions", reactionHandler.GetReactions)
			protected.POST("/chats/:id/messages/:msgId/reactions", reactionHandler.AddReaction)
			protected.DELETE("/chats/:id/messages/:msgId/reactions", reactionHandler.RemoveReaction)

			protected.GET("/ai/chat", func(c *gin.Context) {
				userID := c.GetUint("user_id")
				chat, err := chatService.GetOrCreateAIChat(userID)
				if err != nil {
					c.JSON(500, gin.H{"error": err.Error()})
					return
				}
				c.JSON(200, gin.H{"data": chat})
			})
		}
	}

	router.GET("/ws", wsHandler.HandleWebSocket)

	// SPA fallback
	router.NoRoute(func(c *gin.Context) {
		c.File("./dist/index.html")
	})

	port := ":" + cfg.ServerPort
	log.Printf("Сервер запущен на порту %s в режиме %s", port, cfg.Env)

	if err := router.Run(port); err != nil {
		log.Fatal("Ошибка запуска сервера:", err)
	}
}

func seedAIBot(userRepo repo2.UserRepository) {
	_, err := userRepo.FindByUsername("Ассистент")
	if err == nil {
		return
	}

	bot := &entity2.User{
		Username: "Ассистент",
		Email:    "ai@messengermax.local",
		Password: "",
		IsActive: true,
		IsBot:    true,
	}
	bot.HashPassword("none")
	if err := userRepo.Create(bot); err != nil {
		log.Printf("Ошибка создания AI-ассистента: %v", err)
		return
	}
	log.Println("AI-ассистент создан")
}
