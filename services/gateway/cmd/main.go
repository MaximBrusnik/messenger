package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"messengermax/gateway/internal/api"
	"messengermax/gateway/internal/middleware"
	"messengermax/pkg/config"
	"messengermax/pkg/jwt"
)

func main() {
	cfg := config.Load()
	cfg.ServiceName = "gateway"

	gw, err := api.NewGateway(cfg)
	if err != nil {
		log.Fatal("gateway: dial services: ", err)
	}

	authMW := middleware.NewAuth(jwt.NewManager(cfg.JWTSecret))

	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()
	router.Use(gin.Recovery())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "env": cfg.Env})
	})

	router.Static("/uploads", cfg.FileStorageDir)
	router.Static("/assets", "./dist/assets")

	v1 := router.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		auth.POST("/register", gw.Register)
		auth.POST("/login", gw.Login)
		auth.GET("/verify-email", gw.VerifyEmail)

		protected := v1.Group("/")
		protected.Use(authMW.Required)
		{
			protected.GET("/auth/profile", gw.GetProfile)
			protected.POST("/auth/logout", gw.Logout)
			protected.PUT("/auth/profile", gw.UpdateProfile)
			protected.POST("/auth/change-password", gw.ChangePassword)
			protected.POST("/auth/resend-verification", gw.ResendVerification)
			protected.GET("/auth/settings", gw.GetSettings)
			protected.PUT("/auth/settings", gw.UpdateSettings)

			protected.GET("/chats", gw.GetChats)
			protected.POST("/chats", gw.CreateChat)
			protected.GET("/chats/:id", gw.GetChat)
			protected.GET("/chats/:id/messages", gw.GetMessages)
			protected.POST("/chats/:id/messages", gw.SendMessage)
			protected.PUT("/chats/:id/messages/:msgId", gw.EditMessage)
			protected.DELETE("/chats/:id/messages/:msgId", gw.DeleteMessage)
			protected.POST("/chats/:id/read", gw.MarkAsRead)
			protected.PUT("/chats/:id/pin/:msgId", gw.PinMessage)
			protected.DELETE("/chats/:id/pin", gw.UnpinMessage)
			protected.DELETE("/chats/:id", gw.DeleteChat)
			protected.GET("/ai/chat", gw.GetOrCreateAIChat)

			protected.GET("/users", gw.GetAllUsers)
			protected.GET("/users/search", gw.SearchUsers)
			protected.GET("/users/:id", gw.GetUser)
			protected.GET("/contacts", gw.GetContacts)
			protected.POST("/contacts", gw.AddContact)

			protected.POST("/devices/register", gw.RegisterDevice)
			protected.DELETE("/devices/unregister", gw.UnregisterDevice)

			protected.POST("/upload", gw.Upload)

			protected.GET("/chats/:id/messages/:msgId/reactions", gw.GetReactions)
			protected.POST("/chats/:id/messages/:msgId/reactions", gw.AddReaction)
			protected.DELETE("/chats/:id/messages/:msgId/reactions", gw.RemoveReaction)

			protected.Any("/music/upload", gw.MusicFileProxy)
			protected.Any("/music/:id/stream", gw.MusicFileProxy)
			protected.Any("/music/:id/download", gw.MusicFileProxy)
			protected.GET("/music", gw.MusicList)
			protected.DELETE("/music/:id", gw.MusicDelete)

			admin := protected.Group("/admin")
			admin.Use(func(c *gin.Context) {
				if !gw.IsAdmin(c) {
					c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "нет прав администратора"})
					return
				}
				c.Next()
			})
			{
				admin.GET("/music/pending", gw.MusicPending)
				admin.PUT("/music/:id/approve", gw.MusicApprove)
				admin.PUT("/music/:id/reject", gw.MusicReject)
			}
		}
	}

	router.Any("/ws", gw.WSProxy)
	router.NoRoute(func(c *gin.Context) {
		c.File("./dist/index.html")
	})

	port := ":" + cfg.ServerPort
	log.Printf("API-gateway на порту %s (%s)", port, cfg.Env)
	if err := router.Run(port); err != nil {
		log.Fatal("gateway: ", err)
	}
}
