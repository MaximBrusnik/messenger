package main

import (
	"context"
	"log"

	"google.golang.org/grpc"

	"messengermax/chat/internal/entity"
	"messengermax/chat/internal/repo"
	"messengermax/chat/internal/service"
	"messengermax/pkg/config"
	"messengermax/pkg/grpcsrv"
	"messengermax/pkg/nats"
	"messengermax/pkg/postgres"
	pbchat "messengermax/proto/gen/chat"
	pbuser "messengermax/proto/gen/user"
)

func main() {
	cfg := config.Load()
	cfg.ServiceName = "chat"

	db, err := postgres.New(cfg.Postgres)
	if err != nil {
		log.Fatal("chat: ", err)
	}
	if err := db.AutoMigrate(&entity.Chat{}, &entity.ChatUser{}, &entity.Message{}, &entity.MessageReaction{}); err != nil {
		log.Fatal("chat: migrate: ", err)
	}

	chatRepo := repo.NewChatRepository(db)
	messageRepo := repo.NewMessageRepository(db)
	reactionRepo := repo.NewReactionRepository(db)
	producer := nats.NewProducer(cfg.NATS.URL)
	defer producer.Close()

	botID := resolveBotID(cfg)
	server := service.NewServer(chatRepo, messageRepo, reactionRepo, producer, botID)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Write call-ended system messages into chats.
	consumer := nats.NewConsumer(cfg.NATS.URL, "chat-service", []string{nats.TopicCallEnded}, func(topic, key string, value []byte) {
		server.HandleCallEnded(value)
	})
	go consumer.Run(ctx)

	if err := grpcsrv.Run(cfg.GRPCPort, func(s *grpc.Server) {
		pbchat.RegisterChatServiceServer(s, server)
	}); err != nil {
		log.Fatal("chat: ", err)
	}
}

// resolveBotID looks up the AI assistant user id from the user-service.
// If user-service is unreachable or the bot is missing, chat runs without
// AI support rather than failing to boot.
func resolveBotID(cfg *config.Config) uint {
	conn, err := grpcsrv.Dial(cfg.Services.UserAddr)
	if err != nil {
		log.Printf("chat: user-service unreachable, AI disabled: %v", err)
		return 0
	}
	defer conn.Close()
	client := pbuser.NewUserServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 3_000_000_000)
	defer cancel()
	resp, err := client.ResolveUserByName(ctx, &pbuser.ResolveUserByNameRequest{Username: "Ассистент"})
	if err != nil || resp.GetFound() == false {
		log.Printf("chat: AI assistant not found, AI disabled")
		return 0
	}
	log.Printf("chat: AI assistant id=%d", resp.GetUserId())
	return uint(resp.GetUserId())
}
