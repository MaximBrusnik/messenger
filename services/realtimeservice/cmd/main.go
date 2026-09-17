package main

import (
	"context"
	"log"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"messengermax/pkg/config"
	"messengermax/pkg/grpcsrv"
	"messengermax/pkg/jwt"
	"messengermax/pkg/nats"
	sharedredis "messengermax/pkg/redis"
	pbrealtime "messengermax/proto/gen/realtime"
	pbuser "messengermax/proto/gen/user"

	"messengermax/realtimeservice/internal/consumer"
	handlergrpc "messengermax/realtimeservice/internal/handler/grpc"
	"messengermax/realtimeservice/internal/handler/ws"
	"messengermax/realtimeservice/internal/service"
)

func main() {
	cfg := config.Load()
	cfg.ServiceName = "realtimeservice"

	redisClient := sharedredis.New(cfg.Redis.Addr, cfg.Redis.Pass)
	hubInstance := ws.New(redisClient)
	jwtManager := jwt.NewManager(cfg.JWTSecret)
	producer := nats.NewProducer(cfg.NATS.URL)
	defer producer.Close()

	wsCtrl := ws.NewController(jwtManager, hubInstance, redisClient, producer)
	biz := service.NewServer(redisClient, wsCtrl)

	grpcErr := make(chan error, 1)
	go func() {
		grpcErr <- grpcsrv.Run(cfg.GRPCPort, func(s *grpc.Server) {
			pbrealtime.RegisterRealtimeServiceServer(s, handlergrpc.NewServer(biz))
		})
	}()

	var userClient pbuser.UserServiceClient
	userConn, err := grpcsrv.Dial(cfg.Services.UserAddr)
	if err != nil {
		log.Printf("realtime: userservice unreachable at %s: %v", cfg.Services.UserAddr, err)
	} else {
		defer userConn.Close()
		userClient = pbuser.NewUserServiceClient(userConn)
	}

	// Kafka
	cons := consumer.New(hubInstance, redisClient, userClient)
	handle := func(topic, key string, value []byte) { cons.Handle(topic, key, value) }
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c1 := nats.NewConsumer(cfg.NATS.URL, "realtimeservice-chat", []string{
		nats.TopicMessageCreated,
		nats.TopicMessageEdited,
		nats.TopicMessageDeleted,
		nats.TopicMessagePinned,
		nats.TopicMessageUnpinned,
		nats.TopicReactionAdded,
		nats.TopicReactionRemoved,
		nats.TopicReadReceived,
		nats.TopicChatDeleted,
	}, handle)
	c2 := nats.NewConsumer(cfg.NATS.URL, "realtimeservice-users", []string{nats.TopicUserEvents}, func(t, k string, v []byte) {
		cons.HandleUserEvent(ctx, v)
	})
	go c1.Run(ctx)
	go c2.Run(ctx)

	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.Default()
	router.GET("/ws", wsCtrl.Handle)
	router.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })

	httpErr := make(chan error, 1)
	go func() {
		httpErr <- router.Run(":" + cfg.HTTPPort)
	}()

	log.Printf("realtime: ws on :%s, grpc on :%s", cfg.HTTPPort, cfg.GRPCPort)

	select {
	case err := <-grpcErr:
		log.Fatal("realtime: grpc: ", err)
	case err := <-httpErr:
		log.Fatal("realtime: http: ", err)
	}
}
