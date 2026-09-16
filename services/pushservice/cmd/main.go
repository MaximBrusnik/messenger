package main

import (
	"context"
	"encoding/json"
	"log"

	"google.golang.org/grpc"

	"messengermax/pkg/config"
	"messengermax/pkg/grpcsrv"
	"messengermax/pkg/nats"
	"messengermax/pkg/postgres"
	sharedredis "messengermax/pkg/redis"
	pb "messengermax/proto/gen/push"

	"messengermax/pushservice/internal/entity"
	handlergrpc "messengermax/pushservice/internal/handler/grpc"
	"messengermax/pushservice/internal/integration/fcm"
	"messengermax/pushservice/internal/repo"
	"messengermax/pushservice/internal/service"
)

func main() {
	cfg := config.Load()
	cfg.ServiceName = "pushservice"

	db, err := postgres.New(cfg.Postgres)
	if err != nil {
		log.Fatal("push: ", err)
	}
	if err := db.AutoMigrate(&entity.DeviceToken{}); err != nil {
		log.Fatal("push: migrate: ", err)
	}

	deviceRepo := repo.NewDeviceTokenRepository(db)
	redisClient := sharedredis.New(cfg.Redis.Addr, cfg.Redis.Pass)
	sender := fcm.New(cfg.PushConfig.FCMCredentials, deviceRepo)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handle := func(topic string, key string, value []byte) {
		handleMessageEvent(ctx, redisClient, sender, topic, value)
	}
	consumer := nats.NewConsumer(cfg.NATS.URL, "pushservice", []string{
		nats.TopicMessageCreated,
		nats.TopicMessageEdited,
	}, handle)
	go consumer.Run(ctx)

	if err := grpcsrv.Run(cfg.GRPCPort, func(s *grpc.Server) {
		pb.RegisterPushServiceServer(s, handlergrpc.NewServer(service.NewServer(deviceRepo)))
	}); err != nil {
		log.Fatal("push: ", err)
	}
}

func handleMessageEvent(ctx context.Context, redisClient *sharedredis.Client, sender *fcm.Sender, topic string, value []byte) {
	switch topic {
	case nats.TopicMessageCreated:
		var e nats.EventMessageCreated
		if err := json.Unmarshal(value, &e); err != nil {
			return
		}
		// do not push for system messages or bot replies
		if e.Message.SystemType != "" {
			return
		}
		for _, pid := range e.Participants {
			if pid == e.Message.SenderID {
				continue
			}
			if !redisClient.IsOnline(ctx, uint(pid)) {
				sender.SendPush(ctx, uint(pid), e.ChatID, "Новое сообщение", truncate(e.Message.Text, 80))
			}
		}
	case nats.TopicMessageEdited:
		var e nats.EventMessageEdited
		if err := json.Unmarshal(value, &e); err != nil {
			return
		}
	}
}

func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "…"
}
