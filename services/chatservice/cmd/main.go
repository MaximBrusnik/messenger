package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"

	"messengermax/chatservice/internal/entity"
	handlergrpc "messengermax/chatservice/internal/handler/grpc"
	"messengermax/chatservice/internal/repo"
	"messengermax/chatservice/internal/service"
	"messengermax/pkg/config"
	"messengermax/pkg/grpcsrv"
	"messengermax/pkg/nats"
	"messengermax/pkg/postgres"
	pbchat "messengermax/proto/gen/chat"
	pbuser "messengermax/proto/gen/user"
)

func main() {
	cfg := config.Load()
	cfg.ServiceName = "chatservice"

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
	server := service.NewServer(chatRepo, messageRepo, reactionRepo, producer, botID, func(ctx context.Context) (uint, error) {
		return resolveBotIDOnce(ctx, cfg)
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	consumer := nats.NewConsumer(cfg.NATS.URL, "chatservice", []string{nats.TopicCallEnded}, func(topic, key string, value []byte) {
		server.HandleCallEnded(value)
	})
	go consumer.Run(ctx)

	if err := grpcsrv.Run(cfg.GRPCPort, func(s *grpc.Server) {
		pbchat.RegisterChatServiceServer(s, handlergrpc.NewServer(server))
	}); err != nil {
		log.Fatal("chat: ", err)
	}
}

func resolveBotID(cfg *config.Config) uint {
	id, err := resolveBotIDWithRetry(cfg, 60*time.Second, 2*time.Second)
	if err != nil {
		log.Printf("chat: AI disabled: %v", err)
		return 0
	}
	log.Printf("chat: AI assistant id=%d", id)
	return id
}

func resolveBotIDOnce(ctx context.Context, cfg *config.Config) (uint, error) {
	conn, err := grpcsrv.Dial(cfg.Services.UserAddr)
	if err != nil {
		return 0, fmt.Errorf("userservice unreachable at %s: %w", cfg.Services.UserAddr, err)
	}
	defer conn.Close()
	client := pbuser.NewUserServiceClient(conn)
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	resp, err := client.ResolveUserByName(ctx, &pbuser.ResolveUserByNameRequest{Username: service.BotUsername})
	if err != nil {
		return 0, err
	}
	if !resp.GetFound() {
		return 0, errors.New("AI assistant not found")
	}
	return uint(resp.GetUserId()), nil
}

func resolveBotIDWithRetry(cfg *config.Config, timeout, interval time.Duration) (uint, error) {
	deadline := time.Now().Add(timeout)
	for {
		id, err := resolveBotIDOnce(context.Background(), cfg)
		if err == nil {
			return id, nil
		}
		if time.Now().Add(interval).After(deadline) {
			return 0, err
		}
		log.Printf("chat: assistant resolve failed: %v; retrying in %s", err, interval)
		time.Sleep(interval)
	}
}
