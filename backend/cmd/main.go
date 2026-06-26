package main

import (
	integrationAi "MessangerMax/internal/ai"
	"MessangerMax/internal/config"
	entities "MessangerMax/internal/entity"
	httpHandlers "MessangerMax/internal/handlers"
	messengerLogic "MessangerMax/internal/logic"
	"MessangerMax/internal/middleware"
	repository "MessangerMax/internal/repo"
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
		&entities.User{},
		&entities.Chat{},
		&entities.Message{},
		&entities.ChatUser{},
		&entities.MessageReaction{},
		&entities.Music{},
	); err != nil {
		log.Fatal("Ошибка миграции базы данных:", err)
	}
	log.Println("Миграции базы данных выполнены успешно")

	// Сброс статусов при старте — источник правды теперь in-memory (clients map)
	db.Model(&entities.User{}).Where("status = ?", "online").Update("status", "offline")

	userRepo := repository.NewUserRepository(db)
	chatRepo := repository.NewChatRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	reactionRepo := repository.NewReactionRepository(db)
	musicRepo := repository.NewMusicRepository(db)

	// Создаём AI-ассистента и администратора, если их нет
	seedAIBot(userRepo)
	seedAdmin(userRepo)

	jwtUtils := utils.NewJWTUtils(cfg.JWTSecret)

	wsNotifier := httpHandlers.NewWSNotifier(chatRepo)

	emailService := messengerLogic.NewEmailService(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPFrom, cfg.AppURL)

	geminiClient := integrationAi.NewGeminiClient(cfg.GeminiAPIKey)

	authService := messengerLogic.NewAuthService(userRepo, jwtUtils, emailService)

	userService := messengerLogic.NewUserService(userRepo, chatRepo, wsOnlineTracker{})
	chatService := messengerLogic.NewChatService(chatRepo, messageRepo, userRepo, wsNotifier, geminiClient)
	reactionService := messengerLogic.NewReactionService(reactionRepo, messageRepo, wsNotifier)
	musicService := messengerLogic.NewMusicService(musicRepo, cfg.MusicDir)

	authHandler := httpHandlers.NewAuthHandler(authService)
	userHandler := httpHandlers.NewUserHandler(userService, authService)
	chatHandler := httpHandlers.NewChatHandler(chatService)
	wsHandler := httpHandlers.NewWSHandler(jwtUtils)
	uploadHandler := httpHandlers.NewUploadHandler(cfg.UploadDir)
	reactionHandler := httpHandlers.NewReactionHandler(reactionService)
	musicHandler := httpHandlers.NewMusicHandler(musicService)

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
			protected.PUT("/chats/:id/pin/:msgId", chatHandler.PinMessage)
			protected.DELETE("/chats/:id/pin", chatHandler.UnpinMessage)

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

			protected.GET("/music", musicHandler.List)
			protected.POST("/music/upload", musicHandler.Upload)
			protected.GET("/music/:id/stream", musicHandler.Stream)
			protected.GET("/music/:id/download", musicHandler.Download)
			protected.DELETE("/music/:id", musicHandler.Delete)

			admin := protected.Group("/admin")
			admin.Use(middleware.AdminMiddleware(userRepo))
			{
				admin.GET("/music/pending", musicHandler.ListPending)
				admin.PUT("/music/:id/approve", musicHandler.Approve)
				admin.PUT("/music/:id/reject", musicHandler.Reject)
			}
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

type wsOnlineTracker struct{}

func (wsOnlineTracker) IsOnline(userID uint) bool {
	return httpHandlers.IsUserOnline(userID)
}

func seedAIBot(userRepo repository.UserRepository) {
	_, err := userRepo.FindByUsername("Ассистент")
	if err == nil {
		return
	}

	bot := &entities.User{
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

func seedAdmin(userRepo repository.UserRepository) {
	_, err := userRepo.FindByUsername("Admin")
	if err == nil {
		return
	}

	admin := &entities.User{
		Username: "Admin",
		Email:    "admin@messengermax.local",
		Password: "",
		IsActive: true,
		IsAdmin:  true,
	}
	admin.HashPassword("07062002")
	if err := userRepo.Create(admin); err != nil {
		log.Printf("Ошибка создания администратора: %v", err)
		return
	}
	log.Println("Администратор создан")
}
